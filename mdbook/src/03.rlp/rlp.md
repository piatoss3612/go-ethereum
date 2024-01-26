# rlp 패키지

- RLP(Recursive Length Prefix)는 이더리움에서 사용하는 데이터 인코딩 방식입니다.

## doc.go

- go doc 문서 생성을 위한 내용을 정리해놓은 파일입니다.

### RLP 인코딩 규칙

- 리플렉션(reflection)을 사용하여 값이 가지는 Go 타입에 따라 인코딩 규칙을 적용합니다.
- `Encoder` 인터페이스를 구현하는 타입은 `EncodeRLP` 메서드 호출을 통해 인코딩 규칙을 적용합니다.

| 데이터 타입 | RLP 인코딩 규칙 |
| --- | --- |
| 포인터 | 포인터가 가리키는 값에 대해 인코딩 규칙을 적용합니다. 구조체, 슬라이스, 배열의 nil 포인터는 항상 빈 RLP 리스트로 인코딩됩니다. (단, 슬라이스나 배열의 원소 타입이 byte인 경우는 제외) 다른 타입의 nil 포인터는 빈 RLP 문자열로 인코딩됩니다. |
| 구조체 | 구조체의 모든 공개된(public) 필드를 RLP 리스트로 인코딩합니다. 재귀적인 구조체도 지원합니다. |
| 슬라이스, 배열 | 슬라이스나 배열의 모든 원소를 RLP 리스트로 인코딩합니다. 단, 원소 타입이 uint8 또는 byte인 경우 RLP 문자열로 인코딩합니다. |
| Go 문자열 | RLP 문자열로 인코딩합니다. |
| 부호가 없는 정수 | RLP 문자열로 인코딩합니다. 0은 빈 RLP 문자열로 인코딩합니다. |
| 부호가 있는 정수 | 지원하지 않으며 인코딩 시 에러가 발생합니다. |
| big.Int | 정수와 동일하게 취급합니다. |
| 불리언 | 부호가 없는 정수 0 또는 1로 인코딩합니다. |
| 인터페이스 | 인터페이스가 가리키는 값에 대해 인코딩 규칙을 적용합니다. |
| 부동 소수점, 맵, 채널, 함수 | 지원하지 않습니다. |

### RLP 디코딩 규칙

- `Decoder` 인터페이스를 구현하는 타입은 `DecodeRLP` 메서드 호출을 통해 디코딩 규칙을 적용합니다.

| 대상 타입 | 입력 값 | 디코딩 규칙 |
| --- | --- | --- |
| 포인터 | 포인터가 가리키는 타입에 해당하는 RLP 형식 |포인터가 가리키는 타입에 따라 디코딩 규칙을 적용합니다. 포인터가 nil인 경우, 포인터가 가리키는 타입의 새로운 값을 생성하여 할당합니다. nil이 아닌 경우, 기존의 값을 재사용합니다. 포인터 타입의 구조체 필드는 nil을 가지지 않습니다. (단 nil 태그가 있는 경우는 예외) |
| 구조체 | RLP 리스트 | 디코딩된 RLP 리스트의 원소를 구조체 공개 필드에 할당합니다. 만약 입력 리스트의 원소 개수가 구조체 필드 개수보다 적거나 많은 경우, 에러가 발생합니다. |
| 슬라이스 | RLP 리스트 (원소 타입이 byte인 경우 RLP 문자열) | 디코딩된 RLP 리스트의 원소를 순서대로 슬라이스에 추가합니다. |
| 배열 | RLP 리스트 (원소 타입이 byte인 경우 RLP 문자열) | 디코딩된 RLP 리스트의 원소를 순서대로 배열에 추가합니다. 슬라이스와 달리 배열의 길이는 고정되어 있으므로, 입력 리스트의 원소 개수는 배열의 길이와 일치해야 합니다. |
| Go 문자열 | RLP 문자열 | RLP 문자열을 Go 문자열로 디코딩합니다. |
| 부호가 없는 정수 | RLP 문자열 | 문자열의 바이트 표현은 빅 엔디언(big endian)으로 해석하며 만약 RLP 문자열이 타입이 가지는 비트 크기보다 큰 경우, 에러가 발생합니다. |
| big.Int | RLP 문자열 | 부호가 없는 정수와 동일하게 취급합니다. 다만 길이의 제한이 없습니다. |
| 불리언 | 부호가 없는 정수 0 또는 1 | 정수 0은 false로, 정수 1은 true로 디코딩합니다. |
| 인터페이스 | 인터페이스가 가리키는 타입에 해당하는 RLP 형식 | `[]interface{}`인 경우는 RLP 리스트, `[]byte`인 경우는 RLP 문자열을 사용하여 디코딩합니다. 그 외의 경우는 지원하지 않습니다. |

