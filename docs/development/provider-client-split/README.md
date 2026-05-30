# Provider Client Repository Split

## 목적

이 문서는 `mwosa` 폴리레포 전환 1차 작업에서 provider client 를 독립
repository 로 분리하는 기준과 실행 순서를 정리한다.

이번 범위는 외부 API client 의 분리다. `mwosa` provider adapter, canonical
model, CLI, storage/service 계약은 `mwosa` 안에 남긴다.

## 현재 저장소 상태

- GitHub organization: `awuzag`
- `mwosa` remote: `https://github.com/awuzag/mwosa.git`
- 로컬 branch: `feat/polyrepo-provider-clients`
- `awuzag/kis`, `awuzag/krx`: `clients/kis`, `clients/krx` 히스토리를
  `git subtree split` 으로 보존해 만든 독립 public repository 다.
- `awuzag/opendart`: 기존 `ev3rlit/opendart` 를 이전한 public repository 이며
  기본 branch 는 `main` 이다.
- `awuzag/workspace`: public 작업 repository 이며 기본 branch 는 `main` 이다.

`mwosa` 의 GitHub repository 와 Go module path 는 모두 `github.com/awuzag/mwosa`
기준으로 맞춘다. KIS, KRX, OpenDART 는 외부 module dependency 로 사용하고,
Datago 계열 client module 은 이번 범위에서 계속 `mwosa` repository 안에 둔다.

## 분리 대상

| client | 현재 경로 | 대상 repository | 현재 module path | 1차 target module path | 상태 |
| --- | --- | --- | --- | --- | --- |
| KIS | `clients/kis` | `awuzag/kis` | `github.com/ev3rlit/mwosa/clients/kis` | `github.com/awuzag/kis` | 분리 완료, `v0.1.0` |
| KRX | `clients/krx` | `awuzag/krx` | `github.com/ev3rlit/mwosa/clients/krx` | `github.com/awuzag/krx` | 분리 완료, `v0.1.0` |
| OpenDART | 외부 repository | `awuzag/opendart` | `github.com/ev3rlit/opendart` | `github.com/awuzag/opendart` | 이전 완료, `v1.2.0` |
| ECOS | 범위 밖 | `awuzag/ecos` | 별도 개발 중 | 범위 밖 | 이번 작업에서 제외 |

## 책임 경계

독립 client repository 가 소유한다.

- endpoint path, request/response type, provider-native field name
- 인증 토큰 발급 또는 API key 주입에 필요한 provider-native HTTP 요청
- pagination, provider-native error parsing, remote error context
- OpenAPI 또는 공식 문서 기반 codegen input/output
- fake HTTP transport, `httptest`, module 단위 테스트

`mwosa` 안에 남긴다.

- `providers/<id>` adapter
- provider config loading, registry, role profile, routing
- canonical model 변환과 storage/service 계약
- CLI command, output format, stdout/stderr 계약
- provider raw snapshot 저장 계약
- cross-provider fallback, provenance, unsupported capability 처리

따라서 `mwosa` 는 분리된 client package 를 import 하되, service layer 에 client
type 을 직접 노출하지 않는다. service 는 계속 `providers/core` 의 role
interface 만 본다.

## KIS 분리 절차

KIS 는 `clients/kis` 전체를 `awuzag/kis` 로 분리했다.

```bash
tmp=$(mktemp -d /tmp/mwosa-kis-split.XXXXXX)
git clone --no-hardlinks /Users/danghamo/Documents/gituhb/mwosa "$tmp/mwosa"
cd "$tmp/mwosa"
git subtree split --prefix=clients/kis -b split-kis
git checkout split-kis
git remote add kis https://github.com/awuzag/kis.git
```

분리 후 첫 정리 commit 에서 수행할 일:

- `go.mod` module path 를 `github.com/awuzag/kis` 로 변경한다.
- `internal/codegen/kisopenapi` 가 생성하는 import path 를 새 module path 로
  바꾼다.
- 생성 파일은 generator 수정 후 `go generate ./...` 로 갱신한다.
- 문서의 `clients/kis` 경로 표현은 새 repository 기준으로 바꾼다.

검증:

```bash
GOWORK=off go test ./...
GOWORK=off go mod verify
git log --oneline --follow -- go.mod
```

리허설 결과:

