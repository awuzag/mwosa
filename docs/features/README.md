# Features

`docs/features` 는 구현 예정 기능의 목적, 범위, 완료 기준을 정리하는 공간이다.
안정된 아키텍처 계약은 `docs/architectures` 에 두고, 이 디렉터리에는 실제 작업
단위로 옮길 수 있는 피처 문서를 둔다.

## 우선순위

| 우선순위 | 피처 | 상태 | 이유 |
| --- | --- | --- | --- |
| P0 | [Strategy Optimization](strategy-optimization/README.md) | Backtester / StrategySpec 이후 진행 | MDD 와 Calmar 중심으로 전략 파라미터를 자동 검증해 `mwosa` 를 실전형 전략 연구 도구로 확장한다. |
| P1 | Backtester | 별도 worktree 에서 설계 중 | YAML 전략 스펙, deterministic backtest engine, 실행 리포트가 `Strategy Optimization` 의 선행 조건이다. |
| P2 | [Indicators](indicators/README.md) | 계획 | 백테스터와 전략 스펙이 사용할 순수 지표 계산 API 를 제공한다. |
| P3 | [jq Screening Strategies](jq-screening-strategies/README.md) | 계획 / 진행 | 탐색형 스크리닝과 저장된 jq 전략 실행 흐름을 제공한다. |
| P4 | [ETF JSON Screening](etf-json-screening/README.md) | 계획 | ETF 데이터를 JSON/NDJSON/CSV 로 다루기 쉽게 만들고 도메인 스크리닝을 시작한다. |
| P5 | [DART Financial Valuation](dart-financial-valuation/README.md) | 제안 | DART 재무제표와 가격 데이터를 결합해 주식 스크리닝에 밸류에이션 가중치를 추가한다. |
| P6 | [Database Backend Selection](database-backend-selection/README.md) | 제안 | SQLite 기본값을 유지하면서 Docker PostgreSQL 같은 URL 기반 저장소를 선택할 수 있게 한다. |

## 진행 기준

- P0 는 중요도가 가장 높지만, 구현 순서는 백테스터와 전략 스펙 스키마 완료 이후다.
- 피처 문서는 구현을 시작하기 전에 선행 조건, 제외 범위, 완료 기준을 먼저 고정한다.
- CLI 결과는 `json`, `ndjson`, `csv`, `table` 의 stdout 계약을 명확히 둔다.
- diagnostics, progress, warning 은 stderr 로 분리한다.
- 수익 보장 표현은 쓰지 않고, 검증 가능한 가설과 리스크를 함께 기록한다.
