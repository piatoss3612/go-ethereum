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

package hexutil

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"strconv"

	"github.com/holiman/uint256"
)

var (
	bytesT  = reflect.TypeOf(Bytes(nil))
	bigT    = reflect.TypeOf((*Big)(nil))
	uintT   = reflect.TypeOf(Uint(0))
	uint64T = reflect.TypeOf(Uint64(0))
	u256T   = reflect.TypeOf((*uint256.Int)(nil))
)

// Bytes는 0x 접두사가 있는 JSON 문자열로 마샬링/언마샬링됩니다.
// 빈 슬라이스는 "0x"로 마샬링됩니다.
type Bytes []byte

// MarshalText는 encoding.TextMarshaler를 구현합니다.
func (b Bytes) MarshalText() ([]byte, error) {
	result := make([]byte, len(b)*2+2)
	copy(result, `0x`)
	hex.Encode(result[2:], b)
	return result, nil
}

// UnmarshalJSON은 json.Unmarshaler를 구현합니다.
func (b *Bytes) UnmarshalJSON(input []byte) error {
	if !isString(input) {
		return errNonString(bytesT)
	}
	return wrapTypeError(b.UnmarshalText(input[1:len(input)-1]), bytesT)
}

// UnmarshalText는 encoding.TextUnmarshaler를 구현합니다.
func (b *Bytes) UnmarshalText(input []byte) error {
	raw, err := checkText(input, true)
	if err != nil {
		return err
	}
	dec := make([]byte, len(raw)/2)
	if _, err = hex.Decode(dec, raw); err != nil {
		err = mapError(err)
	} else {
		*b = dec
	}
	return err
}

// String은 b의 16진수 인코딩을 반환합니다.
func (b Bytes) String() string {
	return Encode(b)
}

// ImplementsGraphQLType은 Bytes가 특정한 GraphQL 타입을 구현하는지 여부를 반환합니다.
func (b Bytes) ImplementsGraphQLType(name string) bool { return name == "Bytes" }

// UnmarshalGraphQL은 제공된 GraphQL 쿼리 데이터를 Bytes로 변환합니다.
func (b *Bytes) UnmarshalGraphQL(input interface{}) error {
	var err error
	switch input := input.(type) {
	case string:
		data, err := Decode(input)
		if err != nil {
			return err
		}
		*b = data
	default:
		err = fmt.Errorf("unexpected type %T for Bytes", input)
	}
	return err
}

// UnmarshalFixedJSON은 0x 접두사가 있는 JSON 문자열을 디코딩합니다. out의 길이는 필요한 입력 길이를 결정합니다.
// 이 함수는 고정 크기 타입의 UnmarshalJSON 메서드를 구현하는 데 주로 사용됩니다.
func UnmarshalFixedJSON(typ reflect.Type, input, out []byte) error {
	if !isString(input) {
		return errNonString(typ)
	}
	return wrapTypeError(UnmarshalFixedText(typ.String(), input[1:len(input)-1], out), typ)
}

// UnmarshalFixedText는 0x 접두사가 있는 문자열을 디코딩합니다. out의 길이는 필요한 입력 길이를 결정합니다.
// 이 함수는 고정 크기 타입의 UnmarshalText 메서드를 구현하는 데 주로 사용됩니다.
func UnmarshalFixedText(typname string, input, out []byte) error {
	raw, err := checkText(input, true)
	if err != nil {
		return err
	}
	if len(raw)/2 != len(out) {
		return fmt.Errorf("hex string has length %d, want %d for %s", len(raw), len(out)*2, typname)
	}
	// out을 수정하기 전에 구문을 사전 확인합니다.
	for _, b := range raw {
		if decodeNibble(b) == badNibble {
			return ErrSyntax
		}
	}
	hex.Decode(out, raw)
	return nil
}

// UnmarshalFixedUnprefixedText는 0x 접두사가 있거나 없는 문자열을 디코딩합니다. out의 길이는 필요한 입력 길이를 결정합니다.
// 이 함수는 고정 크기 타입의 UnmarshalText 메서드를 구현하는 데 주로 사용됩니다.
func UnmarshalFixedUnprefixedText(typname string, input, out []byte) error {
	raw, err := checkText(input, false)
	if err != nil {
		return err
	}
	if len(raw)/2 != len(out) {
		return fmt.Errorf("hex string has length %d, want %d for %s", len(raw), len(out)*2, typname)
	}
	// Pre-verify syntax before modifying out.
	for _, b := range raw {
		if decodeNibble(b) == badNibble {
			return ErrSyntax
		}
	}
	hex.Decode(out, raw)
	return nil
}

// Big은 0x 접두사가 있는 JSON 문자열로 마샬링/언마샬링됩니다.
// 0은 "0x0"으로 마샬링됩니다.
//
// 음수는 현재 지원되지 않습니다. 음수를 마샬링하려고 하면 오류가 발생합니다.
// 256비트보다 큰 값은 Unmarshal에서 거부되지만 오류 없이 마샬링됩니다.
type Big big.Int

