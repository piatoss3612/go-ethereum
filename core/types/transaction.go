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
	"bytes"
	"errors"
	"io"
	"math/big"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

var (
	ErrInvalidSig           = errors.New("invalid transaction v, r, s values")
	ErrUnexpectedProtection = errors.New("transaction type does not supported EIP-155 protected signatures")
	ErrInvalidTxType        = errors.New("transaction type not valid in this context")
	ErrTxTypeNotSupported   = errors.New("transaction type not supported")
	ErrGasFeeCapTooLow      = errors.New("fee cap less than base fee")
	errShortTypedTx         = errors.New("typed transaction too short")
	errInvalidYParity       = errors.New("'yParity' field must be 0 or 1")
	errVYParityMismatch     = errors.New("'v' and 'yParity' fields do not match")
	errVYParityMissing      = errors.New("missing 'yParity' or 'v' field in transaction")
)

// 트랜잭션 타입
const (
	LegacyTxType     = 0x00 // Legacy
	AccessListTxType = 0x01 // EIP-2930
	DynamicFeeTxType = 0x02 // EIP-1559
	BlobTxType       = 0x03 // EIP-4844
)

// Transaction은 이더리움 트랜잭션입니다.
type Transaction struct {
	inner TxData    // 트랜잭션의 핵심 내용
	time  time.Time // 로컬에서 처음 확인한 시간 (스팸 방지)

	// 캐시
	hash atomic.Value
	size atomic.Value
	from atomic.Value
}

// NewTx는 새 트랜잭션을 생성합니다.
func NewTx(inner TxData) *Transaction {
	tx := new(Transaction)
	tx.setDecoded(inner.copy(), 0)
	return tx
}

// TxData는 트랜잭션의 기본 데이터를 나타내기 위한 인터페이스입니다.
//
// 이 인터페이스는 DynamicFeeTx, LegacyTx, AccessListTx에 의해 구현됩니다.
type TxData interface {
	txType() byte // 트랜잭션 타입 ID를 반환합니다.
	copy() TxData // 깊은 복사본을 만들어 새로운 TxData를 반환합니다.

	chainID() *big.Int
	accessList() AccessList
	data() []byte
	gas() uint64
	gasPrice() *big.Int
	gasTipCap() *big.Int
	gasFeeCap() *big.Int
	value() *big.Int
	nonce() uint64
	to() *common.Address

	rawSignatureValues() (v, r, s *big.Int)
	setSignatureValues(chainID, v, r, s *big.Int)

	// effectiveGasPrice는 트랜잭션이 지불하는 가스 가격을 계산합니다. 트랜잭션이 포함된 블록의 baseFee가 주어집니다.
	//
	// 다른 TxData 메서드와 달리, 반환된 *big.Int는 계산된 값의 독립적인 복사본이어야 합니다.
	// 즉, 호출자는 결과를 변경할 수 있습니다. 메서드 구현은 'dst'를 사용하여 결과를 저장할 수도 있습니다.
	effectiveGasPrice(dst *big.Int, baseFee *big.Int) *big.Int

	encode(*bytes.Buffer) error
	decode([]byte) error
}

// EncodeRLP은 rlp.Encoder를 구현합니다.
func (tx *Transaction) EncodeRLP(w io.Writer) error {
	if tx.Type() == LegacyTxType {
		return rlp.Encode(w, tx.inner)
	}

	// 레거시 트랜잭션이 아니라면, EIP-2718 타입화된 트랜잭션입니다.
	buf := encodeBufferPool.Get().(*bytes.Buffer)
	defer encodeBufferPool.Put(buf)
	buf.Reset()
	if err := tx.encodeTyped(buf); err != nil {
		return err
	}
	return rlp.Encode(w, buf.Bytes())
}

// encodeTyped는 w에 타입화된 트랜잭션의 정규 인코딩을 작성합니다.
func (tx *Transaction) encodeTyped(w *bytes.Buffer) error {
	w.WriteByte(tx.Type())
	return tx.inner.encode(w)
}

