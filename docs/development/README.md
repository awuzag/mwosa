# Development Collaboration Guide

## 목적

이 문서는 `mwosa` 를 함께 구현할 때의 개발 협업 기준을 정리한다.

아키텍처 문서는 레이어와 책임을 설명하고, 이 문서는 구현 중 어떤 결정을 지금 고정하고 어떤 결정을 나중으로 미룰지 정한다.

## 기본 원칙

- CLI core 는 provider 의 구현 방식을 모른다.
- service 는 provider role interface 에만 의존한다.
- provider adapter 는 provider client module 과 `mwosa` role interface 사이를 연결한다.
- provider 가 REST API, SDK, 파일, scraping 중 무엇을 쓰는지는 provider 구현 단계에서 정한다.
- 구현 라이브러리 선택은 필요가 보일 때 작게 결정하고, core contract 로 새지 않게 한다.

## HTTP Client 선택

provider 가 REST API 를 사용할 수도 있지만, 모든 provider 가 REST API 를 쓰는 것은 아니다. 따라서 HTTP client 라이브러리는 지금 공통 기술로 고정하지 않는다.

초기 기준은 다음과 같다.

- 기본값은 Go 표준 라이브러리 `net/http` 로 본다.
- retry/backoff 정책이 반복되면 `go-retryablehttp` 같은 얇은 wrapper 를 검토한다.
- REST request 작성 코드가 지나치게 장황해지면 `resty` 나 `req` 같은 고수준 client 를 검토한다.
- `fasthttp` 같은 성능 특화 client 는 측정된 병목이 있을 때만 검토한다.

지켜야 할 경계:

- provider role interface 에 특정 HTTP client type 을 노출하지 않는다.
- service request/result 에 HTTP 라이브러리 전용 옵션을 넣지 않는다.
- timeout, retry, rate limit, provenance 는 provider config 와 provider result 의 의미로 표현한다.
- REST provider 테스트는 가능하면 `httptest` 나 provider-local fake transport 로 작성한다.

## 테스트 기준

테스트는 Go 표준 `testing` 을 기반으로 작성하고, assertion 은 `testify` 를 사용한다.

특히 `packages/indicators` 같은 계산 패키지는 함수별 단위 테스트가 필수다. 계산 함수는 기준 fixture 와 비교해 정확성을 보장해야 하며, warm-up 구간, invalid input, floating point 허용 오차를 테스트 안에 명시한다.

기준:

- 입력 준비나 선행 조건 검증은 `require` 로 실패 시 즉시 중단한다.
- 계산 결과 비교는 `assert` 로 여러 값의 차이를 한 번에 확인한다.
- 외부 라이브러리를 참고하더라도 `mwosa` 의 public result type 과 fixture 기준을 테스트의 source of truth 로 둔다.

## Provider 추가 흐름

새 provider 를 붙일 때는 아래 순서로 진행한다.

1. 필요한 role interface 를 먼저 확인한다.
2. provider 가 지원하는 capability 와 제한사항을 profile 로 적는다.
3. provider 구현체가 REST 인지 SDK 인지 파일 기반인지 확인한다.
4. `mwosa` repository root 의 `go.work` 에 포함되는 `clients/<client-name>` 독립 Go module 로 provider client 를 만든다.
5. 구현체에 맞는 transport/library 를 provider client module 안에서 선택한다.
6. request builder, fake transport, 응답 파서, provider-native error 를 client module 단위 테스트로 검증한다.
7. 단위 테스트를 통과한 client 만 CLI adapter 에 연결한다.
8. provider response 를 canonical data 로 변환한다.
9. 실패는 성공처럼 숨기지 않고 provider 이름, operation, symbol, market 같은 맥락을 붙여 반환한다.
10. service 에는 provider 별 adapter type 이 아니라 role interface 와 router 결과만 전달한다.

## 문서 갱신 기준

한 provider 에만 필요한 선택은 provider 문서에 적는다.

여러 provider 에 반복되는 선택은 이 문서에 올린다.

core contract 에 영향을 주는 선택은 `docs/architectures/layers/README.md` 또는 `docs/architectures/provider/README.md` 에 반영한다.

기술 스택으로 고정할 만큼 반복 사용이 확인된 선택은 `docs/architectures/tech-stack/README.md` 로 옮긴다.

## 관련 문서

- `docs/development/cobra-completion/README.md`
- `docs/development/git-branching/README.md`
- `docs/development/pre-commit/README.md`
- `docs/development/release/README.md`