// MarshalText는 encoding.TextMarshaler를 구현합니다.
func (b Big) MarshalText() ([]byte, error) {
	return []byte(EncodeBig((*big.Int)(&b))), nil
}

// UnmarshalJSON은 json.Unmarshaler를 구현합니다.
func (b *Big) UnmarshalJSON(input []byte) error {
	if !isString(input) {
		return errNonString(bigT)
	}
	return wrapTypeError(b.UnmarshalText(input[1:len(input)-1]), bigT)
}

// UnmarshalText는 encoding.TextUnmarshaler를 구현합니다.
func (b *Big) UnmarshalText(input []byte) error {
	raw, err := checkNumberText(input)
	if err != nil {
		return err
	}
	if len(raw) > 64 {
		return ErrBig256Range
	}
	words := make([]big.Word, len(raw)/bigWordNibbles+1)
	end := len(raw)
	for i := range words {
		start := end - bigWordNibbles
		if start < 0 {
			start = 0
		}
		for ri := start; ri < end; ri++ {
			nib := decodeNibble(raw[ri])
			if nib == badNibble {
				return ErrSyntax
			}
			words[i] *= 16
			words[i] += big.Word(nib)
		}
		end = start
	}
	var dec big.Int
	dec.SetBits(words)
	*b = (Big)(dec)
	return nil
}

// ToInt는 b를 big.Int로 변환합니다.
func (b *Big) ToInt() *big.Int {
	return (*big.Int)(b)
}

// String은 b의 16진수 인코딩을 반환합니다.
func (b *Big) String() string {
	return EncodeBig(b.ToInt())
}

// ImplementsGraphQLType은 Big이 특정한 GraphQL 타입을 구현하는지 여부를 반환합니다.
func (b Big) ImplementsGraphQLType(name string) bool { return name == "BigInt" }

// UnmarshalGraphQL은 제공된 GraphQL 쿼리 데이터를 Big으로 변환합니다.
func (b *Big) UnmarshalGraphQL(input interface{}) error {
	var err error
	switch input := input.(type) {
	case string:
		return b.UnmarshalText([]byte(input))
	case int32:
		var num big.Int
		num.SetInt64(int64(input))
		*b = Big(num)
	default:
		err = fmt.Errorf("unexpected type %T for BigInt", input)
	}
	return err
}

// U256은 0x 접두사가 있는 JSON 문자열로 마샬링/언마샬링됩니다.
// 0은 "0x0"으로 마샬링됩니다.
type U256 uint256.Int

// U256은 0x 접두사가 있는 JSON 문자열로 마샬링/언마샬링됩니다.
func (b U256) MarshalText() ([]byte, error) {
	u256 := (*uint256.Int)(&b)
	return []byte(u256.Hex()), nil
}

// UnmarshalJSON은 json.Unmarshaler를 구현합니다.
func (b *U256) UnmarshalJSON(input []byte) error {
	// The uint256.Int.UnmarshalJSON method accepts "dec", "0xhex"; we must be
	// more strict, hence we check string and invoke SetFromHex directly.

	// uint256.Int.UnmarshalJSON 메서드는 "dec", "0xhex"를 허용합니다.
	// 더 엄격한 방식으로 입력을 확인해야 하므로, 입력이 문자열인지 확인합니다.
	if !isString(input) {
		return errNonString(u256T)
	}
	// uint256.Int는 빈 문자열("")을 '0'으로 허용하지 않으므로, 입력이 빈 문자열인지 확인합니다.
	// 길이가 2인 경우는 "0x"이므로, 0으로 설정합니다.
	if len(input) == 2 {
		(*uint256.Int)(b).Clear()
		return nil
	}
	err := (*uint256.Int)(b).SetFromHex(string(input[1 : len(input)-1]))
	if err != nil {
		return &json.UnmarshalTypeError{Value: err.Error(), Type: u256T}
	}
	return nil
}

// UnmarshalText는 encoding.TextUnmarshaler를 구현합니다.
func (b *U256) UnmarshalText(input []byte) error {
	// uint256.Int.UnmarshalText 메서드는 "dec", "0xhex"를 허용합니다.
	// 더 엄격한 방식으로 입력을 확인하고, SetFromHex를 직접 호출합니다.
	return (*uint256.Int)(b).SetFromHex(string(input))
}

// String은 b의 16진수 인코딩을 반환합니다.
func (b *U256) String() string {
	return (*uint256.Int)(b).Hex()
}

// Uint64는 0x 접두사가 있는 JSON 문자열로 마샬링/언마샬링됩니다.
// 0은 "0x0"으로 마샬링됩니다.
type Uint64 uint64

// MarshalText는 encoding.TextMarshaler를 구현합니다.
func (b Uint64) MarshalText() ([]byte, error) {
	buf := make([]byte, 2, 10)
	copy(buf, `0x`)
	buf = strconv.AppendUint(buf, uint64(b), 16)
	return buf, nil
}

