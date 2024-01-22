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

package bitutil

import "errors"

var (
	// errMissingData는 비트셋 헤더가 참조하는 바이트가 입력 데이터의 길이를 초과하는 위치를 가리킬 때, 압축 해제에서 반환됩니다.
	errMissingData = errors.New("missing bytes on input")

	// errUnreferencedData는 압축 해제에서 모든 바이트가 사용되지 않은 경우 반환됩니다.
	errUnreferencedData = errors.New("extra bytes on input")

	// errExceededTarget는 비트셋 헤더에 대상 버퍼 공간보다 더 많은 비트가 정의된 경우, 압축 해제에서 반환됩니다.
	errExceededTarget = errors.New("target data size exceeded")

	// errZeroContent는 압축 해제에서, 압축된 데이터 바이트가 0인 경우(잘못된 압축) 반환됩니다.
	errZeroContent = errors.New("zero byte in input content")
)

// CompressBytes와 DecompressBytes가 구현한 압축 알고리즘은 많은 zero 바이트값을 포함하는 희소 입력 데이터에 최적화되어 있습니다.
// 이 알고리즘은 비압축 데이터 길이를 알아야만 압축을 해제할 수 있습니다.
//
// 압축은 다음과 같이 작동합니다.
//
//   if data only contains zeroes,
//       CompressBytes(data) == nil
//   otherwise if len(data) <= 1,
//       CompressBytes(data) == data
//   otherwise:
//       CompressBytes(data) == append(CompressBytes(nonZeroBitset(data)), nonZeroBytes(data)...)
//       where
//         nonZeroBitset(data) is a bit vector with len(data) bits (MSB first):
//             nonZeroBitset(data)[i/8] && (1 << (7-i%8)) != 0  if data[i] != 0
//             len(nonZeroBitset(data)) == (len(data)+7)/8
//         nonZeroBytes(data) contains the non-zero bytes of data in the same order

// CompressBytes는 희소 비트셋 알고리즘(sparse bitset algorithm)에 따라 입력 바이트 슬라이스를 압축합니다.
// 결과가 원래 입력보다 큰 경우 압축이 수행되지 않습니다.
func CompressBytes(data []byte) []byte {
	if out := bitsetEncodeBytes(data); len(out) < len(data) {
		return out
	}
	cpy := make([]byte, len(data)) // 길이가 길거나 같은 경우 원래 데이터를 복사하여 반환합니다.
	copy(cpy, data)
	return cpy
}

// bitsetEncodeBytes는 희소 비트셋 알고리즘(sparse bitset algorithm)에 따라 입력 바이트 슬라이스를 압축합니다.
func bitsetEncodeBytes(data []byte) []byte {
	// 빈 슬라이스는 nil로 압축됩니다.
	if len(data) == 0 {
		return nil
	}
	// 바이트 슬라이스의 길이가 1이면 nil로 압축(0)되거나 단일 바이트를 유지합니다.
	if len(data) == 1 {
		if data[0] == 0 {
			return nil
		}
		return data
	}
	nonZeroBitset := make([]byte, (len(data)+7)/8) // 1바이트당 8비트를 사용합니다.
	nonZeroBytes := make([]byte, 0, len(data))     // 0이 아닌 바이트를 수집합니다.

	for i, b := range data {
		if b != 0 {
			nonZeroBytes = append(nonZeroBytes, b)
			nonZeroBitset[i/8] |= 1 << byte(7-i%8) // i번째 비트가 0이 아니므로, i/8번째 바이트의 (7-i%8)번째 비트를 1로 설정합니다.
		}
	}
	if len(nonZeroBytes) == 0 { // 모든 바이트가 0이면 nil로 압축합니다.
		return nil
	}
	return append(bitsetEncodeBytes(nonZeroBitset), nonZeroBytes...) // 재귀적으로 압축합니다.
}

// DecompressBytes는 주어진 타겟 사이즈로 데이터를 압축 해제합니다.
// 입력 데이터가 타겟 사이즈와 일치한다면, 압축이 수행되지 않았다는 것을 의미합니다.
func DecompressBytes(data []byte, target int) ([]byte, error) {
	if len(data) > target {
		return nil, errExceededTarget
	}
	if len(data) == target {
		cpy := make([]byte, len(data))
		copy(cpy, data)
		return cpy, nil
	}
	return bitsetDecodeBytes(data, target)
}

// bitsetDecodeBytes는 주어진 타겟 사이즈로 데이터를 압축 해제합니다.
func bitsetDecodeBytes(data []byte, target int) ([]byte, error) {
	out, size, err := bitsetDecodePartialBytes(data, target)
	if err != nil {
		return nil, err
	}
	if size != len(data) {
		return nil, errUnreferencedData
	}
	return out, nil
}

// bitsetDecodePartialBytes는 주어진 목표 크기로 데이터의 압축을 해제하지만, 모든 입력 바이트를 사용하도록 강제하지 않습니다.
// 이 함수는 압축 해제된 출력 외에도, 출력에 대응하는 압축된 입력 데이터의 길이도 반환합니다. (입력 슬라이스가 길 수 있기 때문입니다.)
func bitsetDecodePartialBytes(data []byte, target int) ([]byte, int, error) {
	// 무한 재귀를 피하기 위해 타겟이 0인 경우를 확인합니다.
	if target == 0 {
		return nil, 0, nil
	}
	// data가 비어있는 경우, 또는 target이 1인 경우를 처리합니다.
	decomp := make([]byte, target)
	if len(data) == 0 {
		return decomp, 0, nil
	}
	if target == 1 {
		decomp[0] = data[0] // 입력 슬라이스를 참조하지 않고, 갑을 복사합니다.
		if data[0] != 0 {
			return decomp, 1, nil
		}
		return decomp, 0, nil
	}
	// 비트셋을 압축 해제하고, 0이 아닌 바이트를 분리합니다.
	// Decompress the bitset of set bytes and distribute the non zero bytes
	nonZeroBitset, ptr, err := bitsetDecodePartialBytes(data, (target+7)/8)
	if err != nil {
		return nil, ptr, err
	}
	// 비트셋을 사용하여 0이 아닌 바이트를 복원합니다.
	for i := 0; i < 8*len(nonZeroBitset); i++ {
		if nonZeroBitset[i/8]&(1<<byte(7-i%8)) != 0 {
			// 입력 데이터가 부족하면 에러를 반환합니다.
			if ptr >= len(data) {
				return nil, 0, errMissingData
			}
			if i >= len(decomp) {
				return nil, 0, errExceededTarget
			}
			// 데이터는 0이 아니어야 합니다.
			if data[ptr] == 0 {
				return nil, 0, errZeroContent
			}
			decomp[i] = data[ptr]
			ptr++
		}
	}
	return decomp, ptr, nil
}
