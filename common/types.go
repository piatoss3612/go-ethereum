// Copyright 2015 The go-ethereum Authors
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

package common

import (
	"bytes"
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"reflect"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"golang.org/x/crypto/sha3"
)

// 해시와 주소의 길이 (바이트 단위)
const (
	// 해시의 길이는 32바이트
	HashLength = 32
	// 주소의 길이는 20바이트
	AddressLength = 20
)

var (
	hashT    = reflect.TypeOf(Hash{})
	addressT = reflect.TypeOf(Address{})

	// MaxAddress는 가능한 주소 값의 최대값을 나타냅니다.
	MaxAddress = HexToAddress("0xffffffffffffffffffffffffffffffffffffffff")

	// MaxHash는 가능한 해시 값의 최대값을 나타냅니다.
	MaxHash = HexToHash("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
)

// Hash는 임의의 데이터의 Keccak256 해시를 나타냅니다. (32바이트)
type Hash [HashLength]byte

// BytesToHash sets b to hash.
// BytesToHash는 b를 해시로 설정합니다.
// 만약 b의 길이가 HashLength보다 크다면, b는 왼쪽에서부터 잘립니다.
func BytesToHash(b []byte) Hash {
	var h Hash
	h.SetBytes(b)
	return h
}

// BigToHash는 big.Int의 바이트 표현을 해시로 설정합니다.
// 만약 b의 길이가 HashLength보다 크다면, b는 왼쪽에서부터 잘립니다.
func BigToHash(b *big.Int) Hash { return BytesToHash(b.Bytes()) }

// HexToHash는 s의 바이트 표현을 해시로 설정합니다.
// 만약 s의 길이가 HashLength보다 크다면, s는 왼쪽에서부터 잘립니다.
func HexToHash(s string) Hash { return BytesToHash(FromHex(s)) }

// Cmp는 두 해시를 비교합니다. (0: 같음, -1: h < other, +1: h > other)
func (h Hash) Cmp(other Hash) int {
	return bytes.Compare(h[:], other[:])
}

// Bytes는 해시의 바이트 표현을 반환합니다.
func (h Hash) Bytes() []byte { return h[:] }

// Big은 해시를 big.Int로 변환합니다.
func (h Hash) Big() *big.Int { return new(big.Int).SetBytes(h[:]) }

// Hex는 해시를 16진수 문자열로 변환합니다.
func (h Hash) Hex() string { return hexutil.Encode(h[:]) }

// TerminalString은 log.TerminalStringer를 구현하며, 로깅 중 콘솔 출력을 위한 문자열을 포맷합니다.
func (h Hash) TerminalString() string {
	return fmt.Sprintf("%x..%x", h[:3], h[29:])
}

// String은 fmt.Stringer를 구현하며, 로깅 중 파일에 전체 로깅을 할 때 사용됩니다.
func (h Hash) String() string {
	return h.Hex()
}

// Format은 fmt.Formatter를 구현하며, 해시는 %v, %s, %q, %x, %X, %d 포맷 동사를 지원합니다.
func (h Hash) Format(s fmt.State, c rune) {
	hexb := make([]byte, 2+len(h)*2)
	copy(hexb, "0x")
	hex.Encode(hexb[2:], h[:])

	switch c {
	case 'x', 'X':
		if !s.Flag('#') {
			hexb = hexb[2:]
		}
		if c == 'X' {
			hexb = bytes.ToUpper(hexb)
		}
		fallthrough
	case 'v', 's':
		s.Write(hexb)
	case 'q':
		q := []byte{'"'}
		s.Write(q)
		s.Write(hexb)
		s.Write(q)
	case 'd':
		fmt.Fprint(s, ([len(h)]byte)(h))
	default:
		fmt.Fprintf(s, "%%!%c(hash=%x)", c, h)
	}
}

// UnmarshalText는 16진수 형식의 텍스트 입력을 해시로 변환합니다.
func (h *Hash) UnmarshalText(input []byte) error {
	return hexutil.UnmarshalFixedText("Hash", input, h[:])
}

// UnmarshalJSON은 16진수 형식의 json 입력을 해시로 변환합니다.
func (h *Hash) UnmarshalJSON(input []byte) error {
	return hexutil.UnmarshalFixedJSON(hashT, input, h[:])
}

// MarshalText는 h의 16진수 표현을 반환합니다.
func (h Hash) MarshalText() ([]byte, error) {
	return hexutil.Bytes(h[:]).MarshalText()
}

// SetBytes는 바이트열 b를 해시로 설정합니다.
// 만약 b의 길이가 HashLength보다 크다면, b는 왼쪽에서부터 잘립니다.
func (h *Hash) SetBytes(b []byte) {
	if len(b) > len(h) {
		b = b[len(b)-HashLength:]
	}

	copy(h[HashLength-len(b):], b)
}

// Generate는 testing/quick 패키지의 Generator 인터페이스를 구현합니다.
// Generate는 임의의 해시를 생성하여 반환합니다.
func (h Hash) Generate(rand *rand.Rand, size int) reflect.Value {
	m := rand.Intn(len(h))
	for i := len(h) - 1; i > m; i-- {
		h[i] = byte(rand.Uint32())
	}
	return reflect.ValueOf(h)
}

// Scan은 database/sql 패키지의 Scanner 인터페이스를 구현합니다.
// Scan은 src를 해시로 변환합니다.
func (h *Hash) Scan(src interface{}) error {
	srcB, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("can't scan %T into Hash", src)
	}
	if len(srcB) != HashLength {
		return fmt.Errorf("can't scan []byte of len %d into Hash, want %d", len(srcB), HashLength)
	}
	copy(h[:], srcB)
	return nil
}

