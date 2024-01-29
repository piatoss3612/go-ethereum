# params 패키지

## version.go

- go-ethereum의 버전 정보를 담고 있는 상수들이 정의되어 있습니다.
- 시맨틱 버저닝(Semantic Versioning)을 따릅니다.

---

## bootnodes.go

- go-ethereum의 부트스트랩 노드의 URI 정보를 담고 있는 상수들이 정의되어 있습니다.
- `enode://`로 시작하는 URI 형식을 따릅니다.
- DNS를 통해 노드의 IP 주소를 얻어오는 기능도 제공합니다. (EIP-1459)

---

## config.go

- go-ethereum의 기본 설정 정보를 담고 있는 상수들이 정의되어 있습니다.
- 제네시스 해시에 따라 일부 설정 정보가 달라집니다.

### ChainConfig 구조체

```go
// ChainConfig는 블록 체인 설정을 결정하는 핵심 구성입니다.
//
// ChainConfig는 블록에 따라 데이터베이스에 저장됩니다. 이는
// 제네시스 블록으로 식별되는 모든 네트워크는 자체 설정을 가질 수 있음을 의미합니다.
type ChainConfig struct {
	ChainID *big.Int `json:"chainId"` // chainId는 현재 체인을 식별하고 재생 방지를 위해 사용됩니다.

	HomesteadBlock *big.Int `json:"homesteadBlock,omitempty"` // Homestead 전환 블록 (nil = 포크 없음, 0 = 이미 홈스테드)

	DAOForkBlock   *big.Int `json:"daoForkBlock,omitempty"`   // TheDAO 하드 포크 전환 블록 (nil = 포크 없음)
	DAOForkSupport bool     `json:"daoForkSupport,omitempty"` // 노드가 DAO 하드 포크를 지원하거나 반대하는지 여부

	// EIP150 implements the Gas price changes (https://github.com/ethereum/EIPs/issues/150)
	EIP150Block *big.Int `json:"eip150Block,omitempty"` // EIP150 HF 블록 (nil = 포크 없음)
	EIP155Block *big.Int `json:"eip155Block,omitempty"` // EIP155 HF 블록
	EIP158Block *big.Int `json:"eip158Block,omitempty"` // EIP158 HF 블록

	ByzantiumBlock      *big.Int `json:"byzantiumBlock,omitempty"`      // Byzantium 전환 블록 (nil = 포크 없음, 0 = 이미 byzantium)
	ConstantinopleBlock *big.Int `json:"constantinopleBlock,omitempty"` // Constantinople 전환 블록 (nil = 포크 없음, 0 = 이미 constantinople)
	PetersburgBlock     *big.Int `json:"petersburgBlock,omitempty"`     // Petersburg 전환 블록 (nil = constantinople과 동일)
	IstanbulBlock       *big.Int `json:"istanbulBlock,omitempty"`       // Istanbul 전환 블록 (nil = 포크 없음, 0 = 이미 istanbul)
	MuirGlacierBlock    *big.Int `json:"muirGlacierBlock,omitempty"`    // Eip-2384 (난이도 폭탄 지연) 스위치 블록 (nil = 포크 없음, 0 = 이미 활성화됨)
	BerlinBlock         *big.Int `json:"berlinBlock,omitempty"`         // Berlin 스위치 블록 (nil = 포크 없음, 0 = 이미 berlin)
	LondonBlock         *big.Int `json:"londonBlock,omitempty"`         // London 스위치 블록 (nil = 포크 없음, 0 = 이미 london)
	ArrowGlacierBlock   *big.Int `json:"arrowGlacierBlock,omitempty"`   // Eip-4345 (난이도 폭탄 지연) 스위치 블록 (nil = 포크 없음, 0 = 이미 활성화됨)
	GrayGlacierBlock    *big.Int `json:"grayGlacierBlock,omitempty"`    // Eip-5133 (난이도 폭탄 지연) 스위치 블록 (nil = 포크 없음, 0 = 이미 활성화됨)
	MergeNetsplitBlock  *big.Int `json:"mergeNetsplitBlock,omitempty"`  // The Merge 이후의 가상 포크를 네트워크 분할기로 사용

	// 포크 스케줄링은 블록에서 타임 스탬프로 전환되었습니다.

	ShanghaiTime *uint64 `json:"shanghaiTime,omitempty"` // Shanghai 스위치 시간 (nil = 포크 없음, 0 = 이미 shanghai)
	CancunTime   *uint64 `json:"cancunTime,omitempty"`   // Cancun 스위치 시간 (nil = 포크 없음, 0 = 이미 cancun)
	PragueTime   *uint64 `json:"pragueTime,omitempty"`   // Prague 스위치 시간 (nil = 포크 없음, 0 = 이미 prague)
	VerkleTime   *uint64 `json:"verkleTime,omitempty"`   // Verkle 스위치 시간 (nil = 포크 없음, 0 = 이미 verkle)

	// TerminalTotalDifficulty는 컨센서스 업그레이드를 트리거하는 네트워크가 도달한 총 난이도량입니다.
	TerminalTotalDifficulty *big.Int `json:"terminalTotalDifficulty,omitempty"`

	// TerminalTotalDifficultyPassed는 네트워크가 이미 터미널 총 난이도를 통과했음을 지정하는 플래그입니다.
	// 그 목적은 TTD를 로컬로 보지 않고도 레거시 동기화를 비활성화하는 것입니다(장기적으로 안전함).
	TerminalTotalDifficultyPassed bool `json:"terminalTotalDifficultyPassed,omitempty"`

	// 다양한 컨센서스 엔진
	Ethash *EthashConfig `json:"ethash,omitempty"`
	Clique *CliqueConfig `json:"clique,omitempty"`
}
```

---

## dao.go

- go-ethereum의 DAO 하드 포크 관련 상수들이 정의되어 있습니다.

---

## denomination.go

- 이더 단위를 나타내는 상수들이 정의되어 있습니다.

```go
const (
	Wei   = 1
	GWei  = 1e9
	Ether = 1e18
)
```

---

## network_params.go

- 이더리움 클라이언트 사이의 네트워크 통신에 사용되는 상수들이 정의되어 있습니다.

---

## protocol_params.go

- 이더리움 프로토콜에 사용되는 상수들이 정의되어 있습니다.
- EVM의 가스 계산(storae, tx, call, keccak256, )에 사용되는 고정값들도 정의되어 있습니다.

### SstoreResetGas 변천사

1. [EIP-2200](https://eips.ethereum.org/EIPS/eip-2200): 5000
2. [EIP-2929](https://eips.ethereum.org/EIPS/eip-2929): 5000 - 2100 = 2900
3. [EIP-3529](https://eips.ethereum.org/EIPS/eip-3529): 5000 - 2100 + 1900 = 4800