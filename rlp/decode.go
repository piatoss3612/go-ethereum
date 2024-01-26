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
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/big"
	"reflect"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/rlp/internal/rlpstruct"
	"github.com/holiman/uint256"
)

//lint:ignore ST1012 EOL is not an error.

// EOL은 스트리밍 중 현재 리스트의 마지막에 도달했을 때 반환됩니다.
var EOL = errors.New("rlp: end of list")

var (
	ErrExpectedString   = errors.New("rlp: expected String or Byte")
	ErrExpectedList     = errors.New("rlp: expected List")
	ErrCanonInt         = errors.New("rlp: non-canonical integer format")
	ErrCanonSize        = errors.New("rlp: non-canonical size information")
	ErrElemTooLarge     = errors.New("rlp: element is larger than containing list")
	ErrValueTooLarge    = errors.New("rlp: value size exceeds available input length")
	ErrMoreThanOneValue = errors.New("rlp: input contains more than one value")

	// internal errors
	errNotInList     = errors.New("rlp: call of ListEnd outside of any list")
	errNotAtEOL      = errors.New("rlp: call of ListEnd not positioned at EOL")
	errUintOverflow  = errors.New("rlp: uint overflow")
	errNoPointer     = errors.New("rlp: interface given to Decode must be a pointer")
	errDecodeIntoNil = errors.New("rlp: pointer given to Decode must not be nil")
	errUint256Large  = errors.New("rlp: value too large for uint256")

	streamPool = sync.Pool{
		New: func() interface{} { return new(Stream) },
	}
)

// Decoder는 사용자 정의 RLP 디코딩 규칙이 필요한 유형 또는 내부(private) 필드로 디코딩해야하는 유형에 의해 구현됩니다.
//
// DecodeRLP 메서드는 주어진 Stream에서 하나의 값을 읽어야합니다. 덜 읽거나 더 읽는 것은 금지되지 않았지만 혼란을 야기할 수 있습니다.
type Decoder interface {
	DecodeRLP(*Stream) error
}

// Decode는 r에서 RLP로 인코딩된 데이터를 구문 분석하고 val이 가리키는 값에 결과를 저장합니다.
// 디코딩 규칙에 대한 것은 패키지 수준 문서를 참조하십시오. Val은 nil이 아닌 포인터여야합니다.
//
// r이 ByteReader를 구현하지 않으면 Decode는 자체 버퍼링을 수행합니다.
//
// Decode는 모든 리더에 대해 입력 제한을 설정하지 않으며, 따라서 거대한 값 크기로 인한 패닉에 취약할 수 있습니다.
// 입력 제한이 필요한 경우 다음을 사용하십시오.
//
// NewStream(r, limit).Decode(val)
func Decode(r io.Reader, val interface{}) error {
	stream := streamPool.Get().(*Stream) // 스트림 풀에서 스트림 가져오기
	defer streamPool.Put(stream)         // 스트림 풀에 스트림 반환

	stream.Reset(r, 0)        // 스트림을 r로 초기화
	return stream.Decode(val) // val에 스트림을 디코딩
}

// DecodeBytes는 b에서 RLP 데이터를 val로 구문 분석합니다. 디코딩 규칙에 대한 것은 패키지 수준 문서를 참조하십시오.
// 입력은 정확히 하나의 값을 포함해야 하며 추가 데이터가 없어야합니다.
func DecodeBytes(b []byte, val interface{}) error {
	r := (*sliceReader)(&b)

	stream := streamPool.Get().(*Stream)
	defer streamPool.Put(stream)

	stream.Reset(r, uint64(len(b)))
	if err := stream.Decode(val); err != nil {
		return err
	}
	if len(b) > 0 {
		return ErrMoreThanOneValue
	}
	return nil
}

type decodeError struct {
	msg string
	typ reflect.Type
	ctx []string
}

func (err *decodeError) Error() string {
	ctx := ""
	if len(err.ctx) > 0 {
		ctx = ", decoding into "
		for i := len(err.ctx) - 1; i >= 0; i-- {
			ctx += err.ctx[i]
		}
	}
	return fmt.Sprintf("rlp: %s for %v%s", err.msg, err.typ, ctx)
}

func wrapStreamError(err error, typ reflect.Type) error {
	switch err {
	case ErrCanonInt:
		return &decodeError{msg: "non-canonical integer (leading zero bytes)", typ: typ}
	case ErrCanonSize:
		return &decodeError{msg: "non-canonical size information", typ: typ}
	case ErrExpectedList:
		return &decodeError{msg: "expected input list", typ: typ}
	case ErrExpectedString:
		return &decodeError{msg: "expected input string or byte", typ: typ}
	case errUintOverflow:
		return &decodeError{msg: "input string too long", typ: typ}
	case errNotAtEOL:
		return &decodeError{msg: "input list has too many elements", typ: typ}
	}
	return err
}

func addErrorContext(err error, ctx string) error {
	if decErr, ok := err.(*decodeError); ok {
		decErr.ctx = append(decErr.ctx, ctx)
	}
	return err
}

