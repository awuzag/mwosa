---
id: REQ.A.01
title: Aggregate 실행 리소스 요구사항
type: requirements
status: draft
tags: [requirements, aggregate, cli, mongodb, pipeline]
source: local
created: 2026-07-12
updated: 2026-07-12
---

# Aggregate 실행 리소스 요구사항

## 기본 정보

- Requirements ID: `REQ.A.01`
- 프로젝트: `mwosa`
- 주요 사용자: 투자 데이터를 수집·가공·비교하는 리서처와 CLI 운영자
- 주요 이해관계자: provider 개발자, storage 개발자, 분석 기능 개발자
- 설계 범위: 저장 가능한 Aggregate 정의, 검증, 실행, 출력, 실행 기록과 재사용
- 제외 범위: 투자 판단 자동화, 임의 코드 실행, 작업 예약, 웹 편집기

이 문서에서 `Aggregate`는 DDD의 도메인 Aggregate가 아니다. 여러 데이터 원천과
가공 단계를 하나의 명세로 저장하고 실행하는 `mwosa` 리소스를 뜻한다.

## 근거 문서

- [Aggregate Architecture](../../architectures/aggregate/README.md)
- [Layer Architecture](../../architectures/layers/README.md)
- [Provider Architecture](../../architectures/provider/README.md)
- [Technology Stack](../../architectures/tech-stack/README.md)
- [Aggregate 실행 예시](../../../examples/aggregate/e2e-trading-scenarios/README.md)

근거 문서와 현재 구현은 요구사항 후보를 찾는 자료다. 이 요구사항 문서에서
확정하지 않은 구현 세부사항을 이미 결정된 요구사항으로 간주하지 않는다.

## 문제 정의

### 사용자가 겪는 문제

1. provider API 호출, 로컬 데이터 조회, jq 변환과 MongoDB aggregation이 Python
   스크립트, 셸 명령과 일회성 쿼리로 흩어져 있다.
2. 같은 분석을 다시 실행할 때 어떤 데이터 원천, 파라미터와 가공 규칙을
   사용했는지 확인하기 어렵다.
3. 실시간 API 호출과 저장 데이터 조회가 명시적으로 구분되지 않으면 결과의
   출처와 최신성을 판단하기 어렵다.
4. 중간 단계의 실패나 누락을 빈 결과로 처리하면 사용자는 정상 결과와 실패를
   구분할 수 없다.
5. 결과만 남고 실행 명세와 단계별 근거가 남지 않으면 이후 screen, backtest와
   report에서 안전하게 재사용하기 어렵다.

### 현재 방식의 한계

- 스크립트마다 입력 형식, 오류 처리와 출력 방식이 다르다.
- provider 인증, retry, pagination 정책이 호출 코드에 중복될 수 있다.
- 중간 결과의 이름과 수명주기가 일정하지 않아 단계 조합을 추적하기 어렵다.
- 실행 당시 명세, 파라미터, 단계별 행 수와 오류를 한 번에 조회할 수 없다.
- 사람이 읽는 표와 자동화용 JSON·NDJSON·CSV 출력을 별도로 만들어야 한다.

### 해결해야 하는 것

사용자는 데이터 처리 절차를 YAML로 정의하고 저장한 뒤 이름으로 검증·실행할
수 있어야 한다. `mwosa`는 입력 원천과 처리 단계를 명시적으로 실행하고, 명세
버전, 파라미터, 단계별 출처, 성공·실패와 결과를 조회 가능한 기록으로 남겨야
한다.

### 해결하지 않는 것

- 사용자를 대신해 매수, 매도, 보유 여부를 판단하지 않는다.
- Python이나 셸 코드를 저장하고 실행하는 범용 자동화 도구를 만들지 않는다.
- MongoDB aggregation을 감싸는 독자적인 대형 DSL을 만들지 않는다.
- 첫 버전에서 모든 provider API, screen, backtest와 report를 대체하지 않는다.
- Aggregate 내부에서 provider별 rate limit, retry와 pagination 정책을 다시
  정의하지 않는다.

