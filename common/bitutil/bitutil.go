// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Adapted from: https://golang.org/src/crypto/cipher/xor.go

// bitutil 패키지는 빠른 비트 연산을 구현합니다.
package bitutil

import (
	"runtime"
	"unsafe"
)

const wordSize = int(unsafe.Sizeof(uintptr(0)))
const supportsUnaligned = runtime.GOARCH == "386" || runtime.GOARCH == "amd64" || runtime.GOARCH == "ppc64" || runtime.GOARCH == "ppc64le" || runtime.GOARCH == "s390x" // 해당 아키텍처가 비정렬 메모리 접근을 지원하는지 확인

// XORBytes 함수는 a와 b의 바이트를 XOR합니다. 결과를 저장할 dst의 공간이 충분하다고 가정합니다.
// XOR 연산을 수행한 바이트 수를 반환합니다.
func XORBytes(dst, a, b []byte) int {
	if supportsUnaligned { // 비정렬 메모리 접근을 지원하는 경우
		return fastXORBytes(dst, a, b)
	}
	return safeXORBytes(dst, a, b)
}

// fastXORBytes는 대량의 XOR 연산을 수행합니다. 비정렬 메모리 접근을 지원하는 아키텍처에서만 동작합니다.
func fastXORBytes(dst, a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	w := n / wordSize
	if w > 0 {
		dw := *(*[]uintptr)(unsafe.Pointer(&dst))
		aw := *(*[]uintptr)(unsafe.Pointer(&a))
		bw := *(*[]uintptr)(unsafe.Pointer(&b))
		for i := 0; i < w; i++ {
			dw[i] = aw[i] ^ bw[i]
		}
	}
	for i := n - n%wordSize; i < n; i++ {
		dst[i] = a[i] ^ b[i]
	}
	return n
}

// safeXORBytes는 하나씩 XOR 연산을 수행합니다. 비정렬 메모리 접근을 지원하는지 여부와 상관없이 모든 아키텍처에서 동작합니다.
func safeXORBytes(dst, a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ { // 하나씩 XOR 연산을 수행합니다.
		dst[i] = a[i] ^ b[i]
	}
	return n
}

// ANDBytes는 a와 b의 바이트를 AND 연산합니다. 결과를 저장할 dst의 공간이 충분하다고 가정합니다.
// AND 연산을 수행한 바이트 수를 반환합니다.
func ANDBytes(dst, a, b []byte) int {
	if supportsUnaligned { // 비정렬 메모리 접근을 지원하는 경우
		return fastANDBytes(dst, a, b)
	}
	return safeANDBytes(dst, a, b)
}

// fastANDBytes는 대량의 AND 연산을 수행합니다. 비정렬 메모리 접근을 지원하는 아키텍처에서만 동작합니다.
func fastANDBytes(dst, a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	w := n / wordSize
	if w > 0 {
		dw := *(*[]uintptr)(unsafe.Pointer(&dst))
		aw := *(*[]uintptr)(unsafe.Pointer(&a))
		bw := *(*[]uintptr)(unsafe.Pointer(&b))
		for i := 0; i < w; i++ {
			dw[i] = aw[i] & bw[i]
		}
	}
	for i := n - n%wordSize; i < n; i++ {
		dst[i] = a[i] & b[i]
	}
	return n
}

// safeANDBytes는 하나씩 AND 연산을 수행합니다. 비정렬 메모리 접근을 지원하는지 여부와 상관없이 모든 아키텍처에서 동작합니다.
func safeANDBytes(dst, a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		dst[i] = a[i] & b[i]
	}
	return n
}

// ORBytes는 a와 b의 바이트를 OR 연산합니다. 결과를 저장할 dst의 공간이 충분하다고 가정합니다.
// OR 연산을 수행한 바이트 수를 반환합니다.
func ORBytes(dst, a, b []byte) int {
	if supportsUnaligned { // 비정렬 메모리 접근을 지원하는 경우
		return fastORBytes(dst, a, b)
	}
	return safeORBytes(dst, a, b)
}

// fastORBytes는 대량의 OR 연산을 수행합니다. 비정렬 메모리 접근을 지원하는 아키텍처에서만 동작합니다.
func fastORBytes(dst, a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	w := n / wordSize
	if w > 0 {
		dw := *(*[]uintptr)(unsafe.Pointer(&dst))
		aw := *(*[]uintptr)(unsafe.Pointer(&a))
		bw := *(*[]uintptr)(unsafe.Pointer(&b))
		for i := 0; i < w; i++ {
			dw[i] = aw[i] | bw[i]
		}
	}
	for i := n - n%wordSize; i < n; i++ {
		dst[i] = a[i] | b[i]
	}
	return n
}

// safeORBytes는 하나씩 OR 연산을 수행합니다. 비정렬 메모리 접근을 지원하는지 여부와 상관없이 모든 아키텍처에서 동작합니다.
func safeORBytes(dst, a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		dst[i] = a[i] | b[i]
	}
	return n
}

// TestBytes는 입력 바이트 슬라이스에 설정된 비트가 있는지 확인합니다.
func TestBytes(p []byte) bool {
	if supportsUnaligned { // 비정렬 메모리 접근을 지원하는 경우
		return fastTestBytes(p)
	}
	return safeTestBytes(p)
}

// fastTestBytes는 대량의 비트를 확인합니다. 비정렬 메모리 접근을 지원하는 아키텍처에서만 동작합니다.
func fastTestBytes(p []byte) bool {
	n := len(p)
	w := n / wordSize
	if w > 0 {
		pw := *(*[]uintptr)(unsafe.Pointer(&p))
		for i := 0; i < w; i++ {
			if pw[i] != 0 {
				return true
			}
		}
	}
	for i := n - n%wordSize; i < n; i++ {
		if p[i] != 0 {
			return true
		}
	}
	return false
}

// safeTestBytes는 하나씩 비트를 확인합니다. 모든 아키텍처에서 동작합니다.
func safeTestBytes(p []byte) bool {
	for i := 0; i < len(p); i++ {
		if p[i] != 0 {
			return true
		}
	}
	return false
}
