// Copyright 2023 The go-ethereum Authors
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
	"crypto/sha256"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

// BlobTx는 EIP-4844 트랜잭션을 나타냅니다.
type BlobTx struct {
	ChainID    *uint256.Int
	Nonce      uint64
	GasTipCap  *uint256.Int // a.k.a. maxPriorityFeePerGas
	GasFeeCap  *uint256.Int // a.k.a. maxFeePerGas
	Gas        uint64
	To         common.Address
	Value      *uint256.Int
	Data       []byte
	AccessList AccessList
	BlobFeeCap *uint256.Int // a.k.a. maxFeePerBlobGas
	BlobHashes []common.Hash

	// blob 트랜잭션은 선택적으로 blob을 포함할 수 있습니다. BlobTx가 서명을 위해 트랜잭션을 생성하는 데 사용될 때 이 필드를 설정해야만 합니다.
	Sidecar *BlobTxSidecar `rlp:"-"`

	// 서명 값
	V *uint256.Int `json:"v" gencodec:"required"`
	R *uint256.Int `json:"r" gencodec:"required"`
	S *uint256.Int `json:"s" gencodec:"required"`
}

// BlobTxSidecar는 blob 트랜잭션의 blob을 포함합니다.
type BlobTxSidecar struct {
	Blobs       []kzg4844.Blob       // blob 풀이 필요한 blob
	Commitments []kzg4844.Commitment // blob 풀이 필요한 Commitments
	Proofs      []kzg4844.Proof      // blob 풀이 필요한 Proofs
}

// BlobHashes는 주어진 blob의 blob 해시를 계산합니다.
func (sc *BlobTxSidecar) BlobHashes() []common.Hash {
	h := make([]common.Hash, len(sc.Commitments))
	for i := range sc.Blobs {
		h[i] = blobHash(&sc.Commitments[i])
	}
	return h
}

// encodedSize는 사이드카 요소의 RLP 크기를 계산합니다. 이는 BlobTxSidecar의 인코딩된 크기를 반환하지 않습니다.
// 그저 tx.Size()를 위한 유틸리티 함수입니다.
func (sc *BlobTxSidecar) encodedSize() uint64 {
	var blobs, commitments, proofs uint64
	for i := range sc.Blobs {
		blobs += rlp.BytesSize(sc.Blobs[i][:])
	}
	for i := range sc.Commitments {
		commitments += rlp.BytesSize(sc.Commitments[i][:])
	}
	for i := range sc.Proofs {
		proofs += rlp.BytesSize(sc.Proofs[i][:])
	}
	return rlp.ListSize(blobs) + rlp.ListSize(commitments) + rlp.ListSize(proofs)
}

// blobTxWithBlobs는 blob이 존재할 때 트랜잭션의 인코딩에 사용됩니다.
type blobTxWithBlobs struct {
	BlobTx      *BlobTx
	Blobs       []kzg4844.Blob
	Commitments []kzg4844.Commitment
	Proofs      []kzg4844.Proof
}

// copy는 트랜잭션 데이터의 깊은 복사본을 생성하여 반환합니다.
func (tx *BlobTx) copy() TxData {
	cpy := &BlobTx{
		Nonce: tx.Nonce,
		To:    tx.To,
		Data:  common.CopyBytes(tx.Data),
		Gas:   tx.Gas,
		// 이하의 값들은 아래에서 복사됩니다.
		AccessList: make(AccessList, len(tx.AccessList)),
		BlobHashes: make([]common.Hash, len(tx.BlobHashes)),
		Value:      new(uint256.Int),
		ChainID:    new(uint256.Int),
		GasTipCap:  new(uint256.Int),
		GasFeeCap:  new(uint256.Int),
		BlobFeeCap: new(uint256.Int),
		V:          new(uint256.Int),
		R:          new(uint256.Int),
		S:          new(uint256.Int),
	}
	copy(cpy.AccessList, tx.AccessList)
	copy(cpy.BlobHashes, tx.BlobHashes)

	if tx.Value != nil {
		cpy.Value.Set(tx.Value)
	}
	if tx.ChainID != nil {
		cpy.ChainID.Set(tx.ChainID)
	}
	if tx.GasTipCap != nil {
		cpy.GasTipCap.Set(tx.GasTipCap)
	}
	if tx.GasFeeCap != nil {
		cpy.GasFeeCap.Set(tx.GasFeeCap)
	}
	if tx.BlobFeeCap != nil {
		cpy.BlobFeeCap.Set(tx.BlobFeeCap)
	}
	if tx.V != nil {
		cpy.V.Set(tx.V)
	}
	if tx.R != nil {
		cpy.R.Set(tx.R)
	}
	if tx.S != nil {
		cpy.S.Set(tx.S)
	}
	if tx.Sidecar != nil {
		cpy.Sidecar = &BlobTxSidecar{
			Blobs:       append([]kzg4844.Blob(nil), tx.Sidecar.Blobs...),
			Commitments: append([]kzg4844.Commitment(nil), tx.Sidecar.Commitments...),
			Proofs:      append([]kzg4844.Proof(nil), tx.Sidecar.Proofs...),
		}
	}
	return cpy
}

