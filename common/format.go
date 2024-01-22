// Copyright 2016 The go-ethereum Authors
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
	"regexp"
	"strings"
	"time"
)

// PrettyDuration은 time.Duration 값의 기간을 표시하는 문자열의 불필요한 정밀도를 잘라내어 예쁘게 표시합니다.
type PrettyDuration time.Duration

var prettyDurationRe = regexp.MustCompile(`\.[0-9]{4,}`) // 소수점 아래 4자리 이상의 숫자를 찾습니다.

// String은 Stringer 인터페이스를 구현하며, 소수점 아래 3자리까지 반올림하여 기간 값을 예쁘게 표시합니다.
func (d PrettyDuration) String() string {
	label := time.Duration(d).String()
	if match := prettyDurationRe.FindString(label); len(match) > 4 {
		label = strings.Replace(label, match, match[:4], 1)
	}
	return label
}

// PrettyAge는 time.Duration 값을 예쁘게 표시한 것으로, 값을 년/월/주를 포함한 하나의 최상위 단위(a single most significant unit)로 반올림합니다.
type PrettyAge time.Time

// ageUnits는 나이를 예쁘게 표시하는 데 사용되는 단위 목록입니다.
var ageUnits = []struct {
	Size   time.Duration
	Symbol string
}{
	{12 * 30 * 24 * time.Hour, "y"}, // year
	{30 * 24 * time.Hour, "mo"},     // month
	{7 * 24 * time.Hour, "w"},       // week
	{24 * time.Hour, "d"},           // day
	{time.Hour, "h"},                // hour
	{time.Minute, "m"},              // minute
	{time.Second, "s"},              // second
}

// String은 Stringer 인터페이스를 구현하며, 최상위 시간 단위로 반올림된 기간 값을 예쁘게 표시합니다.
func (t PrettyAge) String() string {
	// 현재 시간과의 차이를 계산하고 예외 상황인 0을 처리합니다.
	diff := time.Since(time.Time(t))
	if diff < time.Second {
		return "0"
	}
	// 년/월/주를 포함한 하나의 최상위 단위로 반올림합니다. (prec은 년/월/주가 결과에 포함되었는지 여부를 추적합니다.)
	result, prec := "", 0

	for _, unit := range ageUnits {
		if diff > unit.Size {
			result = fmt.Sprintf("%s%d%s", result, diff/unit.Size, unit.Symbol)
			diff %= unit.Size

			if prec += 1; prec >= 3 {
				break
			}
		}
	}
	return result
}
