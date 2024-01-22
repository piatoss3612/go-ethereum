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

package common

import (
	"fmt"
)

// StorageSize는 사용자 친화적인 포맷을 지원하기 위해 float 값을 래핑한 별칭 타입이다.
type StorageSize float64

// String은 stringer 인터페이스를 구현하였다. (소수점 2자리까지 표시)
func (s StorageSize) String() string {
	if s > 1099511627776 {
		return fmt.Sprintf("%.2f TiB", s/1099511627776) // 테비바이트
	} else if s > 1073741824 {
		return fmt.Sprintf("%.2f GiB", s/1073741824) // 기비바이트
	} else if s > 1048576 {
		return fmt.Sprintf("%.2f MiB", s/1048576) // 메비바이트
	} else if s > 1024 {
		return fmt.Sprintf("%.2f KiB", s/1024) // 킬로바이트
	} else {
		return fmt.Sprintf("%.2f B", s) // 바이트
	}
}

// TerminalString은 log.TerminalStringer를 구현하였으며, 로깅 중 콘솔 출력을 위한 문자열을 포맷합니다.
func (s StorageSize) TerminalString() string {
	if s > 1099511627776 {
		return fmt.Sprintf("%.2fTiB", s/1099511627776)
	} else if s > 1073741824 {
		return fmt.Sprintf("%.2fGiB", s/1073741824)
	} else if s > 1048576 {
		return fmt.Sprintf("%.2fMiB", s/1048576)
	} else if s > 1024 {
		return fmt.Sprintf("%.2fKiB", s/1024)
	} else {
		return fmt.Sprintf("%.2fB", s)
	}
}
