// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package rlp

import (
	"errors"
	"fmt"
	"io"
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum/rlp/internal/rlpstruct"
	"github.com/holiman/uint256"
)

var (
	// 흔히 사용되는 인코딩된 값들입니다.
	// EncodeRLP를 구현할 때 유용합니다.

	// EmptyString은 빈 문자열의 인코딩입니다.
	EmptyString = []byte{0x80}
	// EmptyList는 빈 리스트의 인코딩입니다.
	EmptyList = []byte{0xC0}
)

var ErrNegativeBigInt = errors.New("rlp: cannot encode negative big.Int")

// Encoder는 사용자 정의 인코딩 규칙이 필요한 타입이나
// private 필드를 인코딩하고 싶은 타입에 의해 구현됩니다.
type Encoder interface {
	// EncodeRLP는 리시버의 RLP 인코딩을 io.Writer에 씁니다.
	// 포인터 메서드로 구현된 경우 nil 포인터에 대해서도 호출될 수 있습니다.
	//
	// 구현체는 유효한 RLP를 생성해야 합니다. io.Writer에 쓰인 데이터는
	// 현시점에서 검증되지는 않지만, 향후 버전에서는 검증될 수 있습니다.
	// 하나의 값만 쓰는 것을 권장하지만, 그렇지 않은 경우도 허용됩니다.
	EncodeRLP(io.Writer) error
}

// Encode는 val의 RLP 인코딩을 w에 씁니다. Encode는 경우에 따라
// 많은 작은 쓰기 작업을 수행할 수 있습니다. w를 버퍼링하는 것을 고려하세요.
//
// 인코딩 규칙에 대한 패키지 수준의 문서를 참조하세요.
func Encode(w io.Writer, val interface{}) error {
	// 최적화: EncodeRLP에 의해 호출될 때 *encBuffer를 재사용합니다.
	if buf := encBufferFromWriter(w); buf != nil { // w가 *encBuffer를 구현하는 경우
		return buf.encode(val)
	}

	buf := getEncBuffer()                   // pool에서 *encBuffer를 가져옵니다.
	defer encBufferPool.Put(buf)            // *encBuffer를 pool에 반환합니다.
	if err := buf.encode(val); err != nil { // 인코딩을 수행합니다.
		return err
	}
	return buf.writeTo(w) // 인코딩된 데이터를 w에 씁니다.
}

// EncodeToBytes는 val의 RLP 인코딩을 반환합니다.
// 인코딩 규칙에 대한 패키지 수준의 문서를 참조하세요.
func EncodeToBytes(val interface{}) ([]byte, error) {
	buf := getEncBuffer()
	defer encBufferPool.Put(buf)

	if err := buf.encode(val); err != nil {
		return nil, err
	}
	return buf.makeBytes(), nil // 인코딩된 데이터를 반환합니다.
}

// EncodeToReader는 val의 RLP 인코딩을 읽을 수 있는 리더를 반환합니다.
// 반환된 size는 인코딩된 데이터의 총 크기입니다.
//
// 인코딩 규칙에 대한 Encode의 문서를 참조하세요.
func EncodeToReader(val interface{}) (size int, r io.Reader, err error) {
	buf := getEncBuffer()
	if err := buf.encode(val); err != nil {
		encBufferPool.Put(buf)
		return 0, nil, err
	}
	// 참고: 여기서 buf를 pool에 반환할 수 없습니다.
	// 왜냐하면 encReader가 buf를 보유하고 있기 때문입니다.
	// 리더가 완전히 소비되었을 때 리더가 buf를 반환합니다.
	return buf.size(), &encReader{buf: buf}, nil
}

type listhead struct {
	offset int // 문자열 데이터에서 이 헤더의 오프셋
	size   int // 인코딩된 데이터의 총 크기(리스트 헤더 포함)
}

// encode는 주어진 버퍼에 head를 씁니다. 버퍼는 적어도 9바이트여야 합니다.
// 인코딩된 바이트를 반환합니다.
func (head *listhead) encode(buf []byte) []byte {
	return buf[:puthead(buf, 0xC0, 0xF7, uint64(head.size))] // 리스트 헤더를 쓰고 헤더의 크기만큼 버퍼를 반환합니다.
}

// headsize는 주어진 크기의 값에 대한 리스트나 문자열 헤더의 크기를 반환합니다.
func headsize(size uint64) int {
	if size < 56 {
		return 1
	}
	return 1 + intsize(size) // 1 + size의 바이트 수
}

