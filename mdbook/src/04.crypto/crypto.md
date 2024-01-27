# crypto 패키지

## crypto.go

- 이더리움에서 사용하는 `keccak256` 해시 생성기의 인터페이스를 정의합니다.
- 비밀키(secp256k1)를 생성, 읽기 및 내보내기 함수를 정의합니다.

### KeccakState 인터페이스

```go
// KeccakState는 sha3.state를 래핑합니다. 일반적인 해시 메서드 외에도, 해시 상태에서 가변 길이의 데이터를 얻는 데도 지원합니다.
// Read는 내부 상태를 복사하지 않기 때문에 Sum보다 빠르지만 내부 상태를 수정합니다.
type KeccakState interface {
	hash.Hash
	Read([]byte) (int, error)
}

// NewKeccakState는 새로운 KeccakState를 생성합니다.
func NewKeccakState() KeccakState {
	return sha3.NewLegacyKeccak256().(KeccakState)
}
```

### 개인키 관련 함수

```go
// GenerateKey는 새로운 개인 키를 생성합니다.
func GenerateKey() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(S256(), rand.Reader)
}
```

```go
// HexToECDSA는 secp256k1 개인 키를 구문 분석합니다.
func HexToECDSA(hexkey string) (*ecdsa.PrivateKey, error) {
	b, err := hex.DecodeString(hexkey) // 0x 접두사가 없어야 합니다.
	if byteErr, ok := err.(hex.InvalidByteError); ok {
		return nil, fmt.Errorf("invalid hex character %q in private key", byte(byteErr))
	} else if err != nil {
		return nil, errors.New("invalid hex data for private key")
	}
	return ToECDSA(b)
}
```

---

## signature_**.go

- ECDSA 서명을 생성하고 검증하는 함수를 정의합니다.

### cgo

- `geth` 이더리움 구현체의 `github.com/ethereum/go-ethereum/crypto/secp256k1` 패키지를 사용합니다.

```go
//go:build !nacl && !js && cgo && !gofuzz
// +build !nacl,!js,cgo,!gofuzz
```

### nocgo

- `btcd` 비트코인 구현체의 `github.com/btcsuite/btcd/btcec/v2/ecdsa` 패키지를 사용합니다.

```go
//go:build nacl || js || !cgo || gofuzz
// +build nacl js !cgo gofuzz
```

