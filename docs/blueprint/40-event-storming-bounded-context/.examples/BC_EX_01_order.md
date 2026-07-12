---
id: BC.EX.01
title: Context 주문 이벤트스토밍과 바운디드 컨텍스트
type: event-storming-bounded-context
status: example
tags: [ddd, event-storming, bounded-context, order, example]
source: local
created: 2026-07-06
updated: 2026-07-10
---

# Context 주문 이벤트스토밍과 바운디드 컨텍스트

## 기본 정보

- BC ID: `BC.EX.01`
- 책임: 주문 생성, 주문 상태 관리, 주문 금액 스냅샷 보존.
- 사용자: 구매자, 운영자.
- 핵심 용어: 주문, 주문 라인, 주문 금액, 주문 상태.
- 제외 책임: 결제 승인, 상품 재고 원장, 쿠폰 발급.

## 연관 태그

- 🏷️ 요구사항 참조: [REQ.A.01](../../00-requirements/.examples/REQ_A_01_order_checkout.md)
- 🏷️ UC 참조: [UC.A.01](../../30-uc/.examples/UC_A_01_place_order.md)
- 🏷️ 서비스 상세 설계 참조: [SD.A.01](../../50-service-design/.examples/order/README.md)
- 🏷️ 도메인 참조: [AGG.A.01](../../50-service-design/.examples/order/A_01_10-domain-model/AGG_A_01_order.md)
- 🏷️ 영속성 참조: [PST.A.01](../../50-service-design/.examples/order/A_01_20-persistence/PST_A_01_order_persistence.md)
- 🏷️ 서비스 참조: [SVC.A.01](../../50-service-design/.examples/order/A_01_30-service/SVC_A_01_order_service.md)
- 🏷️ API 참조: [API.A.01](../../50-service-design/.examples/order/A_01_40-api/API_A_01_place_order.md)
- 🏷️ 시나리오 참조: [SCN.A.01](../../80-sequence/.examples/SCN_A_01_place_order.md)

## 컨텍스트 경계

- 이 BC가 결정하는 것: 주문 생성 가능 여부, 주문 상태, 주문 금액 스냅샷.
- 이 BC가 참조만 하는 것: 상품명, 상품 가격, 쿠폰 할인 결과, 배송지 요약.
- 다른 BC에 위임하는 것: 결제 승인, 쿠폰 유효성 최종 판정, 재고 차감.

## Event Storming Diagram

```mermaid
flowchart LR
  buyer[Actor<br>구매자]
  payment[External System<br>결제 시스템]

  subgraph OrderBC[Context 주문]
    direction LR
    cmdCreate[Command<br>주문 생성]
    order[Aggregate<br>Order]
    evtCreated[Domain Event<br>주문 생성됨]
    policyAmount[Policy<br>주문 금액 변조 차단]
    ruleSnapshot[Business Rule<br>주문 금액 스냅샷 보존]
    hotspotCoupon[Hotspot<br>쿠폰 검증 시점]
    rmSummary[Read Model<br>주문 결제 요약]
  end

  buyer --> cmdCreate
  cmdCreate --> order
  order --> evtCreated
  policyAmount -.-> cmdCreate
  ruleSnapshot -.-> order
  hotspotCoupon -.-> policyAmount
  order --> payment
  order --> rmSummary

  classDef actor fill:#FFFFFF,stroke:#111827,color:#111827,stroke-width:2px;
  classDef command fill:#DBEAFE,stroke:#111827,color:#111827,stroke-width:2px;
  classDef aggregate fill:#FEF3C7,stroke:#111827,color:#111827,stroke-width:2px;
  classDef event fill:#FED7AA,stroke:#111827,color:#111827,stroke-width:2px;
  classDef policy fill:#E9D5FF,stroke:#111827,color:#111827,stroke-width:2px;
  classDef rule fill:#DCFCE7,stroke:#111827,color:#111827,stroke-width:2px;
  classDef hotspot fill:#FEE2E2,stroke:#111827,color:#111827,stroke-width:2px;
  classDef external fill:#FCE7F3,stroke:#111827,color:#111827,stroke-width:2px;
  classDef readmodel fill:#F1F5F9,stroke:#111827,color:#111827,stroke-width:2px;

  class buyer actor;
  class cmdCreate command;
  class order aggregate;
  class evtCreated event;
  class policyAmount policy;
  class ruleSnapshot rule;
  class hotspotCoupon hotspot;
  class payment external;
  class rmSummary readmodel;
  style OrderBC fill:#F9FAFB,stroke:#111827,color:#111827,stroke-width:2px;
```

