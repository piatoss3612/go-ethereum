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

// math 패키지는 정수 계산 도우미 함수를 제공합니다.
package math

import (
	"fmt"
	"math/big"
)

// 큰 정수로 표현된 여러 가지 임계값
var (
	tt255     = BigPow(2, 255)                         // 2^255
	tt256     = BigPow(2, 256)                         // 2^256
	tt256m1   = new(big.Int).Sub(tt256, big.NewInt(1)) // 2^256 - 1
	tt63      = BigPow(2, 63)                          // 2^63
	MaxBig256 = new(big.Int).Set(tt256m1)              // 2^256 - 1
	MaxBig63  = new(big.Int).Sub(tt63, big.NewInt(1))  // 2^63 - 1
)

const (
	// big.Word의 비트 수 (64비트 아키텍처에서 64, 32비트 아키텍처에서 32)
	wordBits = 32 << (uint64(^big.Word(0)) >> 63)
	// big.Word의 바이트 수 (64비트 아키텍처에서 8, 32비트 아키텍처에서 4)
	wordBytes = wordBits / 8
)

// HexOrDecimal256은 big.Int를 16진수 또는 10진수로 변환합니다.
type HexOrDecimal256 big.Int

// NewHexOrDecimal256는 새로운 HexOrDecimal256을 생성합니다.
func NewHexOrDecimal256(x int64) *HexOrDecimal256 {
	b := big.NewInt(x)
	h := HexOrDecimal256(*b)
	return &h
}

// UnmarshalJSON은 json.Unmarshaler를 구현합니다.
//
// UnmarshalText와 유사하지만 실제 10진수를 파싱할 수 있습니다. 따옴표로 묶인 10진수 문자열 뿐만 아니라 실제 10진수도 파싱할 수 있습니다.
func (i *HexOrDecimal256) UnmarshalJSON(input []byte) error {
	if len(input) > 0 && input[0] == '"' {
		input = input[1 : len(input)-1]
	}
	return i.UnmarshalText(input)
}

// UnmarshalText는 encoding.TextUnmarshaler를 구현합니다.
func (i *HexOrDecimal256) UnmarshalText(input []byte) error {
	bigint, ok := ParseBig256(string(input))
	if !ok {
		return fmt.Errorf("invalid hex or decimal integer %q", input)
	}
	*i = HexOrDecimal256(*bigint)
	return nil
}

// MarshalText는 encoding.TextMarshaler를 구현합니다.
func (i *HexOrDecimal256) MarshalText() ([]byte, error) {
	if i == nil {
		return []byte("0x0"), nil
	}
	return []byte(fmt.Sprintf("%#x", (*big.Int)(i))), nil
}

// Decimal256은 big.Int를 10진수 문자열로 변환하는데, "0x" 접두사가 붙은 16진수 문자열도 허용합니다.
type Decimal256 big.Int

// NewDecimal256은 새로운 Decimal256을 생성합니다.
func NewDecimal256(x int64) *Decimal256 {
	b := big.NewInt(x)
	d := Decimal256(*b)
	return &d
}

// UnmarshalText는 encoding.TextUnmarshaler를 구현합니다.
func (i *Decimal256) UnmarshalText(input []byte) error {
	bigint, ok := ParseBig256(string(input))
	if !ok {
		return fmt.Errorf("invalid hex or decimal integer %q", input)
	}
	*i = Decimal256(*bigint)
	return nil
}

// MarshalText는 encoding.TextMarshaler를 구현합니다.
func (i *Decimal256) MarshalText() ([]byte, error) {
	return []byte(i.String()), nil
}

// String은 Stringer를 구현합니다.
func (i *Decimal256) String() string {
	if i == nil {
		return "0"
	}
	return fmt.Sprintf("%#d", (*big.Int)(i))
}

// ParseBig256는 10진수 또는 16진수 구문으로 s를 256비트 정수로 파싱합니다.
// 앞에 0이 붙어있어도 상관없습니다. 빈 문자열은 0으로 파싱됩니다.
func ParseBig256(s string) (*big.Int, bool) {
	if s == "" {
		return new(big.Int), true
	}
	var bigint *big.Int
	var ok bool
	if len(s) >= 2 && (s[:2] == "0x" || s[:2] == "0X") {
		bigint, ok = new(big.Int).SetString(s[2:], 16)
	} else {
		bigint, ok = new(big.Int).SetString(s, 10)
	}
	if ok && bigint.BitLen() > 256 {
		bigint, ok = nil, false
	}
	return bigint, ok
}

// MustParseBig256는 s를 256비트 큰 정수로 파싱하고, 문자열이 유효하지 않으면 패닉을 발생시킵니다.
func MustParseBig256(s string) *big.Int {
	v, ok := ParseBig256(s)
	if !ok {
		panic("invalid 256 bit integer: " + s)
	}
	return v
}

