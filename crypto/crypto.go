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

package crypto

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/rlp"
	"golang.org/x/crypto/sha3"
)

// SignatureLength는 복구 ID와 서명을 전달하는 데 필요한 바이트 길이를 나타냅니다.
const SignatureLength = 64 + 1 // 64 바이트 ECDSA 서명 + 1 바이트 복구 ID

// RecoveryIDOffset는 복구 ID를 포함하는 서명 내 바이트 오프셋을 가리킵니다.
const RecoveryIDOffset = 64

// DigestLength는 서명 다이제스트의 정확한 길이를 설정합니다.
const DigestLength = 32

var (
	secp256k1N, _  = new(big.Int).SetString("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141", 16) // 생성정 G의 유한순환군의 위수
	secp256k1halfN = new(big.Int).Div(secp256k1N, big.NewInt(2))
)

var errInvalidPubkey = errors.New("invalid secp256k1 public key")

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

// HashData는 KeccakState를 사용하여 제공된 데이터를 해시하고 32 바이트 해시를 반환합니다.
func HashData(kh KeccakState, data []byte) (h common.Hash) {
	kh.Reset()
	kh.Write(data)
	kh.Read(h[:])
	return h
}

// Keccak256은 입력 데이터의 Keccak256 해시를 계산하고 반환합니다.
func Keccak256(data ...[]byte) []byte {
	b := make([]byte, 32)
	d := NewKeccakState()
	for _, b := range data {
		d.Write(b)
	}
	d.Read(b)
	return b
}

// Keccak256Hash는 입력 데이터의 Keccak256 해시를 계산하고 내부 Hash 데이터 구조로 변환하여 반환합니다.
func Keccak256Hash(data ...[]byte) (h common.Hash) {
	d := NewKeccakState()
	for _, b := range data {
		d.Write(b)
	}
	d.Read(h[:])
	return h
}

// Keccak512는 입력 데이터의 Keccak512 해시를 계산하고 반환합니다.
func Keccak512(data ...[]byte) []byte {
	d := sha3.NewLegacyKeccak512()
	for _, b := range data {
		d.Write(b)
	}
	return d.Sum(nil)
}

// CreateAddress는 이더리움 주소와 논스를 사용하여 새로운 이더리움 주소를 생성합니다.
func CreateAddress(b common.Address, nonce uint64) common.Address {
	data, _ := rlp.EncodeToBytes([]interface{}{b, nonce})
	return common.BytesToAddress(Keccak256(data)[12:])
}

// CreateAddress2는 주소, 초기 컨트랙트 코드 해시 그리고 설트를 사용하여 이더리움 주소를 생성합니다.
func CreateAddress2(b common.Address, salt [32]byte, inithash []byte) common.Address {
	return common.BytesToAddress(Keccak256([]byte{0xff}, b.Bytes(), salt[:], inithash)[12:])
}

// ToECDSA는 주어진 D 값으로 개인 키를 생성합니다.
func ToECDSA(d []byte) (*ecdsa.PrivateKey, error) {
	return toECDSA(d, true)
}

// ToECDSAUnsafe는 이진 blob을 개인 키로 무작정 변환합니다.
// 입력이 유효하고 원본 인코딩 오류를 피하려는 경우가 아니면 사용하지 않는 것이 좋습니다(0 접두사가 잘림).
func ToECDSAUnsafe(d []byte) *ecdsa.PrivateKey {
	priv, _ := toECDSA(d, false)
	return priv
}

// toECDSA는 주어진 D 값으로 개인 키를 생성합니다.
// strict 매개변수는 키의 길이가 곡선 크기에서 강제되어야 하는지 또는 레거시 인코딩(0 접두사)도 허용할 수 있는지를 제어합니다.
func toECDSA(d []byte, strict bool) (*ecdsa.PrivateKey, error) {
	priv := new(ecdsa.PrivateKey)
	priv.PublicKey.Curve = S256()
	if strict && 8*len(d) != priv.Params().BitSize { // d의 비트 크기가 곡선의 유한체의 크기와 일치하는지 확인합니다.
		return nil, fmt.Errorf("invalid length, need %d bits", priv.Params().BitSize)
	}
	priv.D = new(big.Int).SetBytes(d)

	// priv.D는 위수 N보다 작아야 합니다.
	if priv.D.Cmp(secp256k1N) >= 0 {
		return nil, errors.New("invalid private key, >=N")
	}
	// priv.D는 0 또는 음수일 수 없습니다.
	if priv.D.Sign() <= 0 {
		return nil, errors.New("invalid private key, zero or negative")
	}

	priv.PublicKey.X, priv.PublicKey.Y = priv.PublicKey.Curve.ScalarBaseMult(d)
	if priv.PublicKey.X == nil {
		return nil, errors.New("invalid private key")
	}
	return priv, nil
}

