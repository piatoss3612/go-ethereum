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

// types 패키지는 이더리움 consensus와 관련된 데이터 타입을 포함한다.
package types

import (
	"encoding/binary"
	"fmt"
	"io"
	"math/big"
	"reflect"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rlp"
)

// BlockNonce는 64비트 해시로, (mixHash와 결합하여) 블록에서 충분한 계산이 수행되었음을 증명합니다.
type BlockNonce [8]byte

// EncodeNonce는 주어진 정수를 블록 nonce로 변환합니다. (빅 엔디안)
func EncodeNonce(i uint64) BlockNonce {
	var n BlockNonce
	binary.BigEndian.PutUint64(n[:], i)
	return n
}

// Uint64는 블록 nonce의 정수 값을 반환합니다. (빅 엔디안)
func (n BlockNonce) Uint64() uint64 {
	return binary.BigEndian.Uint64(n[:])
}

// MarshalText는 0x 접두사가 있는 16진수 문자열로 n을 인코딩합니다.
func (n BlockNonce) MarshalText() ([]byte, error) {
	return hexutil.Bytes(n[:]).MarshalText()
}

// UnmarshalText는 16진수 문자열로부터 n을 디코딩합니다.
func (n *BlockNonce) UnmarshalText(input []byte) error {
	return hexutil.UnmarshalFixedText("BlockNonce", input, n[:])
}

//go:generate go run github.com/fjl/gencodec -type Header -field-override headerMarshaling -out gen_header_json.go
//go:generate go run ../../rlp/rlpgen -type Header -out gen_header_rlp.go

// Header는 이더리움 블록체인의 블록 헤더를 나타냅니다.
type Header struct {
	ParentHash  common.Hash    `json:"parentHash"       gencodec:"required"`
	UncleHash   common.Hash    `json:"sha3Uncles"       gencodec:"required"`
	Coinbase    common.Address `json:"miner"`
	Root        common.Hash    `json:"stateRoot"        gencodec:"required"`
	TxHash      common.Hash    `json:"transactionsRoot" gencodec:"required"`
	ReceiptHash common.Hash    `json:"receiptsRoot"     gencodec:"required"`
	Bloom       Bloom          `json:"logsBloom"        gencodec:"required"`
	Difficulty  *big.Int       `json:"difficulty"       gencodec:"required"`
	Number      *big.Int       `json:"number"           gencodec:"required"`
	GasLimit    uint64         `json:"gasLimit"         gencodec:"required"`
	GasUsed     uint64         `json:"gasUsed"          gencodec:"required"`
	Time        uint64         `json:"timestamp"        gencodec:"required"`
	Extra       []byte         `json:"extraData"        gencodec:"required"`
	MixDigest   common.Hash    `json:"mixHash"`
	Nonce       BlockNonce     `json:"nonce"`

	// BaseFee는 EIP-1559에 의해 추가되었으며, 레거시 헤더에서는 무시됩니다.
	BaseFee *big.Int `json:"baseFeePerGas" rlp:"optional"`

	// WithdrawalsHash는 EIP-4895에 의해 추가되었으며, 레거시 헤더에서는 무시됩니다.
	WithdrawalsHash *common.Hash `json:"withdrawalsRoot" rlp:"optional"`

	// BlobGasUsed는 EIP-4844에 의해 추가되었으며, 레거시 헤더에서는 무시됩니다.
	BlobGasUsed *uint64 `json:"blobGasUsed" rlp:"optional"`

	// ExcessBlobGas는 EIP-4844에 의해 추가되었으며, 레거시 헤더에서는 무시됩니다.
	ExcessBlobGas *uint64 `json:"excessBlobGas" rlp:"optional"`

	// ParentBeaconRoot는 EIP-4788에 의해 추가되었으며, 레거시 헤더에서는 무시됩니다.
	ParentBeaconRoot *common.Hash `json:"parentBeaconBlockRoot" rlp:"optional"`
}

// gencodoc을 사용하기 위해 필드 타입을 재정의합니다.
type headerMarshaling struct {
	Difficulty    *hexutil.Big
	Number        *hexutil.Big
	GasLimit      hexutil.Uint64
	GasUsed       hexutil.Uint64
	Time          hexutil.Uint64
	Extra         hexutil.Bytes
	BaseFee       *hexutil.Big
	Hash          common.Hash `json:"hash"` // MarshalJSON에서 Hash() 호출을 추가합니다.
	BlobGasUsed   *hexutil.Uint64
	ExcessBlobGas *hexutil.Uint64
}

