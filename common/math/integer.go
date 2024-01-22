// Copyright 2017 The go-ethereum Authors
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

package math

import (
	"fmt"
	"math/bits"
	"strconv"
)

// 정수형의 임계값을 정의한다.
const (
	MaxInt8   = 1<<7 - 1
	MinInt8   = -1 << 7
	MaxInt16  = 1<<15 - 1
	MinInt16  = -1 << 15
	MaxInt32  = 1<<31 - 1
	MinInt32  = -1 << 31
	MaxInt64  = 1<<63 - 1
	MinInt64  = -1 << 63
	MaxUint8  = 1<<8 - 1
	MaxUint16 = 1<<16 - 1
	MaxUint32 = 1<<32 - 1
	MaxUint64 = 1<<64 - 1
)

// HexOrDecimal64는 uint64를 16진수 또는 10진수로 변환합니다.
type HexOrDecimal64 uint64

// UnmarshalJSON은 json.Unmarshaler를 구현합니다.
//
// UnmarshalText와 유사하지만, 따옴표로 묶인 10진수 문자열 뿐만 아니라 실제 10진수도 파싱할 수 있습니다.
func (i *HexOrDecimal64) UnmarshalJSON(input []byte) error {
	if len(input) > 0 && input[0] == '"' {
		input = input[1 : len(input)-1]
	}
	return i.UnmarshalText(input)
}

// UnmarshalText는 encoding.TextUnmarshaler를 구현합니다.
func (i *HexOrDecimal64) UnmarshalText(input []byte) error {
	int, ok := ParseUint64(string(input))
	if !ok {
		return fmt.Errorf("invalid hex or decimal integer %q", input)
	}
	*i = HexOrDecimal64(int)
	return nil
}

// MarshalText는 encoding.TextMarshaler를 구현합니다.
func (i HexOrDecimal64) MarshalText() ([]byte, error) {
	return []byte(fmt.Sprintf("%#x", uint64(i))), nil
}

// ParseUint64는 10진수 또는 16진수 구문으로 s를 정수로 파싱합니다.
// 앞에 0이 붙어있어도 괜찮습니다. 빈 문자열은 0으로 파싱됩니다.
func ParseUint64(s string) (uint64, bool) {
	if s == "" {
		return 0, true
	}
	if len(s) >= 2 && (s[:2] == "0x" || s[:2] == "0X") {
		v, err := strconv.ParseUint(s[2:], 16, 64)
		return v, err == nil
	}
	v, err := strconv.ParseUint(s, 10, 64)
	return v, err == nil
}

// MustParseUint64는 s를 정수로 파싱하고, 문자열이 유효하지 않으면 패닉합니다.
func MustParseUint64(s string) uint64 {
	v, ok := ParseUint64(s)
	if !ok {
		panic("invalid unsigned 64 bit integer: " + s)
	}
	return v
}

// SafeSub는 x-y가 오버플로우가 발생하는지 확인하고, x-y를 반환합니다.
func SafeSub(x, y uint64) (uint64, bool) {
	diff, borrowOut := bits.Sub64(x, y, 0)
	return diff, borrowOut != 0
}

// SafeAdd는 x+y가 오버플로우가 발생하는지 확인하고, x+y를 반환합니다.
func SafeAdd(x, y uint64) (uint64, bool) {
	sum, carryOut := bits.Add64(x, y, 0)
	return sum, carryOut != 0
}

// SafeMul는 x*y가 오버플로우가 발생하는지 확인하고, x*y를 반환합니다.
func SafeMul(x, y uint64) (uint64, bool) {
	hi, lo := bits.Mul64(x, y)
	return lo, hi != 0
}