## 목표

### 사용자 목표

- 반복 분석 절차를 이름 있는 리소스로 저장하고 필요할 때 다시 실행한다.
- 실행 전에 명세와 실제 실행 계획을 확인한다.
- 실시간 API, 저장된 snapshot, canonical dataset와 로컬 collection 가운데 사용할
  원천을 직접 선택한다.
- 결과뿐 아니라 실행 당시 파라미터, 단계별 출처와 실패 원인을 확인한다.
- 같은 결과를 사람용 표와 자동화용 형식으로 출력하고 다른 분석의 입력으로
  재사용한다.

### 운영 목표

- 실행마다 명세 버전과 `spec_hash`를 남겨 어떤 정의로 실행했는지 확인한다.
- 임시 작업 데이터는 실행별로 격리하고 정해진 기간 뒤 정리한다.
- 실패를 성공이나 빈 결과로 저장하지 않는다.
- 인증 정보와 민감한 provider 응답이 출력, 로그와 실행 기록에 노출되지 않게
  한다.

### 기술 목표

- 기존 CLI, service, provider, storage와 presentation 레이어 경계를 유지한다.
- provider 호출 정책은 provider adapter에 위임한다.
- MongoDB의 pipeline 문법을 가능한 한 유지하고 위험한 쓰기 단계는 차단한다.
- 기본 테스트는 실제 외부 API나 사용자의 로컬 환경에 의존하지 않게 한다.

## 사용자 유형

| Actor ID | 사용자 | 목표 | 주요 행동 |
| --- | --- | --- | --- |
| `ACTOR.A.01-01` | 리서처 | 반복 가능한 데이터 관찰 결과를 만든다. | YAML 작성, 검증, 실행, 결과 확인 |
| `ACTOR.A.01-02` | CLI 운영자 | 저장된 실행을 안전하게 운영하고 문제를 진단한다. | 목록·계획·이력·실패 조회, archive |
| `ACTOR.A.01-03` | 자동화 호출자 | 안정적인 기계 판독 출력을 다음 작업에 전달한다. | 파라미터 지정 실행, JSON·NDJSON·CSV 소비 |
| `ACTOR.A.01-04` | 기능 개발자 | 새 데이터 원천과 처리 단계를 기존 경계 안에 연결한다. | provider role, stage executor, repository 확장 |

## 기능 요구사항

### 리소스 정의와 수명주기

| Req ID | 요구사항 | 우선순위 | 수용 기준 요약 |
| --- | --- | --- | --- |
| `REQ.A.01.FR-001` | 사용자는 `kind: Aggregate` YAML로 이름, 설명, 파라미터, pipeline, 작업 공간과 출력을 정의할 수 있어야 한다. | Must | 필수 필드가 있는 명세를 읽고 canonical spec으로 변환한다. |
| `REQ.A.01.FR-002` | 사용자는 명세를 실행하지 않고 구조, 파라미터 참조, stage 연결과 금지된 동작을 검증할 수 있어야 한다. | Must | 유효한 명세와 잘못된 명세를 구분하고 오류 위치를 반환한다. |
| `REQ.A.01.FR-003` | 사용자는 같은 이름의 Aggregate를 새 버전으로 저장할 수 있어야 한다. | Must | YAML 원문, canonical spec, version과 `spec_hash`를 보존한다. |
| `REQ.A.01.FR-004` | 동일한 canonical spec을 다시 저장했을 때 불필요한 새 버전을 만들지 않아야 한다. | Should | 같은 `spec_hash`에 대한 중복 저장 정책이 일관된다. |
| `REQ.A.01.FR-005` | 사용자는 저장된 Aggregate 목록과 정의, 활성 버전, 버전 목록과 최근 실행 상태를 조회할 수 있어야 한다. | Must | 목록과 상세 조회가 기계 판독 형식을 지원한다. |
| `REQ.A.01.FR-006` | 사용자는 Aggregate를 archive할 수 있고 과거 버전과 실행 기록은 보존되어야 한다. | Must | archive 뒤 신규 실행 정책과 과거 조회 정책이 명확하다. |