// Value는 database/sql/driver 패키지의 Valuer 인터페이스를 구현합니다.
// 해시를 driver.Value로 변환합니다.
func (h Hash) Value() (driver.Value, error) {
	return h[:], nil
}

// ImplementsGraphQLType는 Hash가 특정한 GraphQL 타입을 구현하는지 여부를 반환합니다.
func (Hash) ImplementsGraphQLType(name string) bool { return name == "Bytes32" }

// UnmarshalGraphQL은 제공된 GraphQL 쿼리 데이터를 해시로 변환합니다.
func (h *Hash) UnmarshalGraphQL(input interface{}) error {
	var err error
	switch input := input.(type) {
	case string:
		err = h.UnmarshalText([]byte(input))
	default:
		err = fmt.Errorf("unexpected type %T for Hash", input)
	}
	return err
}

// UnprefixedHash는 0x 접두사 없이 해시를 마샬링할 수 있습니다.
type UnprefixedHash Hash

// UnmarshalText는 16진수 형식의 텍스트 입력을 해시로 변환합니다. 0x 접두사는 선택사항입니다.
func (h *UnprefixedHash) UnmarshalText(input []byte) error {
	return hexutil.UnmarshalFixedUnprefixedText("UnprefixedHash", input, h[:])
}

// MarshalText는 해시를 16진수 문자열로 변환합니다.
func (h UnprefixedHash) MarshalText() ([]byte, error) {
	return []byte(hex.EncodeToString(h[:])), nil
}

/////////// Address

// Address는 이더리움 계정의 20바이트 주소를 나타냅니다.
type Address [AddressLength]byte

// BytesToAddress는 바이트열 b의 값을 주소로 설정합니다.
// 만약 b의 길이가 AddressLength보다 크다면, b는 왼쪽에서부터 잘립니다.
func BytesToAddress(b []byte) Address {
	var a Address
	a.SetBytes(b)
	return a
}

// BigToAddress는 big.Int의 바이트 표현을 주소로 설정합니다.
// 만약 b의 길이가 AddressLength보다 크다면, b는 왼쪽에서부터 잘립니다.
func BigToAddress(b *big.Int) Address { return BytesToAddress(b.Bytes()) }

// HexToAddress는 16진수 문자열 s의 바이트 표현을 주소로 설정합니다.
// 만약 s의 길이가 AddressLength보다 크다면, s는 왼쪽에서부터 잘립니다.
func HexToAddress(s string) Address { return BytesToAddress(FromHex(s)) }

// IsHexAddress는 문자열이 유효한 16진수 인코딩된 이더리움 주소인지 확인합니다.
func IsHexAddress(s string) bool {
	if has0xPrefix(s) { // 0x 접두사가 있는 경우 -> 0x 제거
		s = s[2:]
	}
	return len(s) == 2*AddressLength && isHex(s) // 문자열의 길이가 40이고, 16진수 문자열인지 확인
}

// Cmp는 두 주소를 비교합니다. (0: 같음, -1: a < other, +1: a > other)
func (a Address) Cmp(other Address) int {
	return bytes.Compare(a[:], other[:])
}

// Bytes는 주소의 바이트 표현을 반환합니다.
func (a Address) Bytes() []byte { return a[:] }

// Big는 주소를 big.Int로 변환합니다.
func (a Address) Big() *big.Int { return new(big.Int).SetBytes(a[:]) }

// Hex는 EIP55 호환성을 갖는 16진수 문자열 표현을 반환합니다.
func (a Address) Hex() string {
	return string(a.checksumHex())
}

