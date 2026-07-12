---
id: API.A.01
title: 주문 생성 API 설계
type: service-design-api-endpoint
status: example
tags: [service-design, api, order, checkout, example]
source: local
created: 2026-07-06
updated: 2026-07-10
service_design: SD.A.01
api_design: SD.A.0140
domain_model: SD.A.0110
persistence: SD.A.0120
service: SD.A.0130
---

# API.A.01 주문 생성

## 기본 정보

| 항목 | 값 |
| --- | --- |
| Method / Path | `POST /orders` (`/api/v1` Base URL) |
| operationId | `placeOrder` |
| 역할 | 구매자가 확인한 장바구니·배송지·쿠폰을 기준으로 주문과 금액 스냅샷을 생성한다. |
| API 유형 | Command |
| 인증 | 웹 Session cookie + CSRF token + Origin |
| 권한 | 현재 사용자가 소유한 장바구니와 사용할 수 있는 배송지·쿠폰만 허용 |
| 노출 범위 | public |
| 멱등성 | `Idempotency-Key` 필수 |
| 캐시 | 성공·오류 응답 `no-store` |
| 호환성 | `/api/v1`, deprecation 없음 |

## HTTP 계약 원장

- 완전한 OpenAPI 문서: [openapi/openapi.yaml](openapi/openapi.yaml)
- 이 Endpoint의 Path Item: [openapi/paths/API_A_01_place_order.yaml](openapi/paths/API_A_01_place_order.yaml)
- 공통 component: [schemas](openapi/components/schemas.yaml), [parameters](openapi/components/parameters.yaml), [headers](openapi/components/headers.yaml), [responses](openapi/components/responses.yaml), [security schemes](openapi/components/security-schemes.yaml)

요청·응답 필드, required 조건, header, `201`과 오류 상태, ProblemDetails와 wire 예시는 OpenAPI를 기준으로 한다.

## 연관 문서

- [REQ.A.01 주문 결제 요구사항](../../../../00-requirements/.examples/REQ_A_01_order_checkout.md)
- [UC.A.01 주문 및 결제 사용자 목표](../../../../30-uc/.examples/UC_A_01_place_order.md)
- [BC.EX.01 Context 주문](../../../../40-event-storming-bounded-context/.examples/BC_EX_01_order.md)
- [AGG.A.01 주문 Aggregate](../A_01_10-domain-model/AGG_A_01_order.md)
- [PST.A.01 주문 영속성](../A_01_20-persistence/PST_A_01_order_persistence.md)
- [SVC.A.01 주문 서비스](../A_01_30-service/SVC_A_01_order_service.md)
- [SCN.A.01 주문 생성 시퀀스](../../../../80-sequence/.examples/SCN_A_01_place_order.md)

## 책임과 경계

- 이 API는 주문 생성 가능 여부, 서버 계산 금액과 주문 시점 스냅샷을 확정한다.
- 구매자 식별자는 request body에서 받지 않고 검증된 Session의 `user_id`를 사용한다.
- 결제 승인, 상품 재고 원장 변경, 쿠폰 발급은 이 API가 담당하지 않는다.
- 재고·쿠폰·배송지 Context의 결과를 어떤 방식으로 고정할지는 아래 확인 필요 항목에서 별도로 관리한다.

## 보안과 개인정보

- `SessionCookie`, `CsrfToken`, `WebOrigin`을 모두 만족해야 Handler를 호출한다.
- 장바구니, 배송지와 쿠폰의 소유·사용 가능 여부를 현재 `user_id` 기준으로 검증한다.
- 다른 사용자의 참조와 존재 여부는 공개 응답에서 구분하지 않는다.
- Session·CSRF 원문, 배송지 내용, 전체 request body를 로그·trace·metric label에 남기지 않는다.
- `expectedTotal`은 위변조 방지용 비교값이며 서버 계산 결과를 대체하지 않는다.

## 처리 규칙

1. 인증, CSRF, Origin, request schema와 Idempotency-Key를 검증한다.
2. 현재 사용자의 장바구니, 배송지와 선택한 쿠폰의 사용 가능 여부를 확인한다.
3. 상품 가격과 할인 결과를 서버에서 다시 계산한다.
4. 계산한 총액이 `expectedTotal`과 다르면 주문을 만들지 않고 최신 금액 확인을 요구한다.
5. `Order(status=created)`와 주문 라인·금액 스냅샷을 만든다.
6. 주문, 주문 라인과 멱등 기록을 저장한 뒤 `201 Created`를 반환한다.

## 상태 변경과 트랜잭션

- 시작 상태: 주문 Aggregate 없음.
- 성공 종료 상태: `Order.status=created`.
- `orders`, `order_lines`, `order_idempotency_keys`는 같은 로컬 트랜잭션에 저장한다.
- 트랜잭션이 실패하면 주문 ID와 성공 응답을 확정하지 않는다.
- `EVT.A.01`의 유실을 막기 위한 Outbox 저장은 영속성 예시에 추가해야 한다.
- 외부 Context 호출은 DB 트랜잭션 안에서 실행하지 않는 것을 목표로 하지만 재고·쿠폰 확정 순서는 아직 결정되지 않았다.

## 멱등성과 동시성

