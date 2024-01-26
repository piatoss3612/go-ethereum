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

/*
rlp 패키지는 RLP 직렬화 형식을 구현합니다.

RLP (Recursive Linear Prefix)의 목적은 임의로 중첩된 이진 데이터 배열을 인코딩하는 것입니다.
그리고 RLP는 Ethereum에서 객체를 직렬화하는 데 사용되는 주요 인코딩 방법입니다.
RLP의 주목적은 구조체를 인코딩하는 것이며, (문자열, 정수, 부동 소수점 등) 원자적인 데이터 타입의
인코딩은 상위 프로토콜에 맡겨집니다. Ethereum에서 정수는 선행하는 0이 없는 빅 엔디언 바이너리 형식으로
표현되어야 합니다. (따라서 정수 값 0은 빈 문자열과 동일합니다.)

RLP 값은 타입 태그로 구분됩니다. 타입 태그는 입력 스트림에서 값 앞에 오며, 뒤따르는 바이트의 크기와
종류를 정의합니다.

# 인코딩 규칙

rlp 패키지는 리플렉션을 사용하며, 값이 가지는 Go 타입에 따라 인코딩합니다.

만약 해당 타입이 Encoder 인터페이스를 구현하고 있다면, EncodeRLP를 호출합니다.
nil 포인터 값에 대해서는 EncodeRLP를 호출하지 않습니다.

포인터를 인코딩하면, 포인터가 가리키는 값이 인코딩 됩니다. 구조체, 슬라이스, 배열 타입의 nil 포인터는
항상 빈 RLP 리스트로 인코딩됩니다. (단, 슬라이스나 배열의 원소 타입이 byte인 경우는 제외합니다.)
다른 타입의 nil 포인터는 빈 문자열로 인코딩됩니다.

구조체 값은 모든 공개 필드의 인코딩된 RLP 리스트로 인코딩됩니다. 재귀적인 구조체 타입도 지원됩니다.

슬라이스나 배열을 인코딩하면, 원소들은 RLP 리스트로 인코딩됩니다. (단, 슬라이스나 배열의 원소 타입이
uint8이나 byte인 경우는 RLP 문자열로 인코딩됩니다.)

Go 문자열은 RLP 문자열로 인코딩됩니다.

부호가 없는 정수 값은 RLP 문자열로 인코딩됩니다. 0은 항상 빈 RLP 문자열로 인코딩됩니다.
big.Int 값은 정수로 취급됩니다. 부호가 있는 정수 (int, int8, int16, ...)는 지원되지 않으며,
인코딩할 때 에러가 발생합니다.

불리언 값은 부호가 없는 정수 0 (false)과 1 (true)로 인코딩됩니다.

인터페이스 값은 인터페이스가 가리키는 값에 따라 인코딩됩니다.

부동 소수점, 맵, 채널, 함수는 지원되지 않습니다.

# 디코딩 규칙

디코딩은 다음과 같은 타입 종속적인 규칙을 사용합니다.

만약 해당 타입이 Decoder 인터페이스를 구현하고 있다면, DecodeRLP를 호출합니다.

포인터에 디코딩할 때, 값은 포인터가 가리키는 타입으로 디코딩됩니다. 만약 포인터가 nil이라면,
포인터가 가리키는 타입의 새로운 값이 할당됩니다. 만약 포인터가 nil이 아니라면, 기존의 값이 재사용됩니다.
rlp 패키지는 포인터 타입의 구조체 필드를 nil로 남겨두지 않습니다. (단, "nil" 태그가 있는 경우는 제외합니다.)

구조체에 디코딩할 때, 입력은 RLP 리스트여야 합니다. 디코딩된 리스트의 원소들은 구조체의 공개 필드에
정의된 순서대로 할당됩니다. 입력 리스트는 각 필드에 대해 원소를 하나씩 포함해야 합니다. 만약
입력 리스트의 원소 개수가 구조체의 필드 개수보다 적거나 많다면, 에러가 발생합니다.

슬라이스에 디코딩할 때, 입력은 RLP 리스트여야 합니다. 결과 슬라이스는 입력 리스트의 원소들을
순서대로 포함합니다. 만약 슬라이스의 원소 타입이 byte라면, 입력은 RLP 문자열이어야 합니다.
배열 타입은 슬라이스와 비슷하게 동작하지만, 입력 리스트의 원소 개수가 배열의 길이와 일치해야 합니다.

Go 문자열로 디코딩할 때, 입력은 RLP 문자열이어야 합니다. 입력 바이트는 그대로 사용되며,
항상 유효한 UTF-8 문자열일 필요는 없습니다.

부호가 없는 정수 타입으로 디코딩할 때, 입력은 RLP 문자열이어야 합니다. 바이트는 정수의 빅 엔디언
바이너리 표현으로 해석됩니다. 만약 RLP 문자열이 타입의 비트 크기보다 크다면, 에러가 발생합니다.
*big.Int 타입도 지원됩니다. 큰 정수에는 크기 제한이 없습니다.

불리언으로 디코딩할 때, 입력은 부호가 없는 정수여야 합니다. 값이 0이면 false, 1이면 true로 디코딩됩니다.

인터페이스로 디코딩할 때, 인터페이스가 가리키는 값은 다음 타입 중 하나로 저장됩니다.

	[]interface{}, for RLP lists
	[]byte, for RLP strings

비어있지 않은 인터페이스 타입은 디코딩할 때 지원되지 않습니다.
부호가 있는 정수, 부동 소수점, 맵, 채널, 함수는 디코딩할 때 지원되지 않습니다.

# 구조체 태그

다른 인코딩 패키지와 마찬가지로, "-" 태그는 필드를 무시합니다.

	type StructWithIgnoredField struct{
	    Ignored uint `rlp:"-"`
	    Field   uint
	}

Go 구조체 값은 RLP 리스트로 인코딩/디코딩됩니다. 필드를 리스트 원소로 매핑하는 두 가지 방법이 있습니다.
"tail" 태그는 마지막 공개 필드에서만 사용할 수 있으며, 리스트의 나머지 원소들을 해당 필드에 슬라이스로 묶어서 할당합니다.

	type StructWithTail struct{
	    Field   uint
	    Tail    []string `rlp:"tail"`
	}

"optional" 태그는 필드가 해당 타입의 기본값이라면 생략할 수 있다는 것을 나타냅니다. 이 태그를
사용하면, 모든 후행 공개 필드도 "optional" 태그를 가져야 합니다.

선택적 필드를 가진 구조체를 인코딩할 때, 출력 RLP 리스트는 마지막으로 0이 아닌 선택적 필드까지의 모든 값을 포함합니다.

구조체로 디코딩할 때, 선택적 필드는 입력 리스트의 끝에서 생략할 수 있습니다. 아래 예제에서는,
원소가 1개, 2개, 3개인 입력 리스트가 모두 허용됩니다.

	type StructWithOptionalFields struct{
	     Required  uint
	     Optional1 uint `rlp:"optional"`
	     Optional2 uint `rlp:"optional"`
	}

"nil", "nilList" 그리고 "nilString" 태그는 포인터 타입의 필드에만 적용되며, 필드 타입의
디코딩 규칙을 변경합니다. "nil" 태그가 없는 일반적인 포인터 필드는, 입력 값의 길이가 정확히
필요한 길이와 일치해야 하며, 디코더는 nil 값을 생성하지 않습니다. "nil" 태그가 설정되면,
크기가 0인 입력 값은 nil 포인터로 디코딩됩니다. 이는 재귀적인 타입에 특히 유용합니다.

	type StructWithNilField struct {
	    Field *[3]byte `rlp:"nil"`
	}

위 예제에서, Field는 두 가지 입력 크기를 허용합니다. 입력 0xC180 (빈 문자열을 포함하는 리스트)는
디코딩 후 Field가 nil로 설정됩니다. 입력 0xC483000000 (3바이트 문자열을 포함하는 리스트)는
Field가 nil이 아닌 배열 포인터로 설정됩니다.

RLP는 두 종류의 비어있는 값 (빈 리스트와 빈 문자열)을 지원합니다. "nil" 태그를 사용할 때,
타입에 허용되는 비어있는 값의 종류는 자동으로 선택됩니다. 부호가 없는 정수, 문자열,
불리언, 바이트 배열/슬라이스인 Go 타입을 가리키는 포인터 필드는 빈 RLP 문자열을 가질 것입니다.
다른 포인터 타입의 필드는 빈 RLP 리스트로 인코딩/디코딩됩니다.

null 값을 명시적으로 지정하기 위해 "nilList"와 "nilString" 구조체 태그를 사용할 수 있습니다.
이 태그를 사용하면, Go nil 포인터 값은 태그가 정의한 빈 RLP 값으로 인코딩/디코딩됩니다.
*/
package rlp