var (
	decoderInterface = reflect.TypeOf(new(Decoder)).Elem()
	bigInt           = reflect.TypeOf(big.Int{})
	u256Int          = reflect.TypeOf(uint256.Int{})
)

func makeDecoder(typ reflect.Type, tags rlpstruct.Tags) (dec decoder, err error) {
	kind := typ.Kind()
	switch {
	case typ == rawValueType:
		return decodeRawValue, nil
	case typ.AssignableTo(reflect.PtrTo(bigInt)):
		return decodeBigInt, nil
	case typ.AssignableTo(bigInt):
		return decodeBigIntNoPtr, nil
	case typ == reflect.PtrTo(u256Int):
		return decodeU256, nil
	case typ == u256Int:
		return decodeU256NoPtr, nil
	case kind == reflect.Ptr:
		return makePtrDecoder(typ, tags)
	case reflect.PtrTo(typ).Implements(decoderInterface):
		return decodeDecoder, nil
	case isUint(kind):
		return decodeUint, nil
	case kind == reflect.Bool:
		return decodeBool, nil
	case kind == reflect.String:
		return decodeString, nil
	case kind == reflect.Slice || kind == reflect.Array:
		return makeListDecoder(typ, tags)
	case kind == reflect.Struct:
		return makeStructDecoder(typ)
	case kind == reflect.Interface:
		return decodeInterface, nil
	default:
		return nil, fmt.Errorf("rlp: type %v is not RLP-serializable", typ)
	}
}

func decodeRawValue(s *Stream, val reflect.Value) error {
	r, err := s.Raw()
	if err != nil {
		return err
	}
	val.SetBytes(r)
	return nil
}

func decodeUint(s *Stream, val reflect.Value) error {
	typ := val.Type()
	num, err := s.uint(typ.Bits())
	if err != nil {
		return wrapStreamError(err, val.Type())
	}
	val.SetUint(num)
	return nil
}

func decodeBool(s *Stream, val reflect.Value) error {
	b, err := s.Bool()
	if err != nil {
		return wrapStreamError(err, val.Type())
	}
	val.SetBool(b)
	return nil
}

func decodeString(s *Stream, val reflect.Value) error {
	b, err := s.Bytes()
	if err != nil {
		return wrapStreamError(err, val.Type())
	}
	val.SetString(string(b))
	return nil
}

func decodeBigIntNoPtr(s *Stream, val reflect.Value) error {
	return decodeBigInt(s, val.Addr())
}

func decodeBigInt(s *Stream, val reflect.Value) error {
	i := val.Interface().(*big.Int)
	if i == nil {
		i = new(big.Int)
		val.Set(reflect.ValueOf(i))
	}

	err := s.decodeBigInt(i)
	if err != nil {
		return wrapStreamError(err, val.Type())
	}
	return nil
}

func decodeU256NoPtr(s *Stream, val reflect.Value) error {
	return decodeU256(s, val.Addr())
}

func decodeU256(s *Stream, val reflect.Value) error {
	i := val.Interface().(*uint256.Int)
	if i == nil {
		i = new(uint256.Int)
		val.Set(reflect.ValueOf(i))
	}

	err := s.ReadUint256(i)
	if err != nil {
		return wrapStreamError(err, val.Type())
	}
	return nil
}

func makeListDecoder(typ reflect.Type, tag rlpstruct.Tags) (decoder, error) {
	etype := typ.Elem()
	if etype.Kind() == reflect.Uint8 && !reflect.PtrTo(etype).Implements(decoderInterface) {
		if typ.Kind() == reflect.Array {
			return decodeByteArray, nil
		}
		return decodeByteSlice, nil
	}
	etypeinfo := theTC.infoWhileGenerating(etype, rlpstruct.Tags{})
	if etypeinfo.decoderErr != nil {
		return nil, etypeinfo.decoderErr
	}
	var dec decoder
	switch {
	case typ.Kind() == reflect.Array:
		dec = func(s *Stream, val reflect.Value) error {
			return decodeListArray(s, val, etypeinfo.decoder)
		}
	case tag.Tail:
		// "tail" 태그가 지정된 슬라이스는 구조체의 마지막 필드로 발생할 수 있으며
		// 모든 나머지 리스트 요소를 포함해야 합니다. 구조체 디코더는 이미 s.List를 호출했으므로
		// 요소를 디코딩하기 위해 직접 진행합니다.
		dec = func(s *Stream, val reflect.Value) error {
			return decodeSliceElems(s, val, etypeinfo.decoder)
		}
	default:
		dec = func(s *Stream, val reflect.Value) error {
			return decodeListSlice(s, val, etypeinfo.decoder)
		}
	}
	return dec, nil
}

func decodeListSlice(s *Stream, val reflect.Value, elemdec decoder) error {
	size, err := s.List()
	if err != nil {
		return wrapStreamError(err, val.Type())
	}
	if size == 0 {
		val.Set(reflect.MakeSlice(val.Type(), 0, 0))
		return s.ListEnd()
	}
	if err := decodeSliceElems(s, val, elemdec); err != nil {
		return err
	}
	return s.ListEnd()
}

