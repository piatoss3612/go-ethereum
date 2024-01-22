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

// compiler 패키지는 ABI 컴파일 출력을 래핑합니다.
package compiler

import (
	"encoding/json"
	"fmt"
)

// --combined-output 형식
type solcOutput struct {
	Contracts map[string]struct {
		BinRuntime                                  string `json:"bin-runtime"`
		SrcMapRuntime                               string `json:"srcmap-runtime"`
		Bin, SrcMap, Abi, Devdoc, Userdoc, Metadata string
		Hashes                                      map[string]string
	}
	Version string
}

// solidity v.0.8부터 ABI, Devdoc 및 Userdoc가 직렬화되는 방식이 변경되었습니다.
type solcOutputV8 struct {
	Contracts map[string]struct {
		BinRuntime            string `json:"bin-runtime"`
		SrcMapRuntime         string `json:"srcmap-runtime"`
		Bin, SrcMap, Metadata string
		Abi                   interface{}
		Devdoc                interface{}
		Userdoc               interface{}
		Hashes                map[string]string
	}
	Version string
}

// ParseCombinedJSON은 solc --combined-output 실행 결과를 직접 가져와 map[string]*Contract 구조체로 구문 분석합니다.
// 제공된 source, language 및 compiler version, 그리고 compiler options는 모두 Contract 구조체로 전달됩니다.
//
// solc 출력에는 ABI, 소스 매핑, 사용자 문서 및 개발자 문서가 포함되어 있어야 합니다.
//
// JSON이 잘못 구성되었거나 데이터가 누락되었거나 JSON 내부가 잘못 구성되었을 경우 오류가 발생합니다.
func ParseCombinedJSON(combinedJSON []byte, source string, languageVersion string, compilerVersion string, compilerOptions string) (map[string]*Contract, error) {
	var output solcOutput
	if err := json.Unmarshal(combinedJSON, &output); err != nil {
		// 오류가 발생하면 새로운 solidity v.0.8.0 규칙으로 출력을 구문 분석하려고 시도합니다.
		return parseCombinedJSONV8(combinedJSON, source, languageVersion, compilerVersion, compilerOptions)
	}
	// 컴파일이 성공하면 컨트랙트를 조립하여 반환합니다.
	contracts := make(map[string]*Contract)
	for name, info := range output.Contracts {
		// 개별 컴파일 결과를 구문 분석합니다.
		var abi, userdoc, devdoc interface{}
		if err := json.Unmarshal([]byte(info.Abi), &abi); err != nil {
			return nil, fmt.Errorf("solc: error reading abi definition (%v)", err)
		}
		if err := json.Unmarshal([]byte(info.Userdoc), &userdoc); err != nil {
			return nil, fmt.Errorf("solc: error reading userdoc definition (%v)", err)
		}
		if err := json.Unmarshal([]byte(info.Devdoc), &devdoc); err != nil {
			return nil, fmt.Errorf("solc: error reading devdoc definition (%v)", err)
		}

		contracts[name] = &Contract{
			Code:        "0x" + info.Bin,
			RuntimeCode: "0x" + info.BinRuntime,
			Hashes:      info.Hashes,
			Info: ContractInfo{
				Source:          source,
				Language:        "Solidity",
				LanguageVersion: languageVersion,
				CompilerVersion: compilerVersion,
				CompilerOptions: compilerOptions,
				SrcMap:          info.SrcMap,
				SrcMapRuntime:   info.SrcMapRuntime,
				AbiDefinition:   abi,
				UserDoc:         userdoc,
				DeveloperDoc:    devdoc,
				Metadata:        info.Metadata,
			},
		}
	}
	return contracts, nil
}

// parseCombinedJSONV8는 solc --combined-output의 출력을 solidity v.0.8.0 이후의 규칙을 사용하여 구문 분석합니다.
func parseCombinedJSONV8(combinedJSON []byte, source string, languageVersion string, compilerVersion string, compilerOptions string) (map[string]*Contract, error) {
	var output solcOutputV8
	if err := json.Unmarshal(combinedJSON, &output); err != nil {
		return nil, err
	}
	// 컴파일이 성공하면 컨트랙트를 조립하여 반환합니다.
	contracts := make(map[string]*Contract)
	for name, info := range output.Contracts {
		contracts[name] = &Contract{
			Code:        "0x" + info.Bin,
			RuntimeCode: "0x" + info.BinRuntime,
			Hashes:      info.Hashes,
			Info: ContractInfo{
				Source:          source,
				Language:        "Solidity",
				LanguageVersion: languageVersion,
				CompilerVersion: compilerVersion,
				CompilerOptions: compilerOptions,
				SrcMap:          info.SrcMap,
				SrcMapRuntime:   info.SrcMapRuntime,
				AbiDefinition:   info.Abi,
				UserDoc:         info.Userdoc,
				DeveloperDoc:    info.Devdoc,
				Metadata:        info.Metadata,
			},
		}
	}
	return contracts, nil
}
