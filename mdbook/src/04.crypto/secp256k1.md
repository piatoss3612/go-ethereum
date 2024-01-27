# secp256k1 패키지

- secp256k1 타원 곡선 및 곡선 상의 점에 대한 연산을 수행하는 함수를 제공합니다.

## curve.go

- 코블리츠 타원 곡선과 곡선 상의 점에 대한 연산(덧셈, 2배, 스칼라 곱)을 수행하는 함수를 정의합니다.
- 자코비안 좌표계를 사용합니다.

### BitCurve 구조체

- secp256k1 `y²=x³+7` 방정식을 사용합니다.
- 싱글톤 패턴을 사용합니다.

```go
// BitCurve는 a=0인 코블리츠 타원곡선을 나타냅니다. (y²=x³+B)
// http://www.hyperelliptic.org/EFD/g1p/auto-shortw.html 참조
type BitCurve struct {
	P       *big.Int // 체의 위수
	N       *big.Int // 생성점의 위수
	B       *big.Int // BitCurve 방정식의 상수 (y²=x³+B)
	Gx, Gy  *big.Int // 생성점의 (x,y)
	BitSize int      // 유한체의 비트 수
}

var theCurve = new(BitCurve)

func init() {
	// See SEC 2 section 2.7.1
	// curve parameters taken from:
	// http://www.secg.org/sec2-v2.pdf
	theCurve.P, _ = new(big.Int).SetString("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEFFFFFC2F", 0)  // 유한체의 위수
	theCurve.N, _ = new(big.Int).SetString("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141", 0)  // 생성점의 위수
	theCurve.B, _ = new(big.Int).SetString("0x0000000000000000000000000000000000000000000000000000000000000007", 0)  // BitCurve 방정식의 상수 7 (y²=x³+7)
	theCurve.Gx, _ = new(big.Int).SetString("0x79BE667EF9DCBBAC55A06295CE870B07029BFCDB2DCE28D959F2815B16F81798", 0) // 생성점의 x좌표
	theCurve.Gy, _ = new(big.Int).SetString("0x483ADA7726A3C4655DA4FBFC0E1108A8FD17B448A68554199C47D08FFB10D4B8", 0) // 생성점의 y좌표
	theCurve.BitSize = 256                                                                                           // 유한체의 비트 수
}
```

### scalar_mult_cgo.go

- `C` 라이브러리를 임포트하여 스칼라 곱셈을 수행합니다.

```go
//go:build !gofuzz && cgo
// +build !gofuzz,cgo
```

> 사이드 채널 공격에 대해 언급.

### scalar_mult_nocgo.go

- 미구현.

```go
//go:build gofuzz || !cgo
// +build gofuzz !cgo
```

---

## secp256.go

- `C` 라이브러리를 임포트하여 secp256k1 타원 곡선 상의 점에 대한 서명 및 검증 함수를 정의합니다.

---

## 그 외

- `libsecp256k1` C 라이브러리를 불러와 사용하기 위한 파일들로 보입니다.