func decodeSliceElems(s *Stream, val reflect.Value, elemdec decoder) error {
	i := 0
	for ; ; i++ {
		// 필요하다면 슬라이스 크기를 늘립니다.
		if i >= val.Cap() {
			newcap := val.Cap() + val.Cap()/2
			if newcap < 4 {
				newcap = 4
			}
			newv := reflect.MakeSlice(val.Type(), val.Len(), newcap)
			reflect.Copy(newv, val)
			val.Set(newv)
		}
		if i >= val.Len() {
			val.SetLen(i + 1)
		}
		// decode into element
		if err := elemdec(s, val.Index(i)); err == EOL {
			break
		} else if err != nil {
			return addErrorContext(err, fmt.Sprint("[", i, "]"))
		}
	}
	if i < val.Len() {
		val.SetLen(i)
	}
	return nil
}

func decodeListArray(s *Stream, val reflect.Value, elemdec decoder) error {
	if _, err := s.List(); err != nil {
		return wrapStreamError(err, val.Type())
	}
	vlen := val.Len()
	i := 0
	for ; i < vlen; i++ {
		if err := elemdec(s, val.Index(i)); err == EOL {
			break
		} else if err != nil {
			return addErrorContext(err, fmt.Sprint("[", i, "]"))
		}
	}
	if i < vlen {
		return &decodeError{msg: "input list has too few elements", typ: val.Type()}
	}
	return wrapStreamError(s.ListEnd(), val.Type())
}

func decodeByteSlice(s *Stream, val reflect.Value) error {
	b, err := s.Bytes()
	if err != nil {
		return wrapStreamError(err, val.Type())
	}
	val.SetBytes(b)
	return nil
}

func decodeByteArray(s *Stream, val reflect.Value) error {
	kind, size, err := s.Kind()
	if err != nil {
		return err
	}
	slice := byteArrayBytes(val, val.Len())
	switch kind {
	case Byte:
		if len(slice) == 0 {
			return &decodeError{msg: "input string too long", typ: val.Type()}
		} else if len(slice) > 1 {
			return &decodeError{msg: "input string too short", typ: val.Type()}
		}
		slice[0] = s.byteval
		s.kind = -1
	case String:
		if uint64(len(slice)) < size {
			return &decodeError{msg: "input string too long", typ: val.Type()}
		}
		if uint64(len(slice)) > size {
			return &decodeError{msg: "input string too short", typ: val.Type()}
		}
		if err := s.readFull(slice); err != nil {
			return err
		}
		// 단일 바이트 인코딩을 사용해야하는 입력을 거부합니다.
		if size == 1 && slice[0] < 128 {
			return wrapStreamError(ErrCanonSize, val.Type())
		}
	case List:
		return wrapStreamError(ErrExpectedString, val.Type())
	}
	return nil
}

func makeStructDecoder(typ reflect.Type) (decoder, error) {
	fields, err := structFields(typ)
	if err != nil {
		return nil, err
	}
	for _, f := range fields {
		if f.info.decoderErr != nil {
			return nil, structFieldError{typ, f.index, f.info.decoderErr}
		}
	}
	dec := func(s *Stream, val reflect.Value) (err error) {
		if _, err := s.List(); err != nil {
			return wrapStreamError(err, typ)
		}
		for i, f := range fields {
			err := f.info.decoder(s, val.Field(f.index))
			if err == EOL {
				if f.optional {
					// 필드가 선택 사항이므로 마지막 필드에 도달하기 전에 리스트의 끝에 도달하는 것이 허용됩니다.
					// 모든 남은 디코딩되지 않은 필드는 해당 타입의 제로 값으로 설정됩니다.
					zeroFields(val, fields[i:])
					break
				}
				return &decodeError{msg: "too few elements", typ: typ}
			} else if err != nil {
				return addErrorContext(err, "."+typ.Field(f.index).Name)
			}
		}
		return wrapStreamError(s.ListEnd(), typ)
	}
	return dec, nil
}

func zeroFields(structval reflect.Value, fields []field) {
	for _, f := range fields {
		fv := structval.Field(f.index)
		fv.Set(reflect.Zero(fv.Type()))
	}
}

// makePtrDecoder는 포인터의 요소 유형으로 디코딩하는 디코더를 생성합니다.
func makePtrDecoder(typ reflect.Type, tag rlpstruct.Tags) (decoder, error) {
	etype := typ.Elem()
	etypeinfo := theTC.infoWhileGenerating(etype, rlpstruct.Tags{})
	switch {
	case etypeinfo.decoderErr != nil:
		return nil, etypeinfo.decoderErr
	case !tag.NilOK:
		return makeSimplePtrDecoder(etype, etypeinfo), nil
	default:
		return makeNilPtrDecoder(etype, etypeinfo, tag), nil
	}
}

func makeSimplePtrDecoder(etype reflect.Type, etypeinfo *typeinfo) decoder {
	return func(s *Stream, val reflect.Value) (err error) {
		newval := val
		if val.IsNil() {
			newval = reflect.New(etype)
		}
		if err = etypeinfo.decoder(s, newval.Elem()); err == nil {
			val.Set(newval)
		}
		return err
	}
}

