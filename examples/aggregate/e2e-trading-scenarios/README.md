# Aggregate E2E Trading Scenarios

이 폴더는 Aggregate를 실제 HTS 관찰 화면에 가깝게 검증하기 위한 예제 묶음이다.
목표는 추천이나 판단이 아니라 provider 호출, 저장 데이터, snapshot, 이전 run을
명시적으로 골라 데이터 테이블을 반복 생성하는 것이다.

## 공통 실행 순서

```bash
mwosa validate aggregate examples/aggregate/e2e-trading-scenarios/<file>.aggregate.yaml -o json
mwosa update aggregate <name> --file examples/aggregate/e2e-trading-scenarios/<file>.aggregate.yaml -o json
mwosa inspect aggregate-plan <name> --view stages -o json
mwosa run aggregate <name> --alias <alias> -o table
mwosa history aggregate --name <name> -o table
mwosa inspect aggregate-run <alias> --view stages -o json
```

JSON, NDJSON, CSV smoke는 같은 run 명령에 `-o json`, `-o ndjson`, `-o csv`를
붙여 확인한다. credential 값은 명령이나 출력에 넣지 않는다.

## Taskfile 실행

저장소 루트에서 `Taskfile.yml`을 사용한다. 로컬 secret은 Git에 커밋하지 않는
`.env.local`, `.mwosa.local.env`, 기존 `mwosa` config 파일 중 하나를 사용한다.
dev 설치 바이너리는 실행 위치의 `.mwosa/config.json`을 우선 만들 수 있으므로,
이미 등록해 둔 사용자 config를 쓰려면 `MWOSA_CONFIG`를 지정한다. 이 환경에서는
`/Users/danghamo/.config/mwosa/config.json`에 KIS/KRX/Datago 일부 credential이
마스킹된 상태로 확인됐다.

```bash
task test:unit
task test:integration
task test:e2e:live
task install
task aggregate:smoke:fixture
MWOSA_CONFIG="$HOME/.config/mwosa/config.json" task aggregate:smoke:live
```

`aggregate:smoke:fixture`는 secret 없이 `scripts/aggregate/seed_priority_fixture.js`로
fixture 데이터를 넣고 1순위 후보 테이블을 출력한다. `aggregate:smoke:live`는 전체
종목 universe가 아니라 `aggregate_live_symbols` fixture 2개 종목만 KIS raw live
호출에 사용한다.

`test:e2e:live`는 설치된 `mwosa`가 아니라 현재 checkout에서 임시 CLI 바이너리를
빌드해 `go test -tags=e2e ./cli`로 실행한다. Taskfile의 dotenv 설정 때문에
`.env.local` 또는 `.mwosa.local.env`에 둔 `MWOSA_KRX_AUTH_KEY`를 사용할 수 있고,
테스트 내부에서는 `MWOSA_LIVE_E2E=1`과 `MWOSA_KRX_AUTH_KEY`가 없으면 skip한다.
직접 실행할 때는 다음처럼 명시적으로 opt-in 한다.

```bash
MWOSA_LIVE_E2E=1 go test -tags=e2e ./cli -run TestAggregateLiveE2EKRXMarketSnapshot -count=1
```

설치 CLI smoke는 설치된 사용 환경을 확인하는 용도이고, Go live E2E는 개발 중인
소스가 KRX Aggregate live 시나리오를 통과하는지 확인하는 용도다. KIS live smoke는
KIS 시스템 점검 또는 네트워크 실패가 있을 수 있어 현재 개발용 live E2E 범위에 넣지
않는다.
Go live E2E의 임시 CLI는 development build로 실행되므로 MongoDB database 이름은
hostname prefix가 붙은 개발용 이름으로 분리된다.

## 시나리오

| 파일 | provider/data | HTS형 화면 | live 호출 | 현재 기대 |
| --- | --- | --- | --- | --- |
| `kis-daily-candidate-board.aggregate.yaml` | KIS raw, `instruments`, `valuation_snapshots` | 장중/일봉 후보 관찰 | 있음 | KIS credential과 로컬 universe 필요 |
| `priority-candidate-board-fixture.aggregate.yaml` | local fixture, valuation fixture | 1순위 후보 테이블 출력 계약 | 없음 | credential 없이 출력 계약 검증 |
| `kis-daily-raw-live-smoke.aggregate.yaml` | KIS raw, local fixture universe | 2개 종목 KIS raw live 연결 확인 | 있음 | KIS credential과 네트워크 연결 필요 |
| `krx-market-snapshot.aggregate.yaml` | KRX raw | 시장별 상승률/거래대금 후보 | 있음 | KRX auth와 서비스 승인 필요 |
| `datago-etf-watch.aggregate.yaml` | Datago 수집 `daily_bars` | ETF NAV/괴리/거래대금 | 없음 | 로컬 Datago ETF daily 필요 |
| `opendart-catalyst-watch.aggregate.yaml` | 저장 OpenDART filings, `instruments` | 최근 공시 촉매 테이블 | 없음 | 선행 OpenDART sync 필요 |
| `reuse-previous-candidate-run.aggregate.yaml` | `aggregate_run`, `candidate_observations` | 직전 후보 대비 변화 | 없음 | 이전 `kis-daily-candidate-board` run 필요 |
| `raw-snapshot-replay.aggregate.yaml` | `provider_raw_snapshots` | 저장 raw payload 재조회 | 없음 | 저장 snapshot 필요 |

