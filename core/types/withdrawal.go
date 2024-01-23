// Copyright 2022 The go-ethereum Authors
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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rlp"
)

//go:generate go run github.com/fjl/gencodec -type Withdrawal -field-override withdrawalMarshaling -out gen_withdrawal_json.go
//go:generate go run ../../rlp/rlpgen -type Withdrawal -out gen_withdrawal_rlp.go

// Withdrawal은 합의 레이어로부터 검증자의 출금 작업을 나타냅니다.
type Withdrawal struct {
	Index     uint64         `json:"index"`          // 합의 레이어에 의해 발행된 단조 증가식 식별자
	Validator uint64         `json:"validatorIndex"` // 출금과 관련된 검증자의 인덱스
	Address   common.Address `json:"address"`        // 출금된 이더가 전송되는 주소
	Amount    uint64         `json:"amount"`         // 출금액 (Gwei 단위)
}

// gencodec을 위한 필드 유형 재정의
type withdrawalMarshaling struct {
	Index     hexutil.Uint64
	Validator hexutil.Uint64
	Amount    hexutil.Uint64
}

// Withdrawals implements DerivableList for withdrawals.

// Withdrawals는 머클루트를 계산하기 위해 필요한 인터페이스를 구현합니다.
type Withdrawals []*Withdrawal

// Len은 s의 길이를 반환합니다.
func (s Withdrawals) Len() int { return len(s) }

// EncodeIndex는 i번째 출금을 w에 인코딩합니다. 이는 오류를 확인하지 않습니다. 왜냐하면 *Withdrawal은
// 디코딩 또는 이 패키지의 공개 API를 통해 구성된 유효한 출금만 포함하기 때문입니다.
func (s Withdrawals) EncodeIndex(i int, w *bytes.Buffer) {
	rlp.Encode(w, s[i])
}