// makeNilPtrDecoder는 빈 값을 nil로 디코딩하는 디코더를 생성합니다. 비어있지 않은 값은
// makePtrDecoder와 마찬가지로 요소 유형의 값을 디코딩합니다.
//
// 이 디코더는 구조체 태그 "nil"이 있는 포인터 유형의 구조체 필드에 사용됩니다.
func makeNilPtrDecoder(etype reflect.Type, etypeinfo *typeinfo, ts rlpstruct.Tags) decoder {
	typ := reflect.PtrTo(etype)
	nilPtr := reflect.Zero(typ)

	// nil 포인터가 디코딩되는 값의 종류를 결정합니다.
	nilKind := typeNilKind(etype, ts)

	return func(s *Stream, val reflect.Value) (err error) {
		kind, size, err := s.Kind()
		if err != nil {
			val.Set(nilPtr)
			return wrapStreamError(err, typ)
		}
		// 비어있는 값은 nil 포인터로 처리합니다.
		if kind != Byte && size == 0 {
			if kind != nilKind {
				return &decodeError{
					msg: fmt.Sprintf("wrong kind of empty value (got %v, want %v)", kind, nilKind),
					typ: typ,
				}
			}
			// s.Kind를 다시 설정합니다. 이는 입력 위치가 다음 값을 읽어야 하기 때문에 중요합니다. (아무것도 읽지 않더라도)
			s.kind = -1
			val.Set(nilPtr)
			return nil
		}
		newval := val
		if val.IsNil() {
			newval = reflect.New(etype)
		}
		if err = etypeinfo.decoder(s, newval.Elem()); err == nil {
			val.Set(newval)
		}
		return err
	}
}

var ifsliceType = reflect.TypeOf([]interface{}{})

func decodeInterface(s *Stream, val reflect.Value) error {
	if val.Type().NumMethod() != 0 {
		return fmt.Errorf("rlp: type %v is not RLP-serializable", val.Type())
	}
	kind, _, err := s.Kind()
	if err != nil {
		return err
	}
	if kind == List {
		slice := reflect.New(ifsliceType).Elem()
		if err := decodeListSlice(s, slice, decodeInterface); err != nil {
			return err
		}
		val.Set(slice)
	} else {
		b, err := s.Bytes()
		if err != nil {
			return err
		}
		val.Set(reflect.ValueOf(b))
	}
	return nil
}

func decodeDecoder(s *Stream, val reflect.Value) error {
	return val.Addr().Interface().(Decoder).DecodeRLP(s)
}

// Kind는 RLP 스트림에 포함된 값의 종류를 나타냅니다.
type Kind int8

const (
	Byte   Kind = iota // 단일 바이트 값
	String             // 문자열 (또는 바이트 슬라이스)
	List               // 리스트
)

func (k Kind) String() string {
	switch k {
	case Byte:
		return "Byte"
	case String:
		return "String"
	case List:
		return "List"
	default:
		return fmt.Sprintf("Unknown(%d)", k)
	}
}

// ByteReader는 스트림의 모든 입력 리더에 의해 구현되어야합니다.
// e.g. bufio.Reader 및 bytes.Reader.
type ByteReader interface {
	io.Reader
	io.ByteReader
}

// Stream은 입력 스트림의 파편적 디코딩에 사용할 수 있습니다.
// 이는 입력이 매우 크거나 유형에 대한 디코딩 규칙이 입력 구조에 따라 다른 경우 유용합니다.
// Stream은 내부 버퍼를 유지하지 않습니다. 값을 디코딩 한 후 입력 리더는 다음 값에 대한 유형 정보 바로 앞에 위치합니다.
//
// 리스트를 디코딩하다가 입력 위치가 목록의 선언 된 길이에 도달하면 모든 작업은 EOL 오류를 반환합니다.
// 리스트의 마지막은 ListEnd를 사용하여 알려야합니다.
//
// Stream은 동시 접근에 대해 안전하지 않습니다.
type Stream struct {
	r ByteReader

	remaining uint64   // r에서 읽어야하는 남은 바이트 수
	size      uint64   // 캐시된 값의 크기
	kinderr   error    // 지난 readKind에서 발생한 오류
	stack     []uint64 // 리스트 크기
	uintbuf   [32]byte // 정수 디코딩을 위한 보조 버퍼
	kind      Kind     // 캐시된 값의 종류
	byteval   byte     // 타입 태그의 단일 바이트 값
	limited   bool     // 입력 제한이 적용되는 경우 true
}

// NewStream은 r에서 읽어들이는 새로운 디코딩 스트림을 생성합니다.
//
// r이 ByteReader 인터페이스를 구현하는 경우 Stream은 버퍼링을 추가하지 않습니다.
//
// 최상위 값이 아닌 경우(non-toplevel values) Stream은
// 포함된 리스트(enclosing list)에 맞지 않는 값에 대해 ErrElemTooLarge를 반환합니다.
//
// Stream은 선택적 입력 제한을 지원합니다. 제한이 설정된 경우
// 모든 최상위 값의 크기는 남은 입력 길이와 비교됩니다.
// 입력 길이를 초과하는 값을 만나면 Stream 작업은 ErrValueTooLarge를 반환합니다.
// 제한은 inputLimit에 대해 0이 아닌 값을 전달하여 설정할 수 있습니다.
//
// r이 bytes.Reader 또는 strings.Reader인 경우
// 명시적 제한이 제공되지 않는 한 입력 제한은 r의 기본 데이터의 길이로 설정됩니다.
func NewStream(r io.Reader, inputLimit uint64) *Stream {
	s := new(Stream)
	s.Reset(r, inputLimit)
	return s
}

