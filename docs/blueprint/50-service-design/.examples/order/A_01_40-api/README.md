---
id: SD.A.0140
title: Context 주문 API 설계 예시
type: service-design-api
status: example
tags: [service-design, api, openapi, order, example]
source: local
created: 2026-07-10
updated: 2026-07-10
service_design: SD.A.01
domain_model: SD.A.0110
persistence: SD.A.0120
service: SD.A.0130
---

# Context 주문 API 설계 예시

## 역할

주문 생성 API를 예로 들어 OpenAPI HTTP 계약과 Markdown 설계 노트를 분리하는 방법을 보여준다.

## 계약 원장

| 문서 | 역할 |
| --- | --- |
| [OpenAPI 진입 문서](openapi/openapi.yaml) | API Base URL, Path, security scheme 등록 |
| [주문 생성 Path Item](openapi/paths/API_A_01_place_order.yaml) | 요청·응답·헤더·오류·예시의 정확한 계약 |
| [주문 생성 설계 노트](API_A_01_place_order.md) | 책임·트랜잭션·멱등성·보안·운영 판단 |
| [주문 생성 시퀀스](../../../../80-sequence/.examples/SCN_A_01_place_order.md) | 구매자부터 Event 발행까지의 참여자 처리 순서 |

## 공통 HTTP 원칙

- API Base URL은 `/api/v1`이고 주문 생성 Path는 `/orders`다.
- 성공 응답은 `data`와 `meta.requestId`를 분리한다.
- 오류는 `application/problem+json`과 안정적인 `code`를 사용한다.
- 웹 주문 생성은 Session cookie, CSRF token, Origin 검사를 모두 통과해야 한다.
- `Idempotency-Key`가 같은 동일 요청은 최초 `201` 결과와 `Location`을 재사용한다.
- 요청·응답 body와 오류 예시는 Markdown에 복제하지 않고 OpenAPI에서 관리한다.

## Endpoint 목록

| API ID | Method / Path | 역할 | 계약 | 설계 노트 |
| --- | --- | --- | --- | --- |
| `API.A.01` | `POST /orders` | 주문 생성 | [OpenAPI](openapi/paths/API_A_01_place_order.yaml) | [Markdown](API_A_01_place_order.md) |

## 검증

`archive` 저장소 루트에서 실행한다.

```bash
npx --yes @redocly/cli@2.38.0 lint \
  --config blueprint/50-service-design/.examples/order/A_01_40-api/openapi/redocly.yaml \
  blueprint/50-service-design/.examples/order/A_01_40-api/openapi/openapi.yaml

npx --yes @redocly/cli@2.38.0 bundle \
  blueprint/50-service-design/.examples/order/A_01_40-api/openapi/openapi.yaml \
  --output /tmp/A_01_order.openapi.bundle.yaml
```

코드 생성기는 검증된 bundle을 입력으로 사용하고 bundle을 직접 수정하지 않는다.
