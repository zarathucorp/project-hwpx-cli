# HWPX CLI Research Notes

## What the Hancom articles imply

- [HWP format structure](https://tech.hancom.com/%ed%95%9c-%ea%b8%80-%eb%ac%b8%ec%84%9c-%ed%8c%8c%ec%9d%bc-%ed%98%95%ec%8b%9d-hwp-%ed%8f%ac%eb%a7%b7-%ea%b5%ac%ec%a1%b0-%ec%82%b4%ed%8e%b4%eb%b3%b4%ea%b8%b0/) shows why legacy `.hwp` is harder: it is a binary CFB container with record-oriented streams, so extraction requires header parsing and record-size handling.
- [HWPX format structure](https://tech.hancom.com/hwpxformat/) establishes the better CLI target for macOS/Linux: HWPX is a ZIP package of XML parts, with `mimetype`, `version.xml`, `settings.xml`, `Contents/`, `BinData/`, and `META-INF/`.
- The same article highlights that `Contents/content.hpf` is the package index. `metadata` holds title/creator data, `manifest` maps package items, and `spine` defines reading order.
- `Contents/header.xml` stores shared document properties and style mappings. `Contents/section*.xml` stores body content by section, and text is primarily carried in `<hp:t>` nodes under paragraph runs.
- [Python HWP parsing](https://tech.hancom.com/python-hwp-parsing-1/) is useful only as a contrast: binary HWP readers must decode tagged records, variable-length payloads, and stream positions carefully.
- [Python HWPX parsing](https://tech.hancom.com/python-hwpx-parsing-1/) suggests the practical extraction workflow for a CLI: open ZIP, extract namespaces, read `header.xml`, read `content.hpf`, resolve `spine`, then walk section XML and binary attachments.

## MVP scope

- Inspect package metadata, manifest, spine, sections, and binary payload paths
- Validate structural integrity of `.hwpx` files or unpacked directories
- Extract plain text in spine order
- Unpack `.hwpx` archives to editable directories
- Repack validated directories back into `.hwpx`

## Why Node.js for this repo

## Why Go for this repo

- macOS/Linux CLI 배포에 일반적으로 많이 쓰이고 단일 바이너리 배포가 쉽다
- 표준 라이브러리만으로 ZIP/XML 처리와 테스트 구성이 가능하다
- Windows 지원을 나중에 붙일 때도 크로스 컴파일 경로가 단순하다

## Deliberately out of scope for v0.1

- Legacy `.hwp` binary parsing
- Fine-grained XML editing commands
- Rendering fidelity checks against Hancom Office
- Windows packaging or installer support