// NewListStream은 주어진 길이의 인코딩된 리스트에 위치한 것처럼 동작하는 새로운 스트림을 생성합니다.
func NewListStream(r io.Reader, len uint64) *Stream {
	s := new(Stream)
	s.Reset(r, len)
	s.kind = List
	s.size = len
	return s
}

// Bytes는 RLP 문자열을 읽고 해당 내용을 바이트 슬라이스로 반환합니다.
// 입력이 RLP 문자열을 포함하지 않으면 반환 된 오류는 ErrExpectedString이 됩니다.
func (s *Stream) Bytes() ([]byte, error) {
	kind, size, err := s.Kind()
	if err != nil {
		return nil, err
	}
	switch kind {
	case Byte:
		s.kind = -1 // Kind 다시 설정
		return []byte{s.byteval}, nil
	case String:
		b := make([]byte, size)
		if err = s.readFull(b); err != nil {
			return nil, err
		}
		if size == 1 && b[0] < 128 {
			return nil, ErrCanonSize
		}
		return b, nil
	default:
		return nil, ErrExpectedString
	}
}

// ReadBytes는 다음 RLP 값을 디코딩하고 결과를 b에 저장합니다.
// 값 크기는 len(b)와 정확히 일치해야합니다.
func (s *Stream) ReadBytes(b []byte) error {
	kind, size, err := s.Kind()
	if err != nil {
		return err
	}
	switch kind {
	case Byte:
		if len(b) != 1 {
			return fmt.Errorf("input value has wrong size 1, want %d", len(b))
		}
		b[0] = s.byteval
		s.kind = -1 // Kind 다시 설정
		return nil
	case String:
		if uint64(len(b)) != size {
			return fmt.Errorf("input value has wrong size %d, want %d", size, len(b))
		}
		if err = s.readFull(b); err != nil {
			return err
		}
		if size == 1 && b[0] < 128 {
			return ErrCanonSize
		}
		return nil
	default:
		return ErrExpectedString
	}
}

// Raw는 RLP 유형 정보를 포함한 원시 인코딩 된 값을 읽습니다.
func (s *Stream) Raw() ([]byte, error) {
	kind, size, err := s.Kind()
	if err != nil {
		return nil, err
	}
	if kind == Byte {
		s.kind = -1 // rearm Kind
		return []byte{s.byteval}, nil
	}
	// 원래 헤더는 이미 사용되었으며 더 이상 사용할 수 없습니다.
	// 내용을 읽고 그 앞에 새 헤더를 넣습니다.
	start := headsize(size)
	buf := make([]byte, uint64(start)+size)
	if err := s.readFull(buf[start:]); err != nil {
		return nil, err
	}
	if kind == String {
		puthead(buf, 0x80, 0xB7, size)
	} else {
		puthead(buf, 0xC0, 0xF7, size)
	}
	return buf, nil
}

// Uint는 최대 8 바이트의 RLP 문자열을 읽고 해당 내용을 부호없는 정수로 반환합니다.
// 입력이 RLP 문자열을 포함하지 않으면 반환 된 오류는 ErrExpectedString이 됩니다.
//
// Deprecated: s.Uint64을 대신 사용하십시오.
func (s *Stream) Uint() (uint64, error) {
	return s.uint(64)
}

func (s *Stream) Uint64() (uint64, error) {
	return s.uint(64)
}

func (s *Stream) Uint32() (uint32, error) {
	i, err := s.uint(32)
	return uint32(i), err
}

func (s *Stream) Uint16() (uint16, error) {
	i, err := s.uint(16)
	return uint16(i), err
}

func (s *Stream) Uint8() (uint8, error) {
	i, err := s.uint(8)
	return uint8(i), err
}

func (s *Stream) uint(maxbits int) (uint64, error) {
	kind, size, err := s.Kind()
	if err != nil {
		return 0, err
	}
	switch kind {
	case Byte:
		if s.byteval == 0 {
			return 0, ErrCanonInt
		}
		s.kind = -1 // Kind 다시 설정
		return uint64(s.byteval), nil
	case String:
		if size > uint64(maxbits/8) {
			return 0, errUintOverflow
		}
		v, err := s.readUint(byte(size))
		switch {
		case err == ErrCanonSize:
			// Adjust error because we're not reading a size right now.
			return 0, ErrCanonInt
		case err != nil:
			return 0, err
		case size > 0 && v < 128:
			return 0, ErrCanonSize
		default:
			return v, nil
		}
	default:
		return 0, ErrExpectedString
	}
}

