---
id: PAGE.A.02
page_id: PAGE.A.02
title: 상품 상세 페이지
type: page-spec
status: example
path: /products/{productId}
parent_pages: [상품 목록, 주문 결제 페이지]
child_pages: [장바구니, 주문 결제 페이지]
actors: [구매자]
tags: [page, sitemap, product, example]
source: local
created: 2026-07-06
updated: 2026-07-06
---

# 상품 상세 페이지

## 페이지 소개

구매자가 상품 정보, 가격, 옵션, 재고 상태를 확인하고 구매 행동으로 이어지는 페이지다.

## 스크린샷

![상품 상세 이동 예시](../../20-ui/.examples/assets/UI_A_01_order_checkout_wireframe/order-checkout-wireframe.svg)

## 연관 사이트맵

```mermaid
flowchart LR
    ProductList["상품 목록"]
    ProductDetail["PAGE.A.02 상품 상세"]
    Checkout["PAGE.A.01 주문 결제"]

    ProductList -->|"상품 선택"| ProductDetail
    Checkout -->|"상품명/Detail 클릭"| ProductDetail
    ProductDetail -->|"바로 구매 / 결제 복귀"| Checkout

    click Checkout "./PAGE_A_01_order_checkout.md"
```

[PAGE.A.01](./PAGE_A_01_order_checkout.md)

## 연관 태그

🏷️ 요구사항 참조: [REQ.A.01](../../00-requirements/.examples/REQ_A_01_order_checkout.md) | 이동 출발 페이지: [PAGE.A.01](./PAGE_A_01_order_checkout.md)
