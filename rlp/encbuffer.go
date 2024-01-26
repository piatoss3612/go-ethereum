// Copyright 2022 The go-ethereum Authors
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
	"encoding/binary"
	"io"
	"math/big"
	"reflect"
	"sync"

	"github.com/holiman/uint256"
)

type encBuffer struct {
	str     []byte     // 문자열 데이터, 리스트 헤더를 제외한 모든 것을 포함
	lheads  []listhead // 모든 리스트 헤더
	lhsize  int        // 모든 인코딩된 리스트 헤더의 크기의 합
	sizebuf [9]byte    // uint 인코딩을 위한 보조 버퍼
}

// 글로벌 encBuffer 풀
var encBufferPool = sync.Pool{
	New: func() interface{} { return new(encBuffer) },
}

func getEncBuffer() *encBuffer {
	buf := encBufferPool.Get().(*encBuffer)
	buf.reset()
	return buf
}

func (buf *encBuffer) reset() {
	buf.lhsize = 0
	buf.str = buf.str[:0]
	buf.lheads = buf.lheads[:0]
}

// size는 인코딩된 데이터의 길이를 반환합니다.
func (buf *encBuffer) size() int {
	return len(buf.str) + buf.lhsize // 문자열 데이터 길이 + 리스트 헤더 크기의 합
}

// makeBytes는 인코더 출력을 생성합니다.
func (buf *encBuffer) makeBytes() []byte {
	out := make([]byte, buf.size())
	buf.copyTo(out)
	return out
}

func (buf *encBuffer) copyTo(dst []byte) {
	strpos := 0
	pos := 0
	for _, head := range buf.lheads {
		// 헤더 전에 문자열 데이터를 쓴다.
		n := copy(dst[pos:], buf.str[strpos:head.offset])
		pos += n
		strpos += n
		// 헤더를 쓴다.
		enc := head.encode(dst[pos:])
		pos += len(enc) // 헤더 크기만큼 증가
	}
	// 마지막 리스트 헤더 이후의 문자열 데이터를 복사합니다.
	copy(dst[pos:], buf.str[strpos:])
}

// writeTo는 인코더 출력을 w에 씁니다.
func (buf *encBuffer) writeTo(w io.Writer) (err error) {
	strpos := 0
	for _, head := range buf.lheads { // 순서가 뒤바뀐 것처럼 보일 수 있는데, 오프셋을 따지기 때문에 0번째 리스트 헤더 - 0번째 리스트 페이로드, 1번째 리스트 헤더 ... 순서로 쓰여진다.
		// 헤더 전에 문자열 데이터를 쓴다.
		if head.offset-strpos > 0 {
			n, err := w.Write(buf.str[strpos:head.offset])
			strpos += n
			if err != nil {
				return err
			}
		}
		// 헤더를 쓴다.
		enc := head.encode(buf.sizebuf[:])
		if _, err = w.Write(enc); err != nil {
			return err
		}
	}
	if strpos < len(buf.str) {
		// 마지막 리스트 헤더 이후의 문자열 데이터를 쓴다. (마지막 리스트의 페이로드)
		_, err = w.Write(buf.str[strpos:])
	}
	return err
}

// Write는 io.Writer를 구현하고 b를 직접 출력에 추가합니다.
func (buf *encBuffer) Write(b []byte) (int, error) {
	buf.str = append(buf.str, b...)
	return len(b), nil
}

// writeBool는 b를 정수 0 (false) 또는 1 (true)로 씁니다.
func (buf *encBuffer) writeBool(b bool) {
	if b {
		buf.str = append(buf.str, 0x01)
	} else {
		buf.str = append(buf.str, 0x80)
	}
}

func (buf *encBuffer) writeUint64(i uint64) {
	if i == 0 {
		buf.str = append(buf.str, 0x80) // 길이가 0인 문자열
	} else if i < 128 {
		// 단일 바이트로 인코딩되는 경우
		buf.str = append(buf.str, byte(i))
	} else { // 128보다 크거나 같은 경우
		s := putint(buf.sizebuf[1:], i)
		buf.sizebuf[0] = 0x80 + byte(s)
		buf.str = append(buf.str, buf.sizebuf[:s+1]...)
	}
}