// Bool은 최대 1 바이트의 RLP 문자열을 읽고 해당 내용을 부울 값으로 반환합니다.
// 입력이 RLP 문자열을 포함하지 않으면 반환 된 오류는 ErrExpectedString이 됩니다.
func (s *Stream) Bool() (bool, error) {
	num, err := s.uint(8)
	if err != nil {
		return false, err
	}
	switch num {
	case 0:
		return false, nil
	case 1:
		return true, nil
	default:
		return false, fmt.Errorf("rlp: invalid boolean value: %d", num)
	}
}

// List는 RLP 리스트를 디코딩을 시작합니다. 입력이 리스트를 포함하지 않으면
// 반환 된 오류는 ErrExpectedList가 됩니다. 리스트의 끝에 도달하면
// 모든 Stream 작업은 EOL을 반환합니다.
func (s *Stream) List() (size uint64, err error) {
	kind, size, err := s.Kind()
	if err != nil {
		return 0, err
	}
	if kind != List {
		return 0, ErrExpectedList
	}

	// 새 크기를 스택에 푸시하기 전에 외부 리스트에서 내부 리스트의 크기를 제거합니다.
	// 이렇게하면 ListEnd 호출 후 남은 외부 리스트 크기가 올바르게 유지됩니다.
	if inList, limit := s.listLimit(); inList {
		s.stack[len(s.stack)-1] = limit - size
	}
	s.stack = append(s.stack, size)
	s.kind = -1
	s.size = 0
	return size, nil
}

// ListEnd는 상위 리스트로 돌아갑니다.
// 입력 리더는 리스트의 끝에 위치해야합니다.
func (s *Stream) ListEnd() error {
	// 현재 리스트에 더 이상 데이터가 남아 있지 않도록합니다.
	if inList, listLimit := s.listLimit(); !inList {
		return errNotInList
	} else if listLimit > 0 {
		return errNotAtEOL
	}
	s.stack = s.stack[:len(s.stack)-1] // 제거
	s.kind = -1
	s.size = 0
	return nil
}

// MoreDataInList는 현재 리스트 컨텍스트에 더 읽을 데이터가 있는지보고합니다.
func (s *Stream) MoreDataInList() bool {
	_, listLimit := s.listLimit()
	return listLimit > 0
}

// BigInt는 임의 크기의 정수 값을 디코딩합니다.
func (s *Stream) BigInt() (*big.Int, error) {
	i := new(big.Int)
	if err := s.decodeBigInt(i); err != nil {
		return nil, err
	}
	return i, nil
}

func (s *Stream) decodeBigInt(dst *big.Int) error {
	var buffer []byte
	kind, size, err := s.Kind()
	switch {
	case err != nil:
		return err
	case kind == List:
		return ErrExpectedString
	case kind == Byte:
		buffer = s.uintbuf[:1]
		buffer[0] = s.byteval
		s.kind = -1 // Kind 다시 설정
	case size == 0:
		// 길이가 0인 경우 읽지 않습니다.
		s.kind = -1
	case size <= uint64(len(s.uintbuf)):
		// s.uintbuf보다 작은 정수의 경우 버퍼를 할당하지 않을 수 있습니다.
		buffer = s.uintbuf[:size]
		if err := s.readFull(buffer); err != nil {
			return err
		}
		// 단일 바이트 인코딩을 사용해야하는 입력을 거부합니다.
		if size == 1 && buffer[0] < 128 {
			return ErrCanonSize
		}
	default:
		// 큰 정수의 경우 임시 버퍼가 필요합니다.
		buffer = make([]byte, size)
		if err := s.readFull(buffer); err != nil {
			return err
		}
	}

	// 선행 0 바이트 거부
	if len(buffer) > 0 && buffer[0] == 0 {
		return ErrCanonInt
	}
	// 정수 바이트를 설정합니다.
	dst.SetBytes(buffer)
	return nil
}

// ReadUint256는 다음 값을 uint256으로 디코딩합니다.
func (s *Stream) ReadUint256(dst *uint256.Int) error {
	var buffer []byte
	kind, size, err := s.Kind()
	switch {
	case err != nil:
		return err
	case kind == List:
		return ErrExpectedString
	case kind == Byte:
		buffer = s.uintbuf[:1]
		buffer[0] = s.byteval
		s.kind = -1 // Kind 다시 설정
	case size == 0:
		// 길이가 0인 경우 읽지 않습니다.
		s.kind = -1
	case size <= uint64(len(s.uintbuf)):
		// 모든 가능한 uint256 값이 s.uintbuf에 들어갑니다.
		buffer = s.uintbuf[:size]
		if err := s.readFull(buffer); err != nil {
			return err
		}
		// 단일 바이트 인코딩을 사용해야하는 입력을 거부합니다.
		if size == 1 && buffer[0] < 128 {
			return ErrCanonSize
		}
	default:
		return errUint256Large
	}

	// 선행 0 바이트 거부
	if len(buffer) > 0 && buffer[0] == 0 {
		return ErrCanonInt
	}
	// 정수 바이트를 설정합니다.
	dst.SetBytes(buffer)
	return nil
}

