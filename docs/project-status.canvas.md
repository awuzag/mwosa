---
type: canvas
version: 2
defaultStyle: boardmark.editorial.soft
viewport:
  x: 0
  y: 0
  zoom: 0.74
---

::: note {"id":"legend","at":{"x":-1120,"y":-760,"w":620},"style":{"bg":{"color":"#F8FAFC"},"stroke":{"color":"#64748B"}}}

# mwosa 프로젝트 관리판

기준: 2026-04-26

이 캔버스는 문서 의사결정 목록이 아니라 **프로젝트를 어떻게 진행할지 보는 관리판**이다. 현재 상태, 일정 흐름, 다음 작업, 주요 과제, 아직 정하지 않은 항목, 범위 밖 항목을 한눈에 보도록 구성한다.

색상 기준:

- 파랑: 프로젝트 일정관리
- 초록: 앞으로 해야 할 태스크
- 주황: 프로젝트 주요 과제
- 노랑: 아직 정하지 않은 항목
- 빨강: 범위 밖
- 회색: 기준 문서와 현재 상태

:::

::: note {"id":"current-state","at":{"x":-460,"y":-760,"w":620},"style":{"bg":{"color":"#F8FAFC"},"stroke":{"color":"#64748B"}}}

# 현재 상태

한 줄 요약: **설계 문서는 꽤 많이 정리됐고, 구현은 이제 시작 직전 단계**다.

| 항목 | 상태 |
| --- | --- |
| 제품 정체성 | 결정됨 |
| 장기 CLI 표면 | 결정됨 |
| 스윙 MVP 흐름 | 결정됨 |
| 지표/추세 아키텍처 | 초안 작성됨 |
| canonical schema v1 | 결정됨 |
| provider / layer 구조 | 설계 기준 정리됨 |
| Go 구현 | 초기 모듈 상태 |
| provider 실제 패키지 | 구현 전 |

현재 `go.mod` 와 Cobra 의존성은 있지만, `cmd/mwosa` 와 root package 구현 골격은 아직 만들어지지 않았다.

:::

::: note {"id":"source-map","at":{"x":200,"y":-760,"w":620},"style":{"bg":{"color":"#F8FAFC"},"stroke":{"color":"#64748B"}}}

# 기준 문서 맵

프로젝트 판단의 기준 문서:

- `README.md`: 제품 정체성, 장기 CLI 도움말 표면
- `docs/swing-cli-minimum-requirements.md`: 스윙 MVP 흐름
- `docs/canonical-schema.md`: canonical record, key, 저장/삭제 규칙
- `docs/architectures/tech-stack/README.md`: Go, Cobra, SQLite 결정
- `docs/architectures/layers/README.md`: 레이어 책임, 금지 방향, 디렉터리 예시
- `docs/architectures/provider/README.md`: provider role/router/adapter 구조
- `docs/architectures/provider/group-role-registration-patterns.md`: Datago 처럼 group 별 활용신청이 필요한 provider 구조 비교
- `docs/architectures/indicator/README.md`: 지표와 추세 계산 레이어
- `docs/architectures/packages/README.md`: 독립 core package 경계
- `docs/architectures/packages/indicators/README.md`: 투자 보조지표 계산 패키지 논의
- `docs/architectures/packages/backtest/README.md`: 독립 백테스트 시뮬레이션 엔진 경계
- `docs/architectures/packages/backtest/trading-engine-layers.md`: eolmasa 에서 가져온 백테스트 엔진 레이어 결정
- `docs/architectures/packages/backtest/references/strategy-asset-schema-references.md`: QuantConnect/KIS 전략 에셋 스키마 레퍼런스
- `docs/features/indicators/README.md`: 보조지표 패키지 구현 태스크
- `docs/features/backtester/README.md`: YAML 기반 백테스터 구조 분석과 첫 spike 범위
- `docs/development/README.md`: 개발 협업 기준, HTTP client 선택 기준
- `docs/providers/datago/README.md`: `datago` provider 와 첫 provider group 구현 계획

:::

::: note {"id":"timeline-overview","at":{"x":-1120,"y":-212,"w":620},"style":{"bg":{"color":"#E8F1FF"},"stroke":{"color":"#2563EB"}}}

# 프로젝트 일정관리