// String은 EIP55 호환성을 갖는 16진수 문자열 표현을 반환합니다.
func (a Address) String() string {
	return a.Hex()
}

// checksumHex는 EIP55 호환성을 갖는 16진수 문자열 표현을 바이트열로 반환합니다.
func (a *Address) checksumHex() []byte {
	buf := a.hex() // 16진수 형식의 주소 (0x 접두사 포함, 모두 소문자)

	// compute checksum
	sha := sha3.NewLegacyKeccak256()
	sha.Write(buf[2:])              // 0x를 제외한 소문자 주소를 해시 함수의 입력으로 사용
	hash := sha.Sum(nil)            // keccak256 해시
	for i := 2; i < len(buf); i++ { // 접두사 0x를 제외하고 EIP55 호환성을 위해 16진수 문자열을 대문자로 변환
		hashByte := hash[(i-2)/2] // buf[i]의 대응하는 해시 바이트
		if i%2 == 0 {             // 짝수 인덱스의 경우
			hashByte = hashByte >> 4 // 상위 4비트 추출
		} else {
			hashByte &= 0xf // 하위 4비트 추출
		}
		if buf[i] > '9' && hashByte > 7 { // buf[i]가 숫자가 아니고, hashByte가 0x8 이상인 경우
			buf[i] -= 32 // 대문자로 변환
		}
	}
	return buf[:]
}

// hex는 0x 접두사를 포함한 16진수 문자열 표현의 바이트열을 반환합니다.
func (a Address) hex() []byte {
	var buf [len(a)*2 + 2]byte
	copy(buf[:2], "0x")
	hex.Encode(buf[2:], a[:])
	return buf[:]
}

// Format은 fmt.Formatter를 구현하며, 주소는 %v, %s, %q, %x, %X, %d 포맷 동사를 지원합니다.
func (a Address) Format(s fmt.State, c rune) {
	switch c {
	case 'v', 's':
		s.Write(a.checksumHex())
	case 'q':
		q := []byte{'"'}
		s.Write(q)
		s.Write(a.checksumHex())
		s.Write(q)
	case 'x', 'X':
		// %x는 체크섬을 비활성화합니다.
		hex := a.hex()
		if !s.Flag('#') {
			hex = hex[2:]
		}
		if c == 'X' {
			hex = bytes.ToUpper(hex)
		}
		s.Write(hex)
	case 'd':
		fmt.Fprint(s, ([len(a)]byte)(a))
	default:
		fmt.Fprintf(s, "%%!%c(address=%x)", c, a)
	}
}

// SetBytes는 바이트열 b의 값을 주소로 설정합니다.
// 만약 b의 길이가 AddressLength보다 크다면, b는 왼쪽에서부터 잘립니다.
func (a *Address) SetBytes(b []byte) {
	if len(b) > len(a) {
		b = b[len(b)-AddressLength:]
	}
	copy(a[AddressLength-len(b):], b)
}

// MarshalText는 주소의 16진수 문자열 표현을 반환합니다.
func (a Address) MarshalText() ([]byte, error) {
	return hexutil.Bytes(a[:]).MarshalText()
}

// UnmarshalText는 16진수 형식의 텍스트 입력을 해시로 변환합니다.
func (a *Address) UnmarshalText(input []byte) error {
	return hexutil.UnmarshalFixedText("Address", input, a[:])
}

// UnmarshalJSON은 16진수 형식의 json 입력을 해시로 변환합니다.
func (a *Address) UnmarshalJSON(input []byte) error {
	return hexutil.UnmarshalFixedJSON(addressT, input, a[:])
}

// Scan은 database/sql 패키지의 Scanner 인터페이스를 구현합니다.
// Scan은 src를 주소로 변환합니다.
func (a *Address) Scan(src interface{}) error {
	srcB, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("can't scan %T into Address", src)
	}
	if len(srcB) != AddressLength {
		return fmt.Errorf("can't scan []byte of len %d into Address, want %d", len(srcB), AddressLength)
	}
	copy(a[:], srcB)
	return nil
}

// Value는 database/sql/driver 패키지의 Valuer 인터페이스를 구현합니다.
// 주소를 driver.Value로 변환합니다.
func (a Address) Value() (driver.Value, error) {
	return a[:], nil
}

// ImplementsGraphQLType는 Hash가 지정된 GraphQL 타입을 구현하는지 여부를 반환합니다.
func (a Address) ImplementsGraphQLType(name string) bool { return name == "Address" }

