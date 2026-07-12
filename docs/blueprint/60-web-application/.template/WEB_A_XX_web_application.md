---
id: WEB.A.XX
title: 웹 애플리케이션 설계 이름
type: web-application-design
status: draft
tags: [web-application, frontend, nextjs]
source: local
created: YYYY-MM-DD
updated: YYYY-MM-DD
page_ids: []
ui_ids: []
---

# 웹 애플리케이션 설계 이름

## 기본 정보

- Web Application ID: `WEB.A.XX`
- 적용 route:
- 적용 액터:
- 연결 Page:
- 연결 UI:
- 기준 프레임워크:

## 설계 목표

- 이 설계로 해결할 웹 구현 문제를 적는다.
- 사용자 경험, 배포, 성능, 보안 가운데 중요한 판단 기준을 적는다.

## 범위와 경계

### 포함 범위

-

### 제외 범위

-

### 원천 문서

| 판단 대상 | 기준 문서 | 이 문서에서 결정할 내용 |
| --- | --- | --- |
| 페이지 목적과 이동 | [PAGE.A.XX](../../10-sitemap/PAGE_A_XX_name.md) | route와 layout 배치 |
| 화면 구성과 상태 | [UI.A.XX](../../20-ui/UI_A_XX_name.md) | 컴포넌트와 상태 화면 연결 |
| 사용자 목표 | [UC.A.XX](../../30-uc/UC_A_XX_name.md) | 사용자 행동 진입점 |
| API 계약 | [서비스 상세 설계](../../50-service-design/README.md) | 조회와 mutation 호출 경계 |

## Route와 PAGE/UI 연결

| Route group | URL | Page 파일 | PAGE | UI | 접근 조건 |
| --- | --- | --- | --- | --- | --- |
|  |  |  |  |  |  |

## App Router 구조

```text
src/app/
└── (actor)/
    └── example/
        ├── layout.tsx
        ├── page.tsx
        ├── loading.tsx
        ├── error.tsx
        ├── not-found.tsx
        └── _components/
```

### 폴더 책임

| 위치 | 책임 | 소유하지 않는 것 |
| --- | --- | --- |
| `page.tsx` |  |  |
| `layout.tsx` |  |  |
| `_components` |  |  |
| `features` |  |  |
| `lib/api` |  |  |

## Server Component와 Client Component

| 컴포넌트 | 유형 | 근거 | 입력 데이터 | 사용자 행동 |
| --- | --- | --- | --- | --- |
|  | Server |  |  |  |
|  | Client |  |  |  |

- 독립 조회의 동시 시작 방법:
- `Suspense` 경계:
- Client Component에 전달할 최소 필드:
- Server Action 또는 Route Handler 사용 여부:

## Layout과 반응형 기준

- 공통 셸:
- 작은 화면:
- 넓은 화면:
- keyboard와 focus:
- 액터별 차이:

## 로딩, 오류, 빈 결과

| 상태 | 구현 위치 | 화면 표시 | 재시도 또는 다음 행동 |
| --- | --- | --- | --- |
| 로딩 |  |  |  |
| 오류 |  |  |  |
| 리소스 없음 |  |  |  |
| 빈 결과 |  |  |  |
| 최신성 지연 |  |  |  |

## 상태와 데이터 소유권

| 상태 | 유형 | 소유 위치 | 생명주기 | persistence |
| --- | --- | --- | --- | --- |
|  | 서버 상태 |  |  | 없음 |
|  | URL 상태 | URL | route 생명주기 | 없음 |
|  | 폼 상태 | 폼 컴포넌트 | 제출 또는 이탈까지 | 없음 |
|  | 로컬 UI 상태 | UI 컴포넌트 | 컴포넌트까지 | 없음 |

### 공유 Client 상태 도입 판단

- URL 상태로 둘 수 없는 이유:
- 폼 또는 local state로 끝낼 수 없는 이유:
- query cache 데이터를 복제하지 않는지:
- provider 범위:
- reset 조건:
- persistence와 migration:

## 조회와 Mutation

| 사용자 행동 | 조회 또는 Mutation | API | 멱등성 | 완료 뒤 갱신 대상 |
| --- | --- | --- | --- | --- |
|  |  |  |  |  |

## 캐시, 재시도, 무효화

- server cache 대상:
- client query cache 대상:
- cache하지 않을 데이터:
- query key:
- polling 종료 조건:
- 자동 재시도 허용 조건:
- mutation 성공 뒤 무효화 대상:
- 서버 기준 최신 시각 필드:

## 인증과 보안

- session 확인 위치:
- 권한 확인 위치:
- 민감 정보 최소화:
- 로그인 후 복귀 위치 검증:
- 감사 또는 trace 정보:

## 관찰 가능성

- 브라우저 오류:
- 서버 렌더링 오류:
- API latency:
- trace ID 전달:
- 사용자 경험 지표:

## 앱 분리 판단

| 분리 신호 | 현재 상태 | 측정 근거 | 결정 |
| --- | --- | --- | --- |
| 배포 주기 |  |  |  |
| 보안 경계 |  |  |  |
| 장애 영향 |  |  |  |
| 번들과 성능 |  |  |  |

## 검증 기준

-

## 연관 태그

🏷️ 요구사항 참조: [REQ.A.XX](../../00-requirements/REQ_A_XX_name.md) | 페이지 참조: [PAGE.A.XX](../../10-sitemap/PAGE_A_XX_name.md) | UI 참조: [UI.A.XX](../../20-ui/UI_A_XX_name.md) | UC 참조: [UC.A.XX](../../30-uc/UC_A_XX_name.md) | 서비스 참조: [서비스 상세 설계](../../50-service-design/README.md) | 시나리오 참조: [SCN.A.XX](../../80-sequence/SCN_A_XX_name.md)

## 확인 필요

-
