// Copyright 2021 The go-ethereum Authors
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
	"bytes"
	"fmt"
	"math"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"golang.org/x/crypto/sha3"
)

// hasherPool은 rlpHash를 위한 LegacyKeccak256 해시 함수를 보관합니다.
var hasherPool = sync.Pool{
	New: func() interface{} { return sha3.NewLegacyKeccak256() },
}

// encodeBufferPool holds temporary encoder buffers for DeriveSha and TX encoding.

// encodeBufferPool은 DeriveSha 및 TX 인코딩을 위한 임시 인코더 버퍼를 보관합니다.
var encodeBufferPool = sync.Pool{
	New: func() interface{} { return new(bytes.Buffer) },
}

// getPooledBuffer는 풀에서 버퍼를 회수하고 요청된 크기의 바이트 슬라이스를 만듭니다.
//
// 호출자는 사용 후 *bytes.Buffer 객체를 encodeBufferPool로 반환해야만 합니다!
// 반환된 바이트 슬라이스는 버퍼를 반환한 후에는 사용해서는 안 됩니다.
func getPooledBuffer(size uint64) ([]byte, *bytes.Buffer, error) {
	if size > math.MaxInt {
		return nil, nil, fmt.Errorf("can't get buffer of size %d", size)
	}
	buf := encodeBufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	buf.Grow(int(size))
	b := buf.Bytes()[:int(size)]
	return b, buf, nil
}

// rlpHash는 x를 인코딩하고 인코딩된 바이트를 해시합니다.
func rlpHash(x interface{}) (h common.Hash) {
	sha := hasherPool.Get().(crypto.KeccakState)
	defer hasherPool.Put(sha)
	sha.Reset()
	rlp.Encode(sha, x)
	sha.Read(h[:])
	return h
}

// prefixedRlpHash는 x를 rlp 인코딩하기 전에 해시에 접두사를 작성합니다.
// 이 함수는 typed transactions에 사용됩니다.
func prefixedRlpHash(prefix byte, x interface{}) (h common.Hash) {
	sha := hasherPool.Get().(crypto.KeccakState)
	defer hasherPool.Put(sha)
	sha.Reset()
	sha.Write([]byte{prefix})
	rlp.Encode(sha, x)
	sha.Read(h[:])
	return h
}

// TrieHasher는 파생 가능한 목록(derivable list)의 해시를 계산하는 데 사용되는 도구입니다.
// 이 인터페이스는 프로젝트 내부에서만 사용되므로 외부에서는 사용하지 마십시오.
type TrieHasher interface {
	Reset()
	Update([]byte, []byte) error
	Hash() common.Hash
}

// DerivableList는 DeriveSha의 입력입니다.
// 'Transactions' 및 'Receipts' 타입에서 이 인터페이스를 구현합니다.
// 이 인터페이스는 프로젝트 내부에서만 사용되므로 외부에서는 사용하지 마십시오.
type DerivableList interface {
	Len() int
	EncodeIndex(int, *bytes.Buffer)
}

func encodeForDerive(list DerivableList, i int, buf *bytes.Buffer) []byte {
	buf.Reset()
	list.EncodeIndex(i, buf)
	// It's really unfortunate that we need to perform this copy.
	// StackTrie holds onto the values until Hash is called, so the values
	// written to it must not alias.
	// (어쩔 수 없이 복사를 수행해야 한다고 하는데 정확히 무슨 말인지 모르겠습니다.)
	return common.CopyBytes(buf.Bytes())
}

// DeriveSha는 블록 헤더의 트랜잭션, 영수증 및 출금의 머클루트를 계산합니다.
func DeriveSha(list DerivableList, hasher TrieHasher) common.Hash {
	hasher.Reset()

	valueBuf := encodeBufferPool.Get().(*bytes.Buffer)
	defer encodeBufferPool.Put(valueBuf)

	// StackTrie requires values to be inserted in increasing hash order, which is not the
	// order that `list` provides hashes in. This insertion sequence ensures that the
	// order is correct.

	// StackTrie는 값이 증가하는 해시 순서로 삽입되어야 합니다. 이 삽입 순서는 순서가 올바르도록 보장합니다.(?)
	//
	// 해시 함수에서 반환된 오류는 어짜피 해시 함수가 오류가 발생한 경우 잘못된 해시를 생성되기 때문에 생략됩니다.
	var indexBuf []byte
	for i := 1; i < list.Len() && i <= 0x7f; i++ {
		indexBuf = rlp.AppendUint64(indexBuf[:0], uint64(i))
		value := encodeForDerive(list, i, valueBuf)
		hasher.Update(indexBuf, value)
	}
	if list.Len() > 0 {
		indexBuf = rlp.AppendUint64(indexBuf[:0], 0)
		value := encodeForDerive(list, 0, valueBuf)
		hasher.Update(indexBuf, value)
	}
	for i := 0x80; i < list.Len(); i++ {
		indexBuf = rlp.AppendUint64(indexBuf[:0], uint64(i))
		value := encodeForDerive(list, i, valueBuf)
		hasher.Update(indexBuf, value)
	}
	return hasher.Hash()
}