### 구조체 필드 태그

| 태그 | 설명 |
| --- | --- |
| `rlp:"-"` | 해당 필드를 무시합니다. |
| `rlp:"tail"` | 해당 필드가 마지막 공개 필드임을 나타냅니다. 리스트의 나머지 원소를 해당 필드에 슬라이스로 묶어서 할당합니다. |
| `rlp:"optional"` | 해당 필드가 타입의 기본값을 가지는 경우, RLP 리스트에 추가하지 않습니다. optional 태그를 사용할 경우, 후행 필드에 대해서도 optional 태그를 사용해야 합니다. |
| `rlp:"nil"`, `rlp:"nilList"`, `rlp:"nilString"` | 크기가 0인 RLP 문자열 또는 리스트를 nil로 디코딩합니다. 그렇지 않은 경우는 타입에 따라 디코딩 규칙을 적용합니다. |

---

## raw.go

- `RawValue` 타입은 RLP 인코딩된 값을 저장하는 타입입니다.
- 크기를 계산하는 `**Size` 함수와 인코딩된 값을 분리하는 `**Split` 함수를 제공합니다.

---

## safe.go / unsafe.go

- `reflect.Value` 타입의 값을 바이트 슬라이스로 안전하게/불안전하게 변환하는 함수를 제공합니다.

### safe인 경우

```go
//go:build nacl || js || !cgo
// +build nacl js !cgo
```

- cgo를 사용하지 않는 경우, 또는 nacl, js 환경인 경우 safe.go 파일이 컴파일됩니다.

### unsafe인 경우

```go
//go:build nacl || js || !cgo
// +build nacl js !cgo
```

- cgo를 사용하는 경우, unsafe.go 파일이 컴파일됩니다.

---

## typecache.go

- `typeCache` 타입 정보와 관련 디코더, 인코더를 캐싱하는 구조체입니다.
- 동적으로 인코딩/디코딩을 수행하기 위해 `reflect.Type` 타입을 사용합니다.
- `rlp/internal/rlpstruct` 패키지를 사용하여 구조체 필드 정보를 캐싱합니다.

### typeCache 타입

```go
type typeCache struct {
	cur atomic.Value

	// 이 뮤텍스는 쓰기를 동기화합니다.
	mu   sync.Mutex
	next map[typekey]*typeinfo
}

var theTC = newTypeCache() // 싱글톤 패턴으로 단일 인스턴스를 사용합니다.
```

### typeinfo, typekey 타입

```go
// typeinfo는 타입 캐시의 항목입니다.
type typeinfo struct {
	decoder    decoder
	decoderErr error // makeDecoder의 오류
	writer     writer
	writerErr  error // makeWriter의 오류
}

// typekey는 typeCache의 타입 키입니다. 구조체 태그는 다른 디코더를 생성할 수 있기 때문에 포함됩니다.
type typekey struct {
	reflect.Type
	rlpstruct.Tags
}

type decoder func(*Stream, reflect.Value) error // decoder는 디코딩 규칙을 적용하는 함수입니다.

type writer func(reflect.Value, *encBuffer) error // writer는 인코딩 규칙을 적용하는 함수입니다.
```

