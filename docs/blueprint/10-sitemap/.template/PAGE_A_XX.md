---
id: PAGE.A.XX
page_id: PAGE.A.XX
title: 페이지 이름
type: page-spec
status: draft
path: /example/path
parent_pages: []
child_pages: []
actors: []
tags: [page, sitemap, ui, ddd]
source: local
created: YYYY-MM-DD
updated: YYYY-MM-DD
---

# 페이지 이름

## 페이지 소개

이 페이지가 사용자에게 어떤 화면이고, 어떤 판단이나 행동을 위해 존재하는지 한두 문장으로 설명한다.

## 스크린샷

![페이지 스크린샷](../20-ui/assets/UI_A_XX_name/example.png)

## 연관 사이트맵

```mermaid
flowchart LR
    Parent["PAGE.A.PP 상위 페이지"]
    Current["PAGE.A.XX 현재 페이지"]
    Detail["PAGE.A.YY 이동 가능 페이지"]
    Profile["PAGE.A.ZZ 권한 필요 페이지"]

    Parent -->|"진입"| Current
    Current -->|"항목 선택"| Detail
    Current -->|"보호된 기능 선택 / 로그인 필요"| Profile

    click Parent "./PAGE_A_PP_parent.md"
    click Detail "./PAGE_A_YY_related_detail.md"
    click Profile "./PAGE_A_ZZ_related_profile.md"
```

[PAGE.A.PP](./PAGE_A_PP_parent.md) [PAGE.A.YY](./PAGE_A_YY_related_detail.md) [PAGE.A.ZZ](./PAGE_A_ZZ_related_profile.md)

## 연관 태그

🏷️ 요구사항 참조: [REQ.A.XX](../00-requirements/REQ_A_XX_name.md) | 플로우 참조: FLOW.A.XX | UI 참조: [UI.A.XX](../20-ui/UI_A_XX_name.md) | UC 참조: [UC.A.XX](../30-uc/UC_A_XX_name.md) | 영속성 참조: [SD.A.XX20](../50-service-design/A_XX_name/A_XX_20-persistence/README.md) | 서비스 참조: [SD.A.XX30](../50-service-design/A_XX_name/A_XX_30-service/README.md) | 시나리오 참조: [SCN.A.XX](../80-sequence/SCN_A_XX_name.md) | API·CLI 참조: [SD.A.XX40](../50-service-design/A_XX_name/A_XX_40-api/README.md)
