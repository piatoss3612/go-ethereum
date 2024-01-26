// Copyright 2020 The go-ethereum Authors
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

package rlp

type listIterator struct {
	data []byte
	next []byte
	err  error
}

// NewListIterator는 data로 표시된 (list)에 대한 반복자를 생성합니다.
// TODO: 더 이상 사용되지 않으므로 이 구현을 제거하는 것이 좋습니다.
func NewListIterator(data RawValue) (*listIterator, error) {
	k, t, c, err := readKind(data)
	if err != nil {
		return nil, err
	}
	if k != List {
		return nil, ErrExpectedList
	}
	it := &listIterator{
		data: data[t : t+c],
	}
	return it, nil
}

// Next는 반복자를 한 단계 전진시키고 아직 끝이 아니면 true를 반환합니다.
func (it *listIterator) Next() bool {
	if len(it.data) == 0 {
		return false
	}
	_, t, c, err := readKind(it.data)
	it.next = it.data[:t+c]
	it.data = it.data[t+c:]
	it.err = err
	return true
}

// Value는 현재 값을 반환합니다.
func (it *listIterator) Value() []byte {
	return it.next
}

func (it *listIterator) Err() error {
	return it.err
}
