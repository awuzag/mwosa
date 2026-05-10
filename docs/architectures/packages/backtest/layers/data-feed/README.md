# DataFeed

`DataFeed`는 엔진에 시장 데이터를 공급한다. 백테스트에서는 과거 데이터를 재생하고, 페이퍼 트레이딩과 실거래에서는 현재 데이터를 공급한다.

초기 구현은 [거래 원칙](../../../trading-principles/README.md)에 따라 OHLCV, 거래량, 거래대금, 최근 고가/저가, 이동평균 계산에 필요한 원천 데이터를 우선한다.

다중 타임프레임 전략에서는 [타임프레임 설계](../../timeframes/README.md)에 따라 일봉, 5분봉, 1분봉, tick 데이터를 공급한다. 단, 어떤 타임프레임을 어떤 판단에 사용할지는 `DataFeed`가 아니라 `TimeframePolicy`와 `EvaluationScheduler`의 책임이다.

## 핵심 질문

무엇을 알고 있나?

## 책임

- 외부 데이터 출처에서 원천 데이터를 가져온다.
- 엔진이 사용할 공통 데이터 모델로 정규화한다.
- 과거 데이터를 시간순으로 재생한다.
- 실시간 또는 준실시간 데이터를 공급한다.
- tick 또는 짧은 주기 데이터를 분봉으로 집계한다.
- 요청된 타임프레임의 데이터를 같은 기준으로 조회할 수 있게 한다.

## 입력

- 브로커 API 응답
- 데이터 vendor 파일 또는 API 응답
- 외부 리서치 데이터
- 로컬 DB에 저장된 과거 일봉, 분봉, tick 데이터

## 출력

- `DailyBar`
- `IntradayBar`
- `Tick`
- `Timeframe`이 명시된 시장 데이터 이벤트
- 거래 가능 상태 이벤트
- 데이터 품질 경고

## 하위 책임

- `SourceConnector`: 외부 출처에서 원천 데이터를 가져온다.
- `MarketDataStore`: 원천 데이터와 정규화 데이터를 저장한다.
- `HistoricalFeed`: 백테스트용 데이터를 시간순으로 재생한다.
- `LiveFeed`: 페이퍼와 실거래용 현재 데이터를 공급한다.
- `BarBuilder`: tick 또는 짧은 주기 데이터를 1분봉, 3분봉, 5분봉으로 집계한다.

## 구현 후보

- 직접 구현: `DataFeed`, `HistoricalFeed`, `LiveFeed`, `BarBuilder`의 도메인 인터페이스는 직접 정의해야 한다.
- [Alpaca Market Data API](https://docs.alpaca.markets/docs/about-market-data-api): 미국 주식과 ETF의 실시간 및 과거 시장 데이터 연동 후보.
- [Alpaca Trading API](https://docs.alpaca.markets/): 페이퍼 트레이딩과 주문 실행을 함께 붙일 수 있는 API 후보.
- [Interactive Brokers API](https://www.interactivebrokers.com/campus/ibkr-api-page/webapi-doc/): 미국 ETF 실거래와 계좌 데이터 연동 후보.
- [gorilla/websocket](https://github.com/gorilla/websocket): 실시간 체결이나 시세 스트림을 받을 때 조사할 Go WebSocket 라이브러리.
- [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite): 로컬 개발과 작은 데이터셋 저장에 적합한 CGo-free SQLite 드라이버 후보.
- [DuckDB Go](https://duckdb.org/docs/stable/clients/go): 일봉, 분봉, 리서치 결과를 분석형 쿼리로 다루기 위한 저장소 후보.

## 조사할 자료

- 미국 ETF 일봉, 분봉, 실시간 체결 데이터를 제공하는 브로커 또는 데이터 vendor
- API 호출 제한, 실시간 구독 제한, 과거 분봉 조회 가능 기간
- ETF 마스터, AUM, 비용률, 추종 지수 같은 기준 데이터 범위
- 분봉 저장 스키마와 압축, 파티셔닝 방식
- 장중 데이터 누락, 중복, 지연을 탐지하는 방법

## 설계 쟁점

- 일봉과 분봉을 같은 인터페이스로 다룰지, 별도 이벤트 타입으로 나눌지 정해야 한다.
- 상위, 진입, 실행 타임프레임을 한 컨텍스트로 묶어 규칙에 전달할 방법이 필요하다.
- 분봉 전략은 데이터 지연과 API 제한을 전략 결과에 반영해야 한다.
- 백테스트에서는 그 시점에 알 수 있었던 데이터만 공급해야 한다.
- 실시간 tick으로 분봉을 만들 때 정규장, 프리마켓, 애프터마켓, 휴장일 처리를 정의해야 한다.