## Element Catalog

| 유형 | 식별자 | 이름 | 소속 컨텍스트 | 설명 |
| --- | --- | --- | --- | --- |
| Actor | ACTOR.EX.01 | 구매자 | Context 외부 | 주문 결제 페이지에서 주문 생성을 요청한다. |
| Command | CMD.EX.01 | 주문 생성 | Context 주문 | 주문 생성 요청을 처리한다. |
| Aggregate | AGG.EX.01 | Order | Context 주문 | 주문 라인, 주문 금액 스냅샷, 주문 상태를 보존한다. |
| Domain Event | EVT.EX.01 | 주문 생성됨 | Context 주문 | 주문이 생성된 뒤 발행된다. |
| Policy | POLICY.EX.01 | 주문 금액 변조 차단 | Context 주문 | 클라이언트가 보낸 금액을 신뢰하지 않고 서버 계산 금액만 허용한다. |
| Business Rule | RULE.EX.01 | 주문 금액 스냅샷 보존 | Context 주문 | 주문 생성 시점의 금액과 주문 라인을 이후 변경과 분리해 보존한다. |
| Hotspot | HOTSPOT.EX.01 | 쿠폰 검증 시점 | Context 주문 | 쿠폰 할인 결과를 주문 생성 전 조회 모델에 포함할지 결정이 필요하다. |
| External System | EXT.EX.01 | 결제 시스템 | Context 외부 | 결제 승인과 실패 응답을 제공한다. |
| Read Model | RM.EX.01 | 주문 결제 요약 | Context 주문 | 주문 결제 화면에 필요한 요약 정보를 제공한다. |

## Element Evidence

| 요소 | 근거 문서 | 근거 내용 |
| --- | --- | --- |
| ACTOR.EX.01 구매자 | [UC.A.01](../../30-uc/.examples/UC_A_01_place_order.md) | 구매자가 주문 결제 페이지에서 주문을 확정하는 주체다. |
| CMD.EX.01 주문 생성 | [UC.A.01](../../30-uc/.examples/UC_A_01_place_order.md), [API.A.01](../../50-service-design/.examples/order/A_01_40-api/API_A_01_place_order.md) | 주문 확정 행위가 주문 생성 요청으로 이어진다. |
| AGG.EX.01 Order | [REQ.A.01](../../00-requirements/.examples/REQ_A_01_order_checkout.md), [UC.A.01](../../30-uc/.examples/UC_A_01_place_order.md) | 주문 라인, 주문 금액, 주문 상태를 보존해야 한다. |
| EVT.EX.01 주문 생성됨 | [UC.A.01](../../30-uc/.examples/UC_A_01_place_order.md) | 주문 확정 이후 주문 생성 결과가 남는다. |
| POLICY.EX.01 주문 금액 변조 차단 | [REQ.A.01](../../00-requirements/.examples/REQ_A_01_order_checkout.md), [UC.A.01](../../30-uc/.examples/UC_A_01_place_order.md) | 주문 생성 시 클라이언트 입력 금액을 그대로 신뢰하면 안 된다. |
| RULE.EX.01 주문 금액 스냅샷 보존 | [REQ.A.01](../../00-requirements/.examples/REQ_A_01_order_checkout.md), [UC.A.01](../../30-uc/.examples/UC_A_01_place_order.md) | 주문 후에도 당시 주문 금액과 주문 라인을 재현할 수 있어야 한다. |
| HOTSPOT.EX.01 쿠폰 검증 시점 | [REQ.A.01](../../00-requirements/.examples/REQ_A_01_order_checkout.md), [UC.A.01](../../30-uc/.examples/UC_A_01_place_order.md) | 쿠폰 할인 결과를 조회 모델에 포함할지 동기 검증으로만 처리할지 결정해야 한다. |
| EXT.EX.01 결제 시스템 | [REQ.A.01](../../00-requirements/.examples/REQ_A_01_order_checkout.md) | 결제 승인은 주문 BC 내부 책임이 아니라 외부 시스템 연동이다. |
| RM.EX.01 주문 결제 요약 | [UC.A.01](../../30-uc/.examples/UC_A_01_place_order.md) | 주문 전 금액, 주문 라인, 할인 결과를 확인해야 한다. |

