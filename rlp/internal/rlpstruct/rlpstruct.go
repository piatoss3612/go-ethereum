// Copyright 2022 The go-ethereum Authors
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

// rlpstruct 패키지는 구조체 처리를 위한 RLP 인코딩/디코딩을 구현합니다.
//
// 특히, 이 패키지는 필드 필터링, 구조체 태그 그리고 nil 값 결정에 관한 모든 규칙을 처리합니다.
package rlpstruct

import (
	"fmt"
	"reflect"
	"strings"
)

// Field는 구조체 필드를 나타냅니다.
type Field struct {
	Name     string
	Index    int
	Exported bool
	Type     Type
	Tag      string
}

// Type는 Go 타입의 속성을 나타냅니다.
type Type struct {
	Name      string
	Kind      reflect.Kind
	IsEncoder bool  // 타입이 rlp.Encoder를 구현하는지 여부
	IsDecoder bool  // 타입이 rlp.Decoder를 구현하는지 여부
	Elem      *Type // Ptr, Slice, Array의 Kind 값에 대해서는 nil이 아니어야 합니다.
}

// DefaultNilValue는 t의 nil 포인터가 빈 문자열 또는 빈 리스트로 인코딩/디코딩되는지 여부를 결정합니다.
func (t Type) DefaultNilValue() NilKind {
	k := t.Kind
	if isUint(k) || k == reflect.String || k == reflect.Bool || isByteArray(t) {
		return NilKindString
	}
	return NilKindList
}

// NilKind는 nil 포인터 대신 인코딩되는 RLP 값입니다.
type NilKind uint8

const (
	NilKindString NilKind = 0x80 // 빈 문자열
	NilKindList   NilKind = 0xC0 // 빈 리스트
)

// Tags는 구조체 태그를 나타냅니다.
type Tags struct {
	// rlp:"nil" controls whether empty input results in a nil pointer.
	// nilKind is the kind of empty value allowed for the field.

	// rlp:"nil"은 빈 입력이 nil 포인터로 결과를 내는지 여부를 결정합니다.
	// nilKind는 필드에 허용되는 빈 값의 종류입니다.
	NilKind NilKind
	NilOK   bool

	// rlp:"optional"은 입력 리스트에서 필드가 누락되는 것을 허용합니다.
	// 이것이 설정되면, 이후의 모든 필드도 선택적이어야 합니다.
	Optional bool

	// rlp:"tail" controls whether this field swallows additional list elements. It can
	// only be set for the last field, which must be of slice type.

	// rlp:"tail"은 이 필드가 추가 리스트 요소를 허용하는지 여부를 결정합니다. 이것은
	// 마지막 필드에만 설정할 수 있으며, 슬라이스 타입이어야 합니다.
	Tail bool

	// rlp:"-"은 필드를 무시합니다.
	Ignored bool
}

// TagError는 잘못된 구조체 태그에 대해 발생합니다.
type TagError struct {
	StructType string

	// These are set by this package.
	Field string
	Tag   string
	Err   string
}

func (e TagError) Error() string {
	field := "field " + e.Field
	if e.StructType != "" {
		field = e.StructType + "." + e.Field
	}
	return fmt.Sprintf("rlp: invalid struct tag %q for %s (%s)", e.Tag, field, e.Err)
}

// ProcessFields는 주어진 구조체 필드를 필터링하여 인코딩/디코딩에 고려해야 하는
// 필드만 반환합니다.
func ProcessFields(allFields []Field) ([]Field, []Tags, error) {
	lastPublic := lastPublicField(allFields)

	// 모든 공개된 필드와 태그를 수집합니다. (private 필드는 무시됩니다.)
	var fields []Field
	var tags []Tags
	for _, field := range allFields {
		if !field.Exported {
			continue
		}
		ts, err := parseTag(field, lastPublic)
		if err != nil {
			return nil, nil, err
		}
		if ts.Ignored {
			continue
		}
		fields = append(fields, field)
		tags = append(tags, ts)
	}

	// 선택적 필드 일관성을 검증합니다. 선택적 필드가 하나라도 존재하면,
	// 그 이후의 모든 필드도 선택적이어야 합니다. 참고: 선택적 + tail은
	// 지원됩니다.
	var anyOptional bool
	var firstOptionalName string
	for i, ts := range tags {
		name := fields[i].Name
		if ts.Optional || ts.Tail {
			if !anyOptional {
				firstOptionalName = name
			}
			anyOptional = true
		} else {
			if anyOptional {
				msg := fmt.Sprintf("must be optional because preceding field %q is optional", firstOptionalName)
				return nil, nil, TagError{Field: name, Err: msg}
			}
		}
	}
	return fields, tags, nil
}

func parseTag(field Field, lastPublic int) (Tags, error) {
	name := field.Name
	tag := reflect.StructTag(field.Tag)
	var ts Tags
	for _, t := range strings.Split(tag.Get("rlp"), ",") {
		switch t = strings.TrimSpace(t); t {
		case "":
			// empty tag is allowed for some reason
		case "-":
			ts.Ignored = true
		case "nil", "nilString", "nilList":
			ts.NilOK = true
			if field.Type.Kind != reflect.Ptr {
				return ts, TagError{Field: name, Tag: t, Err: "field is not a pointer"}
			}
			switch t {
			case "nil":
				ts.NilKind = field.Type.Elem.DefaultNilValue()
			case "nilString":
				ts.NilKind = NilKindString
			case "nilList":
				ts.NilKind = NilKindList
			}
		case "optional":
			ts.Optional = true
			if ts.Tail {
				return ts, TagError{Field: name, Tag: t, Err: `also has "tail" tag`}
			}
		case "tail":
			ts.Tail = true
			if field.Index != lastPublic {
				return ts, TagError{Field: name, Tag: t, Err: "must be on last field"}
			}
			if ts.Optional {
				return ts, TagError{Field: name, Tag: t, Err: `also has "optional" tag`}
			}
			if field.Type.Kind != reflect.Slice {
				return ts, TagError{Field: name, Tag: t, Err: "field type is not slice"}
			}
		default:
			return ts, TagError{Field: name, Tag: t, Err: "unknown tag"}
		}
	}
	return ts, nil
}

func lastPublicField(fields []Field) int {
	last := 0
	for _, f := range fields {
		if f.Exported {
			last = f.Index
		}
	}
	return last
}

func isUint(k reflect.Kind) bool {
	return k >= reflect.Uint && k <= reflect.Uintptr
}

func isByte(typ Type) bool {
	return typ.Kind == reflect.Uint8 && !typ.IsEncoder
}

func isByteArray(typ Type) bool {
	return (typ.Kind == reflect.Slice || typ.Kind == reflect.Array) && isByte(*typ.Elem)
}