// MarshalBinary은 트랜잭션의 정규 인코딩을 반환합니다.
// 레거시 트랜잭션의 경우 RLP 인코딩을 반환합니다. EIP-2718 타입화된 트랜잭션의 경우 타입과 페이로드를 반환합니다.
func (tx *Transaction) MarshalBinary() ([]byte, error) {
	if tx.Type() == LegacyTxType {
		return rlp.EncodeToBytes(tx.inner)
	}
	var buf bytes.Buffer
	err := tx.encodeTyped(&buf)
	return buf.Bytes(), err
}

// DecodeRLP은 rlp.Decoder를 구현합니다.
func (tx *Transaction) DecodeRLP(s *rlp.Stream) error {
	kind, size, err := s.Kind()
	switch {
	case err != nil:
		return err
	case kind == rlp.List:
		// 레거시 트랜잭션을 디코딩합니다.
		var inner LegacyTx
		err := s.Decode(&inner)
		if err == nil {
			tx.setDecoded(&inner, rlp.ListSize(size))
		}
		return err
	case kind == rlp.Byte:
		return errShortTypedTx
	default:
		// EIP-2718 트랜잭션을 디코딩합니다.
		// 먼저 tx 페이로드를 임시 버퍼에 읽습니다.
		b, buf, err := getPooledBuffer(size)
		if err != nil {
			return err
		}
		defer encodeBufferPool.Put(buf)
		if err := s.ReadBytes(b); err != nil {
			return err
		}
		// 내부 트랜잭션을 디코딩합니다.
		inner, err := tx.decodeTyped(b)
		if err == nil {
			tx.setDecoded(inner, size)
		}
		return err
	}
}

// UnmarshalBinary은 트랜잭션의 정규 인코딩을 디코딩합니다.
// 레거시 RLP 트랜잭션과 EIP-2718 타입화된 트랜잭션을 모두 지원합니다.
func (tx *Transaction) UnmarshalBinary(b []byte) error {
	if len(b) > 0 && b[0] > 0x7f {
		// It's a legacy transaction.
		var data LegacyTx
		err := rlp.DecodeBytes(b, &data)
		if err != nil {
			return err
		}
		tx.setDecoded(&data, uint64(len(b)))
		return nil
	}
	// EIP-2718 트랜잭션을 디코딩합니다.
	inner, err := tx.decodeTyped(b)
	if err != nil {
		return err
	}
	tx.setDecoded(inner, uint64(len(b)))
	return nil
}

// decodeTyped는 정규 형식에서 타입화된 트랜잭션을 디코딩합니다.
func (tx *Transaction) decodeTyped(b []byte) (TxData, error) {
	if len(b) <= 1 {
		return nil, errShortTypedTx
	}
	var inner TxData
	switch b[0] {
	case AccessListTxType:
		inner = new(AccessListTx)
	case DynamicFeeTxType:
		inner = new(DynamicFeeTx)
	case BlobTxType:
		inner = new(BlobTx)
	default:
		return nil, ErrTxTypeNotSupported
	}
	err := inner.decode(b[1:])
	return inner, err
}

// setDecoded는 디코딩을 마친 후 내부 트랜잭션과 크기를 설정합니다.
func (tx *Transaction) setDecoded(inner TxData, size uint64) {
	tx.inner = inner
	tx.time = time.Now()
	if size > 0 {
		tx.size.Store(size)
	}
}

