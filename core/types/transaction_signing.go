// Copyright 2016 The go-ethereum Authors
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
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

var ErrInvalidChainId = errors.New("invalid chain id for signer")

// sigCache는 서명자와 함께 파생된 발신자를 캐시하는 데 사용됩니다.
type sigCache struct {
	signer Signer
	from   common.Address
}

// MakeSigner는 주어진 체인 설정 및 블록 번호를 기반으로 Signer를 생성하여 반환합니다.
func MakeSigner(config *params.ChainConfig, blockNumber *big.Int, blockTime uint64) Signer {
	var signer Signer
	switch {
	case config.IsCancun(blockNumber, blockTime): // Cancun
		signer = NewCancunSigner(config.ChainID)
	case config.IsLondon(blockNumber): // London
		signer = NewLondonSigner(config.ChainID)
	case config.IsBerlin(blockNumber): // Berlin
		signer = NewEIP2930Signer(config.ChainID)
	case config.IsEIP155(blockNumber): // EIP155
		signer = NewEIP155Signer(config.ChainID)
	case config.IsHomestead(blockNumber): // Homestead
		signer = HomesteadSigner{}
	default: // 그 외의 경우 (이더리움 프론티어)
		signer = FrontierSigner{}
	}
	return signer
}

// LatestSigner는 주어진 체인 구성에 대해 사용 가능한 '가장 허용 범위가 넓은' 서명자(Signer)를 반환합니다.
// 구체적으로, 이 함수는 모든 유형의 트랜잭션을 지원하도록 합니다.
// 이는 해당 포크가 체인 구성의 어떤 블록 번호(또는 시간)에서 발생할 예정인지 여부와 관계없이 가능합니다.
//
// 현재 블록 번호를 알 수 없는 트랜잭션을 처리하는 코드에서 이 함수를 사용하면 됩니다.
// 현재 블록 번호를 사용할 수 있는 경우 MakeSigner를 사용하십시오.
func LatestSigner(config *params.ChainConfig) Signer {
	if config.ChainID != nil {
		if config.CancunTime != nil { // Cancun
			return NewCancunSigner(config.ChainID)
		}
		if config.LondonBlock != nil { // London
			return NewLondonSigner(config.ChainID)
		}
		if config.BerlinBlock != nil { // Berlin
			return NewEIP2930Signer(config.ChainID)
		}
		if config.EIP155Block != nil { // EIP155
			return NewEIP155Signer(config.ChainID)
		}
	}
	return HomesteadSigner{} // Homestead
}

// LatestSignerForChainID는 사용 가능한 '가장 허용 범위가 넓은' 서명자(Signer)를 반환합니다.
// chainID가 nil이 아닌 경우, 이 함수는 EIP-155 재생 방지 및 모든 EIP-2718 트랜잭션 유형을 지원합니다.
//
// 현재 블록 번호와 포크 구성이 알려지지 않은 트랜잭션 처리 코드에서 이 함수를 사용하면 됩니다.
// ChainConfig가 있는 경우 LatestSigner를 사용하십시오.
// ChainConfig가 있고 현재 블록 번호를 알고 있는 경우 MakeSigner를 사용하십시오.
func LatestSignerForChainID(chainID *big.Int) Signer {
	if chainID == nil {
		return HomesteadSigner{}
	}
	return NewCancunSigner(chainID)
}

// SignTx는 주어진 서명자와 개인 키를 사용하여 트랜잭션에 서명합니다.
func SignTx(tx *Transaction, s Signer, prv *ecdsa.PrivateKey) (*Transaction, error) {
	h := s.Hash(tx)                    // 서명 해시 생성 (Signer에 따라 다르게 생성됨)
	sig, err := crypto.Sign(h[:], prv) // 개인 키로 서명 (직렬화된 서명 데이터 반환)
	if err != nil {
		return nil, err
	}
	return tx.WithSignature(s, sig) // 트랜잭션에 서명 데이터 추가 (V, R, S 값 설정 + 서명자의 체인 ID 설정)
}

// SignNewTx는 트랜잭션을 생성하고 서명합니다.
func SignNewTx(prv *ecdsa.PrivateKey, s Signer, txdata TxData) (*Transaction, error) {
	tx := NewTx(txdata)
	h := s.Hash(tx)
	sig, err := crypto.Sign(h[:], prv)
	if err != nil {
		return nil, err
	}
	return tx.WithSignature(s, sig)
}