// Hash는 헤더의 블록 해시를 반환합니다. 이는 단순히 RLP 인코딩 결과의 keccak256 해시입니다.
func (h *Header) Hash() common.Hash {
	return rlpHash(h)
}

var headerSize = common.StorageSize(reflect.TypeOf(Header{}).Size()) // 584 bytes

// Size는 모든 내부 컨텐츠에 의해 사용되는 근사 메모리 크기를 반환합니다.
// 이는 다양한 캐시의 메모리 소비를 근사화하고 제한하는 데 사용됩니다.
func (h *Header) Size() common.StorageSize {
	var baseFeeBits int
	if h.BaseFee != nil {
		baseFeeBits = h.BaseFee.BitLen()
	}
	// 헤더 크기 + extraData 크기 + (difficulty 비트 수 + number 비트 수 + baseFee 비트 수) / 8 (바이트 수)
	return headerSize + common.StorageSize(len(h.Extra)+(h.Difficulty.BitLen()+h.Number.BitLen()+baseFeeBits)/8)
}

// SanityCheck는 몇 가지 기본적인 것들을 확인합니다.
// 이러한 체크는 '정상적인' 프로덕션 값을 체크한다기 보다는, 주로 범위가 정해지지 않은 필드(big.Int 등)가
// 처리 오버헤드를 추가하기 위해 정크 데이터로 채워지는 것을 방지하는 데 사용됩니다.
func (h *Header) SanityCheck() error {
	if h.Number != nil && !h.Number.IsUint64() {
		return fmt.Errorf("too large block number: bitlen %d", h.Number.BitLen())
	}
	if h.Difficulty != nil {
		if diffLen := h.Difficulty.BitLen(); diffLen > 80 {
			return fmt.Errorf("too large block difficulty: bitlen %d", diffLen)
		}
	}
	if eLen := len(h.Extra); eLen > 100*1024 {
		return fmt.Errorf("too large block extradata: size %d", eLen)
	}
	if h.BaseFee != nil {
		if bfLen := h.BaseFee.BitLen(); bfLen > 256 {
			return fmt.Errorf("too large base fee: bitlen %d", bfLen)
		}
	}
	return nil
}

// EmptyBody는 헤더를 완성하는 추가적인 'body'가 없는 경우 true를 반환합니다.
// 즉, 트랜잭션이 없고, 엉클도 없고, 출금도 없습니다.
func (h *Header) EmptyBody() bool {
	if h.WithdrawalsHash != nil {
		return h.TxHash == EmptyTxsHash && *h.WithdrawalsHash == EmptyWithdrawalsHash
	}
	return h.TxHash == EmptyTxsHash && h.UncleHash == EmptyUncleHash
}

// EmptyReceipts는 이 헤더/블록에 영수증이 없는 경우 true를 반환합니다.
func (h *Header) EmptyReceipts() bool {
	return h.ReceiptHash == EmptyReceiptsHash
}

// Body는 블록의 데이터 컨텐츠(트랜잭션과 엉클)를 함께 저장하고
// 이동시키기 위한 간단한(가변, 비안전) 데이터 컨테이너입니다.
type Body struct {
	Transactions []*Transaction
	Uncles       []*Header
	Withdrawals  []*Withdrawal `rlp:"optional"`
}

// Block은 이더리움 블록을 나타냅니다.
//
// Block 타입은 '불변'이 되려고 하며, 이를 위해 특정 캐시를 포함합니다.
// 블록 불변성에 대한 규칙은 다음과 같습니다.
//
//   - 블록이 생성될 때 모든 파라미터를 복사하여 블록을 생성합니다. 이는 파라미터로부터 블록을 독립적으로 만듭니다.
//
//   - 헤더 데이터는 전부 복사하여 사용합니다. 헤더에 대한 변경 사항은 블록의 캐시된 hash와 size를 완전히 망가뜨릴 수 있습니다.
//
//   - 새로운 body 데이터가 블록에 첨부되면, 블록의 얕은 복사본이 반환됩니다.
//     이는 블록 수정이 경쟁 조건 없이 이루어지도록 보장합니다.
//
//   - 블록의 body 데이터에 대해서는 복사하지 않습니다. 왜냐하면 이는 캐시에 영향을 주지 않을 뿐만 아니라, 너무 비싸기 때문입니다.
type Block struct {
	header       *Header
	uncles       []*Header
	transactions Transactions
	withdrawals  Withdrawals

	// 캐시
	hash atomic.Value
	size atomic.Value

	// eth 패키지에서 사용되는 필드로, 피어 간 블록 릴레이를 추적합니다.
	ReceivedAt   time.Time
	ReceivedFrom interface{}
}