// puthead는 buf에 리스트나 문자열 헤더를 씁니다.
// buf는 적어도 9바이트여야 합니다.
func puthead(buf []byte, smalltag, largetag byte, size uint64) int {
	if size < 56 {
		buf[0] = smalltag + byte(size)
		return 1
	}
	sizesize := putint(buf[1:], size) // size를 1번 인덱스부터 씁니다.
	buf[0] = largetag + byte(sizesize)
	return sizesize + 1
}

var encoderInterface = reflect.TypeOf(new(Encoder)).Elem()

// makeWriter는 주어진 타입에 대한 writer 함수를 생성합니다.
func makeWriter(typ reflect.Type, ts rlpstruct.Tags) (writer, error) {
	kind := typ.Kind()
	switch {
	// 특별한 타입들
	case typ == rawValueType: // []byte의 별칭 타입 (rawValue)
		return writeRawValue, nil
	case typ.AssignableTo(reflect.PtrTo(bigInt)): // *big.Int
		return writeBigIntPtr, nil
	case typ.AssignableTo(bigInt): // big.Int
		return writeBigIntNoPtr, nil
	case typ == reflect.PtrTo(u256Int): // *uint256.Int
		return writeU256IntPtr, nil
	case typ == u256Int: // uint256.Int
		return writeU256IntNoPtr, nil
	// 그 외의 타입들
	case kind == reflect.Ptr: // 포인터 타입
		return makePtrWriter(typ, ts)
	case reflect.PtrTo(typ).Implements(encoderInterface): // Encoder 인터페이스를 구현하는 포인터 타입
		return makeEncoderWriter(typ), nil
	case isUint(kind): // 부호 없는 정수 타입
		return writeUint, nil
	case kind == reflect.Bool: // 부울 타입
		return writeBool, nil
	case kind == reflect.String: // 문자열 타입
		return writeString, nil
	case kind == reflect.Slice && isByte(typ.Elem()): // []byte 타입
		return writeBytes, nil
	case kind == reflect.Array && isByte(typ.Elem()): // [N]byte 타입 (배열)
		return makeByteArrayWriter(typ), nil
	case kind == reflect.Slice || kind == reflect.Array: // byte 슬라이스나 배열이 아닌 슬라이스나 배열
		return makeSliceWriter(typ, ts)
	case kind == reflect.Struct: // 구조체
		return makeStructWriter(typ)
	case kind == reflect.Interface: // 인터페이스
		return writeInterface, nil
	default:
		return nil, fmt.Errorf("rlp: type %v is not RLP-serializable", typ) // 그 외는 직렬화할 수 없는 타입
	}
}

func writeRawValue(val reflect.Value, w *encBuffer) error {
	w.str = append(w.str, val.Bytes()...) // rawValue는 헤더가 미리 계산되어 있습니다. 그대로 str에 이어붙입니다.
	return nil
}

func writeUint(val reflect.Value, w *encBuffer) error {
	w.writeUint64(val.Uint())
	return nil
}

func writeBool(val reflect.Value, w *encBuffer) error {
	w.writeBool(val.Bool())
	return nil
}

func writeBigIntPtr(val reflect.Value, w *encBuffer) error {
	ptr := val.Interface().(*big.Int)
	if ptr == nil {
		w.str = append(w.str, 0x80) // 빈 문자열 헤더를 씁니다.
		return nil
	}
	if ptr.Sign() == -1 { // 음수인 경우 에러를 반환합니다.
		return ErrNegativeBigInt
	}
	w.writeBigInt(ptr)
	return nil
}

func writeBigIntNoPtr(val reflect.Value, w *encBuffer) error {
	i := val.Interface().(big.Int)
	if i.Sign() == -1 { // 음수인 경우 에러를 반환합니다.
		return ErrNegativeBigInt
	}
	w.writeBigInt(&i)
	return nil
}

func writeU256IntPtr(val reflect.Value, w *encBuffer) error {
	ptr := val.Interface().(*uint256.Int)
	if ptr == nil {
		w.str = append(w.str, 0x80) // 빈 문자열 헤더를 씁니다.
		return nil
	}
	w.writeUint256(ptr)
	return nil
}

func writeU256IntNoPtr(val reflect.Value, w *encBuffer) error {
	i := val.Interface().(uint256.Int)
	w.writeUint256(&i)
	return nil
}