func (buf *encBuffer) writeBytes(b []byte) {
	if len(b) == 1 && b[0] <= 0x7F {
		// 단일 바이트로 인코딩되는 경우 문자열 헤더가 필요 없다.
		buf.str = append(buf.str, b[0])
	} else {
		buf.encodeStringHeader(len(b))
		buf.str = append(buf.str, b...)
	}
}

func (buf *encBuffer) writeString(s string) {
	buf.writeBytes([]byte(s))
}

// wordBytes는 big.Word의 바이트 수입니다.
const wordBytes = (32 << (uint64(^big.Word(0)) >> 63)) / 8

// writeBigInt는 i를 정수로 씁니다.
func (buf *encBuffer) writeBigInt(i *big.Int) {
	bitlen := i.BitLen()
	if bitlen <= 64 { // 64비트 이하의 정수는 uint64로 인코딩
		buf.writeUint64(i.Uint64())
		return
	}
	// 64비트보다 큰 정수는 i.Bits()로부터 인코딩
	// 최소 바이트 길이는 bitlen을 8의 배수로 올림한 것을 8로 나눈 것이다.
	length := ((bitlen + 7) & -8) >> 3                 // i의 바이트 길이
	buf.encodeStringHeader(length)                     // 문자열 헤더를 쓴다.
	buf.str = append(buf.str, make([]byte, length)...) // 문자열 헤더를 제외한 문자열 데이터를 쓰기 위해 문자열 데이터의 길이만큼 0을 추가한다.
	index := length                                    // 문자열 데이터가 끝나는 인덱스 + 1
	bytesBuf := buf.str[len(buf.str)-length:]          // 문자열을 쓰기 위해 임시 버퍼를 초기화한다.
	for _, d := range i.Bits() {                       // big.Int.Bits()는 리틀 엔디언으로 big.Word 슬라이스를 반환한다. (앞에서부터 읽어야 한다.)
		for j := 0; j < wordBytes && index > 0; j++ { // 8바이트 big.Word d에서 1바이트씩 읽어서 버퍼의 뒤에서부터 쓴다.
			index--
			bytesBuf[index] = byte(d)
			d >>= 8
		}
	}
}

// writeUint256 writes z as an integer.
// writeUint256는 z를 정수로 씁니다.
func (buf *encBuffer) writeUint256(z *uint256.Int) {
	bitlen := z.BitLen()
	if bitlen <= 64 {
		buf.writeUint64(z.Uint64())
		return
	}
	nBytes := byte((bitlen + 7) / 8)
	var b [33]byte
	binary.BigEndian.PutUint64(b[1:9], z[3])
	binary.BigEndian.PutUint64(b[9:17], z[2])
	binary.BigEndian.PutUint64(b[17:25], z[1])
	binary.BigEndian.PutUint64(b[25:33], z[0])
	b[32-nBytes] = 0x80 + nBytes
	buf.str = append(buf.str, b[32-nBytes:]...)
}

// list는 헤더 스택에 새로운 리스트 헤더를 추가합니다. 헤더의 인덱스를 반환합니다.
// 리스트의 내용을 인코딩한 후에 이 인덱스로 listEnd를 호출하십시오.
func (buf *encBuffer) list() int {
	buf.lheads = append(buf.lheads, listhead{offset: len(buf.str), size: buf.lhsize}) // offset: 리스트의 시작 위치
	return len(buf.lheads) - 1
}