// extblock은 블록의 외부 표현입니다. eth 프로토콜 등에서 사용됩니다.
type extblock struct {
	Header      *Header
	Txs         []*Transaction
	Uncles      []*Header
	Withdrawals []*Withdrawal `rlp:"optional"`
}

// NewBlock은 새로운 블록을 생성합니다. 입력 데이터는 복사되므로, 입력 데이터의 변경은 블록에 영향을 주지 않습니다.
//
// 헤더의 TxHash, UncleHash, ReceiptHash, Bloom 값은 입력된 txs, uncles, receipts로부터 유도되므로 생성 시에는 생략됩니다.
func NewBlock(header *Header, txs []*Transaction, uncles []*Header, receipts []*Receipt, hasher TrieHasher) *Block {
	b := &Block{header: CopyHeader(header)}

	// TODO: panic if len(txs) != len(receipts)
	if len(txs) == 0 {
		b.header.TxHash = EmptyTxsHash
	} else {
		b.header.TxHash = DeriveSha(Transactions(txs), hasher)
		b.transactions = make(Transactions, len(txs))
		copy(b.transactions, txs)
	}

	if len(receipts) == 0 {
		b.header.ReceiptHash = EmptyReceiptsHash
	} else {
		b.header.ReceiptHash = DeriveSha(Receipts(receipts), hasher)
		b.header.Bloom = CreateBloom(receipts)
	}

	if len(uncles) == 0 {
		b.header.UncleHash = EmptyUncleHash
	} else {
		b.header.UncleHash = CalcUncleHash(uncles)
		b.uncles = make([]*Header, len(uncles))
		for i := range uncles {
			b.uncles[i] = CopyHeader(uncles[i])
		}
	}

	return b
}

// NewBlockWithWithdrawals는 출금을 포함하는 새로운 블록을 생성합니다. 입력 데이터는 복사되므로, 입력 데이터의 변경은 블록에 영향을 주지 않습니다.
//
// 헤더의 TxHash, UncleHash, ReceiptHash, Bloom 값은 입력된 txs, uncles, receipts로부터 유도되므로 생성 시에는 생략됩니다.
func NewBlockWithWithdrawals(header *Header, txs []*Transaction, uncles []*Header, receipts []*Receipt, withdrawals []*Withdrawal, hasher TrieHasher) *Block {
	b := NewBlock(header, txs, uncles, receipts, hasher)

	if withdrawals == nil {
		b.header.WithdrawalsHash = nil
	} else if len(withdrawals) == 0 {
		b.header.WithdrawalsHash = &EmptyWithdrawalsHash
	} else {
		h := DeriveSha(Withdrawals(withdrawals), hasher)
		b.header.WithdrawalsHash = &h
	}

	return b.WithWithdrawals(withdrawals)
}

// CopyHeader는 블록 헤더의 깊은 복사본을 생성합니다.
func CopyHeader(h *Header) *Header {
	cpy := *h
	if cpy.Difficulty = new(big.Int); h.Difficulty != nil {
		cpy.Difficulty.Set(h.Difficulty)
	}
	if cpy.Number = new(big.Int); h.Number != nil {
		cpy.Number.Set(h.Number)
	}
	if h.BaseFee != nil {
		cpy.BaseFee = new(big.Int).Set(h.BaseFee)
	}
	if len(h.Extra) > 0 {
		cpy.Extra = make([]byte, len(h.Extra))
		copy(cpy.Extra, h.Extra)
	}
	if h.WithdrawalsHash != nil {
		cpy.WithdrawalsHash = new(common.Hash)
		*cpy.WithdrawalsHash = *h.WithdrawalsHash
	}
	if h.ExcessBlobGas != nil {
		cpy.ExcessBlobGas = new(uint64)
		*cpy.ExcessBlobGas = *h.ExcessBlobGas
	}
	if h.BlobGasUsed != nil {
		cpy.BlobGasUsed = new(uint64)
		*cpy.BlobGasUsed = *h.BlobGasUsed
	}
	if h.ParentBeaconRoot != nil {
		cpy.ParentBeaconRoot = new(common.Hash)
		*cpy.ParentBeaconRoot = *h.ParentBeaconRoot
	}
	return &cpy
}