## 한계 분류

- 현재 구현 한계: `provider_raw` 공통 dispatch는 raw adapter를 구현한 provider만 실행한다. KIS와 KRX는 공통 raw adapter가 있고, Datago/OpenDART는 현재 Aggregate raw live stage로 직접 호출하지 않는다.
- provider 자체 한계: Datago 일별 데이터는 일반적으로 D-1 영업일 기준이다. KIS는 심볼 단위 일봉 조회라 전체 시장 fan-out은 universe 크기와 credential 상태에 영향을 받는다.
- 데이터/credential 환경 한계: KIS, KRX live smoke는 credential과 승인 서비스가 없으면 실패가 정상이다. 이 경우 snapshot 또는 local collection 시나리오만 실행한다.
- 추가 구현 필요: OpenDART filings/events와 Datago ETF daily를 Aggregate에서 provider role로 직접 호출하려면 `provider` stage의 role 범위를 `quote` 밖으로 넓혀야 한다.

## 검증 메모

2026-07-04에 로컬 `mwosa-mongodb` 컨테이너의 `mwosa` database를 사용해 smoke를
수행했다. MongoDB storage는 `init storage --mongodb-uri mongodb://127.0.0.1:27017
--mongodb-database mwosa`로 초기화했다.

| 시나리오 | validate | update | plan | run | output smoke | inspect/history | 메모 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| kis-daily-candidate-board | 성공 | 성공 | 성공 | 실패 | table 실패 출력 확인 | 성공 | alias `e2e-kis-20260704`; 기존 검증에서는 worktree local config에 KIS credential이 없어 `provider_raw` 단계에서 KIS provider 미등록으로 실패 |
| priority-candidate-board-fixture | 성공 | 성공 | 성공 | 성공 | table/json/ndjson/csv 성공 | 성공 | alias `e2e-priority-fixture-20260704`; fixture 2행으로 `#`, `코드`, `종목`, `등락%`, `시총(조)`, `거래대금(억)`, `거래량x20`, `52주고점%`, `종가위치%`, `RSI`, `ADX`, `ATR%`, `추세`, `메모/라벨` 출력 계약 검증 |
| kis-daily-raw-live-smoke | 성공 | 성공 | 성공 | 실패 | table 실패 출력 확인 | 성공 | alias `e2e-kis-raw-live-20260704c`; 전역 config로 KIS credential은 확인됐고 provider registry도 활성화됐으나 KIS OAuth token endpoint `openapi.koreainvestment.com:9443` 연결 거부로 `daily_raw` 실패 |
| krx-market-snapshot | 성공 | 성공 | 성공 | 성공 | table 성공, KOSPI/KOSDAQ 포함 | 성공 | alias `e2e-krx-market-live-20260704c`; `kospi_raw` 1행, `kosdaq_raw` 1행, `kospi_rows` 946행, `kosdaq_rows` 1822행, 최종 30행 확인 |
| datago-etf-watch | 성공 | 성공 | 성공 | 성공 | table/json/ndjson/csv 성공 | 성공 | alias `e2e-datago-etf-20260704`; 로컬 `daily_bars` 0행이라 결과 0행 |
| opendart-catalyst-watch | 성공 | 성공 | 성공 | 성공 | table 성공 | 성공 | alias `e2e-opendart-catalyst-20260704`; 로컬 `opendart_filings` 0행이라 결과 0행 |
| reuse-previous-candidate-run | 성공 | 성공 | 성공 | 실패 | table 실패 출력 확인 | 성공 | alias `e2e-reuse-previous-20260704`; 선행 `kis-daily-candidate-board` 성공 run이 없어 `aggregate_run` 단계 실패 |
| raw-snapshot-replay | 성공 | 성공 | 성공 | 성공 | table 성공 | 성공 | alias `e2e-raw-snapshot-20260704`; 저장 snapshot 0행이라 결과 0행 |

실패 run도 `history aggregate`와 `inspect aggregate-run --view stages`에서 stage별
실패 원인이 재현된다. 예를 들어 최신 KIS live smoke 실패 run은 `universe` 2행 성공
뒤 `daily_raw` stage에 OAuth token endpoint 연결 거부가 남는다.

출력 포맷 smoke는 `datago-etf-watch`로 확인했다. 결과가 0행인 환경에서는 table은
run summary table을 출력하고, json은 run detail JSON을 출력하며, ndjson/csv는 빈
stdout으로 정상 종료한다.
