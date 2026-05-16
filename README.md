# mwosa

> AI야, 그래서 뭐 사?

mwosa는 시장 데이터를 모으고, 정리하고, 전략을 개발하고 검증하기 위한 리서치 CLI입니다.

여러 금융 데이터 provider 를 하나의 verb-first 명령 체계로 다루고, 주식과 ETF 데이터를 로컬 저장소에 정규화해서 보관합니다. 그 데이터를 바탕으로 스크리닝, 전략 저장, 반복 평가 흐름을 키워 갑니다.

`mwosa` 는 "그래서 뭐 사?"라는 질문에 바로 종목을 찍어 주는 도구가 아닙니다. 자동매매봇도 아니고, 실제 주문을 실행하는 런타임도 아닙니다. 판단에 필요한 데이터와 실험 결과를 일관된 CLI와 구조화 출력으로 제공하는 도구입니다.

초기 주력 흐름은 한국 주식/ETF 스윙 리서치입니다. `mwosa` 는 추세, 거래량, 변동성, 모멘텀, 가격 위치 같은 조건을 계산하고, 그 계산에 사용한 provider 와 원천 데이터를 함께 드러내는 방향으로 설계합니다. 스윙 흐름을 먼저 잘 만들되, 같은 구조로 다양한 시장 데이터와 지표 조회까지 확장합니다.

## 정체성

`mwosa` 는 provider 기반 데이터 CLI에서 출발했습니다. 앞으로의 중심은 시장 데이터 기반 전략 리서치 CLI입니다.

역할은 아래와 같습니다.

- provider 를 추가하고 관리한다.
- 종목 정보, ETF 정보, 일봉과 분봉 같은 시장 데이터를 수집한다.
- 수집한 데이터를 로컬 저장소에 정규화해서 보관한다.
- 여러 종목과 ETF를 같은 기준으로 스크리닝한다.
- 전략을 등록하고 저장한다.
- 저장된 전략을 여러 종목과 기간에 대해 반복 평가한다.
- CAGR, MDD, Sharpe, 거래 수, 승률, 손익비 같은 지표로 결과를 비교한다.
- 좋은 후보 전략, 종목, 파라미터를 찾아낸다.
- 검증 결과를 사람이 다시 볼 수 있는 리포트와 구조화 출력으로 남긴다.

현재 구현은 provider 관리, 일봉 수집/조회, 로컬 SQLite 저장소, jq 기반 ETF 스크리닝, 저장된 스크리닝 전략 실행을 중심으로 한다. 분봉, 전략 registry 의 확장, 배치 백테스트, 성과 지표 비교, 리포트 출력은 이 흐름 위에 붙일 역할이다.

KIS 같은 브로커 API 는 실시간 실행과 계좌 연동에는 중요하지만, 긴 기간과 다종목을 반복하는 백테스트에는 rate limit 이 병목이 된다. 그래서 `mwosa` 는 provider API 를 매번 직접 두드리는 방식보다, 로컬에 정규화된 데이터를 쌓고 그 위에서 반복 실험하는 구조를 우선한다.

## 리서치 흐름

```text
provider
  -> instrument metadata
  -> daily bars / minute bars
  -> local normalized storage
  -> screen
  -> strategy registry
  -> batch backtest / evaluation
  -> report / structured output
```

현재는 `daily bars -> local storage -> screen -> saved screening run` 흐름이 먼저 구현되어 있다. 나머지는 같은 흐름을 넓히는 방향이다.

## 목표

- 여러 provider API를 하나의 CLI로 사용
- 주식, ETF, 지수, 거시 지표, 뉴스 데이터 조회
- 시장 데이터를 로컬 SQLite 정본으로 정규화
- 스윙 트레이딩에 필요한 추세와 지표 계산 방향 확장
- 여러 후보군을 대상으로 스크리닝과 반복 실험 수행
- 전략 후보를 저장하고 성과를 비교하는 흐름 제공
- AI 에이전트가 읽기 좋은 JSON 출력 지원
- 리서치, 비교, 스크리닝, 계산을 하나의 명령 체계로 통합

## 최소 사용 조건

`mwosa` 로 실제 시장 데이터를 조회하려면 최소 1개 이상의 provider 가 활성화되어 있어야 한다.