func sanityCheckSignature(v *big.Int, r *big.Int, s *big.Int, maybeProtected bool) error {
	if isProtectedV(v) && !maybeProtected {
		return ErrUnexpectedProtection
	}

	var plainV byte
	if isProtectedV(v) {
		chainID := deriveChainId(v).Uint64()
		plainV = byte(v.Uint64() - 35 - 2*chainID)
	} else if maybeProtected {
		// EIP-155 서명만이 선택적으로 보호될 수 있습니다. 이 v 값이 보호되지 않았다고 결정했다면, 그 값은 27 또는 28이어야 합니다.
		plainV = byte(v.Uint64() - 27)
	} else {
		// 서명이 선택적으로 보호되지 않는다면, v 값이 이미 복구 ID와 같아야 한다고 가정합니다.
		plainV = byte(v.Uint64())
	}
	if !crypto.ValidateSignatureValues(plainV, r, s, false) {
		return ErrInvalidSig
	}

	return nil
}

func isProtectedV(V *big.Int) bool {
	if V.BitLen() <= 8 {
		v := V.Uint64()
		return v != 27 && v != 28 && v != 1 && v != 0
	}
	// 27이나 28이 아닌 것은 보호된 것으로 간주됩니다.
	return true
}

// Protected는 트랜잭션을 재실행하는 것을 방지하는지 여부를 나타냅니다.
func (tx *Transaction) Protected() bool {
	switch tx := tx.inner.(type) {
	case *LegacyTx:
		return tx.V != nil && isProtectedV(tx.V)
	default:
		return true // 레거시 트랜잭션이 아니라면, 트랜잭션은 항상 보호됩니다.
	}
}

// Type은 트랜잭션 타입을 반환합니다.
func (tx *Transaction) Type() uint8 {
	return tx.inner.txType()
}

// ChainId는 EIP155에 따라 트랜잭션의 체인 ID를 반환합니다. 반환 값은 항상 nil이 아닙니다.
// 재실행이 방지되지 않은 레거시 트랜잭션의 경우, 반환 값은 0입니다.
func (tx *Transaction) ChainId() *big.Int {
	return tx.inner.chainID()
}

// Data는 트랜잭션의 입력 데이터를 반환합니다.
func (tx *Transaction) Data() []byte { return tx.inner.data() }

// AccessList는 트랜잭션의 액세스 목록을 반환합니다.
func (tx *Transaction) AccessList() AccessList { return tx.inner.accessList() }

// Gas는 트랜잭션의 가스 한도를 반환합니다.
func (tx *Transaction) Gas() uint64 { return tx.inner.gas() }

// GasPrice는 트랜잭션의 가스 가격을 반환합니다.
func (tx *Transaction) GasPrice() *big.Int { return new(big.Int).Set(tx.inner.gasPrice()) }

// GasTipCap는 트랜잭션의 가스 당 gasTipCap을 반환합니다.
func (tx *Transaction) GasTipCap() *big.Int { return new(big.Int).Set(tx.inner.gasTipCap()) }

// GasFeeCap는 트랜잭션의 가스 당 fee cap을 반환합니다.
func (tx *Transaction) GasFeeCap() *big.Int { return new(big.Int).Set(tx.inner.gasFeeCap()) }

// Value는 트랜잭션의 이더 양을 반환합니다.
func (tx *Transaction) Value() *big.Int { return new(big.Int).Set(tx.inner.value()) }

// Nonce는 트랜잭션의 발신자 계정 nonce를 반환합니다.
func (tx *Transaction) Nonce() uint64 { return tx.inner.nonce() }

// To는 트랜잭션의 수신자 주소를 반환합니다.
// 계약 생성 트랜잭션의 경우, To는 nil을 반환합니다.
func (tx *Transaction) To() *common.Address {
	return copyAddressPtr(tx.inner.to())
}

// Cost는 (gas * gasPrice) + (blobGas * blobGasPrice) + value를 반환합니다.
func (tx *Transaction) Cost() *big.Int {
	total := new(big.Int).Mul(tx.GasPrice(), new(big.Int).SetUint64(tx.Gas()))
	if tx.Type() == BlobTxType {
		total.Add(total, new(big.Int).Mul(tx.BlobGasFeeCap(), new(big.Int).SetUint64(tx.BlobGas())))
	}
	total.Add(total, tx.Value())
	return total
}