### 실행 준비

| Req ID | 요구사항 | 우선순위 | 수용 기준 요약 |
| --- | --- | --- | --- |
| `REQ.A.01.FR-007` | 사용자는 저장된 이름과 선택적인 version 또는 `spec_hash`로 실행 대상을 지정할 수 있어야 한다. | Must | 지정 조건과 일치하지 않거나 archive된 대상은 명시적으로 거부한다. |
| `REQ.A.01.FR-008` | 명세는 타입이 있는 기본 파라미터를 정의하고, 사용자는 실행 시 `--param key=value`로 값을 덮어쓸 수 있어야 한다. | Must | 미정의 파라미터, 타입 오류와 해결되지 않은 참조를 실행 전에 거부한다. |
| `REQ.A.01.FR-009` | 사용자는 실행 전에 적용된 파라미터, stage 순서, 입력 참조, 임시 결과와 MongoDB pipeline을 확인할 수 있어야 한다. | Must | 계획 조회는 데이터 원천을 호출하거나 stage를 실행하지 않는다. |
| `REQ.A.01.FR-010` | 사용자는 실행에 사람이 식별할 수 있는 alias를 선택적으로 지정할 수 있어야 한다. | Should | 중복 alias를 거부하고 실행 ID와 함께 조회할 수 있다. |

### Pipeline 실행

| Req ID | 요구사항 | 우선순위 | 수용 기준 요약 |
| --- | --- | --- | --- |
| `REQ.A.01.FR-011` | 각 stage는 고유한 이름과 명시적인 type을 가져야 하며 선언 순서와 입력 의존성에 따라 실행되어야 한다. | Must | 중복 이름, 알 수 없는 type과 존재하지 않는 입력을 실행 전에 거부한다. |
| `REQ.A.01.FR-012` | pipeline은 canonical provider 호출, provider raw 호출, local collection, local dataset, snapshot과 이전 Aggregate 실행 결과를 입력으로 선택할 수 있어야 한다. | Must | 실시간 호출과 저장 데이터 조회가 stage type으로 구분된다. |
| `REQ.A.01.FR-013` | provider stage는 기존 provider router와 adapter를 사용해야 한다. | Must | Aggregate service가 provider client 구현 타입과 rate limit·retry 정책을 직접 소유하지 않는다. |
| `REQ.A.01.FR-014` | stage는 앞 stage의 각 행을 입력으로 반복 호출하는 fan-out을 선언할 수 있어야 한다. | Should | 반복 대상과 `${each.*}` 참조를 실행 전에 검증하고 실행 범위를 제한할 수 있다. |
| `REQ.A.01.FR-015` | 각 stage 결과는 같은 실행 안에서 다음 stage가 참조할 수 있는 이름 있는 중간 결과로 저장되어야 한다. | Must | stage alias가 실행별 물리 collection으로 격리되고 충돌하지 않는다. |
| `REQ.A.01.FR-016` | 사용자는 MongoDB aggregation pipeline으로 저장된 중간 결과를 결합·집계·정렬할 수 있어야 한다. | Must | stage alias와 허용된 local collection만 참조하고 `$out`, `$merge` 같은 쓰기 단계는 차단한다. |
| `REQ.A.01.FR-017` | 사용자는 jq로 stage의 행을 변환할 수 있어야 한다. | Must | jq 오류를 해당 stage의 실행 실패로 기록하며 숨은 셸 실행을 허용하지 않는다. |
| `REQ.A.01.FR-018` | 어느 stage든 실패하면 전체 실행을 실패로 처리하고 이미 확인된 단계 정보와 오류 원인을 기록해야 한다. | Must | 실패가 빈 성공으로 변환되지 않고 stage 이름, type과 원인 맥락이 남는다. |
| `REQ.A.01.FR-019` | 사용자는 실행 취소 신호를 전달할 수 있어야 하며 취소 뒤 추가 stage 실행을 중단해야 한다. | Must | `context.Context` 취소가 provider, storage와 jq 실행 경계까지 전달된다. |

