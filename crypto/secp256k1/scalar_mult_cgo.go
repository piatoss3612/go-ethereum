// Copyright 2015 Jeffrey Wilcke, Felix Lange, Gustav Simonsson. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be found in
// the LICENSE file.

//go:build !gofuzz && cgo
// +build !gofuzz,cgo

package secp256k1

import (
	"math/big"
	"unsafe"
)

/*

#include "libsecp256k1/include/secp256k1.h"

extern int secp256k1_ext_scalar_mul(const secp256k1_context* ctx, const unsigned char *point, const unsigned char *scalar);

*/
import "C"

func (BitCurve *BitCurve) ScalarMult(Bx, By *big.Int, scalar []byte) (*big.Int, *big.Int) {
	// scala는 정확이 32바이트여야 합니다. 스칼라가 32바이트인 경우에도
	// 사이드 채널 공격을 피하기 위해 항상 패딩합니다.
	if len(scalar) > 32 {
		panic("can't handle scalars > 256 bits")
	}
	// 참고: 잠재적인 타이밍 문제
	padded := make([]byte, 32)
	copy(padded[32-len(scalar):], scalar)
	scalar = padded

	// C에서 곱셈을 수행하고 point를 업데이트합니다.
	point := make([]byte, 64)
	readBits(Bx, point[:32])
	readBits(By, point[32:])

	pointPtr := (*C.uchar)(unsafe.Pointer(&point[0]))
	scalarPtr := (*C.uchar)(unsafe.Pointer(&scalar[0]))
	res := C.secp256k1_ext_scalar_mul(context, pointPtr, scalarPtr)

	// 결과를 언패킹하고 임시 변수를 지웁니다.
	x := new(big.Int).SetBytes(point[:32])
	y := new(big.Int).SetBytes(point[32:])
	for i := range point {
		point[i] = 0
	}
	for i := range padded {
		scalar[i] = 0
	}
	if res != 1 {
		return nil, nil
	}
	return x, y
}
