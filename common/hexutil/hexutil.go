// Copyright 2016 The go-ethereum Authors
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

/*
hexutil 패키지는 0x 접두사가 있는 16진수 인코딩을 구현합니다.
hexutil 인코딩은 이더리움 RPC API에서 바이너리 데이터를 JSON 형식의 페이로드로 전송하는 데 사용됩니다.

# 인코딩 규칙

모든 16진수 데이터는 "0x" 접두사를 가져야 합니다.

바이트 슬라이스의 경우 16진수 데이터는 짝수 길이여야 합니다. 빈 바이트 슬라이스는 "0x"로 인코딩됩니다.

정수는 최소한의 숫자를 사용하여 인코딩됩니다.(앞에 0이 붙지 않은 숫자). 그들의 인코딩은 홀수 길이일 수 있습니다. 숫자 0은 "0x0"으로 인코딩됩니다.
*/
package hexutil

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
)

const uintBits = 32 << (uint64(^uint(0)) >> 63) // 64

// Errors
var (
	ErrEmptyString   = &decError{"empty hex string"}
	ErrSyntax        = &decError{"invalid hex string"}
	ErrMissingPrefix = &decError{"hex string without 0x prefix"}
	ErrOddLength     = &decError{"hex string of odd length"}
	ErrEmptyNumber   = &decError{"hex string \"0x\""}
	ErrLeadingZero   = &decError{"hex number with leading zero digits"}
	ErrUint64Range   = &decError{"hex number > 64 bits"}
	ErrUintRange     = &decError{fmt.Sprintf("hex number > %d bits", uintBits)}
	ErrBig256Range   = &decError{"hex number > 256 bits"}
)

type decError struct{ msg string }

func (err decError) Error() string { return err.msg }

// Decode는 0x 접두사가 있는 16진수 문자열을 바이트열로 디코딩합니다.
func Decode(input string) ([]byte, error) {
	if len(input) == 0 {
		return nil, ErrEmptyString
	}
	if !has0xPrefix(input) {
		return nil, ErrMissingPrefix
	}
	b, err := hex.DecodeString(input[2:])
	if err != nil {
		err = mapError(err)
	}
	return b, err
}

// MustDecode는 0x 접두사가 있는 16진수 문자열을 바이트열로 디코딩합니다. 잘못된 입력에 대해서는 패닉이 발생합니다.
func MustDecode(input string) []byte {
	dec, err := Decode(input)
	if err != nil {
		panic(err)
	}
	return dec
}

// Encode는 0x 접두사가 있는 16진수 문자열로 b를 인코딩합니다.
func Encode(b []byte) string {
	enc := make([]byte, len(b)*2+2)
	copy(enc, "0x")
	hex.Encode(enc[2:], b)
	return string(enc)
}

// DecodeUint64는 0x 접두사가 있는 16진수 문자열을 숫자로 디코딩합니다.
func DecodeUint64(input string) (uint64, error) {
	raw, err := checkNumber(input)
	if err != nil {
		return 0, err
	}
	dec, err := strconv.ParseUint(raw, 16, 64)
	if err != nil {
		err = mapError(err)
	}
	return dec, err
}

// MustDecodeUint64는 0x 접두사가 있는 16진수 문자열을 숫자로 디코딩합니다. 잘못된 입력에 대해서는 패닉이 발생합니다.
func MustDecodeUint64(input string) uint64 {
	dec, err := DecodeUint64(input)
	if err != nil {
		panic(err)
	}
	return dec
}

// EncodeUint64는 0x 접두사가 있는 16진수 문자열로 i를 인코딩합니다.
func EncodeUint64(i uint64) string {
	enc := make([]byte, 2, 10)
	copy(enc, "0x")
	return string(strconv.AppendUint(enc, i, 16))
}

var bigWordNibbles int // big.Word에 필요한 nibble 수 (16 또는 8), init() 함수에서 초기화됩니다.

