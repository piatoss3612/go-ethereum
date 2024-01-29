# 이더리움의 역사 (History of Ethereum)

## 0. Genesis

### 이더리움 백서 발표

- 2013년 11월 27일 (UTC) [Ethereum: A Next-Generation Smart Contract and Decentralized Application Platform](https://ethereum.org/whitepaper)

### 이더리움 황서 발표

- 2014년 4월 1일 (UTC) [Ethereum Yellow Paper](https://ethereum.github.io/yellowpaper/paper.pdf)

---

## 0.5 Olympic

- 생략

---

## 1. Frontier

> 하드보일드한 개척자들의 시대

### 1.1 이더리움 프론티어 릴리즈

- 2015년 7월 30일 (UTC) [Ethereum Frontier Release](https://blog.ethereum.org/2015/07/30/ethereum-launches/)
- Olympic 테스트넷에서의 테스트를 거쳐, 이더리움 프론티어가 릴리즈되었습니다.
- 블록당 가스 한도를 5000으로 설정하여 얼리 어답터들(개발자)만이 제한된 트랜잭션을 생성할 수 있었습니다.

### 1.2 프론티어 해동 (Frontier Thawing)

- 2015년 9월 7일 (UTC) [Frontier Thawing](https://blog.ethereum.org/2015/09/07/ethereum-thawing/)
- 블록 번호: 200,000
- 기존에 5,000으로 고정되었던 블록당 가스 한도는 최대 3,141,592까지 증가할 수 있도록 변경되었습니다.
- 블록당 가스 한도 증가량은 최대 (부모 블록의 가스 한도/1024)로 제한됩니다.
- 기본 가스비를 51 gwei로 설정하였습니다.
- 트랜잭션을 생성하기 위한 최소 가스비를 21,000으로 설정하였습니다.
- 향후 PoS(Proof of Stake)로 전환을 위해 난이도 폭탄(Difficulty Bomb)을 추가하였습니다.

> 난이도 폭탄이란, PoS 전환을 위해 채굴이 불가능한 난이도를 가진 블록을 추가하는 것을 의미합니다. 이는 채굴자들이 PoS로 전환되는 시점에 PoW로 채굴을 이행함으로 인해 포크가 발생하는 것을 방지하기 위함입니다.

- [The Thawing Frontier](https://blog.ethereum.org/2015/08/04/the-thawing-frontier)

### 1.3 최초의 EIP (Ethereum Improvement Proposal)

- 2015년 10월 27일 (UTC)
- [EIP-1](https://eips.ethereum.org/EIPS/eip-1)
- Bitcoin Improvement Proposal을 모방하여 Ethereum Improvement Proposal의 첫 번째 제안서가 등장하였습니다.

---

## 2. Homestead

> 개척자들의 시대가 끝나고, 정착민들이 마을을 일구어 나가는 시대

### 2.1 이더리움 홈스테드 릴리즈

- 2016년 3월 14일 (UTC) [Homestead Release](https://blog.ethereum.org/2016/02/29/homestead-release)
- 블록 번호: 1,150,000
- 선별된 EIP에 대한 프로토콜 업그레이드가 이루어졌습니다.

#### EIP-2: Homestead Hard-fork Changes

- [EIP-2](https://eips.ethereum.org/EIPS/eip-2)

1. 트랜잭션을 통해 컨트랙트를 생성할 때, 가스 비용을 21,000에서 53,000으로 증가시켰습니다. 컨트래트 내부에서 `CREATE` opcode를 통해 컨트랙트를 생성하는 경우는 영향을 받지 않습니다. (컨트랙트 suicide를 통해 자금을 비교적 싼 가격으로 회수할 수 있기 때문)
2. 서명의 `s` 값이 `secp256k1n/2`보다 큰 경우, 유효하지 않은 서명으로 판단합니다. 이는 단순히 `s` 값을 `secp256k1n/2 - s`로 변경하고 `v` 값을 반전시켜 유효한 서명을 만들어냄으로써 트랜잭션 가변성 문제가 발생하는 것을 방지하기 위함입니다.
3. 컨트랙트 생성 시, 컨트랙트 코드를 상태에 저장하기 위한 비용이 모자란 경우 빈 컨트랙트를 생성하는 대신 트랜잭션을 실패시킵니다.
4. 채굴 난이도 조절 알고리즘을 변경하였습니다. (중앙값 대신 평균값을 사용) 이는 채굴 난이도가 급격히 증가하는 것을 방지하기 위함입니다.

```bash
# before
block_diff = parent_diff + parent_diff // 2048 * (1 if block_timestamp - parent_timestamp < 13 else -1) + int(2**((block.number // 100000) - 2))

# after
block_diff = parent_diff + parent_diff // 2048 * max(1 - (block_timestamp - parent_timestamp) // 10, -99) + int(2**((block.number // 100000) - 2))
```

#### EIP-7: DELEGATECALL

- [EIP-7](https://eips.ethereum.org/EIPS/eip-7)

- 새로운 opcode인 `DELEGATECALL`이 `0xf4`로 추가되었습니다.
- `DELEGATECALL`은 `CALL`과 유사하게 동작하지만, 호출된 컨트랙트의 코드가 호출자의 컨텍스트에서 실행됩니다.
- 컨트랙트 형식의 라이브러리를 호출자의 컨텍스트에서 실행하기 위해 추가되었지만, 현재는 별도로 라이브러리를 정의할 수 있기 때문에 사용에 주의가 필요합니다.

#### EIP-8: devp2p Forward Compatibility Requirements for Homestead

- [EIP-8](https://eips.ethereum.org/EIPS/eip-8)

- 향후 업그레이드를 위해 devp2p 프로토콜에 대한 호환성 요구사항을 추가하였습니다.
- 자세한 내용은 너무 복잡하고 길어서 생략합니다.

### 2.2 DAO fork (하드포크)

- 2016년 6월 20일 (UTC) [The DAO](https://blog.ethereum.org/2016/06/17/critical-update-re-dao-vulnerability)
- 블록 번호: 1,920,000
- DAO(Distributed Autonomous Organization)는 계층적인 조직 구조가 아닌, 참여자들에 의해 자율적으로 운영되는 조직을 의미합니다.
- `The DAO`라는 프로젝트의 취약점을 이용한 해킹으로 인해 대량의 이더가 탈취당하는 사건이 발생하였습니다.
- 일부 이더리움 사용자들은 이를 무효화시키기 위해 하드포크를 제안하였고, 이를 반대하는 사용자들은 하드포크를 통해 이더리움의 무결성이 훼손될 수 있다고 주장하였습니다.
- 결국 하드포크가 이루어지면서, 탈취된 이더는 원래 주인에게 돌아가고, 하드포크를 거부하는 사용자들은 이더리움 클래식(Ethereum Classic)이라는 이름으로 기존의 이더리움을 유지하였습니다.

- [The DAO Attack](https://www.coindesk.com/learn/understanding-the-dao-attack/)

#### EIP-779: Hardfork Meta: DAO Fork

- [EIP-779](https://eips.ethereum.org/EIPS/eip-779)

### 2.3 Tangerine Whistle (하드포크)

- 2016년 10월 18일 (UTC) [Tangerine Whistle](https://blog.ethereum.org/2016/10/18/faq-upcoming-ethereum-hard-fork)
- 블록 번호: 2,463,000
- `DoS(Denial of Service)` 공격을 방지하기 위해 하드포크가 이루어졌습니다.

#### EIP-608: Hardfork Meta: Tangerine Whistle

- [EIP-608](https://eips.ethereum.org/EIPS/eip-608)

#### EIP-150: Gas cost changes for IO-heavy operations

- [EIP-150](https://eips.ethereum.org/EIPS/eip-150)

- EXTCODESIZE의 가스비를 20에서 700으로 증가시켰습니다.
- EXTCODECOPY의 가스비를 20에서 700으로 증가시켰습니다.
- BALANCE의 가스비를 20에서 400로 증가시켰습니다.
- SLOAD의 가스비를 50에서 200로 증가시켰습니다.
- CALL, DELEGATECALL, CALLCODE의 가스비를 40에서 700으로 증가시켰습니다.
- SELFDESTRUCT의 가스비를 0에서 5000으로 증가시켰습니다. (SELFDESTRUCT를 통해 새로운 계정에 자금을 보내는 경우, 추가적으로 25000의 가스비가 소모됩니다.)
- 권장 가스 한도를 5.5M로 증가시켰습니다.
- `all but one 64th` 규칙을 적용하여 자식은 부모가 가진 가스의 `63/64`만 사용할 수 있도록 제한하였습니다. 이는 재귀 호출을 통해 가스를 소모하는 것(Call Depth Attack)을 방지하기 위함입니다.

- [EIP-150 and the 63/64 Rule for Gas](https://www.rareskills.io/post/eip-150-and-the-63-64-rule-for-gas)

#### EIP-158: State clearing

- [EIP-158](https://eips.ethereum.org/EIPS/eip-158)

- nonce가 0이고, balance가 0이고, storage root가 비어있는 계정은 빈 계정으로 간주됩니다.
- 빈 계정이 다른 트랜잭션에 의해 접근(touch)된 경우, 이를 제거합니다.
- 가치가 0인(자금이 없는) call이나 selfdestruct는 더이상 계정 생성을 위한 25000의 가스비를 소모하지 않습니다.
- 상태를 정리함으로써 클라이언트의 디스크 부하를 줄이고, 빠른 동기화를 가능하게 하였습니다.

### 2.4 Spurious Dragon (하드포크)

- 2016년 11월 22일 (UTC) [Spurious Dragon](https://blog.ethereum.org/2016/11/18/hard-fork-no-4-spurious-dragon)
- 블록 번호: 2,675,000
- `DoS(Denial of Service)` 공격을 방지하기 위해 하드포크가 이루어졌습니다.

#### EIP-607: Hardfork Meta: Spurious Dragon

- [EIP-607](https://eips.ethereum.org/EIPS/eip-607)

#### EIP-155: Simple replay attack protection

- [EIP-155](https://eips.ethereum.org/EIPS/eip-155)

- 서로 다른 체인에서 발생한 트랜잭션이 재생되지 않도록 하기 위해 `v` 값에 체인 ID를 추가하였습니다.
- v 값은 `35 + (chainID * 2)` 또는 `36 + (chainID * 2)`가 됩니다.
- 또한 트랜잭션 해시를 구할 때 `(nonce, gasprice, startgas, to, value, data, chainid, 0, 0)` 형식으로 해시를 구합니다.
- 하위 호환성을 위해 `v` 값이 27 또는 28인 경우도 여전히 유효합니다.

#### EIP-160: EXP cost increase

- [EIP-160](https://eips.ethereum.org/EIPS/eip-160)

- EXP의 가스비를 `10 + 10 * exponent의 바이트 수`에서 `10 + 50 * exponent의 바이트 수`로 증가시켰습니다.
- 이는 네트워크 자원을 많이 소모하는 EXP opcode의 비용이 저렴하여 DoS 공격에 취약하다는 점을 보완하기 위함입니다.

#### EIP-161: State trie clearing (invariant-preserving alternative)

- [EIP-161](https://eips.ethereum.org/EIPS/eip-161)

- 낮은 가격으로 상태에 저장된 빈 계정을 제거하는 것을 가능하게 하였습니다.
- 불변을 유지하기 위해 일부 극단적인 경우를 제외했다는 점을 빼고는 EIP-158과 동일합니다.

#### EIP-170: Contract code size limit

- [EIP-170](https://eips.ethereum.org/EIPS/eip-170)

- 컨트랙트 코드의 크기를 24576 바이트로 제한하였습니다.
- 아주 큰 컨트랙트 코드를 반복적으로 호출하여 네트워크 자원을 고갈시키는 DoS 공격을 방지하기 위함입니다.

---

## 3. Metropolis