`help`, `version`, `config`, `list providers` 같은 로컬 명령은 provider 없이도 실행할 수 있다. 하지만 `inspect`, `get`, `ensure`, `sync`, `screen`, `compare`, `calc` 처럼 시장 데이터가 필요한 명령은 해당 capability 를 제공하는 provider 가 필요하다.

## 설치

Go 가 설치되어 있다면 태그 기준으로 설치한다.

```bash
go install github.com/ev3rlit/mwosa/cmd/mwosa@v0.1.0
```

Go 없이 사용하는 사용자는 GitHub Release 에서 자기 OS/CPU 에 맞는 archive 를
받아 `mwosa` 실행 파일을 `PATH` 에 둔다. 릴리스 자동화는 macOS, Linux,
Windows 용 바이너리와 `checksums.txt` 를 생성한다.

## 현재 CLI 구조

`mwosa` 는 verb-first 명령 체계를 가진다.

```text
mwosa <verb> [resource] [target] [flags]
```

예를 들어 `inspect AAPL`, `get quote 005930`, `search instruments 반도체`, `compare symbols AAPL MSFT` 처럼 먼저 동사를 쓰고, 그 다음에 대상을 적는다.

현재 소스 기준으로 확인한 주요 명령은 아래와 같다.

```bash
mwosa list providers -o json
mwosa doctor provider datago -o json
mwosa backfill daily --provider datago --security-type etf --from 2024-05-02 --to 2024-05-10 --workers 4 -o json
mwosa get daily 069500 --security-type etf --from 2024-05-02 --to 2024-05-10 -o json
mwosa screen etf --jq '.[:10]' -o json
mwosa create strategy latest-liquidity-leaders --engine jq --input etf_daily_metrics --jq '.[:10]' -o json
mwosa screen strategy latest-liquidity-leaders --alias 2024-05-10-liquidity-leaders -o json
mwosa history screen -o table
mwosa inspect screen 2024-05-10-liquidity-leaders -o json
```

현재 `strategy` 명령은 스크리닝 전략 저장과 실행을 다룬다. 주문 실행 전략이나 실계좌 운용 전략을 뜻하지 않는다.

## 장기 명령 지도

아래 목록은 구현 완료 목록이 아니다. 장기적으로 `mwosa --help` 에서 제공할 명령어 표면과 목표 동작을 정리한 지도다.

### 명령어 압축 원칙

`mwosa` 는 명령어 수를 무작정 늘리기보다, 적은 수의 verb 를 넓은 resource 에 적용하는 방식을 우선한다.

| verb       | 역할                                                   |
| ---------- | ------------------------------------------------------ |
| `inspect`  | 선택한 대상을 한눈에 볼 수 있게 상세 정보를 출력한다.  |
| `list`     | 저장되어 있거나 지원 가능한 대상의 목록을 출력한다.    |
| `search`   | 키워드로 종목, 뉴스, 공시, 지표를 찾는다.              |
| `get`      | provider 또는 로컬 저장소에서 원천 데이터를 조회한다.  |
| `ensure`   | 필요한 데이터가 없으면 가져와서 저장한다.              |
| `sync`     | provider 기준으로 메타데이터나 묶음 데이터를 갱신한다. |
| `calc`     | 가격과 재무 데이터를 바탕으로 계산값을 만든다.         |
| `screen`   | 조건에 맞는 후보를 걸러낸다.                           |
| `compare`  | 여러 대상을 같은 기준으로 비교한다.                    |
| `record`   | 거래, 가설, 메모 같은 사용자 기록을 남긴다.            |
| `create`   | 포트폴리오, universe, 전략, 알림 같은 리소스를 만든다. |
| `update`   | 기존 리소스의 속성을 수정한다.                         |
| `delete`   | 저장된 리소스나 데이터를 삭제한다.                     |
| `validate` | 설정, 전략, 출력, 데이터 정합성을 검사한다.            |

가장 중요한 축은 `inspect` 다. `inspect` 는 "이 대상이 지금 어떤 상태인지 자세히 보여줘" 라는 의미로 사용한다.

```text
mwosa inspect <symbol...>
mwosa inspect --stdin
mwosa inspect instrument <symbol...>
mwosa inspect instrument --stdin
mwosa inspect portfolio <name>
mwosa inspect provider <name>
mwosa inspect tool <name>
```

`mwosa inspect <symbol...>` 은 짧은 기본형이고, `mwosa inspect instrument <symbol...>` 은 명시형이다. 두 명령은 같은 결과를 반환한다.