// MustSignNewTx는 트랜잭션을 생성하고 서명합니다.
// 트랜잭션에 서명할 수 없는 경우 패닉이 발생합니다.
func MustSignNewTx(prv *ecdsa.PrivateKey, s Signer, txdata TxData) *Transaction {
	tx, err := SignNewTx(prv, s, txdata)
	if err != nil {
		panic(err)
	}
	return tx
}

// Sender는 secp256k1 타원 곡선을 사용하여 서명(V, R, S)에서 파생된 주소를 반환하고
// 이 작업이 실패하거나 서명이 잘못된 경우 오류를 반환합니다.
//
// Sender는 서명 방법과 관계없이 주소를 사용할 수 있도록 캐시할 수 있습니다.
// 캐시는 현재 호출에서 사용된 서명자가 캐시된 서명자와 일치하지 않는 경우 무효화됩니다.
func Sender(signer Signer, tx *Transaction) (common.Address, error) {
	if sc := tx.from.Load(); sc != nil {
		sigCache := sc.(sigCache)
		// 이전 호출에서 사용된 서명자가 현재 서명자와 일치하는지 확인합니다.
		if sigCache.signer.Equal(signer) {
			return sigCache.from, nil
		}
	}

	// 서명자가 일치하지 않는 경우 서명자를 다시 계산합니다.
	addr, err := signer.Sender(tx)
	if err != nil {
		return common.Address{}, err
	}
	// 서명자를 캐시합니다.
	tx.from.Store(sigCache{signer: signer, from: addr})
	return addr, nil
}

// Signer는 트랜잭션 서명 처리 기능을 캡슐화합니다. 이 타입의 이름은 약간 오해의 소지가 있습니다.
// 왜냐하면 Signer는 실제로 서명하지 않고 서명을 검증하고 처리하기 위한 것이기 때문입니다.
//
// 참고로 이 인터페이스는 안정적인 API가 아니며 새로운 프로토콜 규칙을 수용하기 위해 언제든지 변경될 수 있습니다.
type Signer interface {
	// Sender는 트랜잭션의 발신자 주소를 반환합니다.
	Sender(tx *Transaction) (common.Address, error)

	// SignatureValues는 주어진 서명에 해당하는 원시 R, S, V 값을 반환합니다.
	SignatureValues(tx *Transaction, sig []byte) (r, s, v *big.Int, err error)
	ChainID() *big.Int

	// Hash는 '서명 해시'를 반환합니다. 즉, 개인 키를 사용하여 서명되기 전의 트랜잭션 해시입니다.
	// 이 해시는 트랜잭션을 고유하게 식별하지는 않습니다.
	Hash(tx *Transaction) common.Hash

	// Equal은 주어진 서명자가 수신자와 동일한지 여부를 반환합니다.
	Equal(Signer) bool
}

type cancunSigner struct{ londonSigner }

// NewCancunSigner는 다음을 허용하는 서명자를 반환합니다.
// - EIP-4844 blob transactions
// - EIP-1559 dynamic fee transactions
// - EIP-2930 access list transactions,
// - EIP-155 replay protected transactions, 그리고
// - legacy Homestead transactions. (모든 유형의 트랜잭션을 지원합니다.)
func NewCancunSigner(chainId *big.Int) Signer {
	return cancunSigner{londonSigner{eip2930Signer{NewEIP155Signer(chainId)}}}
}

func (s cancunSigner) Sender(tx *Transaction) (common.Address, error) {
	if tx.Type() != BlobTxType { // Blob 트랜잭션이 아닌 경우 -> London
		return s.londonSigner.Sender(tx)
	}
	// Blob 트랜잭션인 경우
	V, R, S := tx.RawSignatureValues() // 서명 값 추출 (V는 0 또는 1)
	// Blob 트랜잭션은 복구 ID로 0과 1을 사용하도록 정의되어 있습니다.
	// 27을 더하여 보호되지 않은 Homestead 서명과 동일하게 만듭니다.
	V = new(big.Int).Add(V, big.NewInt(27))
	if tx.ChainId().Cmp(s.chainId) != 0 {
		return common.Address{}, fmt.Errorf("%w: have %d want %d", ErrInvalidChainId, tx.ChainId(), s.chainId)
	}
	return recoverPlain(s.Hash(tx), R, S, V, true)
}

