# OrderExecutor

`OrderExecutor`는 리스크 검증을 통과한 주문을 실행하고 체결 결과를 만든다. 실제 매수와 매도, 그리고 청산 주문은 이 레이어의 책임이다.

## 핵심 질문

실제로 주문을 어떻게 실행할까?

## 책임

- 승인된 주문 의도를 실행 가능한 주문 요청으로 변환한다.
- 백테스트, 페이퍼, 실거래 모드별 실행 구현체를 제공한다.
- 주문 접수, 체결, 부분 체결, 실패를 기록한다.
- 체결 결과를 포트폴리오가 반영할 수 있는 형태로 반환한다.

## 입력

- 리스크 검증을 통과한 주문 후보
- 주문 종류와 유효 기간
- 체결 모델 또는 브로커 API 클라이언트
- 현재 시장 데이터

## 출력

- 체결 결과
- 주문 접수 결과
- 부분 체결 상태
- 실패 사유
- 주문 및 체결 로그

## 구현체 후보

- `BacktestExecutor`: `ExecutionModel`을 사용해 과거 데이터 기준 가상 체결을 만든다.
- `PaperExecutor`: 실제 주문 없이 현재 데이터 기준 가상 체결을 만든다.
- `LiveExecutor`: 브로커 API로 실제 주문을 넣고 응답을 기록한다.

## 구현 후보

- 직접 구현: `OrderExecutor`, `BacktestExecutor`, `PaperExecutor`, `LiveExecutor` 인터페이스와 주문 상태 모델은 직접 구현해야 한다.
- [Alpaca Trading API](https://docs.alpaca.markets/): 미국 주식과 ETF 주문, 페이퍼 트레이딩, 계좌 조회 연동 후보.
- [Interactive Brokers API](https://www.interactivebrokers.com/campus/ibkr-api-page/webapi-doc/): 미국 ETF 실거래 주문과 계좌 조회를 붙일 수 있는 후보.
- Go 표준 `net/http`: REST 주문 API 클라이언트의 기본 구현은 표준 라이브러리로 충분할 가능성이 높다.
- [gorilla/websocket](https://github.com/gorilla/websocket): 실시간 체결 통보나 주문 이벤트 스트림을 받을 때 조사할 후보.
- [Watermill](https://watermill.io/): 주문 접수, 체결, 실패 이벤트를 내부 이벤트 버스로 흘릴 때 검토할 수 있다.

## 조사할 자료

- 미국 ETF 브로커 API의 주문 종류, 정정, 취소, 체결 조회 방식
- 실시간 체결 통보를 받는 방식
- 주문 실패, 부분 체결, 미체결 잔량 처리 방식
- 수동 승인 모드와 자동 승인 모드의 운영 절차
- 주문 로그에 반드시 남겨야 할 필드

## 설계 쟁점

- `OrderExecutor`와 브로커별 `BrokerAdapter`를 같은 개념으로 둘지 분리할지 정해야 한다.
- 실거래에서는 주문 요청과 체결 통보가 비동기일 수 있다.
- 실패한 주문을 자동 재시도할지, 사람이 확인할 때까지 멈출지 정해야 한다.
- 청산 주문은 신규 진입 주문보다 우선순위를 높게 둘 가능성이 크다.