여러 종목을 한 번에 볼 때는 공백으로 symbol 을 나열한다. 다른 명령의 출력이나 AI 에이전트가 만든 symbol 목록을 넘길 때는 `--stdin` 을 사용한다.

### 공통 옵션

| 옵션                       | 설명                                                                                |
| -------------------------- | ----------------------------------------------------------------------------------- |
| `-o, --output <format>`    | 출력 형식을 선택한다. 지원 형식은 `table`, `json`, `ndjson`, `csv` 다. 기본값은 `table` 이다. |
| `--provider <name>`        | 특정 provider 를 명시해서 조회한다. 호환 provider 는 [provider compatibility guide](docs/providers/README.md)를 참고한다. |
| `--prefer-provider <name>` | 가능한 경우 특정 provider 를 우선 사용한다.                                         |
| `--market <market>`        | 조회 시장을 제한한다. 예: `krx`, `nasdaq`, `nyse`                                   |
| `--currency <code>`        | 출력 통화 기준을 지정한다. 예: `KRW`, `USD`                                         |
| `--from <date>`            | 조회 시작일을 지정한다.                                                             |
| `--to <date>`              | 조회 종료일을 지정한다.                                                             |
| `--as-of <date>`           | 특정 기준일의 스냅샷을 조회한다.                                                    |
| `--cache-only`             | 외부 API 호출 없이 로컬 캐시만 사용한다.                                            |
| `--refresh`                | 로컬 캐시가 있어도 provider 에서 새로 가져온다.                                     |
| `--explain`                | 계산값, 필터 통과 이유, provider 선택 이유를 함께 출력한다.                         |
| `--quiet`                  | 사람이 읽는 설명을 줄이고 결과만 출력한다.                                          |
| `--verbose`                | provider 호출, 캐시 적중 여부, 계산 과정 등 진단 정보를 더 보여준다.                |

### 기본 명령

| 명령어                     | 설명                                                           |
| -------------------------- | -------------------------------------------------------------- |
| `mwosa help`               | 전체 도움말을 출력한다.                                        |
| `mwosa help <command>`     | 특정 명령어의 사용법과 옵션을 출력한다.                        |
| `mwosa version`            | CLI 버전, schema 버전, 빌드 정보를 출력한다.                   |
| `mwosa completion <shell>` | `zsh`, `bash`, `fish` 같은 shell 자동완성 스크립트를 출력한다. |

### Inspect

| 명령어                                 | 설명                                                                              |
| -------------------------------------- | --------------------------------------------------------------------------------- |
| `mwosa inspect <symbol...>`            | 하나 이상의 종목, ETF, 지수 같은 투자 대상의 현재 상세 정보를 한 화면에 요약한다. |
| `mwosa inspect --stdin`                | 표준 입력으로 받은 symbol 목록을 inspect 한다.                                    |
| `mwosa inspect instrument <symbol...>` | `inspect <symbol...>` 과 같은 명시형 명령이다.                                    |
| `mwosa inspect instrument --stdin`     | 표준 입력으로 받은 symbol 목록을 instrument 로 명시해 inspect 한다.               |
| `mwosa inspect provider <name>`        | provider 가 지원하는 시장, 자산군, 데이터 종류, 제한사항을 자세히 출력한다.       |
| `mwosa inspect market <market>`        | 시장의 거래 시간, 통화, 지원 provider, 주요 지수를 출력한다.                      |
| `mwosa inspect exchange <exchange>`    | 거래소의 기본 정보, 시간대, 휴장일, 지원 데이터 범위를 출력한다.                  |
| `mwosa inspect universe <name>`        | universe 에 포함된 종목과 주요 통계를 출력한다.                                   |
| `mwosa inspect portfolio <name>`       | 포트폴리오의 보유 종목, 비중, 평가금액, 리스크를 출력한다.                        |
| `mwosa inspect strategy <name>`        | 전략의 조건, 필요한 데이터, 최근 실행 결과를 출력한다.                            |
| `mwosa inspect trade <trade-id>`       | 특정 거래의 진입, 청산, 손익, 메모를 출력한다.                                    |
| `mwosa inspect alert <alert-id>`       | 알림 조건, 상태, 마지막 평가 결과를 출력한다.                                     |
| `mwosa inspect tool <name>`            | agent tool 의 입력 schema, 출력 schema, 예시를 출력한다.                          |
| `mwosa inspect schema <resource>`      | 특정 리소스의 JSON schema 와 필드 의미를 출력한다.                                |
| `mwosa inspect coverage <symbol>`      | 로컬에 저장된 심볼별 일별 데이터 범위를 출력한다. 예: `mwosa inspect coverage 069500 --security-type etf` |
| `mwosa inspect storage`                | 로컬 일별 데이터의 record type, 심볼 수, row 수, 기간 범위를 요약한다. 예: `mwosa inspect storage --security-type etf` |
| `mwosa inspect config`                 | 현재 적용된 설정과 설정 파일 경로를 출력한다.                                     |
| `mwosa inspect auth`                   | provider 별 인증 상태를 요약한다.                                                 |

