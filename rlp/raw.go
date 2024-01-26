// Copyright 2015 The go-ethereum Authors
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
	"io"
	"reflect"
)

// RawValue는 인코딩된 RLP 값을 나타내며 RLP 디코딩을 지연하거나 인코딩을 사전에 계산하는 데 사용할 수 있습니다.
// 디코더는 RawValues의 내용이 유효한 RLP인지 확인하지 않습니다.
type RawValue []byte

var rawValueType = reflect.TypeOf(RawValue{})

// StringSize는 문자열의 인코딩된 크기를 반환합니다.
func StringSize(s string) uint64 {
	switch {
	case len(s) == 0:
		return 1
	case len(s) == 1:
		if s[0] <= 0x7f {
			return 1 // 0x00 ~ 0x7f
		} else {
			return 2 // 0x80 ~ 0xff
		}
	default:
		return uint64(headsize(uint64(len(s))) + len(s))
	}
}

// BytesSize는 바이트 슬라이스의 인코딩된 크기를 반환합니다.
func BytesSize(b []byte) uint64 {
	switch {
	case len(b) == 0:
		return 1
	case len(b) == 1:
		if b[0] <= 0x7f { // 0x00 ~ 0x7f
			return 1
		} else {
			return 2
		}
	default:
		return uint64(headsize(uint64(len(b))) + len(b))
	}
}

// ListSize는 주어진 contentSize를 가진 RLP 리스트의 인코딩된 크기를 반환합니다.
func ListSize(contentSize uint64) uint64 {
	return uint64(headsize(contentSize)) + contentSize
}

// IntSize는 정수 x의 인코딩된 크기를 반환합니다. 참고: 이 함수의 반환 유형은
// 이전 버전과의 호환성을 위해 'int'입니다. 결과는 항상 양수입니다.
func IntSize(x uint64) int {
	if x < 0x80 {
		return 1
	}
	return 1 + intsize(x)
}

// Split는 첫 번째 RLP 값의 내용과 해당 값 이후의 모든 바이트를 rest로 반환합니다.
func Split(b []byte) (k Kind, content, rest []byte, err error) {
	k, ts, cs, err := readKind(b)
	if err != nil {
		return 0, nil, b, err
	}
	return k, b[ts : ts+cs], b[ts+cs:], nil
}

// SplitString은 b를 RLP 문자열의 내용과 문자열 이후의 모든 바이트로 분할합니다.
// SplitString은 b가 RLP 문자열이 아닌 경우 오류를 반환합니다.
func SplitString(b []byte) (content, rest []byte, err error) {
	k, content, rest, err := Split(b)
	if err != nil {
		return nil, b, err
	}
	if k == List {
		return nil, b, ErrExpectedString
	}
	return content, rest, nil
}

// SplitUint64는 b의 시작 부분에 있는 정수를 디코딩합니다.
// 또한 디코딩하고 남은 데이터를 'rest'에 반환합니다.
func SplitUint64(b []byte) (x uint64, rest []byte, err error) {
	content, rest, err := SplitString(b)
	if err != nil {
		return 0, b, err
	}
	switch {
	case len(content) == 0:
		return 0, rest, nil
	case len(content) == 1:
		if content[0] == 0 {
			return 0, b, ErrCanonInt
		}
		return uint64(content[0]), rest, nil
	case len(content) > 8:
		return 0, b, errUintOverflow
	default:
		x, err = readSize(content, byte(len(content)))
		if err != nil {
			return 0, b, ErrCanonInt
		}
		return x, rest, nil
	}
}

// SplitList는 b를 리스트의 내용과 그 외의 나머지 바이트로 분할합니다.
func SplitList(b []byte) (content, rest []byte, err error) {
	k, content, rest, err := Split(b)
	if err != nil {
		return nil, b, err
	}
	if k != List {
		return nil, b, ErrExpectedList
	}
	return content, rest, nil
}

// CountValues는 b에 인코딩된 값의 개수를 계산합니다.
func CountValues(b []byte) (int, error) {
	i := 0
	for ; len(b) > 0; i++ {
		_, tagsize, size, err := readKind(b)
		if err != nil {
			return 0, err
		}
		b = b[tagsize+size:]
	}
	return i, nil
}