func (s cancunSigner) Equal(s2 Signer) bool {
	x, ok := s2.(cancunSigner)
	return ok && x.chainId.Cmp(s.chainId) == 0
}

func (s cancunSigner) SignatureValues(tx *Transaction, sig []byte) (R, S, V *big.Int, err error) {
	txdata, ok := tx.inner.(*BlobTx) // Blob 트랜잭션이 아닌 경우 -> London
	if !ok {
		return s.londonSigner.SignatureValues(tx, sig)
	}
	// txdata의 체인 ID는 0이 아니어야 하며, 서명자의 체인 ID와 일치해야 합니다.
	// txdata의 체인 ID가 0이라는 것은 tx에서 체인 ID가 지정되지 않았음을 의미합니다.
	if txdata.ChainID.Sign() != 0 && txdata.ChainID.ToBig().Cmp(s.chainId) != 0 {
		return nil, nil, nil, fmt.Errorf("%w: have %d want %d", ErrInvalidChainId, txdata.ChainID, s.chainId)
	}
	R, S, _ = decodeSignature(sig)
	V = big.NewInt(int64(sig[64]))
	return R, S, V, nil
}

// Hash는 발신자에 의해 서명될 해시를 반환합니다.
// 이는 트랜잭션을 고유하게 식별하지는 않습니다.
func (s cancunSigner) Hash(tx *Transaction) common.Hash {
	if tx.Type() != BlobTxType {
		return s.londonSigner.Hash(tx)
	}
	return prefixedRlpHash(
		tx.Type(),
		[]interface{}{
			s.chainId,
			tx.Nonce(),
			tx.GasTipCap(),
			tx.GasFeeCap(),
			tx.Gas(),
			tx.To(),
			tx.Value(),
			tx.Data(),
			tx.AccessList(),
			tx.BlobGasFeeCap(),
			tx.BlobHashes(),
		})
}

type londonSigner struct{ eip2930Signer }

// NewLondonSigner는 다음을 허용하는 서명자를 반환합니다.
// - EIP-1559 dynamic fee transactions
// - EIP-2930 access list transactions,
// - EIP-155 replay protected transactions, 그리고
// - legacy Homestead transactions. (EIP-4844 blob transactions은 지원하지 않습니다.)
func NewLondonSigner(chainId *big.Int) Signer {
	return londonSigner{eip2930Signer{NewEIP155Signer(chainId)}}
}

func (s londonSigner) Sender(tx *Transaction) (common.Address, error) {
	if tx.Type() != DynamicFeeTxType { // DynamicFee 트랜잭션이 아닌 경우 -> EIP-2930
		return s.eip2930Signer.Sender(tx)
	}
	// DynamicFee 트랜잭션인 경우
	V, R, S := tx.RawSignatureValues()
	// DynamicFee txs는 복구 ID로 0과 1을 사용하도록 정의되어 있습니다.
	// 27을 더하여 보호되지 않은 Homestead 서명과 동일하게 만듭니다.
	V = new(big.Int).Add(V, big.NewInt(27))
	if tx.ChainId().Cmp(s.chainId) != 0 {
		return common.Address{}, fmt.Errorf("%w: have %d want %d", ErrInvalidChainId, tx.ChainId(), s.chainId)
	}
	return recoverPlain(s.Hash(tx), R, S, V, true)
}

func (s londonSigner) Equal(s2 Signer) bool {
	x, ok := s2.(londonSigner)
	return ok && x.chainId.Cmp(s.chainId) == 0
}

func (s londonSigner) SignatureValues(tx *Transaction, sig []byte) (R, S, V *big.Int, err error) {
	txdata, ok := tx.inner.(*DynamicFeeTx)
	if !ok {
		return s.eip2930Signer.SignatureValues(tx, sig)
	}
	// txdata의 체인 ID는 0이 아니어야 하며, 서명자의 체인 ID와 일치해야 합니다.
	// txdata의 체인 ID가 0이라는 것은 tx에서 체인 ID가 지정되지 않았음을 의미합니다.
	if txdata.ChainID.Sign() != 0 && txdata.ChainID.Cmp(s.chainId) != 0 {
		return nil, nil, nil, fmt.Errorf("%w: have %d want %d", ErrInvalidChainId, txdata.ChainID, s.chainId)
	}
	R, S, _ = decodeSignature(sig)
	V = big.NewInt(int64(sig[64]))
	return R, S, V, nil
}