// FromECDSA는 개인 키를 바이너리 형식으로 내보냅니다.
func FromECDSA(priv *ecdsa.PrivateKey) []byte {
	if priv == nil {
		return nil
	}
	return math.PaddedBigBytes(priv.D, priv.Params().BitSize/8)
}

// UnmarshalPubkey는 바이트를 secp256k1 공개 키로 변환합니다.
func UnmarshalPubkey(pub []byte) (*ecdsa.PublicKey, error) {
	x, y := elliptic.Unmarshal(S256(), pub)
	if x == nil {
		return nil, errInvalidPubkey
	}
	return &ecdsa.PublicKey{Curve: S256(), X: x, Y: y}, nil
}

func FromECDSAPub(pub *ecdsa.PublicKey) []byte {
	if pub == nil || pub.X == nil || pub.Y == nil {
		return nil
	}
	return elliptic.Marshal(S256(), pub.X, pub.Y)
}

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

// LoadECDSA는 주어진 파일에서 secp256k1 개인 키를 로드합니다.
func LoadECDSA(file string) (*ecdsa.PrivateKey, error) {
	fd, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	r := bufio.NewReader(fd)
	buf := make([]byte, 64)
	n, err := readASCII(buf, r)
	if err != nil {
		return nil, err
	} else if n != len(buf) {
		return nil, errors.New("key file too short, want 64 hex characters")
	}
	if err := checkKeyFileEnd(r); err != nil {
		return nil, err
	}

	return HexToECDSA(string(buf))
}

// readASCII는 버퍼가 가득 차거나 인쇄할 수 없는 제어 문자가 나타날 때까지 'buf'로 읽어들입니다.
func readASCII(buf []byte, r *bufio.Reader) (n int, err error) {
	for ; n < len(buf); n++ {
		buf[n], err = r.ReadByte()
		switch {
		case err == io.EOF || buf[n] < '!':
			return n, nil
		case err != nil:
			return n, err
		}
	}
	return n, nil
}

// checkKeyFileEnd는 키 파일의 끝에 추가적인 줄바꿈을 건너뜁니다.
func checkKeyFileEnd(r *bufio.Reader) error {
	for i := 0; ; i++ {
		b, err := r.ReadByte()
		switch {
		case err == io.EOF:
			return nil
		case err != nil:
			return err
		case b != '\n' && b != '\r':
			return fmt.Errorf("invalid character %q at end of key file", b)
		case i >= 2:
			return errors.New("key file too long, want 64 hex characters")
		}
	}
}

// SaveECDSA는 제한적인 권한으로 주어진 파일에 secp256k1 개인 키를 저장합니다. 키 데이터는 16진수로 인코딩되어 저장됩니다.
func SaveECDSA(file string, key *ecdsa.PrivateKey) error {
	k := hex.EncodeToString(FromECDSA(key))
	return os.WriteFile(file, []byte(k), 0600)
}

// GenerateKey는 새로운 개인 키를 생성합니다.
func GenerateKey() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(S256(), rand.Reader)
}

// ValidateSignatureValues는 서명 값이 주어진 체인 규칙과 유효한지 확인합니다.
// v 값은 0 또는 1로 가정됩니다.
func ValidateSignatureValues(v byte, r, s *big.Int, homestead bool) bool {
	if r.Cmp(common.Big1) < 0 || s.Cmp(common.Big1) < 0 {
		return false
	}
	// s 값의 상위 범위를 거부합니다(ECDSA 가변성)
	// secp256k1/libsecp256k1/include/secp256k1.h의 토론 참조
	if homestead && s.Cmp(secp256k1halfN) > 0 { // Homestead: s 값이 N/2보다 크면 거부합니다.
		return false
	}
	// Frontier: r 또는 s가 N보다 크면 거부합니다.
	return r.Cmp(secp256k1N) < 0 && s.Cmp(secp256k1N) < 0 && (v == 0 || v == 1)
}

func PubkeyToAddress(p ecdsa.PublicKey) common.Address {
	pubBytes := FromECDSAPub(&p)
	return common.BytesToAddress(Keccak256(pubBytes[1:])[12:])
}

func zeroBytes(bytes []byte) {
	for i := range bytes {
		bytes[i] = 0
	}
}