func readKind(buf []byte) (k Kind, tagsize, contentsize uint64, err error) {
	if len(buf) == 0 {
		return 0, 0, 0, io.ErrUnexpectedEOF
	}
	b := buf[0]
	switch {
	case b < 0x80: // 단일 바이트 문자열
		k = Byte
		tagsize = 0
		contentsize = 1
	case b < 0xB8: // 길이가 55바이트 이하인 문자열
		k = String
		tagsize = 1
		contentsize = uint64(b - 0x80)
		// 단일 바이트여야 하는 문자열 거부
		if contentsize == 1 && len(buf) > 1 && buf[1] < 128 {
			return 0, 0, 0, ErrCanonSize
		}
	case b < 0xC0: // 길이가 55바이트를 초과하는 문자열
		k = String
		tagsize = uint64(b-0xB7) + 1
		contentsize, err = readSize(buf[1:], b-0xB7)
	case b < 0xF8: // 길이가 55바이트 이하인 리스트
		k = List
		tagsize = 1
		contentsize = uint64(b - 0xC0)
	default: // 길이가 55바이트를 초과하는 리스트
		k = List
		tagsize = uint64(b-0xF7) + 1
		contentsize, err = readSize(buf[1:], b-0xF7)
	}
	if err != nil {
		return 0, 0, 0, err
	}
	// 입력 슬라이스보다 큰 값 거부
	if contentsize > uint64(len(buf))-tagsize {
		return 0, 0, 0, ErrValueTooLarge
	}
	return k, tagsize, contentsize, err
}

func readSize(b []byte, slen byte) (uint64, error) {
	if int(slen) > len(b) {
		return 0, io.ErrUnexpectedEOF
	}
	var s uint64
	switch slen {
	case 1:
		s = uint64(b[0])
	case 2:
		s = uint64(b[0])<<8 | uint64(b[1])
	case 3:
		s = uint64(b[0])<<16 | uint64(b[1])<<8 | uint64(b[2])
	case 4:
		s = uint64(b[0])<<24 | uint64(b[1])<<16 | uint64(b[2])<<8 | uint64(b[3])
	case 5:
		s = uint64(b[0])<<32 | uint64(b[1])<<24 | uint64(b[2])<<16 | uint64(b[3])<<8 | uint64(b[4])
	case 6:
		s = uint64(b[0])<<40 | uint64(b[1])<<32 | uint64(b[2])<<24 | uint64(b[3])<<16 | uint64(b[4])<<8 | uint64(b[5])
	case 7:
		s = uint64(b[0])<<48 | uint64(b[1])<<40 | uint64(b[2])<<32 | uint64(b[3])<<24 | uint64(b[4])<<16 | uint64(b[5])<<8 | uint64(b[6])
	case 8:
		s = uint64(b[0])<<56 | uint64(b[1])<<48 | uint64(b[2])<<40 | uint64(b[3])<<32 | uint64(b[4])<<24 | uint64(b[5])<<16 | uint64(b[6])<<8 | uint64(b[7])
	}
	// 56보다 작은 크기 거부(별도의 크기가 없어야 함) 및 선행 0바이트가 있는 크기 거부
	if s < 56 || b[0] == 0 {
		return 0, ErrCanonSize
	}
	return s, nil
}

// AppendUint64은 uint64 i의 RLP 인코딩을 b에 추가하고 결과 슬라이스를 반환합니다.
func AppendUint64(b []byte, i uint64) []byte {
	if i == 0 {
		return append(b, 0x80)
	} else if i < 128 {
		return append(b, byte(i))
	}
	switch {
	case i < (1 << 8):
		return append(b, 0x81, byte(i))
	case i < (1 << 16):
		return append(b, 0x82,
			byte(i>>8),
			byte(i),
		)
	case i < (1 << 24):
		return append(b, 0x83,
			byte(i>>16),
			byte(i>>8),
			byte(i),
		)
	case i < (1 << 32):
		return append(b, 0x84,
			byte(i>>24),
			byte(i>>16),
			byte(i>>8),
			byte(i),
		)
	case i < (1 << 40):
		return append(b, 0x85,
			byte(i>>32),
			byte(i>>24),
			byte(i>>16),
			byte(i>>8),
			byte(i),
		)

	case i < (1 << 48):
		return append(b, 0x86,
			byte(i>>40),
			byte(i>>32),
			byte(i>>24),
			byte(i>>16),
			byte(i>>8),
			byte(i),
		)
	case i < (1 << 56):
		return append(b, 0x87,
			byte(i>>48),
			byte(i>>40),
			byte(i>>32),
			byte(i>>24),
			byte(i>>16),
			byte(i>>8),
			byte(i),
		)

	default:
		return append(b, 0x88,
			byte(i>>56),
			byte(i>>48),
			byte(i>>40),
			byte(i>>32),
			byte(i>>24),
			byte(i>>16),
			byte(i>>8),
			byte(i),
		)
	}
}