// Hash는 발신자에 의해 서명될 해시를 반환합니다.
// 이는 트랜잭션을 고유하게 식별하지는 않습니다.
func (s londonSigner) Hash(tx *Transaction) common.Hash {
	if tx.Type() != DynamicFeeTxType {
		return s.eip2930Signer.Hash(tx)
	}
	return prefixedRlpHash(
		tx.Type(),
		[]interface{}{
			s.chainId,
			tx.Nonce(),
			tx.GasTipCap(),
			tx.GasFeeCap(),
			tx.Gas(),
			tx.To(),
			tx.Value(),
			tx.Data(),
			tx.AccessList(),
		})
}

type eip2930Signer struct{ EIP155Signer }

// NewEIP2930Signer는 EIP-2930 액세스 목록 트랜잭션, EIP-155 재생 방지 트랜잭션 및
// 레거시 Homestead 트랜잭션을 허용하는 서명자를 반환합니다.
func NewEIP2930Signer(chainId *big.Int) Signer {
	return eip2930Signer{NewEIP155Signer(chainId)}
}

func (s eip2930Signer) ChainID() *big.Int {
	return s.chainId
}

func (s eip2930Signer) Equal(s2 Signer) bool {
	x, ok := s2.(eip2930Signer)
	return ok && x.chainId.Cmp(s.chainId) == 0
}

func (s eip2930Signer) Sender(tx *Transaction) (common.Address, error) {
	V, R, S := tx.RawSignatureValues()
	switch tx.Type() {
	case LegacyTxType: // Legacy 트랜잭션인 경우
		return s.EIP155Signer.Sender(tx) // EIP-155
	case AccessListTxType:
		// 접근 목록 트랜잭션은 복구 ID로 0과 1을 사용하도록 정의되어 있습니다.
		// 27을 더하여 보호되지 않은 Homestead 서명과 동일하게 만듭니다.
		V = new(big.Int).Add(V, big.NewInt(27))
	default:
		return common.Address{}, ErrTxTypeNotSupported
	}
	if tx.ChainId().Cmp(s.chainId) != 0 {
		return common.Address{}, fmt.Errorf("%w: have %d want %d", ErrInvalidChainId, tx.ChainId(), s.chainId)
	}
	return recoverPlain(s.Hash(tx), R, S, V, true)
}

func (s eip2930Signer) SignatureValues(tx *Transaction, sig []byte) (R, S, V *big.Int, err error) {
	switch txdata := tx.inner.(type) {
	case *LegacyTx: // Legacy 트랜잭션인 경우
		return s.EIP155Signer.SignatureValues(tx, sig) // EIP-155
	case *AccessListTx: // 접근 목록 트랜잭션인 경우
		// txdata의 체인 ID는 0이 아니어야 하며, 서명자의 체인 ID와 일치해야 합니다.
		// txdata의 체인 ID가 0이라는 것은 tx에서 체인 ID가 지정되지 않았음을 의미합니다.
		if txdata.ChainID.Sign() != 0 && txdata.ChainID.Cmp(s.chainId) != 0 {
			return nil, nil, nil, fmt.Errorf("%w: have %d want %d", ErrInvalidChainId, txdata.ChainID, s.chainId)
		}
		R, S, _ = decodeSignature(sig)
		V = big.NewInt(int64(sig[64]))
	default:
		return nil, nil, nil, ErrTxTypeNotSupported
	}
	return R, S, V, nil
}

