---
id: REQ.A.01
title: 주문 결제 요구사항 정의
type: requirements
status: example
tags: [requirements, order, checkout, example]
source: local
created: 2026-07-06
updated: 2026-07-06
---

# 주문 결제 요구사항 정의

## 기본 정보

- Requirements ID: `REQ.A.01`
- 프로젝트: 커머스 주문 결제 예제
- 주요 사용자: 구매자
- 설계 범위: 장바구니 상품 확인, 쿠폰 적용 결과 확인, 주문 생성
- 제외 범위: PG 결제 승인, 배송 추적, 환불, 정산

## 문제 정의

구매자는 주문 전 상품, 할인, 최종 결제 금액을 확인하고 주문을 확정해야 한다. 백엔드는 주문 가능 여부, 가격 스냅샷, 쿠폰 유효성, 주문 생성 결과를 일관되게 보장해야 한다.

## 기능 요구사항

| Req ID | 요구사항 | 사용자 | 우선순위 | 연결 Page/UC |
| --- | --- | --- | --- | --- |
| `FR-001` | 구매자는 결제 화면에서 주문 상품과 최종 금액을 확인한다. | 구매자 | Must | [PAGE.A.01](../../10-sitemap/.examples/PAGE_A_01_order_checkout.md), [UC.A.01](../../30-uc/.examples/UC_A_01_place_order.md) |
| `FR-002` | 구매자는 사용 가능한 쿠폰이 적용된 결과를 확인한다. | 구매자 | Must | [PAGE.A.01](../../10-sitemap/.examples/PAGE_A_01_order_checkout.md) |
| `FR-003` | 구매자는 주문 확정 버튼으로 주문을 생성한다. | 구매자 | Must | [API.A.01](../../50-service-design/.examples/order/A_01_40-api/API_A_01_place_order.md) |

## 비기능 요구사항

| Req ID | 요구사항 | 기준 | 연결 문서 |
| --- | --- | --- | --- |
| `NFR-001` | 주문 생성은 중복 요청에 안전해야 한다. | 같은 idempotency key는 같은 결과를 반환한다. | [API.A.01](../../50-service-design/.examples/order/A_01_40-api/API_A_01_place_order.md) |
| `NFR-002` | 주문 금액은 서버 계산 결과를 기준으로 확정한다. | 클라이언트 금액은 검증용으로만 사용한다. | [AGG.A.01](../../50-service-design/.examples/order/A_01_10-domain-model/AGG_A_01_order.md) |

## 연관 태그

🏷️ 플로우 참조: FLOW.A.01 | 페이지 참조: [PAGE.A.01](../../10-sitemap/.examples/PAGE_A_01_order_checkout.md) | UI 참조: [UI.A.01](../../20-ui/.examples/UI_A_01_order_checkout_wireframe.md) | UC 참조: [UC.A.01](../../30-uc/.examples/UC_A_01_place_order.md) | 영속성 참조: [PST.A.01](../../50-service-design/.examples/order/A_01_20-persistence/PST_A_01_order_persistence.md) | 서비스 참조: [SVC.A.01](../../50-service-design/.examples/order/A_01_30-service/SVC_A_01_order_service.md) | 시퀀스 참조: [SCN.A.01](../../80-sequence/.examples/SCN_A_01_place_order.md) | API 참조: [API.A.01](../../50-service-design/.examples/order/A_01_40-api/API_A_01_place_order.md)

## 열린 질문

- 쿠폰 할인 계산 책임을 주문 BC에 둘지 쿠폰 BC에 둘지 결정해야 한다.
- 결제 승인 실패 후 주문을 어떤 상태로 보존할지 결정해야 한다.