- 범위는 `placeOrder + user_id + Idempotency-Key`다.
- canonical request에는 서버 비밀키 HMAC을 적용한 fingerprint만 저장하고 요청 원문은 멱등 저장소에 남기지 않는다.
- 같은 key와 같은 fingerprint는 최초 주문과 `201` 응답을 재사용하고 `Idempotency-Replayed: true`를 반환한다.
- 같은 key와 다른 fingerprint는 새 주문을 만들지 않는다.
- `order_idempotency_keys`의 unique constraint가 중복 주문을 막는다.
- 멱등 보존 TTL과 같은 장바구니의 동시 주문 생성 정책은 아직 확정하지 않았다.

## 예외와 복구 규칙

정확한 HTTP 상태, error code, ProblemDetails schema와 예시는 OpenAPI를 기준으로 한다.

| 업무 조건 | 공개 원칙 | 클라이언트 복구 |
| --- | --- | --- |
| 요청 schema 또는 금액 형식 오류 | 공개 가능한 필드만 violations로 반환 | 입력값을 수정한다. |
| 인증·소유권 검증 실패 | 다른 사용자와 리소스 존재 여부를 숨긴다. | 로그인과 주문 대상을 다시 확인한다. |
| 상품 품절 | 내부 재고 수량을 노출하지 않는다. | 장바구니를 새로 확인한다. |
| 서버 계산 총액 변경 | 최신 서버 계산 금액만 제공한다. | 변경 금액을 확인하고 새 key로 다시 주문한다. |
| 쿠폰 사용 불가 | 만료·소진·대상 불일치를 하나의 공개 의미로 합친다. | 쿠폰을 제거하거나 다시 선택한다. |
| 같은 key의 다른 요청 | 원래 주문을 보존하고 두 번째 주문을 만들지 않는다. | 원래 key의 요청을 재사용하거나 새 주문 의도로 새 key를 사용한다. |
| 필수 의존성 장애 | 성공 응답으로 대체하지 않는다. | `Retry-After` 이후 같은 key로 재시도한다. |

## 도메인과 서비스 매핑

| 계층 | 매핑 |
| --- | --- |
| Command Handler | `PlaceOrderHandler`, `CMD.A.01` |
| Aggregate / Entity | `Order`, `OrderLine`, [AGG.A.01](../A_01_10-domain-model/AGG_A_01_order.md) |
| Repository | `OrderRepository`, `IdempotencyRepository` |
| 외부 Port | 장바구니·배송지·재고·쿠폰 조회 또는 예약 Port |
| Event | `EVT.A.01 주문 생성 완료` |
| Read Model | `RM.A.01 주문 결제 요약` |

`AGG.A.01`, `CMD.A.01`, `EVT.A.01`, `RM.A.01`과 원천 이벤트스토밍의 `AGG.EX.01`, `CMD.EX.01`, `EVT.EX.01`, `RM.EX.01` 사이의 상세화 관계는 예시 문서 전체에서 정리할 필요가 있다.

## 관측성과 운영

- 로그: `request_id`, `trace_id`, `order_id`, 결과, error code와 멱등 replay 여부를 기록한다.
- Metric: `order_place_requests_total{result}`, `order_place_duration_seconds`, `order_idempotency_replays_total`을 사용한다.
- Trace: 소유권 확인, 가격·쿠폰·재고 의존성, DB 트랜잭션을 구분한다.
- Audit: 주문 생성 주체, 주문 ID, 금액과 적용 정책 버전을 기록하되 배송지 원문을 제외한다.
- Rate limit은 `user_id`와 Session 기준으로 적용하고 제한 응답에는 `Retry-After`를 반환한다.
- 인증·주문 Endpoint의 body sampling과 중간 캐시를 비활성화한다.

## 검증 항목

- OpenAPI lint와 bundle이 성공하고 모든 예시가 schema를 통과한다.
- 사용자 식별자를 body에서 바꿀 수 없다.
- 다른 사용자의 장바구니·배송지·쿠폰으로 주문할 수 없다.
- 상품 품절, 금액 변경과 쿠폰 사용 불가가 주문을 부분 저장하지 않는다.
- 같은 key와 같은 요청은 주문과 Event를 중복 생성하지 않는다.
- 같은 key와 다른 요청은 멱등 충돌로 종료한다.
- DB 실패 시 주문, 주문 라인과 멱등 기록이 모두 롤백된다.
- 민감 정보가 응답, 로그, trace와 metric label에 남지 않는다.

## 연관 시퀀스

- [SCN.A.01 주문 생성](../../../../80-sequence/.examples/SCN_A_01_place_order.md)
- 관련 API: `API.A.01`
- 여러 참여자의 Mermaid 다이어그램은 시퀀스 문서에서 관리한다.

## 호환성과 변경 정책

- optional 응답 필드와 새 오류 code 추가는 클라이언트 영향 검토 뒤 `/api/v1`에 추가할 수 있다.
- 기존 필드 삭제·타입 변경·의미 변경은 새 API 버전에서 처리한다.
- deprecation 시 대체 operation과 제거 예정일을 OpenAPI에 함께 기록한다.

## 확인 필요

- `orderNumber` 생성 규칙과 Aggregate·DB 저장 필드를 추가한다.
- 배송지, 쿠폰, 금액 구성의 주문 시점 스냅샷 범위를 확정한다.
- 재고를 조회만 할지 주문 생성과 함께 예약할지 결정한다.
- 쿠폰 사용 예약과 실패 보상 책임을 Context 주문과 Context 쿠폰 중 어디에 둘지 결정한다.
- `EVT.A.01` Outbox와 relay를 영속성·서비스·시퀀스 예시에 반영한다.
- IdempotencyRecord의 request fingerprint, scope, response snapshot과 TTL을 영속성 예시에 추가한다.