// Decode는 값을 디코딩하고 그 결과를 val이 가리키는 값에 저장합니다.
// 디코딩 규칙에 대한 설명은 Decode 함수에 대한 문서를 참조하십시오.
func (s *Stream) Decode(val interface{}) error {
	if val == nil { // val은 nil이 아닌 포인터 유형이어야합니다.
		return errDecodeIntoNil
	}
	rval := reflect.ValueOf(val) // val의 값을 가져옵니다.
	rtyp := rval.Type()          // val의 유형을 가져옵니다.
	if rtyp.Kind() != reflect.Ptr {
		return errNoPointer
	}
	if rval.IsNil() {
		return errDecodeIntoNil
	}
	decoder, err := cachedDecoder(rtyp.Elem()) // 유형에 대한 디코더를 가져옵니다.
	if err != nil {
		return err
	}

	err = decoder(s, rval.Elem()) // 값을 디코딩합니다.
	if decErr, ok := err.(*decodeError); ok && len(decErr.ctx) > 0 {
		// 디코딩 대상 유형을 오류에 추가하여 컨텍스트가 더 의미 있도록합니다.
		decErr.ctx = append(decErr.ctx, fmt.Sprint("(", rtyp.Elem(), ")"))
	}
	return err
}

// Reset은 현재 디코딩 컨텍스트에 대한 모든 정보를 삭제하고 r에서 읽기를 시작합니다.
// 이 메서드는 미리 할당 된 Stream을 많은 디코딩 작업에서 재사용하기위한 것입니다.
//
// r이 ByteReader도 구현하지 않으면 Stream은 자체 버퍼링을 수행합니다.
func (s *Stream) Reset(r io.Reader, inputLimit uint64) {
	if inputLimit > 0 { // 입력 제한이 설정된 경우
		s.remaining = inputLimit
		s.limited = true
	} else {
		// 제한이 설정되지 않은 경우
		// 입력의 길이를 통해 제한을 자동으로 설정합니다.
		switch br := r.(type) {
		case *bytes.Reader:
			s.remaining = uint64(br.Len())
			s.limited = true
		case *bytes.Buffer:
			s.remaining = uint64(br.Len())
			s.limited = true
		case *strings.Reader:
			s.remaining = uint64(br.Len())
			s.limited = true
		default:
			s.limited = false
		}
	}
	// r을 버퍼로 래핑합니다. (버퍼가 없는 경우)
	bufr, ok := r.(ByteReader)
	if !ok {
		bufr = bufio.NewReader(r)
	}
	s.r = bufr
	// 디코딩 컨텍스트를 재설정합니다.
	s.stack = s.stack[:0]
	s.size = 0
	s.kind = -1
	s.kinderr = nil
	s.byteval = 0
	s.uintbuf = [32]byte{}
}

// 반환된 크기는 값을 구성하는 바이트 수입니다.
// kind == Byte의 경우 크기는 값이 타입 태그에 포함되어 있기 때문에 0입니다. (단일 바이트)
//
// Kind의 첫 번째 호출은 입력 리더에서 크기 정보를 읽고 입력 리더의 시작 위치를 값의 실제 바이트 앞에 둡니다.
// 이후 호출은 입력 리더를 전혀 읽지 않고 이전에 캐시 된 정보를 반환합니다.
func (s *Stream) Kind() (kind Kind, size uint64, err error) {
	if s.kind >= 0 { // 캐시 된 정보가 있는 경우
		return s.kind, s.size, s.kinderr
	}

	// 리스트의 마지막을 확인합니다.
	// readKind는 리스트 크기를 확인하고 잘못된 오류를 반환할 수 있으므로 여기에서 수행해야합니다.
	inList, listLimit := s.listLimit()
	if inList && listLimit == 0 {
		return 0, 0, EOL
	}
	// 실제 크기 태그를 읽습니다.
	s.kind, s.size, s.kinderr = s.readKind()
	if s.kinderr == nil {
		// 입력 제한에 대해 실제 값 크기를 확인합니다. 왜냐하면 많은 디코더가 값의 크기와 일치하는
		// 입력 버퍼를 할당하는 것을 요구하기 때문입니다. 여기에서 이를 먼저 확인함으로써
		// 매우 큰 크기를 가지는 입력으로부터 이러한 디코더를 보호합니다.
		if inList && s.size > listLimit {
			s.kinderr = ErrElemTooLarge
		} else if s.limited && s.size > s.remaining {
			s.kinderr = ErrValueTooLarge
		}
	}
	return s.kind, s.size, s.kinderr
}

