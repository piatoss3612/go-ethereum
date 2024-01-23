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

package types

import (
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

type bytesBacked interface {
	Bytes() []byte
}

const (
	// BloomByteLength는 블록 헤더의 로그 블룸에 사용되는 바이트 수를 나타냅니다.
	BloomByteLength = 256

	// BloomBitLength는 헤더 로그 블룸에 사용되는 비트 수를 나타냅니다.
	BloomBitLength = 8 * BloomByteLength
)

// Bloom은 2048 비트 블룸 필터를 나타냅니다.
type Bloom [BloomByteLength]byte

// BytesToBloom은 바이트 슬라이스를 블룸 필터로 변환합니다.
// b가 적당한 크기가 아닌 경우 패닉이 발생합니다.
func BytesToBloom(b []byte) Bloom {
	var bloom Bloom
	bloom.SetBytes(b)
	return bloom
}

// SetBytes는 블룸 필터의 내용을 주어진 바이트로 설정합니다.
// d가 적당한 크기가 아닌 경우 패닉이 발생합니다.
func (b *Bloom) SetBytes(d []byte) {
	if len(b) < len(d) { // b의 길이가 d보다 작으면 패닉 발생
		panic(fmt.Sprintf("bloom bytes too big %d %d", len(b), len(d)))
	}
	copy(b[BloomByteLength-len(d):], d) // d의 길이만큼 b에 복사 (b의 뒤에서부터)
}

// Add는 d를 필터에 추가합니다. Test(d)의 미래 호출은 true를 반환합니다.
func (b *Bloom) Add(d []byte) {
	b.add(d, make([]byte, 6))
}

// add는 Add의 내부 버전으로, 재사용을 위해 임시 버퍼를 사용합니다 (최소 6바이트여야 함)
func (b *Bloom) add(d []byte, buf []byte) {
	i1, v1, i2, v2, i3, v3 := bloomValues(d, buf) // d를 해싱하여 인덱스와 값을 얻습니다.
	b[i1] |= v1                                   // 인덱스에 값을 OR 연산합니다.
	b[i2] |= v2
	b[i3] |= v3
}

// Big는 b를 big.Int로 변환합니다.
// 참고: 블룸 필터를 big.Int로 변환한 다음 GetBytes를 호출하더라도
// 블룸 필터의 바이트와 동일한 바이트를 반환하지 않습니다. 왜냐하면 big.Int는 앞의 비어있는 바이트를 잘라내기 때문입니다.
func (b Bloom) Big() *big.Int {
	return new(big.Int).SetBytes(b[:])
}

// Bytes는 블룸의 바이트 슬라이스를 반환합니다.
func (b Bloom) Bytes() []byte {
	return b[:]
}

// Test는 주어진 토픽이 블룸 필터에 들어 있는지 여부를 반환합니다.
func (b Bloom) Test(topic []byte) bool {
	i1, v1, i2, v2, i3, v3 := bloomValues(topic, make([]byte, 6))
	return v1 == v1&b[i1] &&
		v2 == v2&b[i2] &&
		v3 == v3&b[i3]
}

// MarshalText는 0x 접두사가 있는 16진수 문자열로 b를 인코딩합니다.
func (b Bloom) MarshalText() ([]byte, error) {
	return hexutil.Bytes(b[:]).MarshalText()
}

// UnmarshalText는 0x 접두사가 있는 16진수 문자열을 b로 디코딩합니다.
func (b *Bloom) UnmarshalText(input []byte) error {
	return hexutil.UnmarshalFixedText("Bloom", input, b[:])
}

// CreateBloom은 주어진 트랜잭션 영수증 목록에 대한 블룸 필터를 생성합니다.
func CreateBloom(receipts Receipts) Bloom {
	buf := make([]byte, 6)
	var bin Bloom
	for _, receipt := range receipts {
		for _, log := range receipt.Logs {
			bin.add(log.Address.Bytes(), buf) // 로그를 발생시킨 주소를 해싱하여 블룸 필터에 추가합니다.
			for _, b := range log.Topics {
				bin.add(b[:], buf) // 로그의 토픽을 해싱하여 블룸 필터에 추가합니다.
			}
		}
	}
	return bin
}

// LogsBloom는 주어진 로그에 대한 블룸 바이트를 반환합니다.
func LogsBloom(logs []*Log) []byte {
	buf := make([]byte, 6)
	var bin Bloom
	for _, log := range logs {
		bin.add(log.Address.Bytes(), buf) // 로그를 발생시킨 주소를 해싱하여 블룸 필터에 추가합니다.
		for _, b := range log.Topics {
			bin.add(b[:], buf) // 로그의 토픽을 해싱하여 블룸 필터에 추가합니다.
		}
	}
	return bin[:]
}

// Bloom9은 주어진 데이터에 대한 블룸 필터를 바이트열로 반환합니다.
func Bloom9(data []byte) []byte {
	var b Bloom
	b.SetBytes(data)
	return b.Bytes()
}

// bloomValues는 주어진 데이터에 대해 설정할 바이트 (인덱스-값 쌍)를 반환합니다.
// hashbuf는 6바이트 이상의 임시 버퍼여야 합니다.
func bloomValues(data []byte, hashbuf []byte) (uint, byte, uint, byte, uint, byte) {
	sha := hasherPool.Get().(crypto.KeccakState) // keccak256 해시 함수를 풀에서 가져옵니다.
	sha.Reset()                                  // 해시 함수를 초기화합니다.
	sha.Write(data)                              // 데이터를 해싱합니다. (한 번만 해싱합니다.)
	sha.Read(hashbuf)                            // 해시를 읽습니다. (Sum보다 Read가 더 빠릅니다.)
	hasherPool.Put(sha)                          // 해시 함수를 풀에 반환합니다.
	// 필터에 추가되는 비트 자리를 구합니다. (1을 0~7만큼 왼쪽으로 시프트한 값)
	v1 := byte(1 << (hashbuf[1] & 0x7))
	v2 := byte(1 << (hashbuf[3] & 0x7))
	v3 := byte(1 << (hashbuf[5] & 0x7))
	// 데이터를 필터에 추가하기 위해 OR 연산할 바이트의 인덱스
	i1 := BloomByteLength - uint((binary.BigEndian.Uint16(hashbuf)&0x7ff)>>3) - 1 // v1의 바이트 인덱스는 hashbuf를 uint16 빅 엔디언으로 읽은 값의 하위 11비트를 3만큼 오른쪽으로 시프트한 값을 uint로 변환한 값에서 1을 뺀 값입니다.
	i2 := BloomByteLength - uint((binary.BigEndian.Uint16(hashbuf[2:])&0x7ff)>>3) - 1
	i3 := BloomByteLength - uint((binary.BigEndian.Uint16(hashbuf[4:])&0x7ff)>>3) - 1

	return i1, v1, i2, v2, i3, v3
}

// BloomLookup은 블룸 필터에 특정 토픽이 존재하는지 확인하는 편의 메서드입니다.
func BloomLookup(bin Bloom, topic bytesBacked) bool {
	return bin.Test(topic.Bytes())
}
