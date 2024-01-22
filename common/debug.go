// Copyright 2015 The go-ethereum Authors
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
	"os"
	"runtime"
	"runtime/debug"
	"strings"
)

// Report는 사용자가 github 트래커에 이슈를 제출하도록 요청하는 경고를 발생시킵니다.
func Report(extra ...interface{}) {
	fmt.Fprintln(os.Stderr, "You've encountered a sought after, hard to reproduce bug. Please report this to the developers <3 https://github.com/ethereum/go-ethereum/issues")
	fmt.Fprintln(os.Stderr, extra...)

	_, file, line, _ := runtime.Caller(1) // 이 함수를 호출한 함수의 파일명과 라인 번호를 반환합니다.
	fmt.Fprintf(os.Stderr, "%v:%v\n", file, line)

	debug.PrintStack() // 스택 트레이스를 표준 오류에 출력합니다.

	fmt.Fprintln(os.Stderr, "#### BUG! PLEASE REPORT ####")
}

// PrintDeprecationWarning는 fmt.Println을 사용하여 주어진 문자열을 박스에 담아 출력합니다.
/*
	ex)

	###############################
	#                             #
	# This function is deprecated #
	#                             #
	###############################
*/
func PrintDeprecationWarning(str string) {
	line := strings.Repeat("#", len(str)+4)
	emptyLine := strings.Repeat(" ", len(str))
	fmt.Printf(`
%s
# %s #
# %s #
# %s #
%s

`, line, emptyLine, str, emptyLine, line)
}