### List

`list` 는 "어떤 것들이 있는지"를 확인할 때 사용한다. 상세 정보가 필요하면 같은 대상을 `inspect` 로 본다.

| 명령어                             | 설명                                                                            |
| ---------------------------------- | ------------------------------------------------------------------------------- |
| `mwosa list providers`             | 사용 가능한 provider 목록과 활성화 상태를 출력한다.                             |
| `mwosa list provider-capabilities` | `quote`, `candles`, `news`, `fundamentals` 등 기능별 지원 provider 를 출력한다. |
| `mwosa list markets`               | 지원하는 시장 목록을 출력한다.                                                  |
| `mwosa list exchanges`             | 지원하는 거래소 목록을 출력한다.                                                |
| `mwosa list asset-types`           | `stock`, `etf`, `etn`, `fund`, `crypto`, `index` 같은 자산 유형을 출력한다.     |
| `mwosa list universes`             | 저장된 종목 universe 목록을 출력한다.                                           |
| `mwosa list portfolios`            | 저장된 포트폴리오 목록을 출력한다.                                              |
| `mwosa list strategies`            | 저장된 전략 목록을 출력한다.                                                    |
| `mwosa list trades`                | 저장된 거래 목록을 조건별로 출력한다.                                           |
| `mwosa list alerts`                | 등록된 알림 목록을 출력한다.                                                    |
| `mwosa list tools`                 | agent tool 로 노출 가능한 명령어 목록을 출력한다.                               |

### Search

| 명령어                             | 설명                                                      |
| ---------------------------------- | --------------------------------------------------------- |
| `mwosa search instruments <query>` | 종목명, 티커, ISIN, 키워드로 투자 대상을 검색한다.        |
| `mwosa search news <query>`        | 키워드로 시장 뉴스와 종목 뉴스를 검색한다.                |
| `mwosa search filings <query>`     | 공시, 사업보고서, 10-K, 10-Q 같은 filing 자료를 검색한다. |
| `mwosa search indicators <query>`  | 금리, 환율, 물가, 고용 같은 거시 지표 이름을 검색한다.    |
| `mwosa resolve symbol <query>`     | 사용자 입력을 canonical symbol 로 해석한다.               |

### Get

`get` 은 현재가, OHLCV, 재무제표, 뉴스처럼 원천 데이터에 가까운 값을 조회한다. 조회 결과가 없을 때 자동 수집까지 보장해야 한다면 `ensure` 를 사용한다.

| 명령어                            | 설명                                                      |
| --------------------------------- | --------------------------------------------------------- |
| `mwosa get quote <symbol>`        | 현재가 또는 최신 quote snapshot 을 조회한다.              |
| `mwosa get candles <symbol>`      | 일봉, 주봉, 월봉, 분봉 같은 OHLCV 시계열을 조회한다.      |
| `mwosa get daily <symbol>`        | 저장된 일별 OHLCV 데이터를 조회한다. 데이터가 없으면 `ensure`, `sync`, `backfill` 안내를 출력한다. |
| `mwosa get orderbook <symbol>`    | 호가 잔량과 매수/매도 호가를 조회한다.                    |
| `mwosa get trades <symbol>`       | 최근 체결 내역을 조회한다.                                |
| `mwosa get fundamentals <symbol>` | 시가총액, PER, PBR, EPS 등 기본 지표를 조회한다.          |
| `mwosa get financials <symbol>`   | 손익계산서, 재무상태표, 현금흐름표를 조회한다.            |
| `mwosa get earnings <symbol>`     | 실적 발표일, 예상치, 실제 발표값을 조회한다.              |
| `mwosa get dividends <symbol>`    | 배당 내역과 배당 수익률을 조회한다.                       |
| `mwosa get splits <symbol>`       | 액면분할, 병합 등 split 이벤트를 조회한다.                |
| `mwosa get filings <symbol>`      | 공시, 사업보고서, 10-K, 10-Q 같은 filing 자료를 조회한다. |
| `mwosa get news <symbol>`         | 종목 관련 뉴스를 조회한다.                                |
| `mwosa get macro <indicator>`     | 금리, 환율, 물가, 고용 같은 거시 지표를 조회한다.         |
| `mwosa get fx <pair>`             | 환율 데이터를 조회한다. 예: `USD-KRW`                     |
| `mwosa get index <symbol>`        | 지수 데이터를 조회한다. 예: `KOSPI`, `SPY`, `QQQ`         |