// Hash는 발신자에 의해 서명될 해시를 반환합니다.
// 이는 트랜잭션을 고유하게 식별하지는 않습니다.
func (s eip2930Signer) Hash(tx *Transaction) common.Hash {
	switch tx.Type() {
	case LegacyTxType: // Legacy 트랜잭션인 경우
		return s.EIP155Signer.Hash(tx)
	case AccessListTxType: // 접근 목록 트랜잭션인 경우
		return prefixedRlpHash(
			tx.Type(),
			[]interface{}{
				s.chainId,
				tx.Nonce(),
				tx.GasPrice(),
				tx.Gas(),
				tx.To(),
				tx.Value(),
				tx.Data(),
				tx.AccessList(),
			})
	default:
		// 어떤 타입과도 일치하지 않는 경우, 빈 해시를 반환합니다.
		// 이러한 경우는 어떤 경우에도 발생하지 않아야 하지만, 아마도 누군가가 RPC를 통해 잘못된 json 구조를 보내는 경우가 있을 수 있으므로
		// 노드를 패닉으로 죽이는 대신 빈 해시를 반환하는 것이 더 조심스러울 것입니다.
		return common.Hash{}
	}
}

// EIP155Signer는 EIP-155 규칙을 사용하여 서명자를 구현합니다. 이는 재생 방지 트랜잭션과 보호되지 않은 Homestead 트랜잭션을 모두 허용합니다.
type EIP155Signer struct {
	chainId, chainIdMul *big.Int
}

func NewEIP155Signer(chainId *big.Int) EIP155Signer {
	if chainId == nil {
		chainId = new(big.Int)
	}
	return EIP155Signer{
		chainId:    chainId,
		chainIdMul: new(big.Int).Mul(chainId, big.NewInt(2)),
	}
}

func (s EIP155Signer) ChainID() *big.Int {
	return s.chainId
}

func (s EIP155Signer) Equal(s2 Signer) bool {
	eip155, ok := s2.(EIP155Signer)
	return ok && eip155.chainId.Cmp(s.chainId) == 0
}

var big8 = big.NewInt(8)

func (s EIP155Signer) Sender(tx *Transaction) (common.Address, error) {
	if tx.Type() != LegacyTxType {
		return common.Address{}, ErrTxTypeNotSupported
	}
	if !tx.Protected() {
		return HomesteadSigner{}.Sender(tx)
	}
	if tx.ChainId().Cmp(s.chainId) != 0 {
		return common.Address{}, fmt.Errorf("%w: have %d want %d", ErrInvalidChainId, tx.ChainId(), s.chainId)
	}
	V, R, S := tx.RawSignatureValues()
	V = new(big.Int).Sub(V, s.chainIdMul)
	V.Sub(V, big8)
	return recoverPlain(s.Hash(tx), R, S, V, true)
}

// SignatureValues는 서명 값을 반환합니다. 이 서명은 V가 0 또는 1인 [R || S || V] 형식이어야 합니다.
func (s EIP155Signer) SignatureValues(tx *Transaction, sig []byte) (R, S, V *big.Int, err error) {
	if tx.Type() != LegacyTxType {
		return nil, nil, nil, ErrTxTypeNotSupported
	}
	R, S, V = decodeSignature(sig)
	if s.chainId.Sign() != 0 {
		V = big.NewInt(int64(sig[64] + 35))
		V.Add(V, s.chainIdMul)
	}
	return R, S, V, nil
}

// Hash는 발신자에 의해 서명될 해시를 반환합니다.
// 이는 트랜잭션을 고유하게 식별하지는 않습니다.
func (s EIP155Signer) Hash(tx *Transaction) common.Hash {
	return rlpHash([]interface{}{
		tx.Nonce(),
		tx.GasPrice(),
		tx.Gas(),
		tx.To(),
		tx.Value(),
		tx.Data(),
		s.chainId, uint(0), uint(0),
	})
}

// HomesteadSigner는 Homestead 규칙을 사용하여 서명자를 구현합니다.
type HomesteadSigner struct{ FrontierSigner }

func (s HomesteadSigner) ChainID() *big.Int {
	return nil
}

func (s HomesteadSigner) Equal(s2 Signer) bool {
	_, ok := s2.(HomesteadSigner)
	return ok
}

// SignatureValues는 서명 값을 반환합니다. 이 서명은 V가 0 또는 1인 [R || S || V] 형식이어야 합니다.
func (hs HomesteadSigner) SignatureValues(tx *Transaction, sig []byte) (r, s, v *big.Int, err error) {
	return hs.FrontierSigner.SignatureValues(tx, sig)
}

