# Overview

## 소개

go-ethereum의 rlp 패키지는 Ethereum의 RLP 인코딩과 관련된 기능을 구현한 패키지입니다.

---

## 구조

```bash
📦rlp
 ┣ 📂internal
 ┃ ┗ 📂rlpstruct
 ┃ ┃ ┗ 📜rlpstruct.go
 ┣ 📂rlpgen
 ┃ ┣ 📂testdata
 ┃ ┃ ┣ 📜bigint.in.txt
 ┃ ┃ ┣ 📜bigint.out.txt
 ┃ ┃ ┣ 📜nil.in.txt
 ┃ ┃ ┣ 📜nil.out.txt
 ┃ ┃ ┣ 📜optional.in.txt
 ┃ ┃ ┣ 📜optional.out.txt
 ┃ ┃ ┣ 📜rawvalue.in.txt
 ┃ ┃ ┣ 📜rawvalue.out.txt
 ┃ ┃ ┣ 📜uint256.in.txt
 ┃ ┃ ┣ 📜uint256.out.txt
 ┃ ┃ ┣ 📜uints.in.txt
 ┃ ┃ ┗ 📜uints.out.txt
 ┃ ┣ 📜gen.go
 ┃ ┣ 📜gen_test.go
 ┃ ┣ 📜main.go
 ┃ ┗ 📜types.go
 ┣ 📜decode.go
 ┣ 📜decode_tail_test.go
 ┣ 📜decode_test.go
 ┣ 📜doc.go
 ┣ 📜encbuffer.go
 ┣ 📜encbuffer_example_test.go
 ┣ 📜encode.go
 ┣ 📜encode_test.go
 ┣ 📜encoder_example_test.go
 ┣ 📜iterator.go
 ┣ 📜iterator_test.go
 ┣ 📜raw.go
 ┣ 📜raw_test.go
 ┣ 📜safe.go
 ┣ 📜typecache.go
 ┗ 📜unsafe.go
```