### 데이터 확보와 저장소

| 명령어                            | 설명                                                            |
| --------------------------------- | --------------------------------------------------------------- |
| `mwosa ensure quote <symbol>`     | 최신 quote 가 없으면 provider 에서 받아 저장한다.               |
| `mwosa ensure daily <symbol>`     | 요청 범위의 일별 데이터가 없으면 부족한 날짜를 provider 에서 받아 저장한다. |
| `mwosa ensure candles <symbol>`   | 요청 범위의 candle 데이터가 없으면 부족한 구간만 받아 저장한다. |
| `mwosa sync daily`                | 특정 날짜의 시장/자산군 일별 데이터를 provider 기준으로 일괄 갱신한다. |
| `mwosa sync instrument <symbol>`  | 종목 메타데이터를 provider 기준으로 갱신한다.                   |
| `mwosa sync universe <name>`      | universe 에 포함된 종목의 주요 데이터를 일괄 갱신한다.          |
| `mwosa backfill daily`            | 과거 일별 데이터를 지정한 기간만큼 시장/자산군 단위로 채운다.   |
| `mwosa backfill candles <symbol>` | 과거 시계열 데이터를 지정한 범위만큼 채운다.                    |
| `mwosa verify data <symbol>`      | 저장된 데이터의 중복, 누락, 날짜 정합성을 검사한다.             |
| `mwosa reindex data`              | SQLite 정본 데이터를 기준으로 검색/coverage 보조 구조를 재구축한다. |
| `mwosa prune cache`               | 오래된 캐시나 임시 데이터를 정리한다.                           |
| `mwosa delete data <selector>`    | 특정 종목, 기간, record type 의 로컬 데이터를 삭제한다.         |
| `mwosa export data <selector>`    | 데이터를 `json`, `ndjson`, `csv` 등으로 내보낸다.               |
| `mwosa import data <path>`        | 외부 파일을 canonical 데이터로 가져온다.                        |

`backfill daily` 는 provider 가 기간 조회를 지원하면 날짜를 쪼개지 않고 기간
batch 를 수집한다. Datago ETP 처럼 기간 조회가 page 로 나뉘는 provider 에서는
`--workers` 가 page fetch 병렬도다.

```sh
mwosa backfill daily --security-type etf --from 20240502 --to 20260502 --workers 4
```

### 설정과 인증

설정과 인증도 같은 원칙을 따른다. 현재 상태는 `inspect`, 변경은 `set` 또는 `login/logout`, 검사는 `validate` 로 처리한다.

| 명령어                           | 설명                                                     |
| -------------------------------- | -------------------------------------------------------- |
| `mwosa init config`              | 기본 설정 파일과 데이터 디렉터리 구조를 만든다.          |
| `mwosa get config <key>`         | 특정 설정값을 조회한다.                                  |
| `mwosa set config <key> <value>` | 특정 설정값을 변경한다.                                  |
| `mwosa validate config`          | 설정 파일, 필수 환경변수, provider 인증 정보를 검사한다. |
| `mwosa login provider <name>`    | provider 인증 정보를 등록한다.                           |
| `mwosa logout provider <name>`   | provider 인증 정보를 제거한다.                           |

### Provider 제어

| 명령어                          | 설명                                               |
| ------------------------------- | -------------------------------------------------- |
| `mwosa doctor provider <name>`  | provider 설정, 인증, client 구성 가능성을 진단한다. |
| `mwosa enable provider <name>`  | provider 를 기본 후보에 포함한다.                  |
| `mwosa disable provider <name>` | provider 를 기본 후보에서 제외한다.                |
| `mwosa prefer provider <name>`  | provider 우선순위를 조정한다.                      |