### 출력, 기록과 재사용

| Req ID | 요구사항 | 우선순위 | 수용 기준 요약 |
| --- | --- | --- | --- |
| `REQ.A.01.FR-020` | 사용자는 출력 대상 stage와 기본 출력 형식, 컬럼, 제목, 정렬과 숫자 표시 형식을 명세에 정의할 수 있어야 한다. | Must | 출력 설정이 계산 stage를 대신하지 않고 presentation에만 적용된다. |
| `REQ.A.01.FR-021` | 실행 결과는 `table`, `json`, `ndjson`, `csv`로 출력할 수 있어야 한다. | Must | 같은 결과 행을 형식별 계약에 맞춰 출력하고 기계 판독 출력에 설명 문장을 섞지 않는다. |
| `REQ.A.01.FR-022` | 실행마다 Aggregate 이름, version, `spec_hash`, 적용 파라미터, 상태, stage 요약, 결과 요약과 오류를 저장해야 한다. | Must | 성공과 실패 실행을 모두 이력에서 조회할 수 있다. |
| `REQ.A.01.FR-023` | 각 stage 기록에는 입력 종류, provider·group·operation 또는 collection, 행 수와 변환 규칙의 식별값을 남겨야 한다. | Must | 사용자가 결과의 출처와 처리 단계를 역추적할 수 있다. |
| `REQ.A.01.FR-024` | 사용자는 실행 ID 또는 alias로 파라미터, stage, pipeline, 오류와 결과 행을 조회할 수 있어야 한다. | Must | 큰 결과는 기본 제한과 명시적인 조회 범위를 가진다. |
| `REQ.A.01.FR-025` | Aggregate 실행 결과는 이후 Aggregate의 입력으로 다시 사용할 수 있어야 한다. | Must | 이전 run을 명시적으로 선택하고 원본 실행 식별자를 provenance에 남긴다. |
| `REQ.A.01.FR-026` | screen, backtest와 report가 Aggregate 결과를 입력으로 사용할 수 있는 안정적인 조회 계약을 제공해야 한다. | Should | 다른 기능이 Aggregate storage 구현 세부사항에 직접 의존하지 않는다. |
| `REQ.A.01.FR-027` | 실행별 임시 collection은 TTL 정책으로 정리되어야 하며 영구 재사용 결과와 구분되어야 한다. | Must | 기본 TTL과 실행별 만료 시각이 있고 결과 이력은 임시 collection 삭제 뒤에도 조회할 수 있다. |

## 비기능 요구사항

| Req ID | 요구사항 | 기준 |
| --- | --- | --- |
| `REQ.A.01.NFR-001` | 추적 가능성 | 모든 실행은 정의 version, `spec_hash`, 적용 파라미터와 단계별 provenance를 가진다. |
| `REQ.A.01.NFR-002` | 실패 가시성 | invalid input, 인증 실패, provider 오류, 누락된 입력, MongoDB 오류와 jq 오류를 성공으로 위장하지 않는다. |
| `REQ.A.01.NFR-003` | 출력 안정성 | JSON·NDJSON·CSV stdout은 기계 판독 가능해야 하며 진행 상태와 진단은 stderr로 분리한다. |
| `REQ.A.01.NFR-004` | 보안 | credential, token, provider secret과 민감한 원문을 명세, 출력, 오류, 로그와 실행 기록에 저장하지 않는다. |
| `REQ.A.01.NFR-005` | 실행 격리 | 실행별 임시 collection 이름이 충돌하지 않고 한 실행의 정리가 다른 실행 결과를 삭제하지 않는다. |
| `REQ.A.01.NFR-006` | 자원 제한 | fan-out, 실행 시간, 결과 행 수, 조회 행 수와 임시 데이터 수명에 상한을 적용할 수 있어야 한다. |
| `REQ.A.01.NFR-007` | 레이어 독립성 | service는 provider role과 repository 계약에만 의존하며 provider client와 MongoDB 수명주기를 직접 관리하지 않는다. |
| `REQ.A.01.NFR-008` | 테스트 가능성 | 기본 테스트는 fake provider, fixture와 격리된 storage를 사용하고 실제 외부 API 호출은 opt-in E2E로 분리한다. |
| `REQ.A.01.NFR-009` | 명세 호환성 | `schema_version`을 검증하고 지원하지 않는 버전은 묵시적으로 해석하지 않는다. |
| `REQ.A.01.NFR-010` | 오류 맥락 | 오류에는 Aggregate, run, stage, provider 또는 collection 등 문제를 찾는 데 필요한 맥락을 포함한다. |

