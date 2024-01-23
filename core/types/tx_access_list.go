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
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

//go:generate go run github.com/fjl/gencodec -type AccessTuple -out gen_access_tuple.go

// AccessList은 EIP-2930에 정의된 접근 목록입니다.
type AccessList []AccessTuple

// AccessTuple은 접근 목록의 요소입니다.
type AccessTuple struct {
	Address     common.Address `json:"address"     gencodec:"required"`
	StorageKeys []common.Hash  `json:"storageKeys" gencodec:"required"`
}

// StorageKeys는 접근 목록에 포함된 스토리지 키의 총 개수를 반환합니다.
func (al AccessList) StorageKeys() int {
	sum := 0
	for _, tuple := range al {
		sum += len(tuple.StorageKeys)
	}
	return sum
}

// AccessListTx는 EIP-2930 접근 목록 트랜잭션의 데이터입니다.
type AccessListTx struct {
	ChainID    *big.Int        // 대상 체인 ID
	Nonce      uint64          // 발신자 계정의 nonce
	GasPrice   *big.Int        // 가스당 wei
	Gas        uint64          // 가스 한도
	To         *common.Address `rlp:"nil"` // 수신자의 주소. nil이면 컨트랙트 생성 트랜잭션
	Value      *big.Int        // wei 단위의 이더량
	Data       []byte          // 컨트랙트 생성 트랜잭션의 경우 생성자의 바이트코드. 그 외의 경우 호출 데이터
	AccessList AccessList      // 접근 목록
	V, R, S    *big.Int        // 서명 값
}

// copy는 트랜잭션 데이터의 깊은 복사본을 생성하여 반환합니다.
func (tx *AccessListTx) copy() TxData {
	cpy := &AccessListTx{
		Nonce: tx.Nonce,
		To:    copyAddressPtr(tx.To),
		Data:  common.CopyBytes(tx.Data),
		Gas:   tx.Gas,
		// 이하의 값들은 아래에서 복사됩니다.
		AccessList: make(AccessList, len(tx.AccessList)),
		Value:      new(big.Int),
		ChainID:    new(big.Int),
		GasPrice:   new(big.Int),
		V:          new(big.Int),
		R:          new(big.Int),
		S:          new(big.Int),
	}
	copy(cpy.AccessList, tx.AccessList)
	if tx.Value != nil {
		cpy.Value.Set(tx.Value)
	}
	if tx.ChainID != nil {
		cpy.ChainID.Set(tx.ChainID)
	}
	if tx.GasPrice != nil {
		cpy.GasPrice.Set(tx.GasPrice)
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
	return cpy
}

// innerTx에 대한 접근자
func (tx *AccessListTx) txType() byte           { return AccessListTxType }
func (tx *AccessListTx) chainID() *big.Int      { return tx.ChainID }
func (tx *AccessListTx) accessList() AccessList { return tx.AccessList }
func (tx *AccessListTx) data() []byte           { return tx.Data }
func (tx *AccessListTx) gas() uint64            { return tx.Gas }
func (tx *AccessListTx) gasPrice() *big.Int     { return tx.GasPrice }
func (tx *AccessListTx) gasTipCap() *big.Int    { return tx.GasPrice }
func (tx *AccessListTx) gasFeeCap() *big.Int    { return tx.GasPrice }
func (tx *AccessListTx) value() *big.Int        { return tx.Value }
func (tx *AccessListTx) nonce() uint64          { return tx.Nonce }
func (tx *AccessListTx) to() *common.Address    { return tx.To }

func (tx *AccessListTx) effectiveGasPrice(dst *big.Int, baseFee *big.Int) *big.Int {
	return dst.Set(tx.GasPrice)
}

func (tx *AccessListTx) rawSignatureValues() (v, r, s *big.Int) {
	return tx.V, tx.R, tx.S
}

func (tx *AccessListTx) setSignatureValues(chainID, v, r, s *big.Int) {
	tx.ChainID, tx.V, tx.R, tx.S = chainID, v, r, s
}

func (tx *AccessListTx) encode(b *bytes.Buffer) error {
	return rlp.Encode(b, tx)
}

func (tx *AccessListTx) decode(input []byte) error {
	return rlp.DecodeBytes(input, tx)
}