// RawSignatureValues는 트랜잭션의 V, R, S 서명 값을 반환합니다.
// 반환 값은 호출자에 의해 수정되어서는 안 됩니다.
func (tx *Transaction) RawSignatureValues() (v, r, s *big.Int) {
	return tx.inner.rawSignatureValues()
}

// GasFeeCapCmp는 두 트랜잭션의 fee cap을 비교합니다.
func (tx *Transaction) GasFeeCapCmp(other *Transaction) int {
	return tx.inner.gasFeeCap().Cmp(other.inner.gasFeeCap())
}

// GasFeeCapIntCmp는 트랜잭션의 fee cap을 주어진 fee cap과 비교합니다.
func (tx *Transaction) GasFeeCapIntCmp(other *big.Int) int {
	return tx.inner.gasFeeCap().Cmp(other)
}

// GasTipCapCmp는 두 트랜잭션의 gasTipCap을 비교합니다.
func (tx *Transaction) GasTipCapCmp(other *Transaction) int {
	return tx.inner.gasTipCap().Cmp(other.inner.gasTipCap())
}

// GasTipCapIntCmp는 트랜잭션의 gasTipCap을 주어진 gasTipCap과 비교합니다.
func (tx *Transaction) GasTipCapIntCmp(other *big.Int) int {
	return tx.inner.gasTipCap().Cmp(other)
}

// EffectiveGasTip는 주어진 base fee에 대한 유효한 마이너 gasTipCap을 반환합니다.
// 참고: 유효한 gasTipCap이 음수인 경우, 이 메서드는 실제 음수 값 및 ErrGasFeeCapTooLow를 반환합니다.
func (tx *Transaction) EffectiveGasTip(baseFee *big.Int) (*big.Int, error) {
	if baseFee == nil {
		return tx.GasTipCap(), nil
	}
	var err error
	gasFeeCap := tx.GasFeeCap()
	if gasFeeCap.Cmp(baseFee) == -1 {
		err = ErrGasFeeCapTooLow
	}
	return math.BigMin(tx.GasTipCap(), gasFeeCap.Sub(gasFeeCap, baseFee)), err
}

// EffectiveGasTipValue는 EffectiveGasTip과 동일하지만, 유효한 gasTipCap이 음수인 경우 오류를 반환하지 않습니다.
func (tx *Transaction) EffectiveGasTipValue(baseFee *big.Int) *big.Int {
	effectiveTip, _ := tx.EffectiveGasTip(baseFee)
	return effectiveTip
}

// EffectiveGasTipCmp는 주어진 base fee를 가정하고 두 트랜잭션의 유효한 gasTipCap을 비교합니다.
func (tx *Transaction) EffectiveGasTipCmp(other *Transaction, baseFee *big.Int) int {
	if baseFee == nil {
		return tx.GasTipCapCmp(other)
	}
	return tx.EffectiveGasTipValue(baseFee).Cmp(other.EffectiveGasTipValue(baseFee))
}

// EffectiveGasTipIntCmp는 트랜잭션의 유효한 gasTipCap을 주어진 gasTipCap과 비교합니다.
func (tx *Transaction) EffectiveGasTipIntCmp(other *big.Int, baseFee *big.Int) int {
	if baseFee == nil {
		return tx.GasTipCapIntCmp(other)
	}
	return tx.EffectiveGasTipValue(baseFee).Cmp(other)
}

// BlobGas는 blob 트랜잭션의 blob gas 한도를 반환합니다. blob 트랜잭션이 아니라면 0을 반환합니다.
func (tx *Transaction) BlobGas() uint64 {
	if blobtx, ok := tx.inner.(*BlobTx); ok {
		return blobtx.blobGas()
	}
	return 0
}