func writeBytes(val reflect.Value, w *encBuffer) error {
	w.writeBytes(val.Bytes()) // 바이트 슬라이스를 그대로 씁니다.
	return nil
}

func makeByteArrayWriter(typ reflect.Type) writer {
	switch typ.Len() {
	case 0:
		return writeLengthZeroByteArray
	case 1:
		return writeLengthOneByteArray
	default:
		length := typ.Len()
		return func(val reflect.Value, w *encBuffer) error {
			if !val.CanAddr() {
				// Getting the byte slice of val requires it to be addressable. Make it
				// addressable by copying.
				copy := reflect.New(val.Type()).Elem()
				copy.Set(val)
				val = copy
			}
			slice := byteArrayBytes(val, length)
			w.encodeStringHeader(len(slice))
			w.str = append(w.str, slice...)
			return nil
		}
	}
}

func writeLengthZeroByteArray(val reflect.Value, w *encBuffer) error {
	w.str = append(w.str, 0x80) // 빈 문자열 헤더를 씁니다.
	return nil
}

func writeLengthOneByteArray(val reflect.Value, w *encBuffer) error {
	b := byte(val.Index(0).Uint())
	if b <= 0x7f {
		w.str = append(w.str, b) // 0x00 ~ 0x7f 사이의 값은 헤더가 필요 없습니다.
	} else {
		w.str = append(w.str, 0x81, b) // 0x80 이상의 값은 헤더가 필요합니다.
	}
	return nil
}

func writeString(val reflect.Value, w *encBuffer) error {
	s := val.String()
	if len(s) == 1 && s[0] <= 0x7f {
		// 0x00 ~ 0x7f 사이의 값은 헤더가 필요 없습니다.
		w.str = append(w.str, s[0])
	} else {
		w.encodeStringHeader(len(s)) // 헤더를 씁니다.
		w.str = append(w.str, s...)  // 문자열을 씁니다.
	}
	return nil
}

func writeInterface(val reflect.Value, w *encBuffer) error {
	if val.IsNil() {
		// 빈 리스트를 씁니다. 이는 이전 RLP 인코더와 일관성이 있으며
		// 따라서 어떤 문제도 발생하지 않아야 합니다.
		w.str = append(w.str, 0xC0)
		return nil
	}
	eval := val.Elem()
	writer, err := cachedWriter(eval.Type())
	if err != nil {
		return err
	}
	return writer(eval, w)
}

func makeSliceWriter(typ reflect.Type, ts rlpstruct.Tags) (writer, error) {
	etypeinfo := theTC.infoWhileGenerating(typ.Elem(), rlpstruct.Tags{})
	if etypeinfo.writerErr != nil {
		return nil, etypeinfo.writerErr
	}

	var wfn writer
	if ts.Tail {
		// 구조체의 tail 슬라이스에 대한 writer입니다.
		// w.list는 호출되지 않습니다.
		wfn = func(val reflect.Value, w *encBuffer) error {
			vlen := val.Len()
			for i := 0; i < vlen; i++ {
				if err := etypeinfo.writer(val.Index(i), w); err != nil {
					return err
				}
			}
			return nil
		}
	} else {
		// 일반적인 슬라이스와 배열에 대한 writer입니다.
		wfn = func(val reflect.Value, w *encBuffer) error {
			vlen := val.Len()
			if vlen == 0 {
				w.str = append(w.str, 0xC0)
				return nil
			}
			listOffset := w.list()
			for i := 0; i < vlen; i++ {
				if err := etypeinfo.writer(val.Index(i), w); err != nil {
					return err
				}
			}
			w.listEnd(listOffset)
			return nil
		}
	}
	return wfn, nil
}

func makeStructWriter(typ reflect.Type) (writer, error) {
	fields, err := structFields(typ)
	if err != nil {
		return nil, err
	}
	for _, f := range fields {
		if f.info.writerErr != nil {
			return nil, structFieldError{typ, f.index, f.info.writerErr}
		}
	}

	var writer writer
	firstOptionalField := firstOptionalField(fields)
	if firstOptionalField == len(fields) {
		// optional 필드가 없는 구조체에 대한 writer 함수입니다.
		writer = func(val reflect.Value, w *encBuffer) error {
			lh := w.list()
			for _, f := range fields {
				if err := f.info.writer(val.Field(f.index), w); err != nil {
					return err
				}
			}
			w.listEnd(lh)
			return nil
		}
	} else {
		// optional 필드가 있는 구조체에 대한 writer 함수입니다.
		// optional 필드가 있는 경우, writer는 출력 리스트의 길이를 결정하기 위해 추가적인 검사를 수행해야 합니다.
		writer = func(val reflect.Value, w *encBuffer) error {
			lastField := len(fields) - 1
			for ; lastField >= firstOptionalField; lastField-- {
				if !val.Field(fields[lastField].index).IsZero() {
					break
				}
			}
			lh := w.list()
			for i := 0; i <= lastField; i++ {
				if err := fields[i].info.writer(val.Field(fields[i].index), w); err != nil {
					return err
				}
			}
			w.listEnd(lh)
			return nil
		}
	}
	return writer, nil
}

