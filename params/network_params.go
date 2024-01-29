// Copyright 2017 The go-ethereum Authors
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

package params

// 이 값들은 클라이언트들 사이에서 일관되어야 하는 네트워크 파라미터들이지만,
// 반드시 컨센서스와 관련된 것들은 아니다.

const (
	// BloomBitsBlocks는 서버 측에서 단일 블룸 비트 섹션 벡터가 포함하는 블록 수이다.
	BloomBitsBlocks uint64 = 4096

	// BloomBitsBlocksClient는 라이트 클라이언트 측에서 단일 블룸 비트 섹션 벡터가 포함하는 블록 수이다.
	BloomBitsBlocksClient uint64 = 32768

	// BloomConfirms is the number of confirmation blocks before a bloom section is
	// considered probably final and its rotated bits are calculated.

	// BloomConfirms는 블룸 섹션이 최종 상태로 간주되고 회전된 비트가 계산되기 전의 확인 블록 수이다. (?) - TODO: 이해 안됨
	BloomConfirms = 256

	// CHTFrequency는 CHT를 생성하는 블록 프리퀀시이다.
	CHTFrequency = 32768

	// BloomTrieFrequency는 서버/클라이언트 양쪽에서 BloomTrie를 생성하는 블록 프리퀀시이다.
	BloomTrieFrequency = 32768

	// HelperTrieConfirmations is the number of confirmations before a client is expected
	// to have the given HelperTrie available.

	// HelperTrieConfirmations는 클라이언트가 주어진 HelperTrie를 사용할 수 있게 되는 확인 블록 수이다. (?)
	HelperTrieConfirmations = 2048

	// HelperTrieProcessConfirmations is the number of confirmations before a HelperTrie
	// is generated

	// HelperTrieProcessConfirmations는 HelperTrie가 생성되기 전의 확인 블록 수이다. (?)
	HelperTrieProcessConfirmations = 256

	// CheckpointFrequency는 체크포인트를 생성하는 블록 프리퀀시이다.
	CheckpointFrequency = 32768

	// CheckpointProcessConfirmations는 체크포인트가 생성되기 전의 확인 블록 수이다. (?)
	CheckpointProcessConfirmations = 256

	// FullImmutabilityThreshold is the number of blocks after which a chain segment is
	// considered immutable (i.e. soft finality). It is used by the downloader as a
	// hard limit against deep ancestors, by the blockchain against deep reorgs, by
	// the freezer as the cutoff threshold and by clique as the snapshot trust limit.

	// FullImmutabilityThreshold는 체인 세그먼트가 불변(soft finality)으로 간주되는 블록 수이다. (i.e. soft finality)
	// ??
	FullImmutabilityThreshold = 90000

	// LightImmutabilityThreshold is the number of blocks after which a header chain
	// segment is considered immutable for light client(i.e. soft finality). It is used by
	// the downloader as a hard limit against deep ancestors, by the blockchain against deep
	// reorgs, by the light pruner as the pruning validity guarantee.

	// LightImmutabilityThreshold는 헤더 체인 세그먼트가 라이트 클라이언트에 대해 불변(soft finality)으로 간주되는 블록 수이다.
	// ??
	LightImmutabilityThreshold = 30000
)