// listEnd는 주어진 인덱스의 리스트가 인코딩이 끝났음을 표시합니다.
func (buf *encBuffer) listEnd(index int) {
	lh := &buf.lheads[index]                   // 리스트 헤더 가져오기
	lh.size = buf.size() - lh.offset - lh.size // 리스트 페이로드의 크기 계산
	if lh.size < 56 {                          // 페이로드의 크기가 56보다 작은 경우
		buf.lhsize++ // 헤더 크기는 1바이트
	} else {
		buf.lhsize += 1 + intsize(uint64(lh.size)) // 헤더 크기는 1바이트 + 페이로드 크기의 바이트 수
	}
}

func (buf *encBuffer) encode(val interface{}) error {
	rval := reflect.ValueOf(val)
	writer, err := cachedWriter(rval.Type())
	if err != nil {
		return err
	}
	return writer(rval, buf)
}

func (buf *encBuffer) encodeStringHeader(size int) {
	if size < 56 {
		buf.str = append(buf.str, 0x80+byte(size))
	} else {
		sizesize := putint(buf.sizebuf[1:], uint64(size))
		buf.sizebuf[0] = 0xB7 + byte(sizesize)
		buf.str = append(buf.str, buf.sizebuf[:sizesize+1]...)
	}
}

// encReader는 EncodeToReader에 의해 반환된 io.Reader입니다.
// EOF에서 encbuf를 해제합니다.
type encReader struct {
	buf    *encBuffer // 읽기 작업을 수행하는 버퍼. EOF일 때 nil이 된다.
	lhpos  int        // 읽고 있는 리스트 헤더의 인덱스
	strpos int        // 문자열 버퍼에서 현재 위치
	piece  []byte     // 다음 읽을 조각
}

func (r *encReader) Read(b []byte) (n int, err error) {
	for {
		if r.piece = r.next(); r.piece == nil {
			// EOF를 처음 마주한 경우 인코딩 버퍼를 풀에 다시 돌려줍니다.
			// 이후의 호출은 여전히 오류로 EOF를 반환하지만 버퍼는 더 이상 유효하지 않습니다.
			if r.buf != nil {
				encBufferPool.Put(r.buf)
				r.buf = nil
			}
			return n, io.EOF
		}
		nn := copy(b[n:], r.piece)
		n += nn
		if nn < len(r.piece) {
			// 버퍼의 크기가 작은 경우, 남은 데이터는 읽지 않고 버퍼를 반환합니다.
			r.piece = r.piece[nn:]
			return n, nil
		}
		r.piece = nil
	}
}

// next는 읽을 다음 데이터 조각을 반환합니다.
// EOF에서는 nil을 반환합니다.
func (r *encReader) next() []byte {
	switch {
	case r.buf == nil:
		return nil

	case r.piece != nil:
		// 읽을 데이터가 남아있는 경우
		return r.piece

	case r.lhpos < len(r.buf.lheads):
		// 마지막 리스트 헤더 이전의 경우
		head := r.buf.lheads[r.lhpos]
		sizebefore := head.offset - r.strpos
		if sizebefore > 0 {
			// 헤더 전에 문자열 데이터가 있는 경우
			p := r.buf.str[r.strpos:head.offset]
			r.strpos += sizebefore
			return p
		}
		r.lhpos++
		return head.encode(r.buf.sizebuf[:])

	case r.strpos < len(r.buf.str):
		// 모든 리스트 헤더 이후의 문자열 데이터가 있는 경우
		p := r.buf.str[r.strpos:]
		r.strpos = len(r.buf.str)
		return p

	default:
		return nil
	}
}

func encBufferFromWriter(w io.Writer) *encBuffer {
	switch w := w.(type) {
	case EncoderBuffer:
		return w.buf
	case *EncoderBuffer:
		return w.buf
	case *encBuffer:
		return w
	default:
		return nil
	}
}

// EncoderBuffer는 점진적 인코딩을 위한 버퍼입니다.
//
// 각 타입의 zero value는 사용할 수 없습니다.
// 사용 가능한 버퍼를 얻으려면 NewEncoderBuffer를 사용하거나 Reset을 호출하십시오.
type EncoderBuffer struct {
	buf *encBuffer
	dst io.Writer

	ownBuffer bool
}

