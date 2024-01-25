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

//go:generate go run ../../rlp/rlpgen -type StateAccount -out gen_account_rlp.go

// StateAccount는 이더리움 컨센서스 계정입니다.
// 이 객체들은 메인 계정 트라이에 저장됩니다.
type StateAccount struct {
	Nonce    uint64      // 계정의 nonce
	Balance  *big.Int    // 계정의 잔액
	Root     common.Hash // 스토리지 트라이의 머클루트
	CodeHash []byte      // EVM 코드 해시 (Externally Owned Account는 nil)
}

// NewEmptyStateAccount는 빈 상태 계정을 구성합니다.
func NewEmptyStateAccount() *StateAccount {
	return &StateAccount{
		Balance:  new(big.Int),
		Root:     EmptyRootHash,
		CodeHash: EmptyCodeHash.Bytes(),
	}
}

// Copy는 상태 계정 객체의 깊은 복사본을 반환합니다.
func (acct *StateAccount) Copy() *StateAccount {
	var balance *big.Int
	if acct.Balance != nil {
		balance = new(big.Int).Set(acct.Balance)
	}
	return &StateAccount{
		Nonce:    acct.Nonce,
		Balance:  balance,
		Root:     acct.Root,
		CodeHash: common.CopyBytes(acct.CodeHash),
	}
}

// SlimAccount는 스토리지 트라이의 머클루트가 바이트 슬라이스로 대체된 버전입니다.
// 이 형식은 빈 루트와 코드 해시를 nil 바이트 슬라이스로 대체하는 슬림 형식 또는 전체 컨센서스 형식을 나타낼 수 있습니다.
type SlimAccount struct {
	Nonce    uint64
	Balance  *big.Int
	Root     []byte // 루트가 types.EmptyRootHash와 같으면 Nil
	CodeHash []byte // 해시가 types.EmptyCodeHash와 같으면 Nil
}

// SlimAccountRLP는 상태 계정을 'slim RLP' 형식으로 인코딩합니다.
func SlimAccountRLP(account StateAccount) []byte {
	slim := SlimAccount{
		Nonce:   account.Nonce,
		Balance: account.Balance,
	}
	if account.Root != EmptyRootHash {
		slim.Root = account.Root[:]
	}
	if !bytes.Equal(account.CodeHash, EmptyCodeHash[:]) {
		slim.CodeHash = account.CodeHash
	}
	data, err := rlp.EncodeToBytes(slim)
	if err != nil {
		panic(err)
	}
	return data
}

// FullAccount는 'slim RLP' 형식의 데이터를 디코딩하고 컨센서스 형식 계정을 반환합니다.
func FullAccount(data []byte) (*StateAccount, error) {
	var slim SlimAccount
	if err := rlp.DecodeBytes(data, &slim); err != nil {
		return nil, err
	}
	var account StateAccount
	account.Nonce, account.Balance = slim.Nonce, slim.Balance

	// Interpret the storage root and code hash in slim format.
	if len(slim.Root) == 0 {
		account.Root = EmptyRootHash
	} else {
		account.Root = common.BytesToHash(slim.Root)
	}
	if len(slim.CodeHash) == 0 {
		account.CodeHash = EmptyCodeHash[:]
	} else {
		account.CodeHash = slim.CodeHash
	}
	return &account, nil
}

// FullAccountRLP는 'slim RLP' 형식의 데이터를 full RLP 형식으로 변환합니다.
func FullAccountRLP(data []byte) ([]byte, error) {
	account, err := FullAccount(data)
	if err != nil {
		return nil, err
	}
	return rlp.EncodeToBytes(account)
}