성능 목표의 구체적인 수치와 기본 자원 상한은 아직 정하지 않았다. 근거 없이
임의의 숫자를 요구사항으로 확정하지 않고 열린 질문에서 결정한다.

## 제약 조건

- Aggregate 실행 storage와 aggregation engine은 MongoDB를 사용한다.
- SQLite fallback은 제공하지 않는다.
- CLI는 기존 verb-first 명령 체계를 유지한다.
- 기계 판독용 stdout과 진단용 stderr를 분리한다.
- provider별 인증, pagination, rate limit과 retry는 provider adapter가 소유한다.
- MongoDB client의 수명주기는 `storage/mongodb` runtime이 소유한다.
- pipeline은 선언된 stage와 허용된 변환만 실행하며 임의 셸·Python 실행을
  허용하지 않는다.
- MongoDB aggregation의 쓰기·부수 효과 stage는 기본적으로 허용하지 않는다.
- 실제 provider API를 호출하는 검증은 명시적인 opt-in E2E로 분리한다.

## 1차 검증 시나리오

첫 번째 기능 검증은 투자 추천이 아니라 다음 데이터 처리 능력을 확인한다.

> KOSPI와 KOSDAQ 데이터를 명시한 provider 또는 로컬 원천에서 읽고, 사용자가
> 정의한 pipeline으로 합성·정렬한 뒤 한눈에 확인할 수 있는 후보 관찰 표를
> 출력한다.

이 시나리오는 다음을 한 번에 검증해야 한다.

1. YAML 검증과 version 저장
2. 타입이 있는 실행 파라미터 적용
3. 실시간 provider와 로컬 원천의 명시적 선택
4. stage 중간 결과 저장과 MongoDB aggregation
5. jq 변환
6. `table`과 `json` 출력
7. run, stage provenance와 결과 행 저장
8. history와 run 상세 조회
9. 저장된 이전 run 결과의 재사용
10. 실패 stage 기록과 secret 비노출

후보 선정 규칙, 지표와 라벨은 `mwosa`에 고정된 투자 판단이 아니라 사용자가
명세에 작성한 데이터 처리 규칙이어야 한다.

## 수용 기준

- [ ] 유효한 fixture 명세를 검증하고 저장하면 version과 `spec_hash`를 확인할 수 있다.
- [ ] 잘못된 stage type, 중복 stage 이름, 존재하지 않는 입력과 해결되지 않은
      파라미터 참조를 실행 전에 거부한다.
- [ ] 계획 조회는 실제 provider 호출이나 데이터 변경 없이 적용 파라미터와
      stage 계획을 반환한다.
- [ ] 격리된 MongoDB와 fixture로 local collection, aggregate, jq pipeline을
      실행하고 기대한 행을 출력한다.
- [ ] fake provider로 provider·provider raw stage의 라우팅과 오류 전파를 검증한다.
- [ ] 같은 실행 결과를 table, JSON, NDJSON와 CSV 계약으로 출력한다.
- [ ] 성공 실행을 history와 run 상세 조회에서 다시 확인한다.
- [ ] 실패 실행도 `failed` 상태, 실패 stage와 원인을 포함해 조회할 수 있다.
- [ ] 이전 run 결과를 입력으로 사용하는 후속 Aggregate가 원본 run 식별자를
      provenance에 남긴다.
