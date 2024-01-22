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

// Package common contains various helper functions.
package common

import (
	"encoding/hex"
	"errors"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

// FromHex는 16진수 문자열 s로부터 바이트열을 반환합니다.
// s는 "0x"로 시작할 수 있습니다.
func FromHex(s string) []byte {
	if has0xPrefix(s) { // 0x로 시작하는 경우 - 0x 제거 (hex.DecodeString은 16진수 문자열이 아닌 경우 에러 반환 (x는 16진수가 아님))
		s = s[2:]
	}
	if len(s)%2 == 1 { // 홀수인 경우 - 앞에 0 추가
		s = "0" + s
	}
	return Hex2Bytes(s) // 16진수 문자열을 바이트열로 변환
}

// CopyBytes는 제공된 바이트열의 정확한 복사본을 반환합니다.
func CopyBytes(b []byte) (copiedBytes []byte) {
	if b == nil {
		return nil
	}
	copiedBytes = make([]byte, len(b))
	copy(copiedBytes, b)

	return
}

// has0xPrefix는 str이 '0x' 또는 '0X'로 시작하는지 확인합니다.
func has0xPrefix(str string) bool {
	return len(str) >= 2 && str[0] == '0' && (str[1] == 'x' || str[1] == 'X')
}

// isHexCharacter는 c가 유효한 16진수인지 여부를 반환합니다. (0~9, a~f, A~F)
func isHexCharacter(c byte) bool {
	return ('0' <= c && c <= '9') || ('a' <= c && c <= 'f') || ('A' <= c && c <= 'F')
}

// isHex는 각 바이트가 유효한 16진수 문자열인지 확인합니다.
func isHex(str string) bool {
	if len(str)%2 != 0 { // 문자열 길이가 홀수인 경우
		return false
	}
	for _, c := range []byte(str) {
		if !isHexCharacter(c) {
			return false
		}
	}
	return true
}

// Bytes2Hex는 바이트열 d의 16진수 인코딩을 반환합니다.
func Bytes2Hex(d []byte) string {
	return hex.EncodeToString(d)
}

// Hex2Bytes는 16진수 문자열 str로 표현된 바이트열을 반환합니다.
func Hex2Bytes(str string) []byte {
	h, _ := hex.DecodeString(str)
	return h
}

// Hex2BytesFixed는 지정된 고정 길이 flen 크기의 바이트열을 반환합니다.
func Hex2BytesFixed(str string, flen int) []byte {
	h, _ := hex.DecodeString(str)
	if len(h) == flen {
		return h
	}
	if len(h) > flen { // 길이가 더 긴 경우 - 뒤에서부터 flen만큼 잘라서 반환
		return h[len(h)-flen:]
	}
	hh := make([]byte, flen) // 길이가 더 짧은 경우 - 앞에 0 추가
	copy(hh[flen-len(h):flen], h)
	return hh
}

// ParseHexOrString는 문자열 str을 16진수로 디코딩하려고 시도하지만, 접두사가 누락된 경우 원시 바이트열을 반환합니다.
func ParseHexOrString(str string) ([]byte, error) {
	b, err := hexutil.Decode(str)
	if errors.Is(err, hexutil.ErrMissingPrefix) {
		return []byte(str), nil
	}
	return b, err
}

// RightPadBytes는 길이가 l이 될 때까지 slice를 오른쪽에 0을 채웁니다.
func RightPadBytes(slice []byte, l int) []byte {
	if l <= len(slice) { // 길이가 l보다 크거나 같은 경우 - 그대로 반환
		return slice
	}

	padded := make([]byte, l)
	copy(padded, slice)

	return padded
}

// LeftPadBytes는 길이가 l이 될 때까지 slice를 왼쪽에 0을 채웁니다.
func LeftPadBytes(slice []byte, l int) []byte {
	if l <= len(slice) { // 길이가 l보다 크거나 같은 경우 - 그대로 반환
		return slice
	}

	padded := make([]byte, l)
	copy(padded[l-len(slice):], slice)

	return padded
}

// TrimLeftZeroes는 s의 왼쪽에 있는 0을 제거한 subslice를 반환합니다.
func TrimLeftZeroes(s []byte) []byte {
	idx := 0
	for ; idx < len(s); idx++ {
		if s[idx] != 0 {
			break
		}
	}
	return s[idx:]
}

// TrimRightZeroes는 s의 오른쪽에 있는 0을 제거한 subslice를 반환합니다.
func TrimRightZeroes(s []byte) []byte {
	idx := len(s)
	for ; idx > 0; idx-- {
		if s[idx-1] != 0 {
			break
		}
	}
	return s[:idx]
}