---

## ~iterator.go~

- `deprecated`

---

## encbuffer.go

- `EncoderBuffer`와 `encBuffer` 타입은 RLP 인코딩된 값을 저장하는 버퍼입니다.
- `encBuffer`는 `sync.Pool`을 사용하여 재사용합니다.

### EncoderBuffer 타입

```go
// EncoderBuffer는 점진적 인코딩을 위한 버퍼입니다.
//
// 각 타입의 zero value는 사용할 수 없습니다.
// 사용 가능한 버퍼를 얻으려면 NewEncoderBuffer를 사용하거나 Reset을 호출하십시오.
type EncoderBuffer struct {
	buf *encBuffer
	dst io.Writer

	ownBuffer bool
}
```

### encBuffer 타입

```go
type encBuffer struct {
	str     []byte     // 문자열 데이터, 리스트 헤더를 제외한 모든 것을 포함
	lheads  []listhead // 모든 리스트 헤더
	lhsize  int        // 모든 인코딩된 리스트 헤더의 크기의 합
	sizebuf [9]byte    // uint 인코딩을 위한 보조 버퍼
}
```
### encReader 타입

```go
// encReader는 EncodeToReader에 의해 반환된 io.Reader입니다.
// EOF에서 encbuf를 해제합니다.
type encReader struct {
	buf    *encBuffer // 읽기 작업을 수행하는 버퍼. EOF일 때 nil이 된다.
	lhpos  int        // 읽고 있는 리스트 헤더의 인덱스
	strpos int        // 문자열 버퍼에서 현재 위치
	piece  []byte     // 다음 읽을 조각
}
```

### 리스트 인코딩 순서

1. `NewEncoderBuffer(dst)` 또는 `Reset(dst)`을 호출하여 `EncoderBuffer`를 초기화합니다.
    - `NewEncoderBuffer`의 인수로 전달한 `io.Writer`가 `*encBuffer`를 가지고 있다면 이를 사용하고, 그렇지 않은 경우 버퍼 풀에서 `*encBuffer`를 가져옵니다.
2. `w.List()`를 호출하여 RLP 리스트를 시작합니다. 이 때 리스트 헤더의 인덱스가 반환됩니다.
3. `w.Write***(...)`를 호출하여 리스트에 아이템을 추가합니다.
4. `w.ListEnd(idx)`를 호출하여 리스트를 종료합니다. 이 때 리스트 헤더의 인덱스를 전달합니다.
5. `w.Flush()`를 호출하여 버퍼에 쓴 데이터를 `dst`에 씁니다. (*encBuffer의 writeTo 메서드가 호출됩니다.)
6. 사용이 끝난 `*encBuffer`를 버퍼 풀에 반환합니다.

---

## encode.go

- 커스텀 인코딩을 규칙을 적용하기 위한 `Encoder` 인터페이스를 제공합니다.
- `reflect.Type`에 따라 동적으로 인코딩 규칙을 적용하기 위해 `Encode` 함수와 `write***` 함수가 정의되어 있습니다.

### Encoder 인터페이스

```go
// Encoder는 사용자 정의 인코딩 규칙이 필요한 타입이나
// private 필드를 인코딩하고 싶은 타입에 의해 구현됩니다.
type Encoder interface {
	// EncodeRLP는 리시버의 RLP 인코딩을 io.Writer에 씁니다.
	// 포인터 메서드로 구현된 경우 nil 포인터에 대해서도 호출될 수 있습니다.
	//
	// 구현체는 유효한 RLP를 생성해야 합니다. io.Writer에 쓰인 데이터는
	// 현시점에서 검증되지는 않지만, 향후 버전에서는 검증될 수 있습니다.
	// 하나의 값만 쓰는 것을 권장하지만, 그렇지 않은 경우도 허용됩니다.
	EncodeRLP(io.Writer) error
}
```

