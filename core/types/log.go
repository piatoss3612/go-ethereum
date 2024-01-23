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
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

//go:generate go run ../../rlp/rlpgen -type Log -out gen_log_rlp.go
//go:generate go run github.com/fjl/gencodec -type Log -field-override logMarshaling -out gen_log_json.go

// Log는 컨트랙트 로그 이벤트를 나타냅니다. 이러한 이벤트는 LOG opcode에 의해 생성되고 노드에 의해 저장/인덱싱됩니다.
type Log struct {
	// Consensus 필드:
	// 이벤트를 생성한 컨트랙트의 주소
	Address common.Address `json:"address" gencodec:"required"`
	// 컨트랙트가 제공한 토픽 목록
	Topics []common.Hash `json:"topics" gencodec:"required"`
	// 컨트랙트가 제공한 데이터, 일반적으로 ABI로 인코딩됨
	Data []byte `json:"data" gencodec:"required"`

	// 파생된 필드. 이러한 필드는 노드에 의해 채워지지만 합의에 의해 보호되지는 않습니다.
	// 트랜잭션이 포함된 블록의 번호
	BlockNumber uint64 `json:"blockNumber" rlp:"-"`
	// 트랜잭션의 해시
	TxHash common.Hash `json:"transactionHash" gencodec:"required" rlp:"-"`
	// 블록에서 트랜잭션의 인덱스
	TxIndex uint `json:"transactionIndex" rlp:"-"`
	// 트랜잭션이 포함된 블록의 해시
	BlockHash common.Hash `json:"blockHash" rlp:"-"`
	// 블록에서 로그의 인덱스
	Index uint `json:"logIndex" rlp:"-"`

	// The Removed field is true if this log was reverted due to a chain reorganisation.
	// You must pay attention to this field if you receive logs through a filter query.

	// Removed 필드는 이 로그가 체인 재구성으로 인해 revert되었을 경우 true입니다.
	// 필터 쿼리를 통해 로그를 받는 경우 이 필드에 주의해야 합니다.
	Removed bool `json:"removed" rlp:"-"`
}

type logMarshaling struct {
	Data        hexutil.Bytes
	BlockNumber hexutil.Uint64
	TxIndex     hexutil.Uint
	Index       hexutil.Uint
}