// UnmarshalJSON은 json.Unmarshaler를 구현합니다.
func (b *Uint64) UnmarshalJSON(input []byte) error {
	if !isString(input) {
		return errNonString(uint64T)
	}
	return wrapTypeError(b.UnmarshalText(input[1:len(input)-1]), uint64T)
}

// UnmarshalText는 encoding.TextUnmarshaler를 구현합니다.
func (b *Uint64) UnmarshalText(input []byte) error {
	raw, err := checkNumberText(input)
	if err != nil {
		return err
	}
	if len(raw) > 16 {
		return ErrUint64Range
	}
	var dec uint64
	for _, byte := range raw {
		nib := decodeNibble(byte)
		if nib == badNibble {
			return ErrSyntax
		}
		dec *= 16
		dec += nib
	}
	*b = Uint64(dec)
	return nil
}

// String은 b의 16진수 인코딩을 반환합니다.
func (b Uint64) String() string {
	return EncodeUint64(uint64(b))
}

// ImplementsGraphQLType은 Uint64가 제공된 GraphQL 타입을 구현하는지 여부를 반환합니다.
func (b Uint64) ImplementsGraphQLType(name string) bool { return name == "Long" }

// UnmarshalGraphQL은 제공된 GraphQL 쿼리 데이터를 Uint64로 변환합니다.
func (b *Uint64) UnmarshalGraphQL(input interface{}) error {
	var err error
	switch input := input.(type) {
	case string:
		return b.UnmarshalText([]byte(input))
	case int32:
		*b = Uint64(input)
	default:
		err = fmt.Errorf("unexpected type %T for Long", input)
	}
	return err
}

// Uint는 0x 접두사가 있는 JSON 문자열로 마샬링/언마샬링됩니다.
// 0은 "0x0"으로 마샬링됩니다.
type Uint uint

// MarshalText는 encoding.TextMarshaler를 구현합니다.
func (b Uint) MarshalText() ([]byte, error) {
	return Uint64(b).MarshalText()
}

// UnmarshalJSON은 json.Unmarshaler를 구현합니다.
func (b *Uint) UnmarshalJSON(input []byte) error {
	if !isString(input) {
		return errNonString(uintT)
	}
	return wrapTypeError(b.UnmarshalText(input[1:len(input)-1]), uintT)
}

// UnmarshalText는 encoding.TextUnmarshaler를 구현합니다.
func (b *Uint) UnmarshalText(input []byte) error {
	var u64 Uint64
	err := u64.UnmarshalText(input)
	if u64 > Uint64(^uint(0)) || err == ErrUint64Range {
		return ErrUintRange
	} else if err != nil {
		return err
	}
	*b = Uint(u64)
	return nil
}

// String은 b의 16진수 인코딩을 반환합니다.
func (b Uint) String() string {
	return EncodeUint64(uint64(b))
}

// isString은 문자열이 따옴표로 묶인 문자열인지 여부를 반환합니다.
func isString(input []byte) bool {
	return len(input) >= 2 && input[0] == '"' && input[len(input)-1] == '"'
}

// bytesHave0xPrefix는 입력이 0x 접두사를 가지고 있는지 여부를 반환합니다.
func bytesHave0xPrefix(input []byte) bool {
	return len(input) >= 2 && input[0] == '0' && (input[1] == 'x' || input[1] == 'X')
}

// checkText는 0x 접두사를 제거하고 입력을 반환합니다. wantPrefix가 true이면 0x 접두사가 없으면 오류가 발생합니다.
func checkText(input []byte, wantPrefix bool) ([]byte, error) {
	if len(input) == 0 {
		return nil, nil // empty strings are allowed
	}
	if bytesHave0xPrefix(input) {
		input = input[2:]
	} else if wantPrefix {
		return nil, ErrMissingPrefix
	}
	if len(input)%2 != 0 {
		return nil, ErrOddLength
	}
	return input, nil
}

// checkNumberText는 0x 접두사를 제거하고 입력을 반환합니다.
func checkNumberText(input []byte) (raw []byte, err error) {
	if len(input) == 0 {
		return nil, nil // 빈 문자열은 빈 슬라이스로 반환합니다.
	}
	if !bytesHave0xPrefix(input) {
		return nil, ErrMissingPrefix
	}
	input = input[2:]
	if len(input) == 0 {
		return nil, ErrEmptyNumber
	}
	if len(input) > 1 && input[0] == '0' {
		return nil, ErrLeadingZero
	}
	return input, nil
}

// wrapTypeError는 언마샬링 오류를 json.UnmarshalTypeError로 래핑합니다.
func wrapTypeError(err error, typ reflect.Type) error {
	if _, ok := err.(*decError); ok {
		return &json.UnmarshalTypeError{Value: err.Error(), Type: typ}
	}
	return err
}

// errNonString는 문자열이 아닌 값을 언마샬링하려고 할 때 발생하는 오류입니다.
func errNonString(typ reflect.Type) error {
	return &json.UnmarshalTypeError{Value: "non-string", Type: typ}
}