### Calc

| 명령어                                  | 설명                                                     |
| --------------------------------------- | -------------------------------------------------------- |
| `mwosa calc returns <symbol>`           | 기간 수익률, 누적 수익률, 연환산 수익률을 계산한다.      |
| `mwosa calc indicator <symbol> <name>`  | RSI, MACD, SMA, EMA, ATR 같은 기술적 지표를 계산한다.    |
| `mwosa calc volatility <symbol>`        | 변동성, 연환산 변동성, 구간별 변동성을 계산한다.         |
| `mwosa calc drawdown <symbol>`          | 최대 낙폭과 낙폭 구간을 계산한다.                        |
| `mwosa calc correlation <symbols...>`   | 여러 종목 사이의 상관관계를 계산한다.                    |
| `mwosa calc beta <symbol>`              | 기준 지수 대비 beta 를 계산한다.                         |
| `mwosa calc risk <symbol>`              | VaR, downside risk, 손절 기준 리스크를 계산한다.         |
| `mwosa calc valuation <symbol>`         | 밸류에이션 지표와 과거 구간 대비 위치를 계산한다.        |
| `mwosa calc relative-strength <symbol>` | 기준 지수나 universe 대비 상대강도를 계산한다.           |
| `mwosa calc relative-volume <symbol>`   | 평균 거래량 대비 현재 거래량 수준을 계산한다.            |
| `mwosa calc position-size`              | 진입가, 손절가, 허용 손실 기준으로 수량을 계산한다.      |
| `mwosa calc rr`                         | 진입가, 손절가, 목표가 기준으로 reward/risk 를 계산한다. |

### Screen

| 명령어                          | 설명                                                    |
| ------------------------------- | ------------------------------------------------------- |
| `mwosa screen stocks`           | 조건에 맞는 주식 후보를 찾는다.                         |
| `mwosa screen etfs`             | 조건에 맞는 ETF 후보를 찾는다.                          |
| `mwosa screen universe <name>`  | 저장된 universe 안에서 조건에 맞는 후보를 찾는다.       |
| `mwosa screen momentum`         | 모멘텀, 상대강도, 추세 조건으로 후보를 찾는다.          |
| `mwosa screen value`            | PER, PBR, 배당수익률 등 가치 지표로 후보를 찾는다.      |
| `mwosa screen volatility`       | 변동성, ATR, 낙폭 기준으로 후보를 찾는다.               |
| `mwosa screen liquidity`        | 거래대금, 거래량, 스프레드 기준으로 후보를 찾는다.      |
| `mwosa explain screen <run-id>` | 특정 스크리닝 결과가 왜 통과하거나 탈락했는지 설명한다. |

### Compare

| 명령어                                | 설명                                                        |
| ------------------------------------- | ----------------------------------------------------------- |
| `mwosa compare symbols <symbols...>`  | 여러 종목의 수익률, 변동성, 거래량, 밸류에이션을 비교한다.  |
| `mwosa compare etfs <symbols...>`     | ETF 의 기초지수, 보수, 추적오차, 유동성, 수익률을 비교한다. |
| `mwosa compare sectors <sectors...>`  | 업종 또는 섹터 단위 성과와 위험을 비교한다.                 |
| `mwosa compare providers <symbol>`    | provider 별 동일 데이터의 차이와 신선도를 비교한다.         |
| `mwosa compare portfolios <names...>` | 여러 포트폴리오의 성과와 위험 지표를 비교한다.              |

### 리소스 생성과 수정

포트폴리오, universe, 전략, 알림은 같은 CRUD 계열 verb 를 공유한다.