// BigPow는 a ** b를 큰 정수로 반환합니다.
func BigPow(a, b int64) *big.Int {
	r := big.NewInt(a)
	return r.Exp(r, big.NewInt(b), nil)
}

// BigMax는 x와 y 중 더 큰 값을 반환합니다.
func BigMax(x, y *big.Int) *big.Int {
	if x.Cmp(y) < 0 {
		return y
	}
	return x
}

// BigMin은 x와 y 중 더 작은 값을 반환합니다.
func BigMin(x, y *big.Int) *big.Int {
	if x.Cmp(y) > 0 {
		return y
	}
	return x
}

// FirstBitSet는 최하위 비트부터 시작하여 v의 첫 번째 1 비트의 인덱스를 반환합니다.
func FirstBitSet(v *big.Int) int {
	for i := 0; i < v.BitLen(); i++ {
		if v.Bit(i) > 0 {
			return i
		}
	}
	return v.BitLen()
}

// PaddedBigBytes는 큰 정수를 빅 엔디언 바이트 슬라이스로 인코딩합니다. 슬라이스의 길이는 최소 n바이트입니다.
func PaddedBigBytes(bigint *big.Int, n int) []byte {
	if bigint.BitLen()/8 >= n {
		return bigint.Bytes()
	}
	ret := make([]byte, n)
	ReadBits(bigint, ret)
	return ret
}

// bigEndianByteAt는 빅 엔디언 인코딩에서 위치 n의 바이트를 반환합니다.
// n==0일 경우 최하위 바이트를 반환합니다.
func bigEndianByteAt(bigint *big.Int, n int) byte {
	words := bigint.Bits()
	// 바이트가 속할 word-bucket을 확인합니다.
	i := n / wordBytes
	if i >= len(words) {
		return byte(0)
	}
	word := words[i]
	// 바이트의 오프셋
	shift := 8 * uint(n%wordBytes)

	return byte(word >> shift)
}

// Byte는 리틀 엔디언에서 padlength가 될 때까지 패딩된 바이트열의 n번째 바이트를 반환합니다.
// n==0인 경우 최상위 바이트를 반환합니다.
// 예시: bigint '5', padlength 32, n=31 => 5
func Byte(bigint *big.Int, padlength, n int) byte {
	if n >= padlength {
		return byte(0)
	}
	return bigEndianByteAt(bigint, padlength-1-n)
}

// ReadBits는 bigint의 절대값을 빅 엔디언 바이트로 인코딩합니다. 호출자는 buf에 충분한 공간이 있는지 확인해야 합니다.
// buf가 너무 짧으면 결과가 불완전할 수 있습니다.
func ReadBits(bigint *big.Int, buf []byte) {
	i := len(buf)
	for _, d := range bigint.Bits() {
		for j := 0; j < wordBytes && i > 0; j++ {
			i--
			buf[i] = byte(d)
			d >>= 8
		}
	}
}

// U256은 큰 정수를 256비트 2의 보수 숫자로 인코딩합니다. 이 연산은 파괴적(원본값을 변경)입니다.
func U256(x *big.Int) *big.Int {
	return x.And(x, tt256m1)
}

// U256Bytes는 큰 정수를 256비트 EVM 숫자로 인코딩합니다. 이 연산은 파괴적(원본값을 변경)입니다.
func U256Bytes(n *big.Int) []byte {
	return PaddedBigBytes(U256(n), 32)
}

// S256은 2의 보수 x를 Signed 256비트 숫자로 인코딩합니다.
// x는 256비트를 초과해서는 안됩니다(그렇게 되면 결과는 정의되지 않음). 이 연산은 파괴적(원본값을 변경)이지 않습니다.
//
//	S256(0)        = 0
//	S256(1)        = 1
//	S256(2**255)   = -2**255
//	S256(2**256-1) = -1
func S256(x *big.Int) *big.Int {
	if x.Cmp(tt255) < 0 {
		return x
	}
	return new(big.Int).Sub(x, tt256)
}

// Exp는 제곱을 통한 지수 연산을 구현합니다.
// Exp는 새로운 큰 정수를 반환하며, base 또는 exponent를 변경하지 않습니다. 결과는 256비트로 잘립니다.
//
// Courtesy @karalabe and @chfast
func Exp(base, exponent *big.Int) *big.Int {
	result := big.NewInt(1)

	for _, word := range exponent.Bits() {
		for i := 0; i < wordBits; i++ {
			if word&1 == 1 {
				U256(result.Mul(result, base))
			}
			U256(base.Mul(base, base))
			word >>= 1
		}
	}
	return result
}