현재는 날짜 기반 일정표보다 **단계 기반 일정표**가 더 맞다. 아직 구현량이 적어서, 마감일보다 “무엇부터 정해야 다음 단계로 갈 수 있는지”가 중요하다.

```text
문서 기준 정리
  -> CLI 골격
  -> canonical type
  -> local storage
  -> provider adapter
  -> get daily
  -> 스윙 MVP 흐름
```

현재 위치는 `문서 기준 정리` 와 `CLI 골격` 사이로 보는 게 자연스럽다.

:::

::: note {"id":"phase-1","at":{"x":-460,"y":-212,"w":420},"style":{"bg":{"color":"#E8F1FF"},"stroke":{"color":"#2563EB"}}}

# 1단계: CLI 골격

목표: 실행 가능한 최소 CLI 만들기

- `cmd/mwosa/main.go`
- `app`
- `cli`
- `mwosa version`
- `mwosa help`
- 공통 output flag 골격

완료 기준:

- `go run ./cmd/mwosa version` 이 동작한다.
- 도움말 구조가 장기 CLI 표면과 어긋나지 않는다.

:::

::: note {"id":"phase-2","at":{"x":0,"y":-212,"w":420},"style":{"bg":{"color":"#E8F1FF"},"stroke":{"color":"#2563EB"}}}

# 2단계: Canonical 기반

목표: provider 와 무관한 데이터 계약을 코드로 옮기기

- `canonical`
- `instrument`
- `daily_bar`
- `quote_snapshot`
- canonical key helper
- 날짜/통화/market 정규화 기준

완료 기준:

- 문서의 canonical v1 예시를 Go type 으로 표현할 수 있다.
- key 생성 규칙을 테스트로 확인할 수 있다.

:::

::: note {"id":"phase-3","at":{"x":460,"y":-212,"w":420},"style":{"bg":{"color":"#E8F1FF"},"stroke":{"color":"#2563EB"}}}

# 3단계: 저장소

목표: 로컬 SQLite database 를 정본으로 쓰는 최소 저장 흐름 만들기

- SQLite schema 초안
- daily_bar upsert/query
- database path flag
- coverage/index 확장 준비

완료 기준:

- `daily_bar` 를 SQLite 에 쓰고 다시 읽을 수 있다.
- 단일 database 파일만으로 기본 조회가 가능하다.

:::

::: note {"id":"phase-4","at":{"x":920,"y":-212,"w":420},"style":{"bg":{"color":"#E8F1FF"},"stroke":{"color":"#2563EB"}}}

# 4단계: Provider 연결

목표: provider role/router/adapter 구조를 첫 데이터 흐름에 연결하기

- `providers/core`
- `dailybar.Fetcher`
- `instrument.Searcher`
- provider registry
- provider router
- `providers/datago` adapter 또는 stub
- `datago` provider group 등록

완료 기준:

- `datago` 의 `securitiesProductPrice` group 이 `daily_bar` 후보로 등록된다.
- unsupported capability 가 빈 성공으로 숨겨지지 않는다.

:::

::: note {"id":"phase-5","at":{"x":1380,"y":-212,"w":420},"style":{"bg":{"color":"#E8F1FF"},"stroke":{"color":"#2563EB"}}}

# 5단계: 첫 사용자 흐름

목표: 설계 문서가 실제 CLI 경험으로 이어지는 첫 조각 만들기

- `mwosa get daily <symbol>`
- `mwosa search instruments <query>`
- `mwosa inspect <symbol>` 초안
- table/json 출력
- provider 선택 이유 또는 오류 노출

완료 기준:

- 사람이 읽는 table 과 기계가 읽는 JSON 이 모두 동작한다.
- stdout/stderr 경계가 지켜진다.

:::

::: note {"id":"tasks-now","at":{"x":-1120,"y":371,"w":520},"style":{"bg":{"color":"#EAF7EE"},"stroke":{"color":"#16A34A"}}}

# 앞으로 해야 할 태스크: 지금

바로 시작 가능한 작업:

- `cmd/mwosa` 진입점 만들기
- Cobra root command 만들기
- `version`, `help`, `completion` 최소 연결
- `--output table|json|ndjson|csv` flag 형태 잡기
- `app` 에 조립 지점 만들기
- README 의 명령어 표면과 실제 도움말 구조를 먼저 맞추기

이 그룹은 구현 시작을 위한 “손에 잡히는 첫 작업”이다.