| 명령어                                             | 설명                                                                  |
| -------------------------------------------------- | --------------------------------------------------------------------- |
| `mwosa create universe <name>`                     | 관심 종목 universe 를 만든다.                                         |
| `mwosa update universe <name>`                     | universe 이름, 설명, 기본 필터를 수정한다.                            |
| `mwosa add universe-symbol <universe> <symbol>`    | universe 에 종목을 추가한다.                                          |
| `mwosa remove universe-symbol <universe> <symbol>` | universe 에서 종목을 제거한다.                                        |
| `mwosa create portfolio <name>`                    | 새 포트폴리오를 만든다.                                               |
| `mwosa update portfolio <name>`                    | 포트폴리오 이름, 설명, 목표 통화, 목표 비중을 수정한다.               |
| `mwosa add holding <portfolio> <symbol>`           | 포트폴리오에 보유 종목을 추가한다.                                    |
| `mwosa update holding <portfolio> <symbol>`        | 수량, 평균단가, 목표비중 같은 보유 정보를 수정한다.                   |
| `mwosa remove holding <portfolio> <symbol>`        | 포트폴리오에서 보유 종목을 제거한다.                                  |
| `mwosa rebalance portfolio <name>`                 | 목표 비중 대비 매수/매도 필요 금액을 계산한다.                        |
| `mwosa simulate portfolio <name>`                  | 특정 가격, 환율, 비중 변화가 포트폴리오에 미치는 영향을 계산한다.     |
| `mwosa stress portfolio <name>`                    | 금리, 환율, 지수 하락 같은 시나리오로 포트폴리오 스트레스를 계산한다. |
| `mwosa create strategy <name>`                     | 스크리닝 조건, 진입 조건, 청산 조건을 가진 전략 초안을 만든다.        |
| `mwosa update strategy <name>`                     | 전략 조건, universe, 리스크 설정을 수정한다.                          |
| `mwosa validate strategy <name>`                   | 전략 정의의 필수 항목과 데이터 요구사항을 검사한다.                   |
| `mwosa run backtest <yaml>`                        | YAML 실행 스펙으로 과거 데이터 전략 성과를 검증한다.                  |
| `mwosa update backtest strategy <name>`            | YAML 전략 스펙을 저장하거나 새 버전으로 갱신한다.                     |
| `mwosa compare strategies <names...>`              | 여러 전략의 수익률, 낙폭, 승률, 회전율을 비교한다.                    |
| `mwosa explain backtest <run-id>`                  | 백테스트 결과의 주요 거래, 성과 요인, 한계를 설명한다.                |
| `mwosa create alert <target>`                      | 가격, 지표, 뉴스, 공시 조건에 대한 알림을 만든다.                     |
| `mwosa update alert <alert-id>`                    | 알림 조건과 전달 경로를 수정한다.                                     |
| `mwosa delete alert <alert-id>`                    | 등록된 알림을 삭제한다.                                               |

### 거래 기록과 복기

| 명령어                         | 설명                                              |
| ------------------------------ | ------------------------------------------------- |
| `mwosa record trade`           | 실제 또는 가상 거래를 기록한다.                   |
| `mwosa close trade <trade-id>` | 거래의 청산가, 청산일, 청산 사유를 기록한다.      |
| `mwosa record thesis <symbol>` | 종목별 투자 가설을 기록한다.                      |
| `mwosa record note <target>`   | 종목, 포트폴리오, 전략에 대한 메모를 남긴다.      |
| `mwosa review journal`         | 거래일지와 메모를 기간별로 복기한다.              |
| `mwosa summarize journal`      | 승률, 평균 손익, 룰 준수율, 반복 실수를 요약한다. |

### 알림과 모니터링

| 명령어                        | 설명                                                     |
| ----------------------------- | -------------------------------------------------------- |
| `mwosa watch quote <symbol>`  | 특정 종목의 quote 변화를 터미널에서 지속적으로 보여준다. |
| `mwosa watch screen <name>`   | 저장된 스크리닝 조건을 반복 실행해 변화만 보여준다.      |
| `mwosa test alert <alert-id>` | 알림 조건과 전달 경로가 정상인지 확인한다.               |

### AI 에이전트 지원

| 명령어                         | 설명                                                           |
| ------------------------------ | -------------------------------------------------------------- |
| `mwosa build context <target>` | AI 에이전트가 리서치에 바로 사용할 수 있는 데이터 묶음을 만든다. |
| `mwosa build prompt <target>`  | LLM 에 전달하기 좋은 리서치 프롬프트를 생성한다.               |
| `mwosa validate output <path>` | 저장된 JSON/NDJSON 출력이 schema 를 만족하는지 검사한다.       |

### 진단

| 명령어                            | 설명                                                                 |
| --------------------------------- | -------------------------------------------------------------------- |
| `mwosa doctor`                    | 설정, provider 인증, 데이터 디렉터리, 네트워크 상태를 종합 점검한다. |
| `mwosa trace request <command>`   | 특정 명령이 어떤 provider, cache, normalizer 를 거쳤는지 추적한다.   |
| `mwosa benchmark provider <name>` | provider 응답 시간과 rate limit 상태를 측정한다.                     |