// NewEncoderBuffer는 인코더 버퍼를 생성합니다.
func NewEncoderBuffer(dst io.Writer) EncoderBuffer {
	var w EncoderBuffer
	w.Reset(dst)
	return w
}

// Reset은 버퍼를 비우고 출력 대상을 새로 설정합니다.
func (w *EncoderBuffer) Reset(dst io.Writer) {
	if w.buf != nil && !w.ownBuffer {
		panic("can't Reset derived EncoderBuffer")
	}

	// 출력 대상이 *encBuffer를 가지고 있는 경우, 그것을 사용합니다.
	// w.ownBuffer는 여기서 false로 남겨집니다.
	if dst != nil {
		if outer := encBufferFromWriter(dst); outer != nil {
			*w = EncoderBuffer{outer, nil, false}
			return
		}
	}

	// 풀에서 *encBuffer를 가져옵니다.
	// w.ownBuffer는 여기서 true로 지정됩니다.
	if w.buf == nil {
		w.buf = encBufferPool.Get().(*encBuffer)
		w.ownBuffer = true
	}
	w.buf.reset()
	w.dst = dst
}

// Flush는 인코딩된 RLP 데이터를 출력 writer에 씁니다. 한 번만 호출할 수 있습니다.
// Flush 이후에 버퍼를 재사용하려면 Reset을 호출해야 합니다.
func (w *EncoderBuffer) Flush() error {
	var err error
	if w.dst != nil {
		err = w.buf.writeTo(w.dst)
	}
	// 내부 버퍼를 해제합니다.
	if w.ownBuffer {
		encBufferPool.Put(w.buf)
	}
	*w = EncoderBuffer{}
	return err
}

// ToBytes는 인코딩된 바이트를 반환합니다.
func (w *EncoderBuffer) ToBytes() []byte {
	return w.buf.makeBytes()
}

// AppendToBytes는 인코딩된 바이트를 dst에 추가합니다.
func (w *EncoderBuffer) AppendToBytes(dst []byte) []byte {
	size := w.buf.size()
	out := append(dst, make([]byte, size)...)
	w.buf.copyTo(out[len(dst):])
	return out
}

// Write는 b를 직접 인코더 출력에 추가합니다.
func (w EncoderBuffer) Write(b []byte) (int, error) {
	return w.buf.Write(b)
}

// WriteBool는 b를 정수 0 (false) 또는 1 (true)로 씁니다.
func (w EncoderBuffer) WriteBool(b bool) {
	w.buf.writeBool(b)
}

// WriteUint64은 부호 없는 정수를 인코딩합니다.
func (w EncoderBuffer) WriteUint64(i uint64) {
	w.buf.writeUint64(i)
}

// WriteBigInt는 big.Int를 RLP 문자열로 인코딩합니다.
// 참고: Encode와 달리 i의 부호는 무시됩니다.
func (w EncoderBuffer) WriteBigInt(i *big.Int) {
	w.buf.writeBigInt(i)
}

// WriteUint256는 uint256.Int를 RLP 문자열로 인코딩합니다.
func (w EncoderBuffer) WriteUint256(i *uint256.Int) {
	w.buf.writeUint256(i)
}

// WriteBytes는 b를 RLP 문자열로 인코딩합니다.
func (w EncoderBuffer) WriteBytes(b []byte) {
	w.buf.writeBytes(b)
}

// WriteString은 s를 RLP 문자열로 인코딩합니다.
func (w EncoderBuffer) WriteString(s string) {
	w.buf.writeString(s)
}

// List는 리스트를 시작합니다. 내부 인덱스를 반환합니다.
// 리스트의 내용을 인코딩한 후에 EndList를 호출하여 리스트를 마무리합니다.
func (w EncoderBuffer) List() int {
	return w.buf.list()
}

// ListEnd는 주어진 리스트를 마무리합니다.
func (w EncoderBuffer) ListEnd(index int) {
	w.buf.listEnd(index)
}
