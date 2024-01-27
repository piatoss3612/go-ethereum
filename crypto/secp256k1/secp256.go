// Copyright 2015 Jeffrey Wilcke, Felix Lange, Gustav Simonsson. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be found in
// the LICENSE file.

//go:build !gofuzz && cgo
// +build !gofuzz,cgo

// Package secp256k1 wraps the bitcoin secp256k1 C library.
package secp256k1

/*
#cgo CFLAGS: -I./libsecp256k1
#cgo CFLAGS: -I./libsecp256k1/src/

#ifdef __SIZEOF_INT128__
#  define HAVE___INT128
#  define USE_FIELD_5X52
#  define USE_SCALAR_4X64
#else
#  define USE_FIELD_10X26
#  define USE_SCALAR_8X32
#endif

#ifndef NDEBUG
#  define NDEBUG
#endif

#define USE_ENDOMORPHISM
#define USE_NUM_NONE
#define USE_FIELD_INV_BUILTIN
#define USE_SCALAR_INV_BUILTIN
#include "./libsecp256k1/src/secp256k1.c"
#include "./libsecp256k1/src/modules/recovery/main_impl.h"
#include "ext.h"

typedef void (*callbackFunc) (const char* msg, void* data);
extern void secp256k1GoPanicIllegal(const char* msg, void* data);
extern void secp256k1GoPanicError(const char* msg, void* data);
*/
import "C"

import (
	"errors"
	"math/big"
	"unsafe"
)

var context *C.secp256k1_context

func init() {
	// around 20 ms on a modern CPU.
	context = C.secp256k1_context_create_sign_verify()
	C.secp256k1_context_set_illegal_callback(context, C.callbackFunc(C.secp256k1GoPanicIllegal), nil)
	C.secp256k1_context_set_error_callback(context, C.callbackFunc(C.secp256k1GoPanicError), nil)
}

var (
	ErrInvalidMsgLen       = errors.New("invalid message length, need 32 bytes")
	ErrInvalidSignatureLen = errors.New("invalid signature length")
	ErrInvalidRecoveryID   = errors.New("invalid signature recovery id")
	ErrInvalidKey          = errors.New("invalid private key")
	ErrInvalidPubkey       = errors.New("invalid public key")
	ErrSignFailed          = errors.New("signing failed")
	ErrRecoverFailed       = errors.New("recovery failed")
)

// Sign은 복구 가능한 ECDSA 서명을 생성합니다.
// 생성된 서명은 V가 0 또는 1인 65바이트 [R || S || V] 형식입니다.
//
// 호출자는 msg가 공격자에 의해 직접 선택될 수 없음을 보장해야 합니다.
// 일반적으로 이 함수에 전달하기 전에 입력에 대해 암호 해시 함수를 사용하는 것이 좋습니다.
func Sign(msg []byte, seckey []byte) ([]byte, error) {
	if len(msg) != 32 {
		return nil, ErrInvalidMsgLen
	}
	if len(seckey) != 32 {
		return nil, ErrInvalidKey
	}
	seckeydata := (*C.uchar)(unsafe.Pointer(&seckey[0]))
	if C.secp256k1_ec_seckey_verify(context, seckeydata) != 1 {
		return nil, ErrInvalidKey
	}

	var (
		msgdata   = (*C.uchar)(unsafe.Pointer(&msg[0]))
		noncefunc = C.secp256k1_nonce_function_rfc6979
		sigstruct C.secp256k1_ecdsa_recoverable_signature
	)
	if C.secp256k1_ecdsa_sign_recoverable(context, &sigstruct, msgdata, seckeydata, noncefunc, nil) == 0 {
		return nil, ErrSignFailed
	}

	var (
		sig     = make([]byte, 65)
		sigdata = (*C.uchar)(unsafe.Pointer(&sig[0]))
		recid   C.int
	)
	C.secp256k1_ecdsa_recoverable_signature_serialize_compact(context, sigdata, &recid, &sigstruct)
	sig[64] = byte(recid) // recid를 다시 추가하여 65바이트 sig를 얻습니다.
	return sig, nil
}