## Event Relations

| 출발 | 관계 | 도착 | 설명 |
| --- | --- | --- | --- |
| 구매자 | 요청한다 | 주문 생성 | 구매자가 주문 확정 의사를 전달한다. |
| 주문 생성 | 변경한다 | Order | 주문 생성 가능 여부와 주문 금액 스냅샷을 결정한다. |
| Order | 발행한다 | 주문 생성됨 | 주문 상태와 금액이 확정된 뒤 이벤트를 발행한다. |
| 주문 금액 변조 차단 | 제한한다 | 주문 생성 | 서버 계산 금액과 다른 주문 생성 요청은 허용하지 않는다. |
| 주문 금액 스냅샷 보존 | 규정한다 | Order | 주문 생성 시점의 금액과 라인을 별도 스냅샷으로 남긴다. |
| 쿠폰 검증 시점 | 표시한다 | 주문 금액 변조 차단 | 쿠폰 할인 결과를 언제 확정할지 추가 결정이 필요하다. |
| Order | 요청한다 | 결제 시스템 | 외부 결제 승인 결과를 기다린다. |
| Order | 투영한다 | 주문 결제 요약 | 주문 생성과 결제 화면에 필요한 금액·상태 요약을 제공한다. |

## 유비쿼터스 언어

| 용어 | 의미 | 혼동하기 쉬운 용어 |
| --- | --- | --- |
| 주문 | 구매자가 상품 구매 의사를 확정한 기록 | 결제 |
| 주문 라인 | 주문에 포함된 상품 스냅샷 | 장바구니 항목 |
| 주문 금액 | 주문 생성 시점에 확정한 서버 계산 금액 | 화면 표시 금액 |

## 후속 설계 메모

| 항목 | 메모 | 연결 문서 |
| --- | --- | --- |
| 도메인 모델 | `Order`, 주문 라인, 주문 금액 스냅샷, 주문 상태를 상세화한다. | [AGG.A.01](../../50-service-design/.examples/order/A_01_10-domain-model/AGG_A_01_order.md) |
| 영속성 | 주문 저장소, 주문 상태 변경 트랜잭션, 결제 연동 outbox 여부를 검토한다. | [PST.A.01](../../50-service-design/.examples/order/A_01_20-persistence/PST_A_01_order_persistence.md) |
| 서비스 | 주문 생성 유스케이스와 주문 가능성 검증 위치를 정리한다. | [SVC.A.01](../../50-service-design/.examples/order/A_01_30-service/SVC_A_01_order_service.md) |
| API | 주문 생성 요청/응답과 실패 응답을 정리한다. | [API.A.01](../../50-service-design/.examples/order/A_01_40-api/API_A_01_place_order.md) |
| 발행 Event | `EVT.EX.01` |  |
| 구독 Event | 결제 승인/실패 이벤트 구독 여부를 결정한다. |  |
| 외부 연동 | 결제 시스템 승인 요청과 실패 응답 처리를 정리한다. |  |
| 정책/불변조건 | 주문 생성 전 상품 상태, 금액, 쿠폰 할인 결과를 검증한다. |  |
| 열린 질문 | 쿠폰 할인 결과를 주문 생성 전 조회 모델에 포함할지, 주문 생성 시 동기 검증으로만 처리할지 결정해야 한다. |  |
