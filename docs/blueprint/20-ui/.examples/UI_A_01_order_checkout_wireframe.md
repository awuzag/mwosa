---
id: UI.A.01
title: 주문 결제 와이어프레임
type: ui-asset
status: example
tags: [ui, wireframe, checkout, example]
source: local
created: 2026-07-06
updated: 2026-07-06
---

# 주문 결제 와이어프레임

## 기본 정보

- UI ID: `UI.A.01`
- 연관 Page: [PAGE.A.01](../../10-sitemap/.examples/PAGE_A_01_order_checkout.md)
- 에셋 유형: 와이어프레임
- 파일 경로: [order-checkout-wireframe.svg](assets/UI_A_01_order_checkout_wireframe/order-checkout-wireframe.svg)
- 캡처 조건: 로그인한 구매자, 장바구니 상품 2개, 사용 가능한 쿠폰 1개.

## 연관 태그

🏷️ 요구사항 참조: [REQ.A.01](../../00-requirements/.examples/REQ_A_01_order_checkout.md) | 페이지 참조: [PAGE.A.01](../../10-sitemap/.examples/PAGE_A_01_order_checkout.md) | UC 참조: [UC.A.01](../../30-uc/.examples/UC_A_01_place_order.md) | 영속성 참조: [PST.A.01](../../50-service-design/.examples/order/A_01_20-persistence/PST_A_01_order_persistence.md) | 서비스 참조: [SVC.A.01](../../50-service-design/.examples/order/A_01_30-service/SVC_A_01_order_service.md) | 시나리오 참조: [SCN.A.01](../../80-sequence/.examples/SCN_A_01_place_order.md) | API 참조: [API.A.01](../../50-service-design/.examples/order/A_01_40-api/API_A_01_place_order.md)

## 에셋

![주문 결제 와이어프레임](assets/UI_A_01_order_checkout_wireframe/order-checkout-wireframe.svg)

## 화면 구성

- 좌측: 주문 상품 목록.
- 우측: 금액 요약, 쿠폰 적용 결과, 주문 확정 버튼.
- 상단: 현재 단계와 페이지 제목.

## 화면에 필요한 정보

| 화면 영역 | 필드 | 타입 | 용도 |
| --- | --- | --- | --- |
| 주문 상품 | `items[].productName` | string | 상품명 표시 |
| 주문 상품 | `items[].quantity` | number | 주문 수량 표시 |
| 주문 상품 | `items[].unitPrice` | number | 상품별 금액 표시 |
| 금액 요약 | `summary.subtotalAmount` | number | 상품 합계 표시 |
| 금액 요약 | `summary.discountAmount` | number | 총 할인 금액 표시 |
| 금액 요약 | `summary.totalAmount` | number | 최종 결제 금액 표시 |
| 주문 액션 | `actions.canPlaceOrder` | boolean | 주문 버튼 활성화 |
| 주문 액션 | `actions.disabledReason` | string? | 주문 불가 사유 표시 |

## 설계 반영 사항

- Read Model 후보: `RM.A.01`
- Command 후보: `CMD.A.01`
- Error 후보: `ERR.A.01`, `ERR.A.03`