// DecodeRLP은 RLP 형식으로부터 블록을 디코딩합니다.
func (b *Block) DecodeRLP(s *rlp.Stream) error {
	var eb extblock
	_, size, _ := s.Kind()
	if err := s.Decode(&eb); err != nil {
		return err
	}
	b.header, b.uncles, b.transactions, b.withdrawals = eb.Header, eb.Uncles, eb.Txs, eb.Withdrawals
	b.size.Store(rlp.ListSize(size))
	return nil
}

// EncodeRLP은 블록을 RLP 형식으로 직렬화합니다.
func (b *Block) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, &extblock{
		Header:      b.header,
		Txs:         b.transactions,
		Uncles:      b.uncles,
		Withdrawals: b.withdrawals,
	})
}

// Body는 블록의 헤더를 제외한 내용을 반환합니다.
// 반환된 데이터는 독립적인 복사본이 아닙니다.
func (b *Block) Body() *Body {
	return &Body{b.transactions, b.uncles, b.withdrawals}
}

// body 데이터에 대한 접근자. 해당 값들은 블록의 캐시된 hash/size에 영향을 주지 않기 때문에 복사본을 반환하지 않고 레퍼런스를 반환합니다.

func (b *Block) Uncles() []*Header          { return b.uncles }
func (b *Block) Transactions() Transactions { return b.transactions }
func (b *Block) Withdrawals() Withdrawals   { return b.withdrawals }

func (b *Block) Transaction(hash common.Hash) *Transaction {
	for _, transaction := range b.transactions {
		if transaction.Hash() == hash {
			return transaction
		}
	}
	return nil
}

// Header는 블록 헤더를 반환합니다. (복사본으로)
func (b *Block) Header() *Header {
	return CopyHeader(b.header)
}

// 헤더 값에 대한 접근자. 이들은 복사본을 반환합니다!

func (b *Block) Number() *big.Int     { return new(big.Int).Set(b.header.Number) }
func (b *Block) GasLimit() uint64     { return b.header.GasLimit }
func (b *Block) GasUsed() uint64      { return b.header.GasUsed }
func (b *Block) Difficulty() *big.Int { return new(big.Int).Set(b.header.Difficulty) }
func (b *Block) Time() uint64         { return b.header.Time }

func (b *Block) NumberU64() uint64        { return b.header.Number.Uint64() }
func (b *Block) MixDigest() common.Hash   { return b.header.MixDigest }
func (b *Block) Nonce() uint64            { return binary.BigEndian.Uint64(b.header.Nonce[:]) }
func (b *Block) Bloom() Bloom             { return b.header.Bloom }
func (b *Block) Coinbase() common.Address { return b.header.Coinbase }
func (b *Block) Root() common.Hash        { return b.header.Root }
func (b *Block) ParentHash() common.Hash  { return b.header.ParentHash }
func (b *Block) TxHash() common.Hash      { return b.header.TxHash }
func (b *Block) ReceiptHash() common.Hash { return b.header.ReceiptHash }
func (b *Block) UncleHash() common.Hash   { return b.header.UncleHash }
func (b *Block) Extra() []byte            { return common.CopyBytes(b.header.Extra) }

func (b *Block) BaseFee() *big.Int {
	if b.header.BaseFee == nil {
		return nil
	}
	return new(big.Int).Set(b.header.BaseFee)
}

func (b *Block) BeaconRoot() *common.Hash { return b.header.ParentBeaconRoot }

func (b *Block) ExcessBlobGas() *uint64 {
	var excessBlobGas *uint64
	if b.header.ExcessBlobGas != nil {
		excessBlobGas = new(uint64)
		*excessBlobGas = *b.header.ExcessBlobGas
	}
	return excessBlobGas
}

