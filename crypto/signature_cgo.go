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

//go:build !nacl && !js && cgo && !gofuzz
// +build !nacl,!js,cgo,!gofuzz

package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
)

// Ecrecover는 주어진 서명을 만든 비압축 공개키를 반환합니다.
func Ecrecover(hash, sig []byte) ([]byte, error) {
	return secp256k1.RecoverPubkey(hash, sig)
}

// SigToPub는 주어진 서명을 만든 공개키를 반환합니다.
func SigToPub(hash, sig []byte) (*ecdsa.PublicKey, error) {
	s, err := Ecrecover(hash, sig)
	if err != nil {
		return nil, err
	}

	x, y := elliptic.Unmarshal(S256(), s)
	return &ecdsa.PublicKey{Curve: S256(), X: x, Y: y}, nil
}

// Sign은 ECDSA 서명을 계산합니다.
//
// 이 함수는 서명에 사용되는 개인 키에 대한 정보를 누출할 수 있는 선택된 평문 공격에 취약합니다.
// 호출자는 주어진 다이제스트가 악의적인 사용자에 의해 선택되어서는 안 됨을 인지해야 합니다.
// 일반적인 해결책은 서명을 계산하기 전에 모든 입력을 해시하는 것입니다.
//
// 생성된 서명은 [R || S || V] 형식입니다. 여기서 V는 0 또는 1입니다.
func Sign(digestHash []byte, prv *ecdsa.PrivateKey) (sig []byte, err error) {
	if len(digestHash) != DigestLength {
		return nil, fmt.Errorf("hash is required to be exactly %d bytes (%d)", DigestLength, len(digestHash))
	}
	seckey := math.PaddedBigBytes(prv.D, prv.Params().BitSize/8)
	defer zeroBytes(seckey)
	return secp256k1.Sign(digestHash, seckey)
}

// VerifySignature는 주어진 공개 키가 다이제스트에 대한 서명을 생성했는지 확인합니다.
// 공개 키는 압축(33바이트) 또는 비압축(65바이트) 형식이어야 합니다.
// 서명은 64바이트 [R || S] 형식이어야 합니다.
func VerifySignature(pubkey, digestHash, signature []byte) bool {
	return secp256k1.VerifySignature(pubkey, digestHash, signature)
}

// DecompressPubkey는 33바이트 압축 형식의 공개 키를 구문 분석합니다.
func DecompressPubkey(pubkey []byte) (*ecdsa.PublicKey, error) {
	x, y := secp256k1.DecompressPubkey(pubkey)
	if x == nil {
		return nil, errors.New("invalid public key")
	}
	return &ecdsa.PublicKey{X: x, Y: y, Curve: S256()}, nil
}

// CompressPubkey는 공개 키를 33바이트 압축 형식으로 인코딩합니다.
func CompressPubkey(pubkey *ecdsa.PublicKey) []byte {
	return secp256k1.CompressPubkey(pubkey.X, pubkey.Y)
}

// S256는 secp256k1 곡선의 인스턴스를 반환합니다.
func S256() elliptic.Curve {
	return secp256k1.S256()
}