// BlobGasFeeCap는 blob 트랜잭션의 blob gas 당 blob fee cap을 반환합니다. blob 트랜잭션이 아니라면 nil을 반환합니다.
func (tx *Transaction) BlobGasFeeCap() *big.Int {
	if blobtx, ok := tx.inner.(*BlobTx); ok {
		return blobtx.BlobFeeCap.ToBig()
	}
	return nil
}

// BlobHashes는 blob 트랜잭션의 blob 해시를 반환합니다. blob 트랜잭션이 아니라면 nil을 반환합니다.
func (tx *Transaction) BlobHashes() []common.Hash {
	if blobtx, ok := tx.inner.(*BlobTx); ok {
		return blobtx.BlobHashes
	}
	return nil
}

// BlobTxSidecar는 blob 트랜잭션의 사이드카를 반환합니다. blob 트랜잭션이 아니라면 nil을 반환합니다.
func (tx *Transaction) BlobTxSidecar() *BlobTxSidecar {
	if blobtx, ok := tx.inner.(*BlobTx); ok {
		return blobtx.Sidecar
	}
	return nil
}

// BlobGasFeeCapCmp는 두 트랜잭션의 blob fee cap을 비교합니다.
func (tx *Transaction) BlobGasFeeCapCmp(other *Transaction) int {
	return tx.BlobGasFeeCap().Cmp(other.BlobGasFeeCap())
}

// BlobGasFeeCapIntCmp는 트랜잭션의 blob fee cap을 주어진 blob fee cap과 비교합니다.
func (tx *Transaction) BlobGasFeeCapIntCmp(other *big.Int) int {
	return tx.BlobGasFeeCap().Cmp(other)
}

// WithoutBlobTxSidecar는 blob 사이드카가 제거된 tx의 복사본을 반환합니다.
func (tx *Transaction) WithoutBlobTxSidecar() *Transaction {
	blobtx, ok := tx.inner.(*BlobTx)
	if !ok {
		return tx
	}
	cpy := &Transaction{
		inner: blobtx.withoutSidecar(),
		time:  tx.time,
	}
	// 참고: tx.size 캐시는 사이드카가 크기에 포함되기 때문에 복사되지 않습니다!
	if h := tx.hash.Load(); h != nil {
		cpy.hash.Store(h)
	}
	if f := tx.from.Load(); f != nil {
		cpy.from.Store(f)
	}
	return cpy
}

// SetTime은 트랜잭션의 디코딩 시간을 설정합니다. 이는 테스트에서 임의의 시간을 설정하는 데 사용되거나,
// 디스크에서 오래된 트랜잭션을 로드할 때 트랜잭션 풀에 의해 사용됩니다.
func (tx *Transaction) SetTime(t time.Time) {
	tx.time = t
}

// Time은 트랜잭션이 네트워크에서 처음 확인된 시간을 반환합니다.
// It is a heuristic to prefer mining older txs vs new all other things equal.
func (tx *Transaction) Time() time.Time {
	return tx.time
}

// Hash는 트랜잭션 해시를 반환합니다.
func (tx *Transaction) Hash() common.Hash {
	if hash := tx.hash.Load(); hash != nil { // 캐시된 해시가 있는지 확인합니다.
		return hash.(common.Hash)
	}

	var h common.Hash
	if tx.Type() == LegacyTxType { // 레거시 트랜잭션은 RLP 해시를 사용합니다.
		h = rlpHash(tx.inner)
	} else {
		h = prefixedRlpHash(tx.Type(), tx.inner) // EIP-2718 트랜잭션은 prefix RLP 해시를 사용합니다.
	}
	tx.hash.Store(h) // 해시를 캐시합니다.
	return h
}

