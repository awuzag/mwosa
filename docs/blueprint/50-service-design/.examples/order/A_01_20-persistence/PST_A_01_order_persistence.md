---
id: PST.A.01
title: 주문 영속성 설계
type: persistence-design
status: example
tags: [persistence, database, schema, repository, order, example]
source: local
created: 2026-07-06
updated: 2026-07-06
bounded_context: BC.A.01
service: SVC.A.01
domain_models: [AGG.A.01]
apis: [API.A.01]
---

# 주문 영속성 설계

## 역할

주문 Aggregate를 관계형 데이터베이스에 저장하고, 주문 생성 API와 주문 결제 화면에 필요한 읽기/쓰기 전략을 정의한다.

## 연관 태그

🏷️ BC 참조: [BC.EX.01](../../../../40-event-storming-bounded-context/.examples/BC_EX_01_order.md) | 도메인 참조: [AGG.A.01](../A_01_10-domain-model/AGG_A_01_order.md) | 서비스 참조: [SVC.A.01](../A_01_30-service/SVC_A_01_order_service.md) | API 참조: [API.A.01](../A_01_40-api/API_A_01_place_order.md) | 시나리오 참조: [SCN.A.01](../../../../80-sequence/.examples/SCN_A_01_place_order.md)

## 저장 모델

| 테이블/컬렉션 | 역할 | 연결 도메인 | 비고 |
| --- | --- | --- | --- |
| `orders` | 주문 Aggregate Root 저장 | `Order` | 주문 상태와 금액 스냅샷의 기준 테이블 |
| `order_lines` | 주문 상품 스냅샷 저장 | `OrderLine` | 주문 생성 시점의 상품명, 단가, 수량 보존 |
| `order_idempotency_keys` | 중복 주문 생성 방지 | `OrderRepository` | 같은 key는 같은 주문 결과를 반환 |

## 스키마

| 저장 모델 | 필드 | 타입 | 제약 | 설명 |
| --- | --- | --- | --- | --- |
| `orders` | `order_id` | UUID | PK | 주문 식별자 |
| `orders` | `buyer_id` | UUID | NOT NULL | 구매자 식별자 |
| `orders` | `status` | VARCHAR(32) | NOT NULL | 주문 상태 |
| `orders` | `total_amount` | BIGINT | NOT NULL | 서버가 계산한 최종 금액 |
| `orders` | `currency` | CHAR(3) | NOT NULL | 통화 코드 |
| `orders` | `created_at` | TIMESTAMP | NOT NULL | 주문 생성 시각 |
| `order_lines` | `order_line_id` | UUID | PK | 주문 라인 식별자 |
| `order_lines` | `order_id` | UUID | FK | 주문 식별자 |
| `order_lines` | `product_id` | UUID | NOT NULL | 상품 식별자 |
| `order_lines` | `product_name` | VARCHAR(255) | NOT NULL | 주문 시점 상품명 |
| `order_lines` | `quantity` | INT | NOT NULL | 주문 수량 |
| `order_lines` | `unit_price` | BIGINT | NOT NULL | 주문 시점 단가 |
| `order_idempotency_keys` | `idempotency_key` | VARCHAR(128) | PK | 클라이언트 중복 요청 방지 키 |
| `order_idempotency_keys` | `order_id` | UUID | UNIQUE | 연결된 주문 식별자 |

## Aggregate 매핑

| 도메인 모델 | 저장 모델 | 매핑 방식 | 주의점 |
| --- | --- | --- | --- |
| [AGG.A.01](../A_01_10-domain-model/AGG_A_01_order.md) | `orders`, `order_lines` | `Order` 1개를 `orders` 1행과 `order_lines` N행으로 저장 | 저장 시 Aggregate 전체를 같은 트랜잭션에서 반영 |
| `VO.A.01 Money` | `orders.total_amount`, `orders.currency` | 금액과 통화를 컬럼으로 분리 | 통화가 늘어나면 반올림 규칙을 별도 검토 |

## Repository 설계 근거

| Repository 메서드 | 저장 모델 | 쿼리 기준 | 반환/저장 대상 |
| --- | --- | --- | --- |
| `save(order)` | `orders`, `order_lines` | `order.orderId` | 주문 Aggregate 전체 저장 |
| `findById(orderId)` | `orders`, `order_lines` | `orders.order_id` | 주문 Aggregate 복원 |
| `findByIdempotencyKey(key)` | `order_idempotency_keys`, `orders` | `idempotency_key` | 이미 생성된 주문 결과 조회 |
| `saveIdempotencyKey(key, orderId)` | `order_idempotency_keys` | `idempotency_key` | 중복 주문 생성 방지 기록 저장 |

## 쓰기 전략

| 작업 | 트랜잭션 경계 | 정합성 기준 | 실패 처리 |
| --- | --- | --- | --- |
| 주문 생성 | `orders`, `order_lines`, `order_idempotency_keys` 저장까지 하나의 트랜잭션 | 주문과 주문 라인은 항상 같이 저장된다 | 실패 시 전체 롤백 |
| 멱등 요청 처리 | `order_idempotency_keys.idempotency_key` 유니크 제약 | 같은 key는 하나의 주문만 가진다 | 이미 존재하면 기존 주문 결과 반환 |

## 읽기 전략

| 화면/API | 조회 모델 | 인덱스 | 캐시 여부 |
| --- | --- | --- | --- |
| [API.A.01](../A_01_40-api/API_A_01_place_order.md) | 생성된 주문 결과 | `orders.order_id` | 없음 |
| [UI.A.01](../../../../20-ui/.examples/UI_A_01_order_checkout_wireframe.md) | 주문 결제 요약 `RM.A.01` | 장바구니/구매자 기준 인덱스 필요 | 짧은 TTL 검토 |

## 인덱스

| 저장 모델 | 인덱스 | 목적 |
| --- | --- | --- |
| `orders` | `idx_orders_buyer_created_at (buyer_id, created_at DESC)` | 구매자 주문 조회 |
| `order_lines` | `idx_order_lines_order_id (order_id)` | 주문 Aggregate 복원 |
| `order_idempotency_keys` | `pk_order_idempotency_keys (idempotency_key)` | 중복 요청 방지 |

## 마이그레이션

- `orders`와 `order_lines`를 먼저 생성한 뒤 `order_idempotency_keys`를 추가한다.
- 기존 주문 데이터가 있는 경우 `currency` 기본값과 금액 단위를 사전에 확정한다.

## 확인 필요

- 주문 결제 요약 `RM.A.01`을 주문 서비스 DB에서 직접 구성할지, 장바구니/상품/쿠폰 조회 모델을 조합할지 결정해야 한다.
