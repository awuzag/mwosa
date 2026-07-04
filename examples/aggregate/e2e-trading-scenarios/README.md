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

## 시나리오

| 파일 | provider/data | HTS형 화면 | live 호출 | 현재 기대 |
| --- | --- | --- | --- | --- |
| `kis-daily-candidate-board.aggregate.yaml` | KIS raw, `instruments`, `valuation_snapshots` | 장중/일봉 후보 관찰 | 있음 | KIS credential과 로컬 universe 필요 |
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
| kis-daily-candidate-board | 성공 | 성공 | 성공 | 실패 | table 실패 출력 확인 | 성공 | alias `e2e-kis-20260704`; `universe` 2768행 뒤 `provider_raw` 단계에서 KIS provider 미등록으로 실패 |
| krx-market-snapshot | 성공 | 성공 | 성공 | 실패 | table 실패 출력 확인 | 성공 | alias `e2e-krx-market-20260704`; KRX provider 미등록으로 `kospi_raw` 단계 실패 |
| datago-etf-watch | 성공 | 성공 | 성공 | 성공 | table/json/ndjson/csv 성공 | 성공 | alias `e2e-datago-etf-20260704`; 로컬 `daily_bars` 0행이라 결과 0행 |
| opendart-catalyst-watch | 성공 | 성공 | 성공 | 성공 | table 성공 | 성공 | alias `e2e-opendart-catalyst-20260704`; 로컬 `opendart_filings` 0행이라 결과 0행 |
| reuse-previous-candidate-run | 성공 | 성공 | 성공 | 실패 | table 실패 출력 확인 | 성공 | alias `e2e-reuse-previous-20260704`; 선행 `kis-daily-candidate-board` 성공 run이 없어 `aggregate_run` 단계 실패 |
| raw-snapshot-replay | 성공 | 성공 | 성공 | 성공 | table 성공 | 성공 | alias `e2e-raw-snapshot-20260704`; 저장 snapshot 0행이라 결과 0행 |

실패 run도 `history aggregate`와 `inspect aggregate-run --view stages`에서 stage별
실패 원인이 재현된다. 예를 들어 KIS 실패 run은 `daily_raw` stage에
`provider is not registered`가 남고, KRX 실패 run은 `kospi_raw` stage에 같은
provider 등록 한계가 남는다.

출력 포맷 smoke는 `datago-etf-watch`로 확인했다. 결과가 0행인 환경에서는 table은
run summary table을 출력하고, json은 run detail JSON을 출력하며, ndjson/csv는 빈
stdout으로 정상 종료한다.