// Size는 트랜잭션의 실제 인코딩된 저장공간 크기를 반환합니다.
// 인코딩하고 반환하거나, 이전에 캐시된 값을 반환합니다.
func (tx *Transaction) Size() uint64 {
	if size := tx.size.Load(); size != nil {
		return size.(uint64)
	}

	// 캐시가 존재하지 않으면 인코딩하고 캐시합니다.
	// 모든 tx.inner 값이 RLP로 인코딩된다는 가정하에 실행됩니다.
	c := writeCounter(0)
	rlp.Encode(&c, &tx.inner)
	size := uint64(c)

	// For blob transactions,

	// blob 트랜잭션의 경우, add the size of the blob content and the outer list of the
	// tx + sidecar encoding.
	if sc := tx.BlobTxSidecar(); sc != nil {
		size += rlp.ListSize(sc.encodedSize())
	}

	// 타입화된 트랜잭션의 경우, 인코딩에는 선행하는 타입 바이트도 포함됩니다.
	if tx.Type() != LegacyTxType {
		size += 1
	}

	tx.size.Store(size) // 사이즈를 캐시합니다.
	return size
}

// WithSignature는 주어진 서명을 가진 새 트랜잭션을 반환합니다.
// 이 서명은 V가 0 또는 1인 [R || S || V] 형식이어야 합니다.
func (tx *Transaction) WithSignature(signer Signer, sig []byte) (*Transaction, error) {
	r, s, v, err := signer.SignatureValues(tx, sig)
	if err != nil {
		return nil, err
	}
	cpy := tx.inner.copy()
	cpy.setSignatureValues(signer.ChainID(), v, r, s)
	return &Transaction{inner: cpy, time: tx.time}, nil
}

// Transactions는 머클루트를 계산하기 위해 필요한 인터페이스를 구현합니다.
type Transactions []*Transaction

// Len은 s의 길이를 반환합니다.
func (s Transactions) Len() int { return len(s) }

// EncodeIndex는 i번째 트랜잭션을 w에 인코딩합니다. 트랜잭션이 디코딩되거나
// 이 패키지의 공개 API를 통해 생성된 유효한 txs만 포함된다고 가정하므로 오류를 확인하지 않습니다.
func (s Transactions) EncodeIndex(i int, w *bytes.Buffer) {
	tx := s[i]
	if tx.Type() == LegacyTxType {
		rlp.Encode(w, tx.inner)
	} else {
		tx.encodeTyped(w)
	}
}

// TxDifference는 b에 포함되지 않은 a의 트랜잭션을 반환합니다.
func TxDifference(a, b Transactions) Transactions {
	keep := make(Transactions, 0, len(a))

	remove := make(map[common.Hash]struct{})
	for _, tx := range b {
		remove[tx.Hash()] = struct{}{}
	}

	for _, tx := range a {
		if _, ok := remove[tx.Hash()]; !ok {
			keep = append(keep, tx)
		}
	}

	return keep
}

// HashDifference는 b에 포함되지 않은 a의 해시를 반환합니다.
func HashDifference(a, b []common.Hash) []common.Hash {
	keep := make([]common.Hash, 0, len(a))

	remove := make(map[common.Hash]struct{})
	for _, hash := range b {
		remove[hash] = struct{}{}
	}

	for _, hash := range a {
		if _, ok := remove[hash]; !ok {
			keep = append(keep, hash)
		}
	}

	return keep
}

// TxByNonce는 트랜잭션 목록을 nonce로 정렬할 수 있도록 sort 인터페이스를 구현합니다.
// 이는 일반적으로 하나의 계정에서 트랜잭션을 정렬하는 데만 유용하며, 그렇지 않으면 nonce 비교는 큰 의미가 없습니다.
type TxByNonce Transactions

func (s TxByNonce) Len() int           { return len(s) }
func (s TxByNonce) Less(i, j int) bool { return s[i].Nonce() < s[j].Nonce() }
func (s TxByNonce) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// copyAddressPtr는 주소를 복사합니다. (깊은 복사)
func copyAddressPtr(a *common.Address) *common.Address {
	if a == nil {
		return nil
	}
	cpy := *a
	return &cpy
}