// UnmarshalGraphQL은 제공된 GraphQL 쿼리 데이터를 주소로 변환합니다.
func (a *Address) UnmarshalGraphQL(input interface{}) error {
	var err error
	switch input := input.(type) {
	case string:
		err = a.UnmarshalText([]byte(input))
	default:
		err = fmt.Errorf("unexpected type %T for Address", input)
	}
	return err
}

// UnprefixedAddress는 0x 접두사 없이 주소를 마샬링할 수 있습니다.
type UnprefixedAddress Address

// UnmarshalText는 16진수 형식의 텍스트 입력을 해시로 변환합니다. 0x 접두사는 선택사항입니다.
func (a *UnprefixedAddress) UnmarshalText(input []byte) error {
	return hexutil.UnmarshalFixedUnprefixedText("UnprefixedAddress", input, a[:])
}

// MarshalText는 주소를 16진수 문자열로 변환합니다.
func (a UnprefixedAddress) MarshalText() ([]byte, error) {
	return []byte(hex.EncodeToString(a[:])), nil
}

// MixedcaseAddress는 원래의 문자열을 유지합니다. 이 문자열은 올바른 체크섬을 가질 수도 있고, 아닐 수도 있습니다.
type MixedcaseAddress struct {
	addr     Address
	original string
}

// NewMixedcaseAddress 생성자 (주로 테스트용)
func NewMixedcaseAddress(addr Address) MixedcaseAddress {
	return MixedcaseAddress{addr: addr, original: addr.Hex()}
}

// NewMixedcaseAddressFromString은 주로 단위 테스트를 위해 사용됩니다.
func NewMixedcaseAddressFromString(hexaddr string) (*MixedcaseAddress, error) {
	if !IsHexAddress(hexaddr) {
		return nil, errors.New("invalid address")
	}
	a := FromHex(hexaddr)
	return &MixedcaseAddress{addr: BytesToAddress(a), original: hexaddr}, nil
}

// UnmarshalJSON은 입력을 MixedcaseAddress로 변환합니다.
func (ma *MixedcaseAddress) UnmarshalJSON(input []byte) error {
	if err := hexutil.UnmarshalFixedJSON(addressT, input, ma.addr[:]); err != nil {
		return err
	}
	return json.Unmarshal(input, &ma.original)
}

// MarshalJSON은 원래의 값을 바이트열로 변환합니다.
func (ma MixedcaseAddress) MarshalJSON() ([]byte, error) {
	if strings.HasPrefix(ma.original, "0x") || strings.HasPrefix(ma.original, "0X") {
		return json.Marshal(fmt.Sprintf("0x%s", ma.original[2:]))
	}
	return json.Marshal(fmt.Sprintf("0x%s", ma.original))
}

// Address는 Address 타입의 주소를 반환합니다.
func (ma *MixedcaseAddress) Address() Address {
	return ma.addr
}

// String은 fmt.Stringer를 구현합니다.
func (ma *MixedcaseAddress) String() string {
	if ma.ValidChecksum() {
		return fmt.Sprintf("%s [chksum ok]", ma.original)
	}
	return fmt.Sprintf("%s [chksum INVALID]", ma.original)
}

// ValidChecksum은 주소가 올바른 체크섬을 가지고 있는지 여부를 반환합니다.
func (ma *MixedcaseAddress) ValidChecksum() bool {
	return ma.original == ma.addr.Hex()
}

// Original은 원래의 문자열을 반환합니다.
func (ma *MixedcaseAddress) Original() string {
	return ma.original
}

// AddressEIP55는 커스터마이징된 json marshaller를 가진 Address의 별칭 타입입니다.
type AddressEIP55 Address

// String은 EIP55 형식의 16진수 문자열 표현을 반환합니다.
func (addr AddressEIP55) String() string {
	return Address(addr).Hex()
}

// MarshalJSON은 EIP55 형식의 주소를 바이트열로 변환합니다.
func (addr AddressEIP55) MarshalJSON() ([]byte, error) {
	return json.Marshal(addr.String())
}

// uint64 형식 정수의 별칭 타입입니다.
type Decimal uint64

// isString은 입력이 "..." 형식의 문자열인지 여부를 반환합니다.
func isString(input []byte) bool {
	return len(input) >= 2 && input[0] == '"' && input[len(input)-1] == '"'
}

// UnmarshalJSON은 16진수 형식의 json 입력을 Decimal로 변환합니다.
func (d *Decimal) UnmarshalJSON(input []byte) error {
	if !isString(input) {
		return &json.UnmarshalTypeError{Value: "non-string", Type: reflect.TypeOf(uint64(0))}
	}
	if i, err := strconv.ParseInt(string(input[1:len(input)-1]), 10, 64); err == nil {
		*d = Decimal(i)
		return nil
	} else {
		return err
	}
}