### Encode 함수

- `val`에 대한 동적인 인코딩 규칙을 적용하여 `w`에 씁니다.

```go
// Encode는 val의 RLP 인코딩을 w에 씁니다. Encode는 경우에 따라
// 많은 작은 쓰기 작업을 수행할 수 있습니다. w를 버퍼링하는 것을 고려하세요.
//
// 인코딩 규칙에 대한 패키지 수준의 문서를 참조하세요.
func Encode(w io.Writer, val interface{}) error {
	// 최적화: EncodeRLP에 의해 호출될 때 *encBuffer를 재사용합니다.
	if buf := encBufferFromWriter(w); buf != nil { // w가 *encBuffer를 구현하는 경우
		return buf.encode(val)
	}

	buf := getEncBuffer()                   // pool에서 *encBuffer를 가져옵니다.
	defer encBufferPool.Put(buf)            // *encBuffer를 pool에 반환합니다.
	if err := buf.encode(val); err != nil { // 인코딩을 수행합니다.
		return err
	}
	return buf.writeTo(w) // 인코딩된 데이터를 w에 씁니다.
}
```

- `*encBuffer`의 `encode` 메서드를 호출하여 인코딩을 수행합니다.

```go
func (buf *encBuffer) encode(val interface{}) error {
	rval := reflect.ValueOf(val)
	writer, err := cachedWriter(rval.Type())
	if err != nil {
		return err
	}
	return writer(rval, buf)
}
```

- `cachedWriter` 함수는 `reflect.Type`에 따라 동적으로 인코딩 규칙을 적용하는 함수를 반환합니다.

```go
var theTC = newTypeCache()

func cachedWriter(typ reflect.Type) (writer, error) {
	info := theTC.info(typ)
	return info.writer, info.writerErr
}
```

- 캐시된 `typeinfo`가 존재하지 않는 경우, `encode.go` 파일에 정의된 `makeWriter` 함수와 `makeDecoder` 함수를 호출하여 새로운 `typeinfo`를 생성합니다.

### listHead 타입

```go
type listhead struct {
	offset int // 문자열 데이터에서 이 헤더의 오프셋
	size   int // 인코딩된 데이터의 총 크기(리스트 헤더 포함)
}
```

- `encBuffer`에 저장된 리스트 헤더의 정보를 저장하는 타입입니다.

---

## decode.go

- 커스텀 디코딩을 규칙을 적용하기 위한 `Decoder` 인터페이스를 제공합니다.
- `Stream` 타입을 사용하여 RLP 인코딩된 데이터를 구문 분석합니다.
- `reflect.Type`에 따라 동적으로 디코딩 규칙을 적용하기 위해 `Decode` 함수와 `decode***` 함수가 정의되어 있습니다.

### Decoder 인터페이스

```go
// Decoder는 사용자 정의 RLP 디코딩 규칙이 필요한 유형 또는 내부(private) 필드로 디코딩해야하는 유형에 의해 구현됩니다.
//
// DecodeRLP 메서드는 주어진 Stream에서 하나의 값을 읽어야합니다. 덜 읽거나 더 읽는 것은 금지되지 않았지만 혼란을 야기할 수 있습니다.
type Decoder interface {
	DecodeRLP(*Stream) error
}
```

### Decode 함수

- `r`에서 RLP로 인코딩된 데이터를 구문 분석하고 `val`이 가리키는 값에 결과를 저장합니다.

