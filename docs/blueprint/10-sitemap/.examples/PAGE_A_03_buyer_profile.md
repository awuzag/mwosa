---
id: PAGE.A.03
page_id: PAGE.A.03
title: 구매자 프로필 페이지
type: page-spec
status: example
path: /me
parent_pages: [주문 결제 페이지, 마이페이지]
child_pages: [배송지 관리, 주문 내역]
actors: [구매자]
tags: [page, sitemap, profile, example]
source: local
created: 2026-07-06
updated: 2026-07-06
---

# 구매자 프로필 페이지

## 페이지 소개

구매자가 본인 정보, 기본 배송지, 주문 관련 설정을 확인하거나 수정하는 페이지다.

## 스크린샷

![프로필 이동 예시](../../20-ui/.examples/assets/UI_A_01_order_checkout_wireframe/order-checkout-wireframe.svg)

## 연관 사이트맵

```mermaid
flowchart LR
    MyPage["마이페이지"]
    BuyerProfile["PAGE.A.03 구매자 프로필"]
    Checkout["PAGE.A.01 주문 결제"]

    MyPage -->|"프로필 보기"| BuyerProfile
    Checkout -->|"Profile 클릭"| BuyerProfile
    BuyerProfile -->|"배송지 확인 후 복귀"| Checkout

    click Checkout "./PAGE_A_01_order_checkout.md"
```

[PAGE.A.01](./PAGE_A_01_order_checkout.md)

## 연관 태그

🏷️ 요구사항 참조: [REQ.A.01](../../00-requirements/.examples/REQ_A_01_order_checkout.md) | 이동 출발 페이지: [PAGE.A.01](./PAGE_A_01_order_checkout.md)