func (s *Stream) readKind() (kind Kind, size uint64, err error) {
	b, err := s.readByte()
	if err != nil {
		if len(s.stack) == 0 {
			// 상위 레벨에서 실제 EOF에 대한 오류를 조정합니다. io.EOF는
			// 호출자가 디코딩을 중지할 때 사용됩니다.
			switch err {
			case io.ErrUnexpectedEOF:
				err = io.EOF
			case ErrValueTooLarge:
				err = io.EOF
			}
		}
		return 0, 0, err
	}
	s.byteval = 0
	switch {
	case b < 0x80:
		// 값이 [0x00, 0x7F] 범위에있는 단일 바이트의 경우 해당 바이트는 자체 RLP 인코딩입니다.
		s.byteval = b
		return Byte, 0, nil
	case b < 0xB8:
		// 문자열이 0-55 바이트 길이 인 경우 RLP 인코딩은 문자열의 길이에 0x80을 더한 단일 바이트와 문자열로 구성됩니다.
		// 첫 번째 바이트의 범위는 [0x80, 0xB7]입니다.
		return String, uint64(b - 0x80), nil
	case b < 0xC0:
		// 문자열이 56 바이트 이상인 경우 RLP 인코딩은 0xB7 값과 문자열의 길이를 바이트로 표시한 값의 길이를 더한 단일 바이트와
		// 문자열의 길이, 그리고 문자열로 구성됩니다. 첫 번째 바이트의 범위는 [0xB8, 0xBF]입니다.
		size, err = s.readUint(b - 0xB7)
		if err == nil && size < 56 {
			err = ErrCanonSize
		}
		return String, size, err
	case b < 0xF8:
		// 리스트의 길이를 읽습니다.
		// 모든 항목의 길이를 합한 것이 0-55 바이트 인 경우 RLP 인코딩은 길이에 0xC0을 더한 단일 바이트와
		// 페이로드로 구성됩니다. 첫 번째 바이트의 범위는 [0xC0, 0xF7]입니다.
		return List, uint64(b - 0xC0), nil
	default:
		// 리스트의 길이를 읽습니다.
		// 모든 항목의 길이를 합한 것이 56 바이트 이상인 경우 RLP 인코딩은 0xF7 값과 페이로드의 길이를 바이트로 표시한 값의 길이를 더한 단일 바이트,
		// 페이로드의 길이, 그리고 페이로드로 구성됩니다. 첫 번째 바이트의 범위는 [0xF8, 0xFF]입니다.
		size, err = s.readUint(b - 0xF7)
		if err == nil && size < 56 {
			err = ErrCanonSize
		}
		return List, size, err
	}
}

func (s *Stream) readUint(size byte) (uint64, error) {
	switch size {
	case 0:
		s.kind = -1 // Kind 재설정
		return 0, nil
	case 1:
		b, err := s.readByte()
		return uint64(b), err
	default:
		buffer := s.uintbuf[:8]
		for i := range buffer {
			buffer[i] = 0
		}
		start := int(8 - size)
		if err := s.readFull(buffer[start:]); err != nil {
			return 0, err
		}
		if buffer[start] == 0 {
			// 참고: readUint는 정수 값을 디코딩하는 데도 사용됩니다.
			// 이 경우 오류를 ErrCanonInt로 조정해야합니다.
			return 0, ErrCanonSize
		}
		return binary.BigEndian.Uint64(buffer[:]), nil
	}
}

// readFull은 스트림에서 buf로 읽어들입니다.
func (s *Stream) readFull(buf []byte) (err error) {
	if err := s.willRead(uint64(len(buf))); err != nil {
		return err
	}
	var nn, n int
	for n < len(buf) && err == nil {
		nn, err = s.r.Read(buf[n:])
		n += nn
	}
	if err == io.EOF {
		if n < len(buf) {
			err = io.ErrUnexpectedEOF
		} else {
			// 리더는 읽기가 성공했음에도 불구하고 EOF를 제공 할 수 있습니다.
			// 이러한 경우 io.ReadFull()과 같이 EOF를 버립니다.
			err = nil
		}
	}
	return err
}

// readByte는 기본 스트림에서 단일 바이트를 읽습니다.
func (s *Stream) readByte() (byte, error) {
	if err := s.willRead(1); err != nil {
		return 0, err
	}
	b, err := s.r.ReadByte()
	if err == io.EOF {
		err = io.ErrUnexpectedEOF
	}
	return b, err
}

// willRead는 기본 스트림에서 읽기 전에 호출됩니다. n을 크기 제한과 비교하고
// n이 제한을 초과하지 않으면 제한을 업데이트합니다.
func (s *Stream) willRead(n uint64) error {
	s.kind = -1 // Kind 재설정

	if inList, limit := s.listLimit(); inList {
		if n > limit {
			return ErrElemTooLarge
		}
		s.stack[len(s.stack)-1] = limit - n
	}
	if s.limited {
		if n > s.remaining {
			return ErrValueTooLarge
		}
		s.remaining -= n
	}
	return nil
}

// listLimit는 가장 안쪽 리스트에 남은 데이터 양을 반환합니다.
func (s *Stream) listLimit() (inList bool, limit uint64) {
	if len(s.stack) == 0 {
		return false, 0
	}
	return true, s.stack[len(s.stack)-1]
}

type sliceReader []byte

func (sr *sliceReader) Read(b []byte) (int, error) {
	if len(*sr) == 0 {
		return 0, io.EOF
	}
	n := copy(b, *sr)
	*sr = (*sr)[n:]
	return n, nil
}

func (sr *sliceReader) ReadByte() (byte, error) {
	if len(*sr) == 0 {
		return 0, io.EOF
	}
	b := (*sr)[0]
	*sr = (*sr)[1:]
	return b, nil
}