func (hs HomesteadSigner) Sender(tx *Transaction) (common.Address, error) {
	if tx.Type() != LegacyTxType {
		return common.Address{}, ErrTxTypeNotSupported
	}
	v, r, s := tx.RawSignatureValues()
	return recoverPlain(hs.Hash(tx), r, s, v, true)
}

// FrontierSigner는 프론티어 규칙을 사용하여 서명자를 구현합니다.
type FrontierSigner struct{}

func (s FrontierSigner) ChainID() *big.Int {
	return nil
}

func (s FrontierSigner) Equal(s2 Signer) bool {
	_, ok := s2.(FrontierSigner)
	return ok
}

func (fs FrontierSigner) Sender(tx *Transaction) (common.Address, error) {
	if tx.Type() != LegacyTxType {
		return common.Address{}, ErrTxTypeNotSupported
	}
	v, r, s := tx.RawSignatureValues()
	return recoverPlain(fs.Hash(tx), r, s, v, false)
}

// SignatureValues는 서명 값을 반환합니다. 이 서명은 V가 0 또는 1인 [R || S || V] 형식이어야 합니다.
func (fs FrontierSigner) SignatureValues(tx *Transaction, sig []byte) (r, s, v *big.Int, err error) {
	if tx.Type() != LegacyTxType {
		return nil, nil, nil, ErrTxTypeNotSupported
	}
	r, s, v = decodeSignature(sig)
	return r, s, v, nil
}

// Hash는 발신자에 의해 서명될 해시를 반환합니다.
// 이는 트랜잭션을 고유하게 식별하지는 않습니다.
func (fs FrontierSigner) Hash(tx *Transaction) common.Hash {
	return rlpHash([]interface{}{
		tx.Nonce(),
		tx.GasPrice(),
		tx.Gas(),
		tx.To(),
		tx.Value(),
		tx.Data(),
	})
}

func decodeSignature(sig []byte) (r, s, v *big.Int) {
	if len(sig) != crypto.SignatureLength {
		panic(fmt.Sprintf("wrong size for signature: got %d, want %d", len(sig), crypto.SignatureLength))
	}
	r = new(big.Int).SetBytes(sig[:32])             // 서명의 처음 32바이트는 R 값입니다.
	s = new(big.Int).SetBytes(sig[32:64])           // 서명의 다음 32바이트는 S 값입니다.
	v = new(big.Int).SetBytes([]byte{sig[64] + 27}) // 마지막 바이트는 V 값입니다.
	return r, s, v
}

func recoverPlain(sighash common.Hash, R, S, Vb *big.Int, homestead bool) (common.Address, error) {
	if Vb.BitLen() > 8 {
		return common.Address{}, ErrInvalidSig
	}
	V := byte(Vb.Uint64() - 27)
	if !crypto.ValidateSignatureValues(V, R, S, homestead) {
		return common.Address{}, ErrInvalidSig
	}
	// 비압축 형식으로 서명을 인코딩합니다.
	r, s := R.Bytes(), S.Bytes()
	sig := make([]byte, crypto.SignatureLength)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = V
	// 서명으로부터 공개 키를 복구합니다.
	pub, err := crypto.Ecrecover(sighash[:], sig)
	if err != nil {
		return common.Address{}, err
	}
	if len(pub) == 0 || pub[0] != 4 {
		return common.Address{}, errors.New("invalid public key")
	}
	var addr common.Address
	copy(addr[:], crypto.Keccak256(pub[1:])[12:]) // 공개 키의 끝 20바이트를 사용하여 주소를 생성합니다.
	return addr, nil
}

// deriveChainId는 주어진 v 매개변수에서 체인 ID를 추출합니다.
func deriveChainId(v *big.Int) *big.Int {
	if v.BitLen() <= 64 { // v가 64비트 이하인 경우
		v := v.Uint64()
		if v == 27 || v == 28 { // 레거시 서명
			return new(big.Int) // 체인 ID가 없음
		}
		return new(big.Int).SetUint64((v - 35) / 2) // EIP-155 이후 서명 ({0, 1}에 체인 ID * 2 + 35를 더해서 V를 구했으므로 역으로 계산)
	}
	v = new(big.Int).Sub(v, big.NewInt(35))
	return v.Div(v, big.NewInt(2))
}