:::

::: note {"id":"tasks-next","at":{"x":-560,"y":371,"w":520},"style":{"bg":{"color":"#EAF7EE"},"stroke":{"color":"#16A34A"}}}

# 앞으로 해야 할 태스크: 다음

CLI 골격 다음에 이어질 작업:

- `canonical` type 작성
- canonical key 생성 helper 작성
- `daily_bar` SQLite upsert/query 작성
- provider error code 초안 작성
- table/json formatter 분리
- `get daily` service request/result type 작성

핵심은 provider API 호출보다 먼저 **mwosa 내부 언어**를 코드로 옮겨두는 것이다.

:::

::: note {"id":"tasks-later","at":{"x":0,"y":371,"w":520},"style":{"bg":{"color":"#EAF7EE"},"stroke":{"color":"#16A34A"}}}

# 앞으로 해야 할 태스크: 이후

첫 데이터 흐름이 잡힌 뒤:

- SQLite coverage/index 연결
- `ensure daily`
- `delete data`
- `reindex data`
- `calc relative-strength`
- `calc relative-volume`
- 스크리너 초안
- 포지션 사이즈와 R/R 계산

스윙 MVP는 `get daily` 와 canonical storage 가 먼저 있어야 자연스럽게 붙는다.

:::

::: note {"id":"management-metrics","at":{"x":560,"y":371,"w":520},"style":{"bg":{"color":"#EAF7EE"},"stroke":{"color":"#16A34A"}}}

# 관리할 체크 지표

프로젝트 관리판에서 계속 추적하면 좋은 신호:

- 실행 가능한 CLI command 수
- canonical type 구현 여부
- machine-readable 출력 지원 여부
- provider 없는 로컬 명령 동작 여부
- provider 필요한 명령의 에러 품질
- 문서에서 정한 내용과 실제 명령어/도움말의 불일치
- 범위 밖 기능이 다시 들어오려는 조짐

단순 진행률보다 **설계가 실제 동작으로 연결되는 정도**를 보는 게 좋다.

:::

::: note {"id":"challenge-provider","at":{"x":-1120,"y":992,"w":520},"style":{"bg":{"color":"#FFF1E6"},"stroke":{"color":"#EA580C"}}}

# 주요 과제: Provider 분리

왜 중요한가:

- `mwosa` repository root 는 `go.work` 기반 Go workspace 로 관리한다.
- provider client 는 `clients` 아래의 독립 Go module 로 둔다.
- CLI module 은 `providers/core` 와 provider 별 adapter 를 가진다.
- provider 등록 전 client module 단위 테스트를 먼저 통과시킨다.
- provider-native result 와 canonical record 를 섞으면 나중에 확장이 어려워진다.

위험:

- adapter 가 두꺼워질 수 있다.
- unsupported 기능이 빈 성공처럼 보일 수 있다.
- `datago` 의 group 별 일별 시세 성격을 quote 처럼 오해할 수 있다.

관리 포인트:

- role interface 를 작게 유지한다.
- adapter 는 storage 를 직접 쓰지 않는다.
- fallback 사유를 숨기지 않는다.

:::

::: note {"id":"challenge-storage","at":{"x":-560,"y":992,"w":520},"style":{"bg":{"color":"#FFF1E6"},"stroke":{"color":"#EA580C"}}}

# 주요 과제: SQLite 정본과 인덱스

왜 중요한가:

- canonical body 는 로컬 SQLite database 가 정본이다.
- 보조 index 는 정본 데이터를 대체하지 않는다.
- delete, reindex, ensure 는 이 경계를 전제로 한다.

위험:

- file storage 와 index 를 하나의 repository 처럼 섞을 수 있다.
- index 손상 시 복구 전략이 흐려질 수 있다.
- coverage 기준이 실제 파일과 어긋날 수 있다.

관리 포인트:

- 파일만으로 읽을 수 있는 최소 경로를 먼저 만든다.
- index 는 재구축 가능한 보조 데이터로 둔다.

:::

::: note {"id":"challenge-output","at":{"x":0,"y":992,"w":520},"style":{"bg":{"color":"#FFF1E6"},"stroke":{"color":"#EA580C"}}}

# 주요 과제: 출력 경계

왜 중요한가:

- `table` 은 사람이 읽는 출력이다.
- `json`, `ndjson`, `csv` 는 다른 도구와 AI 에이전트가 다시 처리하는 출력이다.
- machine-readable 출력에 progress/debug 가 섞이면 CLI 신뢰도가 떨어진다.

위험:

- stdout 에 진단 로그가 섞일 수 있다.
- `--explain` 과 `--verbose` 가 결과 모델 없이 문자열로만 붙을 수 있다.
- error 가 성공 형태의 빈 결과로 숨겨질 수 있다.

관리 포인트:

- 결과는 stdout, 진단은 stderr.
- 설명이 필요하면 result model 에 필드로 둔다.

:::

::: note {"id":"challenge-swing","at":{"x":560,"y":992,"w":520},"style":{"bg":{"color":"#FFF1E6"},"stroke":{"color":"#EA580C"}}}

# 주요 과제: 스윙 MVP 연결

왜 중요한가:

스윙 MVP는 기능 5개가 따로 노는 게 아니라 한 거래 흐름으로 이어져야 한다.

```text
스크리너
  -> 진입 체크리스트
  -> 포지션 트래커
  -> 매매일지
  -> 주간 통계
```

위험:

- 스크리너 결과가 포지션 생성으로 이어지지 않을 수 있다.
- R/R 계산과 포지션 기록이 분리될 수 있다.
- 일지와 주간 통계가 같은 거래 키를 공유하지 않을 수 있다.

관리 포인트:

- `position_id` 중심으로 흐름을 묶는다.
- 자동매매보다 기록과 복기를 우선한다.

:::

::: note {"id":"open-decisions-before-code","at":{"x":-1120,"y":1616,"w":520},"style":{"bg":{"color":"#FFF7D6"},"stroke":{"color":"#CA8A04"}}}

# 아직 정할 것: 첫 구현 전

첫 구현을 시작하기 전에 결정하면 좋은 항목:

- `version` 출력에 포함할 정보
- config 파일 포맷을 지금 정할지, env/path 만 먼저 둘지
- `--output` 기본값과 에러 출력 형식
- `get daily` 의 날짜 flag 형식
- provider 없는 상태에서 시장 데이터 명령의 에러 문구
- `datago` 를 외부 package 로 바로 만들지, local stub 으로 시작할지
- provider group 인증 상태를 어떤 명령에서 먼저 보여줄지

:::

::: note {"id":"open-decisions-later","at":{"x":-560,"y":1616,"w":520},"style":{"bg":{"color":"#FFF7D6"},"stroke":{"color":"#CA8A04"}}}

# 나중에 정해도 되는 것

첫 CLI 골격 작업을 막지는 않는 항목:

- logging library
- test assertion library
- table rendering library
- migration/versioning tool
- `handler/cli` 즉시 분리 여부
- `presentation` 과 `format` 분리 여부
- progress renderer 정책
- interactive confirmation 정책

이 항목들은 실제 명령어가 생긴 뒤 정해도 늦지 않다.

:::

::: note {"id":"scope-out","at":{"x":0,"y":1616,"w":520},"style":{"bg":{"color":"#FDECEC"},"stroke":{"color":"#DC2626"}}}

# 범위 밖

초기에는 넣지 않는 항목:

- 종목 추천
- 자동매매
- 브로커 계좌 연동
- 실시간 알림
- 고급 백테스트
- 멀티 계좌/멀티 전략 권한 모델
- 분봉, 틱 데이터
- 호가창 depth
- 주문/계좌 record type
- 지표 계산 결과 캐시

이 노트는 스코프가 커질 때 되돌아볼 방어선이다.

:::

::: note {"id":"first-slice","at":{"x":560,"y":1616,"w":520},"style":{"bg":{"color":"#F8FAFC"},"stroke":{"color":"#64748B"}}}

# 첫 구현 후보

가장 작은 실행 단위:

1. `cmd/mwosa/main.go`
2. `app/run.go`
3. `cli/root.go`
4. `mwosa version`
5. `mwosa completion <shell>`
6. `--output` flag 후보

그 다음 단위:

1. `canonical`
2. `daily_bar` type
3. canonical key helper
4. table/json formatter
5. `mwosa get daily <symbol>` stub

이렇게 나누면 설계 문서를 한 번에 모두 구현하려다 퍼지는 일을 줄일 수 있다.

:::