- [ ] 금지된 MongoDB stage, 미정의 local collection과 중복 alias를 실행 전에
      거부한다.
- [ ] 실행 취소 시 후속 stage를 시작하지 않고 중단 사실과 정리 결과를 확인할
      수 있다.
- [ ] 결과와 오류에 테스트 credential 또는 secret이 포함되지 않는다.

## 열린 질문

| Question ID | 질문 | 결정이 필요한 이유 |
| --- | --- | --- |
| `REQ.A.01.Q-001` | Aggregate version을 별도 collection에 둘 것인가, 리소스에 embed할 것인가? | 보존 정책, 조회 복잡도와 document 크기에 영향을 준다. |
| `REQ.A.01.Q-002` | archive된 Aggregate의 기존 version 실행을 허용할 것인가? | archive의 사용자 의미를 먼저 고정해야 한다. |
| `REQ.A.01.Q-003` | 기본 fan-out, timeout, 결과 행 수와 조회 행 수 상한은 얼마인가? | 안전한 기본값과 실제 분석 규모를 함께 고려해야 한다. |
| `REQ.A.01.Q-004` | 실행 취소를 `failed`와 구분한 `cancelled` 상태로 저장할 것인가? | 사용자 진단과 재시도 판단에 영향을 준다. |
| `REQ.A.01.Q-005` | 실패 뒤 완료된 stage의 임시 결과를 언제까지 보존할 것인가? | 진단 가능성과 storage 비용 사이의 기준이 필요하다. |
| `REQ.A.01.Q-006` | local collection과 MongoDB stage 허용 목록을 어디서 관리할 것인가? | 명세 검증과 운영 보안의 기준 문서가 필요하다. |
| `REQ.A.01.Q-007` | 날짜, 숫자와 boolean 파라미터를 pipeline literal에 치환할 때 타입을 어떻게 보존할 것인가? | 문자열 치환으로 인한 쿼리 의미 변경을 막아야 한다. |
| `REQ.A.01.Q-008` | live 입력을 사용한 실행의 재현 가능성을 어느 수준까지 보장할 것인가? | 실행 기록, raw snapshot 저장과 재실행의 관계를 정해야 한다. |
| `REQ.A.01.Q-009` | 결과가 큰 실행의 저장 상한과 export 정책은 무엇인가? | `aggregate_run_items` 증가와 CLI 응답 크기를 제한해야 한다. |
| `REQ.A.01.Q-010` | screen, backtest와 report가 run 결과를 참조하는 공개 service 계약은 무엇인가? | storage collection 직접 의존을 막아야 한다. |

## 확인 필요

- 현재 구현된 명령과 stage가 이 요구사항을 얼마나 충족하는지는 별도 gap
  분석으로 확인한다.
- `RULE.md`와 Aggregate 아키텍처는 MongoDB를 기준으로 하지만
  `docs/architectures/tech-stack/README.md`에는 SQLite를 canonical source로 둔
  과거 결정이 남아 있다. 전체 저장소 기준 문서는 별도 작업에서 정리한다.
- 현재 코드에 존재한다는 이유만으로 요구사항의 우선순위나 수용 기준을
  완료로 표시하지 않는다.
- 열린 질문을 결정하기 전에는 해당 구현 선택을 장기 계약으로 고정하지 않는다.

## 후속 문서

- `UC.A.01`: Aggregate 정의·검증·실행·진단의 사용자 목표
- `BC.A.01`: 실행 리소스, 실행 기록과 stage 책임 경계
- `SD.A.0110`: Aggregate 정의와 실행 도메인 모델
- `SD.A.0120`: Aggregate version, run, stage와 result 영속성
- `SD.A.0130`: 검증, 계획, 실행과 이력 조회 서비스
- `SD.A.0140`: Aggregate CLI 명령과 출력·오류 계약
- `SCN.A.01`: 저장된 Aggregate 실행 처리 시퀀스