func (b *Block) BlobGasUsed() *uint64 {
	var blobGasUsed *uint64
	if b.header.BlobGasUsed != nil {
		blobGasUsed = new(uint64)
		*blobGasUsed = *b.header.BlobGasUsed
	}
	return blobGasUsed
}

// Size는 블록의 실제 RLP 인코딩된 크기를 반환합니다.
// 캐시된 값이 있으면, 이를 반환하거나, 그렇지 않으면 인코딩하여 크기를 계산합니다.
func (b *Block) Size() uint64 {
	if size := b.size.Load(); size != nil {
		return size.(uint64)
	}
	c := writeCounter(0)
	rlp.Encode(&c, b)       // 블록을 인코딩하여 크기를 계산합니다.
	b.size.Store(uint64(c)) // 값을 캐시합니다.
	return uint64(c)
}

// SanityCheck는 범위가 정해지지 않은 필드가 처리 오버헤드를 추가하기 위해 정크 데이터로 채워지는 것을 방지하는 데 사용됩니다.
func (b *Block) SanityCheck() error {
	return b.header.SanityCheck()
}

type writeCounter uint64 // io.Writer를 구현합니다. 쓰여진 바이트 수를 세기 위해 사용됩니다.

func (c *writeCounter) Write(b []byte) (int, error) {
	*c += writeCounter(len(b))
	return len(b), nil
}

func CalcUncleHash(uncles []*Header) common.Hash {
	if len(uncles) == 0 {
		return EmptyUncleHash
	}
	return rlpHash(uncles)
}

// NewBlockWithHeader는 주어진 헤더 데이터로 블록을 생성합니다. 헤더 데이터는 복사되며, 입력된 헤더와 필드 값의 변경은 블록에 영향을 주지 않습니다.
func NewBlockWithHeader(header *Header) *Block {
	return &Block{header: CopyHeader(header)}
}

// WithSeal은 b의 데이터를 그대로 사용하지만, 헤더를 포장된(sealed) 헤더로 교체한 새로운 블록을 반환합니다.
func (b *Block) WithSeal(header *Header) *Block {
	return &Block{
		header:       CopyHeader(header),
		transactions: b.transactions,
		uncles:       b.uncles,
		withdrawals:  b.withdrawals,
	}
}

// WithBody는 주어진 트랜잭션과 엉클 컨텐츠를 포함하는 블록의 복사본을 반환합니다.
func (b *Block) WithBody(transactions []*Transaction, uncles []*Header) *Block {
	block := &Block{
		header:       b.header,
		transactions: make([]*Transaction, len(transactions)),
		uncles:       make([]*Header, len(uncles)),
		withdrawals:  b.withdrawals,
	}
	copy(block.transactions, transactions)
	for i := range uncles {
		block.uncles[i] = CopyHeader(uncles[i])
	}
	return block
}

// WithWithdrawals는 주어진 출금을 포함하는 블록의 복사본을 반환합니다.
func (b *Block) WithWithdrawals(withdrawals []*Withdrawal) *Block {
	block := &Block{
		header:       b.header,
		transactions: b.transactions,
		uncles:       b.uncles,
	}
	if withdrawals != nil {
		block.withdrawals = make([]*Withdrawal, len(withdrawals))
		copy(block.withdrawals, withdrawals)
	}
	return block
}

// Hash는 블록 헤더의 keccak256 해시를 반환합니다.
// 해시는 첫 호출 시에 계산되고, 그 이후에는 캐시됩니다.
func (b *Block) Hash() common.Hash {
	if hash := b.hash.Load(); hash != nil {
		return hash.(common.Hash)
	}
	v := b.header.Hash()
	b.hash.Store(v)
	return v
}

type Blocks []*Block

// HeaderParentHashFromRLP는 RLP로 인코딩된 헤더의 parentHash를 반환합니다.
// 'header'가 유효하지 않으면, zero hash가 반환됩니다.
func HeaderParentHashFromRLP(header []byte) common.Hash {
	// parentHash는 첫 번째 리스트 요소입니다.
	listContent, _, err := rlp.SplitList(header)
	if err != nil {
		return common.Hash{}
	}
	parentHash, _, err := rlp.SplitString(listContent)
	if err != nil {
		return common.Hash{}
	}
	if len(parentHash) != 32 {
		return common.Hash{}
	}
	return common.BytesToHash(parentHash)
}
