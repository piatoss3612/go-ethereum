// Copyright 2019 The go-ethereum Authors
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

// compiler 패키지는 Solidity와 Vyper 컴파일러 실행 파일(solc; vyper)을 래핑합니다.
package compiler

// Contract는 코드 및 런타임 코드와 함께 컴파일된 컨트랙트에 대한 정보를 포함합니다.
type Contract struct {
	Code        string            `json:"code"`
	RuntimeCode string            `json:"runtime-code"`
	Info        ContractInfo      `json:"info"`
	Hashes      map[string]string `json:"hashes"`
}

// ContractInfo는 컴파일된 컨트랙트에 대한 정보를 포함합니다.
// 이 정보에는 ABI 정의, 소스 매핑, 사용자 및 개발자 문서, 메타데이터 등이 포함됩니다.
//
// 소스, 언어 버전, 컴파일러 버전 및 컴파일러 옵션에 따라 컨트랙트가 컴파일된 방식에 대한 정보를 제공합니다.
type ContractInfo struct {
	Source          string      `json:"source"`
	Language        string      `json:"language"`
	LanguageVersion string      `json:"languageVersion"`
	CompilerVersion string      `json:"compilerVersion"`
	CompilerOptions string      `json:"compilerOptions"`
	SrcMap          interface{} `json:"srcMap"`
	SrcMapRuntime   string      `json:"srcMapRuntime"`
	AbiDefinition   interface{} `json:"abiDefinition"`
	UserDoc         interface{} `json:"userDoc"`
	DeveloperDoc    interface{} `json:"developerDoc"`
	Metadata        string      `json:"metadata"`
}
