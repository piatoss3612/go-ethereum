// Copyright 2017 The go-ethereum Authors
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

//go:build nacl || js || !cgo || gofuzz
// +build nacl js !cgo gofuzz

package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/btcec/v2"
	btc_ecdsa "github.com/btcsuite/btcd/btcec/v2/ecdsa"
)

// Ecrecover는 주어진 서명을 만든 비압축 공개키를 반환합니다.
func Ecrecover(hash, sig []byte) ([]byte, error) {
	pub, err := sigToPub(hash, sig)
	if err != nil {
		return nil, err
	}
	bytes := pub.SerializeUncompressed()
	return bytes, err
}

func sigToPub(hash, sig []byte) (*btcec.PublicKey, error) {
	if len(sig) != SignatureLength {
		return nil, errors.New("invalid signature")
	}
	// 가장 앞에 '복구 ID' v가 있는 btcec 입력 형식으로 변환합니다.
	btcsig := make([]byte, SignatureLength)
	btcsig[0] = sig[RecoveryIDOffset] + 27
	copy(btcsig[1:], sig)

	pub, _, err := btc_ecdsa.RecoverCompact(btcsig, hash)
	return pub, err
}

// SigToPub는 주어진 서명을 만든 공개키를 반환합니다.
func SigToPub(hash, sig []byte) (*ecdsa.PublicKey, error) {
	pub, err := sigToPub(hash, sig)
	if err != nil {
		return nil, err
	}
	return pub.ToECDSA(), nil
}

// Sign은 ECDSA 서명을 계산합니다.
//
// 이 함수는 서명에 사용되는 개인 키에 대한 정보를 누출할 수 있는 선택된 평문 공격에 취약합니다.
// 호출자는 주어진 다이제스트가 악의적인 사용자에 의해 선택되어서는 안 됨을 인지해야 합니다.
// 일반적인 해결책은 서명을 계산하기 전에 모든 입력을 해시하는 것입니다.
//
// 생성된 서명은 [R || S || V] 형식입니다. 여기서 V는 0 또는 1입니다.
func Sign(hash []byte, prv *ecdsa.PrivateKey) ([]byte, error) {
	if len(hash) != 32 {
		return nil, fmt.Errorf("hash is required to be exactly 32 bytes (%d)", len(hash))
	}
	if prv.Curve != btcec.S256() {
		return nil, errors.New("private key curve is not secp256k1")
	}
	// ecdsa.PrivateKey -> btcec.PrivateKey
	var priv btcec.PrivateKey
	if overflow := priv.Key.SetByteSlice(prv.D.Bytes()); overflow || priv.Key.IsZero() {
		return nil, errors.New("invalid private key")
	}
	defer priv.Zero()
	sig, err := btc_ecdsa.SignCompact(&priv, hash, false) // ref uncompressed pubkey
	if err != nil {
		return nil, err
	}
	// 마지막에 '복구 ID' v가 있는 Ethereum 서명 형식으로 변환합니다.
	v := sig[0] - 27
	copy(sig, sig[1:])
	sig[RecoveryIDOffset] = v
	return sig, nil
}

// VerifySignature는 주어진 공개 키가 다이제스트에 대한 서명을 생성했는지 확인합니다.
// 공개 키는 압축(33바이트) 또는 비압축(65바이트) 형식이어야 합니다.
// 서명은 64바이트 [R || S] 형식이어야 합니다.
func VerifySignature(pubkey, hash, signature []byte) bool {
	if len(signature) != 64 {
		return false
	}
	var r, s btcec.ModNScalar
	if r.SetByteSlice(signature[:32]) {
		return false // overflow
	}
	if s.SetByteSlice(signature[32:]) {
		return false
	}
	sig := btc_ecdsa.NewSignature(&r, &s)
	key, err := btcec.ParsePubKey(pubkey)
	if err != nil {
		return false
	}
	// 비정상적인 서명은 거부합니다. libsecp256k1은 이 검사를 수행하지만 btcec는 수행하지 않습니다.
	if s.IsOverHalfOrder() {
		return false
	}
	return sig.Verify(hash, key)
}

// DecompressPubkey는 33바이트 압축 형식의 공개 키를 구문 분석합니다.
func DecompressPubkey(pubkey []byte) (*ecdsa.PublicKey, error) {
	if len(pubkey) != 33 {
		return nil, errors.New("invalid compressed public key length")
	}
	key, err := btcec.ParsePubKey(pubkey)
	if err != nil {
		return nil, err
	}
	return key.ToECDSA(), nil
}

// CompressPubkey는 공개 키를 33바이트 압축 형식으로 인코딩합니다.
// 제공된 PublicKey는 유효해야 합니다. 즉, 각 좌표는 32바이트보다 크지 않아야 하며,
// 필드의 소수보다 작아야 하며, secp256k1 곡선의 점이어야 합니다.
// 이는 elliptic.Unmarshal(See UnmarshalPubkey) 또는 ToECDSA 및 ecdsa.GenerateKey로
// PrivateKey를 구성할 때 PublicKey가 구성된 경우에 해당합니다.
func CompressPubkey(pubkey *ecdsa.PublicKey) []byte {
	// 참고: 좌표는 btcec.ParsePubKey(FromECDSAPub(pubkey))로 유효성을 검사할 수 있습니다.
	var x, y btcec.FieldVal
	x.SetByteSlice(pubkey.X.Bytes())
	y.SetByteSlice(pubkey.Y.Bytes())
	return btcec.NewPublicKey(&x, &y).SerializeCompressed()
}

// S256는 secp256k1 곡선의 인스턴스를 반환합니다.
func S256() elliptic.Curve {
	return btcec.S256()
}
