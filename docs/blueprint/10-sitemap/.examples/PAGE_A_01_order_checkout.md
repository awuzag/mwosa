---
id: PAGE.A.01
page_id: PAGE.A.01
title: 주문 결제 페이지
type: page-spec
status: example
path: /checkout
parent_pages: [장바구니]
child_pages: [주문 완료]
actors: [구매자]
tags: [page, sitemap, checkout, example]
source: local
created: 2026-07-06
updated: 2026-07-06
---

# 주문 결제 페이지

## 페이지 소개

구매자가 주문 상품, 할인, 최종 결제 금액을 확인하고 주문을 확정하는 페이지다.

## 스크린샷

![주문 결제 와이어프레임](../../20-ui/.examples/assets/UI_A_01_order_checkout_wireframe/order-checkout-wireframe.svg)

## 연관 사이트맵

```mermaid
flowchart LR
    Cart["장바구니"]
    Checkout["PAGE.A.01 주문 결제"]
    ProductDetail["PAGE.A.02 상품 상세"]
    BuyerProfile["PAGE.A.03 구매자 프로필"]
    OrderComplete["주문 완료"]

    Cart -->|"결제하기"| Checkout
    Checkout -->|"상품명/Detail 클릭"| ProductDetail
    Checkout -->|"Profile 클릭 / 로그인 필요"| BuyerProfile
    Checkout -->|"주문 성공"| OrderComplete

    click ProductDetail "./PAGE_A_02_product_detail.md"
    click BuyerProfile "./PAGE_A_03_buyer_profile.md"
```

[PAGE.A.02](./PAGE_A_02_product_detail.md) [PAGE.A.03](./PAGE_A_03_buyer_profile.md)

## 연관 태그

🏷️ 요구사항 참조: [REQ.A.01](../../00-requirements/.examples/REQ_A_01_order_checkout.md) | 플로우 참조: FLOW.A.01 | UI 참조: [UI.A.01](../../20-ui/.examples/UI_A_01_order_checkout_wireframe.md) | UC 참조: [UC.A.01](../../30-uc/.examples/UC_A_01_place_order.md) | 영속성 참조: [PST.A.01](../../50-service-design/.examples/order/A_01_20-persistence/PST_A_01_order_persistence.md) | 서비스 참조: [SVC.A.01](../../50-service-design/.examples/order/A_01_30-service/SVC_A_01_order_service.md) | 시나리오 참조: [SCN.A.01](../../80-sequence/.examples/SCN_A_01_place_order.md) | API 참조: [API.A.01](../../50-service-design/.examples/order/A_01_40-api/API_A_01_place_order.md)