// innerTx에 대한 접근자
func (tx *BlobTx) txType() byte           { return BlobTxType }
func (tx *BlobTx) chainID() *big.Int      { return tx.ChainID.ToBig() }
func (tx *BlobTx) accessList() AccessList { return tx.AccessList }
func (tx *BlobTx) data() []byte           { return tx.Data }
func (tx *BlobTx) gas() uint64            { return tx.Gas }
func (tx *BlobTx) gasFeeCap() *big.Int    { return tx.GasFeeCap.ToBig() }
func (tx *BlobTx) gasTipCap() *big.Int    { return tx.GasTipCap.ToBig() }
func (tx *BlobTx) gasPrice() *big.Int     { return tx.GasFeeCap.ToBig() }
func (tx *BlobTx) value() *big.Int        { return tx.Value.ToBig() }
func (tx *BlobTx) nonce() uint64          { return tx.Nonce }
func (tx *BlobTx) to() *common.Address    { tmp := tx.To; return &tmp }
func (tx *BlobTx) blobGas() uint64        { return params.BlobTxBlobGasPerBlob * uint64(len(tx.BlobHashes)) }

func (tx *BlobTx) effectiveGasPrice(dst *big.Int, baseFee *big.Int) *big.Int {
	if baseFee == nil {
		return dst.Set(tx.GasFeeCap.ToBig())
	}
	tip := dst.Sub(tx.GasFeeCap.ToBig(), baseFee)
	if tip.Cmp(tx.GasTipCap.ToBig()) > 0 {
		tip.Set(tx.GasTipCap.ToBig())
	}
	return tip.Add(tip, baseFee)
}

func (tx *BlobTx) rawSignatureValues() (v, r, s *big.Int) {
	return tx.V.ToBig(), tx.R.ToBig(), tx.S.ToBig()
}

func (tx *BlobTx) setSignatureValues(chainID, v, r, s *big.Int) {
	tx.ChainID.SetFromBig(chainID)
	tx.V.SetFromBig(v)
	tx.R.SetFromBig(r)
	tx.S.SetFromBig(s)
}

func (tx *BlobTx) withoutSidecar() *BlobTx {
	cpy := *tx
	cpy.Sidecar = nil
	return &cpy
}

func (tx *BlobTx) encode(b *bytes.Buffer) error {
	if tx.Sidecar == nil {
		return rlp.Encode(b, tx)
	}
	inner := &blobTxWithBlobs{
		BlobTx:      tx,
		Blobs:       tx.Sidecar.Blobs,
		Commitments: tx.Sidecar.Commitments,
		Proofs:      tx.Sidecar.Proofs,
	}
	return rlp.Encode(b, inner)
}

func (tx *BlobTx) decode(input []byte) error {
	// 두 가지 형식을 지원해야 합니다: blob을 포함하는 tx의 네트워크 프로토콜 인코딩 또는 blob이 없는 정규 인코딩.
	//
	// 두 인코딩은 입력 목록의 첫 번째 요소가 리스트인지 확인하여 구분할 수 있습니다.

	outerList, _, err := rlp.SplitList(input)
	if err != nil {
		return err
	}
	firstElemKind, _, _, err := rlp.Split(outerList)
	if err != nil {
		return err
	}

	if firstElemKind != rlp.List {
		return rlp.DecodeBytes(input, tx)
	}
	// blob을 포함하는 tx입니다.
	var inner blobTxWithBlobs
	if err := rlp.DecodeBytes(input, &inner); err != nil {
		return err
	}
	*tx = *inner.BlobTx
	tx.Sidecar = &BlobTxSidecar{
		Blobs:       inner.Blobs,
		Commitments: inner.Commitments,
		Proofs:      inner.Proofs,
	}
	return nil
}

func blobHash(commit *kzg4844.Commitment) common.Hash {
	hasher := sha256.New()
	hasher.Write(commit[:])
	var vhash common.Hash
	hasher.Sum(vhash[:0])
	vhash[0] = params.BlobTxHashVersion
	return vhash
}