- split commit: `e057cee19a1523537a2443cbea5a831c5292c1b9`
- split root file count: 198
- `GOWORK=off go test ./...`: 통과
- `GOWORK=off go mod verify`: 통과
- release tag: `v0.1.0`
- `git log --follow -- go.mod` 에서 `feat: add KIS client module` 과
  `Refactor KIS SDK public services` 계열 히스토리가 보존됨

## KRX 분리 절차

KRX 는 `clients/krx` 전체를 `awuzag/krx` 로 분리했다.

```bash
tmp=$(mktemp -d /tmp/mwosa-krx-split.XXXXXX)
git clone --no-hardlinks /Users/danghamo/Documents/gituhb/mwosa "$tmp/mwosa"
cd "$tmp/mwosa"
git subtree split --prefix=clients/krx -b split-krx
git checkout split-krx
git remote add krx https://github.com/awuzag/krx.git
```

분리 후 첫 정리 commit 에서 수행할 일:

- `go.mod` module path 를 `github.com/awuzag/krx` 로 변경한다.
- `go mod tidy` 로 단독 module checksum 을 정리한다.
- 문서의 `clients/krx` 경로 표현은 새 repository 기준으로 바꾼다.

검증:

```bash
GOWORK=off go mod tidy
GOWORK=off go test ./...
GOWORK=off go mod verify
git log --oneline --follow -- go.mod
```

리허설 결과:

- split commit: `ff28671eb79e2c60116e8d9585bd911cfa721b9b`
- split root file count: 107
- 최초 `GOWORK=off go test ./...` 는 `go.sum` 의 일부 `go.mod` checksum 누락으로
  실패함
- `GOWORK=off go mod tidy` 후 `go test ./...` 와 `go mod verify` 통과
- `git log --follow -- go.mod` 에서 `Add KRX OPEN API client module` 히스토리가
  보존됨
- release tag: `v0.1.0`

## OpenDART 이전 기준

OpenDART 는 기존 `ev3rlit/opendart` repository 를 `awuzag/opendart` 로 이전했다.
module path 도 `github.com/awuzag/opendart` 로 변경했고, `mwosa` 는
`providers/opendart` adapter 에서 `v1.2.0` 을 dependency 로 사용한다.

## mwosa 의존성 전환 순서

KIS/KRX client repository 를 push 한 뒤 `mwosa` 에서는 다음 순서로 dependency 를
전환한다.

1. `go.work` 에서 `./clients/kis`, `./clients/krx` 를 제거한다.
2. `go.mod` 의 client require 를 `github.com/awuzag/kis`,
   `github.com/awuzag/krx` 로 바꾼다.
3. `providers/kis`, `providers/krx` import alias 는 유지하되 import path 만 새
   client module 로 바꾼다.
4. `clients/kis`, `clients/krx` source directory 를 제거한다.
5. `go test ./...` 를 통과시킨다.

외부 KIS/KRX dependency 에 대한 local replace 는 남기지 않는다. Datago 계열
in-repo client module 은 계속 `replace => ./clients/...` 로 묶는다.

## 이슈 이동 기준

`awuzag/workspace` 로 옮길 이슈:

- 여러 repository 를 가로지르는 parent issue 또는 meta issue
- 일정, ADR, provider matrix, migration checklist, release coordination
- `mwosa`, `kis`, `krx`, `opendart` 중 둘 이상의 완료 상태를 묶어야 하는 작업

각 public repository 에 남길 이슈:

- 특정 repository 의 코드, 테스트, 문서 변경으로 닫을 수 있는 구현 이슈
- `kis` SDK API 추가, `krx` checksum/test 정리, `mwosa` adapter import 전환처럼
  소유 경계가 분명한 작업
- live e2e, release tag, module path 변경처럼 repository 단위 검증이 필요한 작업

현재 열린 `awuzag/mwosa` 이슈 중 `type: epic` 후보:

- `#1 시장 국면 해석을 위한 핵심 지표 수집 기반 구축`: workspace 이동 후보
- `#15 ECOS 클라이언트 및 범용 경제지표 조회 기반 구축`: workspace 이동 후보지만,
  ECOS 는 이번 작업 범위에서 제외한다.

이번 provider client split 의 직접 구현 이슈는 아직 열려 있지 않다. 권장 구조:

- workspace: `KIS/KRX/OpenDART client repository split meta`
- mwosa: `Use external KIS/KRX client modules in provider adapters`
- kis: `Initialize KIS SDK repository from mwosa clients/kis split`
- krx: `Initialize KRX SDK repository from mwosa clients/krx split`
- opendart: `Decide transfer and module path policy for OpenDART SDK`