// RecoverPubkey는 서명자의 공개 키를 반환합니다.
// msg는 서명할 메시지의 32바이트 해시여야 합니다.
// sig는 마지막 요소로 복구 ID를 포함하는 65바이트 압축 ECDSA 서명이어야 합니다.
func RecoverPubkey(msg []byte, sig []byte) ([]byte, error) {
	if len(msg) != 32 {
		return nil, ErrInvalidMsgLen
	}
	if err := checkSignature(sig); err != nil {
		return nil, err
	}

	var (
		pubkey  = make([]byte, 65)
		sigdata = (*C.uchar)(unsafe.Pointer(&sig[0]))
		msgdata = (*C.uchar)(unsafe.Pointer(&msg[0]))
	)
	if C.secp256k1_ext_ecdsa_recover(context, (*C.uchar)(unsafe.Pointer(&pubkey[0])), sigdata, msgdata) == 0 {
		return nil, ErrRecoverFailed
	}
	return pubkey, nil
}

// VerifySignature checks that the given pubkey created signature over message.
// The signature should be in [R || S] format.

// VerifySignature는 주어진 공개 키가 메시지에 대한 서명을 생성했는지 확인합니다.
// 서명은 [R || S] 형식이어야 합니다.
func VerifySignature(pubkey, msg, signature []byte) bool {
	if len(msg) != 32 || len(signature) != 64 || len(pubkey) == 0 {
		return false
	}
	sigdata := (*C.uchar)(unsafe.Pointer(&signature[0]))
	msgdata := (*C.uchar)(unsafe.Pointer(&msg[0]))
	keydata := (*C.uchar)(unsafe.Pointer(&pubkey[0]))
	return C.secp256k1_ext_ecdsa_verify(context, sigdata, msgdata, keydata, C.size_t(len(pubkey))) != 0
}

// DecompressPubkey는 33바이트 압축 형식의 공개 키를 구문 분석합니다.
// 공개 키가 유효하면 nil이 아닌 좌표를 반환합니다.
func DecompressPubkey(pubkey []byte) (x, y *big.Int) {
	if len(pubkey) != 33 {
		return nil, nil
	}
	var (
		pubkeydata = (*C.uchar)(unsafe.Pointer(&pubkey[0]))
		pubkeylen  = C.size_t(len(pubkey))
		out        = make([]byte, 65)
		outdata    = (*C.uchar)(unsafe.Pointer(&out[0]))
		outlen     = C.size_t(len(out))
	)
	if C.secp256k1_ext_reencode_pubkey(context, outdata, outlen, pubkeydata, pubkeylen) == 0 {
		return nil, nil
	}
	return new(big.Int).SetBytes(out[1:33]), new(big.Int).SetBytes(out[33:])
}

// CompressPubkey는 공개 키를 33바이트 압축 형식으로 인코딩합니다.
func CompressPubkey(x, y *big.Int) []byte {
	var (
		pubkey     = S256().Marshal(x, y)
		pubkeydata = (*C.uchar)(unsafe.Pointer(&pubkey[0]))
		pubkeylen  = C.size_t(len(pubkey))
		out        = make([]byte, 33)
		outdata    = (*C.uchar)(unsafe.Pointer(&out[0]))
		outlen     = C.size_t(len(out))
	)
	if C.secp256k1_ext_reencode_pubkey(context, outdata, outlen, pubkeydata, pubkeylen) == 0 {
		panic("libsecp256k1 error")
	}
	return out
}

func checkSignature(sig []byte) error {
	if len(sig) != 65 {
		return ErrInvalidSignatureLen
	}
	if sig[64] >= 4 {
		return ErrInvalidRecoveryID
	}
	return nil
}