```go
// Decode는 r에서 RLP로 인코딩된 데이터를 구문 분석하고 val이 가리키는 값에 결과를 저장합니다.
// 디코딩 규칙에 대한 것은 패키지 수준 문서를 참조하십시오. Val은 nil이 아닌 포인터여야합니다.
//
// r이 ByteReader를 구현하지 않으면 Decode는 자체 버퍼링을 수행합니다.
//
// Decode는 모든 리더에 대해 입력 제한을 설정하지 않으며, 따라서 거대한 값 크기로 인한 패닉에 취약할 수 있습니다.
// 입력 제한이 필요한 경우 다음을 사용하십시오.
//
// NewStream(r, limit).Decode(val)
func Decode(r io.Reader, val interface{}) error {
	stream := streamPool.Get().(*Stream) // 스트림 풀에서 스트림 가져오기
	defer streamPool.Put(stream)         // 스트림 풀에 스트림 반환

	stream.Reset(r, 0)        // 스트림을 r로 초기화
	return stream.Decode(val) // val에 스트림을 디코딩
}
```

- `Stream` 입력 스트림을 초기화하고 `Decode` 메서드를 호출하여 디코딩을 수행합니다.

### Stream 타입

```go
// Stream은 입력 스트림의 파편적 디코딩에 사용할 수 있습니다.
// 이는 입력이 매우 크거나 유형에 대한 디코딩 규칙이 입력 구조에 따라 다른 경우 유용합니다.
// Stream은 내부 버퍼를 유지하지 않습니다. 값을 디코딩 한 후 입력 리더는 다음 값에 대한 유형 정보 바로 앞에 위치합니다.
//
// 리스트를 디코딩하다가 입력 위치가 목록의 선언 된 길이에 도달하면 모든 작업은 EOL 오류를 반환합니다.
// 리스트의 마지막은 ListEnd를 사용하여 알려야합니다.
//
// Stream은 동시 접근에 대해 안전하지 않습니다.
type Stream struct {
	r ByteReader

	remaining uint64   // r에서 읽어야하는 남은 바이트 수
	size      uint64   // 캐시된 값의 크기
	kinderr   error    // 지난 readKind에서 발생한 오류
	stack     []uint64 // 리스트 크기
	uintbuf   [32]byte // 정수 디코딩을 위한 보조 버퍼
	kind      Kind     // 캐시된 값의 종류
	byteval   byte     // 타입 태그의 단일 바이트 값
	limited   bool     // 입력 제한이 적용되는 경우 true
}

// Decode는 값을 디코딩하고 그 결과를 val이 가리키는 값에 저장합니다.
// 디코딩 규칙에 대한 설명은 Decode 함수에 대한 문서를 참조하십시오.
func (s *Stream) Decode(val interface{}) error {
	if val == nil { // val은 nil이 아닌 포인터 유형이어야합니다.
		return errDecodeIntoNil
	}
	rval := reflect.ValueOf(val) // val의 값을 가져옵니다.
	rtyp := rval.Type()          // val의 유형을 가져옵니다.
	if rtyp.Kind() != reflect.Ptr {
		return errNoPointer
	}
	if rval.IsNil() {
		return errDecodeIntoNil
	}
	decoder, err := cachedDecoder(rtyp.Elem()) // 유형에 대한 디코더를 가져옵니다.
	if err != nil {
		return err
	}

	err = decoder(s, rval.Elem()) // 값을 디코딩합니다.
	if decErr, ok := err.(*decodeError); ok && len(decErr.ctx) > 0 {
		// 디코딩 대상 유형을 오류에 추가하여 컨텍스트가 더 의미 있도록합니다.
		decErr.ctx = append(decErr.ctx, fmt.Sprint("(", rtyp.Elem(), ")"))
	}
	return err
}
```

-  `cachedDecoder` 함수는 `reflect.Type`에 따라 동적으로 디코딩 규칙을 적용하는 함수를 반환합니다.
- 캐시된 `typeinfo`가 존재하지 않는 경우, `decode.go` 파일에 정의된 `makeDecoder` 함수와 `makeWriter` 함수를 호출하여 새로운 `typeinfo`를 생성합니다.

---

## internal/rlpstruct 패키지

### rlpstruct.go

- 구조체의 각 필드의 (타입, 태그)에 대한 규칙을 처리하고 필드를 필터링하는 데 사용되는 타입과 함수를 정의합니다.