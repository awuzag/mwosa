---
id: UC.A.XX
title: 유스케이스 이름
type: use-case
status: draft
tags: [use-case, flowchart, ddd]
source: local
created: YYYY-MM-DD
updated: YYYY-MM-DD
---

# 유스케이스 이름

## 기본 정보

- UC ID: `UC.A.XX`
- 사용자:
- 기준 페이지:
- 기준 기능:
- 제외 범위:

## 연관 태그

- 🏷️ 플로우 참조: FLOW.A.XX
- 🏷️ 요구사항 참조: [REQ.A.XX](../00-requirements/REQ_A_XX_name.md)
- 🏷️ 페이지 참조: [PAGE.A.XX](../10-sitemap/PAGE_A_XX_name.md)
- 🏷️ UI 참조: [UI.A.XX](../20-ui/UI_A_XX_name.md)
- 🏷️ 영속성 참조: [SD.A.XX20](../50-service-design/A_XX_name/A_XX_20-persistence/README.md)
- 🏷️ 서비스 참조: [SD.A.XX30](../50-service-design/A_XX_name/A_XX_30-service/README.md)
- 🏷️ 시나리오 참조: [SCN.A.XX](../80-sequence/SCN_A_XX_name.md)
- 🏷️ API·CLI 참조: [SD.A.XX40](../50-service-design/A_XX_name/A_XX_40-api/README.md)

## 유스케이스

```mermaid
flowchart LR
    ActorA["액터 A"]
    ActorB["액터 B"]

    subgraph SystemBoundary["시스템 바운더리 이름"]
        UCXX01(("UC.A.XX-01<br/>사용자 목표"))
        UCXX02(("UC.A.XX-02<br/>상위 사용자 목표"))
        UCXX03(("UC.A.XX-03<br/>포함 사용자 목표"))
        UCXX04(("UC.A.XX-04<br/>확장 사용자 목표"))
    end

    ActorA --> UCXX01
    ActorA --> UCXX02
    ActorB --> UCXX04

    UCXX02 -. "include" .-> UCXX03
    UCXX02 -. "extend" .-> UCXX04
```

## 사용자 목표

| UC ID | 액터 | 사용자 목표 | 설명 | 연결 요구사항 |
| --- | --- | --- | --- | --- |
| `UC.A.XX-01` | 액터 A | 사용자 목표 | 사용자가 달성하려는 목표를 명사형으로 적는다. | `REQ.A.XX.FR-001` |
| `UC.A.XX-02` | 액터 A | 상위 사용자 목표 | 여러 선택지가 모이는 상위 목표를 적는다. | `REQ.A.XX.FR-002` |
| `UC.A.XX-03` | 액터 A | 포함 사용자 목표 | 상위 목표에 항상 포함되는 목표를 적는다. | `REQ.A.XX.FR-003` |
| `UC.A.XX-04` | 액터 B | 확장 사용자 목표 | 조건에 따라 확장되는 목표를 적는다. | `REQ.A.XX.FR-004` |