func makePtrWriter(typ reflect.Type, ts rlpstruct.Tags) (writer, error) {
	nilEncoding := byte(0xC0)
	if typeNilKind(typ.Elem(), ts) == String {
		nilEncoding = 0x80
	}

	etypeinfo := theTC.infoWhileGenerating(typ.Elem(), rlpstruct.Tags{})
	if etypeinfo.writerErr != nil {
		return nil, etypeinfo.writerErr
	}

	writer := func(val reflect.Value, w *encBuffer) error {
		if ev := val.Elem(); ev.IsValid() {
			return etypeinfo.writer(ev, w)
		}
		w.str = append(w.str, nilEncoding)
		return nil
	}
	return writer, nil
}

func makeEncoderWriter(typ reflect.Type) writer {
	if typ.Implements(encoderInterface) {
		return func(val reflect.Value, w *encBuffer) error {
			return val.Interface().(Encoder).EncodeRLP(w)
		}
	}
	w := func(val reflect.Value, w *encBuffer) error {
		if !val.CanAddr() {
			// json 패키지는 이 경우에 MarshalJSON을 호출하지 않고 인터페이스를 구현하지 않은 것처럼
			// 값 자체를 인코딩합니다. 우리는 이를 그렇게 처리하고 싶지 않습니다.
			return fmt.Errorf("rlp: unaddressable value of type %v, EncodeRLP is pointer method", val.Type())
		}
		return val.Addr().Interface().(Encoder).EncodeRLP(w)
	}
	return w
}

// putint는 i를 b의 시작 부분에 big endian 바이트 순서로 씁니다.
// i를 표현하는 데 필요한 최소한의 바이트 수만 사용합니다.
func putint(b []byte, i uint64) (size int) {
	switch {
	case i < (1 << 8):
		b[0] = byte(i)
		return 1
	case i < (1 << 16):
		b[0] = byte(i >> 8)
		b[1] = byte(i)
		return 2
	case i < (1 << 24):
		b[0] = byte(i >> 16)
		b[1] = byte(i >> 8)
		b[2] = byte(i)
		return 3
	case i < (1 << 32):
		b[0] = byte(i >> 24)
		b[1] = byte(i >> 16)
		b[2] = byte(i >> 8)
		b[3] = byte(i)
		return 4
	case i < (1 << 40):
		b[0] = byte(i >> 32)
		b[1] = byte(i >> 24)
		b[2] = byte(i >> 16)
		b[3] = byte(i >> 8)
		b[4] = byte(i)
		return 5
	case i < (1 << 48):
		b[0] = byte(i >> 40)
		b[1] = byte(i >> 32)
		b[2] = byte(i >> 24)
		b[3] = byte(i >> 16)
		b[4] = byte(i >> 8)
		b[5] = byte(i)
		return 6
	case i < (1 << 56):
		b[0] = byte(i >> 48)
		b[1] = byte(i >> 40)
		b[2] = byte(i >> 32)
		b[3] = byte(i >> 24)
		b[4] = byte(i >> 16)
		b[5] = byte(i >> 8)
		b[6] = byte(i)
		return 7
	default:
		b[0] = byte(i >> 56)
		b[1] = byte(i >> 48)
		b[2] = byte(i >> 40)
		b[3] = byte(i >> 32)
		b[4] = byte(i >> 24)
		b[5] = byte(i >> 16)
		b[6] = byte(i >> 8)
		b[7] = byte(i)
		return 8
	}
}

// intsize는 i를 저장하는 데 필요한 최소한의 바이트 수를 계산합니다.
func intsize(i uint64) (size int) {
	for size = 1; ; size++ {
		if i >>= 8; i == 0 {
			return size
		}
	}
}