func init() {
	// 이것은 big.Word에 필요한 nibble 수를 계산하는 이상한 방법입니다.
	// 일반적인 방법은 상수를 사용한 산술 연산이지만, go vet은 그러한 방법을 처리할 수 없습니다.
	b, _ := new(big.Int).SetString("FFFFFFFFFF", 16)
	switch len(b.Bits()) {
	case 1:
		bigWordNibbles = 16 // 32 / 2
	case 2:
		bigWordNibbles = 8 // 16 / 2
	default:
		panic("weird big.Word size")
	}
}

// DecodeBig decodes a hex string with 0x prefix as a quantity.
// Numbers larger than 256 bits are not accepted.

// DecodeBig는 0x 접두사가 있는 16진수 문자열을 big.Int로 디코딩합니다.
// 256비트보다 큰 숫자는 허용되지 않습니다.
func DecodeBig(input string) (*big.Int, error) {
	raw, err := checkNumber(input)
	if err != nil {
		return nil, err
	}
	if len(raw) > 64 {
		return nil, ErrBig256Range
	}
	words := make([]big.Word, len(raw)/bigWordNibbles+1)
	end := len(raw)
	for i := range words {
		start := end - bigWordNibbles
		if start < 0 {
			start = 0
		}
		for ri := start; ri < end; ri++ {
			nib := decodeNibble(raw[ri])
			if nib == badNibble {
				return nil, ErrSyntax
			}
			words[i] *= 16
			words[i] += big.Word(nib)
		}
		end = start
	}
	dec := new(big.Int).SetBits(words)
	return dec, nil
}

// MustDecodeBig은 0x 접두사가 있는 16진수 문자열을 big.Int로 디코딩합니다.
// 잘못된 입력에 대해서는 패닉이 발생합니다.
func MustDecodeBig(input string) *big.Int {
	dec, err := DecodeBig(input)
	if err != nil {
		panic(err)
	}
	return dec
}

// EncodeBig은 bigint를 0x 접두사가 있는 16진수 문자열로 인코딩합니다.
func EncodeBig(bigint *big.Int) string {
	if sign := bigint.Sign(); sign == 0 {
		return "0x0"
	} else if sign > 0 {
		return "0x" + bigint.Text(16)
	} else {
		return "-0x" + bigint.Text(16)[1:]
	}
}

// has0xPrefix는 input이 0x 접두사를 가지고 있는지 확인합니다.
func has0xPrefix(input string) bool {
	return len(input) >= 2 && input[0] == '0' && (input[1] == 'x' || input[1] == 'X')
}

// checkNumber는 0x 접두사가 있는 16진수 문자열이 정수로 디코딩될 수 있는지 확인합니다.
func checkNumber(input string) (raw string, err error) {
	if len(input) == 0 {
		return "", ErrEmptyString
	}
	if !has0xPrefix(input) {
		return "", ErrMissingPrefix
	}
	input = input[2:]
	if len(input) == 0 {
		return "", ErrEmptyNumber
	}
	if len(input) > 1 && input[0] == '0' {
		return "", ErrLeadingZero
	}
	return input, nil
}

const badNibble = ^uint64(0) // 64비트의 모든 비트가 1인 상수

// decodeNibble는 16진수 문자를 정수로 디코딩합니다.
func decodeNibble(in byte) uint64 {
	switch {
	case in >= '0' && in <= '9':
		return uint64(in - '0')
	case in >= 'A' && in <= 'F':
		return uint64(in - 'A' + 10)
	case in >= 'a' && in <= 'f':
		return uint64(in - 'a' + 10)
	default:
		return badNibble
	}
}

// mapError는 strconv 또는 hex 패키지에서 발생한 오류를 hexutil 패키지에서 정의한 오류로 매핑합니다.
func mapError(err error) error {
	if err, ok := err.(*strconv.NumError); ok {
		switch err.Err {
		case strconv.ErrRange:
			return ErrUint64Range
		case strconv.ErrSyntax:
			return ErrSyntax
		}
	}
	if _, ok := err.(hex.InvalidByteError); ok {
		return ErrSyntax
	}
	if err == hex.ErrLength {
		return ErrOddLength
	}
	return err
}
