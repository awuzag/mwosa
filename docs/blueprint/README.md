# mwosa Blueprint

이 폴더는 mwosa의 기능을 요구사항부터 처리 시퀀스까지 같은 식별자로 연결해
설계하기 위한 문서 골격이다. DropMong archive에서 검증한 Blueprint의 폴더
구조, 템플릿, 예제만 가져왔으며 DropMong의 실제 설계 문서는 포함하지 않는다.

## 사용 순서

1. `00-requirements`에서 문제, 범위, 수용 기준을 정한다.
2. 사용자 화면이 있는 기능은 `10-sitemap`과 `20-ui`에서 화면 목적과 근거를
   정리한다. CLI 전용 기능이면 두 단계를 생략할 수 있다.
3. `30-uc`에서 사용자가 달성하려는 목표와 예외를 정리한다.
4. `40-event-storming-bounded-context`에서 도메인 사건과 책임 경계를 찾는다.
5. `50-service-design`에서 도메인 모델, 영속성, 서비스, API·CLI 계약을
   구체화한다.
6. 웹 애플리케이션이 필요한 기능은 `60-web-application`에서 구현 경계를
   정리한다.
7. `80-sequence`에서 CLI, 서비스, provider, storage 사이의 처리 순서를
   검증한다.

## 폴더 구조

```text
docs/blueprint/
├── 00-requirements/
├── 10-sitemap/
├── 20-ui/
├── 30-uc/
├── 40-event-storming-bounded-context/
├── 50-service-design/
├── 60-web-application/
└── 80-sequence/
```

각 단계는 다음 규칙을 따른다.

- `.template/`: 새 문서를 시작할 때 복사하는 빈 양식
- `.examples/`: 작성 방식만 보여 주는 예제이며 실제 mwosa 설계의 기준이 아님
- `README.md`: 해당 단계의 역할과 문서 인덱스
- 실제 설계 문서: 숨김 디렉터리 밖에 두고 도메인 또는 기능 식별자로 묶음

## 식별자

문서 안에서는 점 표기, 파일명에서는 밑줄 표기를 사용한다.

| 접두사 | 의미 | 식별자 | 파일명 예시 |
| --- | --- | --- | --- |
| `REQ` | 요구사항 | `REQ.A.01` | `REQ_A_01_aggregate_execution.md` |
| `PAGE` | 페이지 또는 화면 묶음 | `PAGE.A.01` | `PAGE_A_01_run_history.md` |
| `UI` | UI 근거 | `UI.A.01` | `UI_A_01_run_history.md` |
| `UC` | 사용자 목표 | `UC.A.01` | `UC_A_01_run_aggregate.md` |
| `BC` | 바운디드 컨텍스트 | `BC.A.01` | `BC_A_01_aggregate.md` |
| `SD` | 서비스 상세 설계 | `SD.A.01` | `A_01_aggregate/README.md` |
| `AGG` | 도메인 Aggregate | `AGG.A.01` | `AGG_A_01_aggregate_run.md` |
| `PST` | 영속성 설계 | `PST.A.01` | `PST_A_01_aggregate_run.md` |
| `SVC` | 서비스 설계 | `SVC.A.01` | `SVC_A_01_aggregate_executor.md` |
| `API` | API 또는 명령 계약 | `API.A.01` | `API_A_01_run_aggregate.md` |
| `SCN` | 처리 시나리오 | `SCN.A.01` | `SCN_A_01_run_aggregate.md` |
| `WEB` | 웹 애플리케이션 설계 | `WEB.A.01` | `WEB_A_01_run_history.md` |
| `BFF` | 웹 BFF 모듈 | `BFF.A.01` | `BFF_A_01_run_history.md` |

알파벳은 기능 묶음을 나타낸다. 같은 기능은 요구사항부터 시퀀스까지 같은
알파벳과 번호를 유지한다. 예제와 실제 문서가 충돌할 수 있으면 예제에는
`EX` 그룹을 사용한다.

## 서비스 설계 폴더

하나의 도메인 또는 바운디드 컨텍스트는 다음 네 영역으로 나눈다.

```text
50-service-design/
└── A_01_aggregate/
    ├── README.md
    ├── A_01_10-domain-model/
    ├── A_01_20-persistence/
    ├── A_01_30-service/
    └── A_01_40-api/
```

API 영역에는 HTTP API만 두지 않는다. mwosa처럼 CLI가 주 사용자 경계인
프로젝트에서는 명령, 입력 옵션, 출력 형식, 오류 계약도 같은 영역에서
설계할 수 있다.

## 원천 문서 원칙

- 요구사항은 사용자 문제, 제약, 수용 기준을 소유한다.
- 유스케이스는 사용자의 목표와 사전 조건을 소유한다.
- 바운디드 컨텍스트는 용어와 책임 경계를 소유한다.
- 서비스 상세 설계는 도메인 규칙과 기술 계약을 소유한다.
- 시퀀스는 여러 구성 요소의 상호작용만 설명하고 앞 단계 내용을 복제하지
  않는다.
- 기존 아키텍처 계약은 `docs/architectures`가 계속 소유한다. Blueprint는
  해당 문서를 링크하며 같은 결정을 별도 기준으로 다시 만들지 않는다.

## 시작 방법

각 단계의 `.template` 문서를 실제 파일명으로 복사한 다음 자리표시자를
바꾼다. 전체 작성 형태는 `.examples`에서 확인한다. 예제의 주문 도메인과
웹 기술 선택은 구조 설명용이며 mwosa의 실제 요구사항으로 간주하지 않는다.
