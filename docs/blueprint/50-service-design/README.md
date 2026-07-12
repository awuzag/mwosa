# 서비스 상세 설계

바운디드 컨텍스트 또는 독립된 기능 단위로 도메인 모델, 영속성, 서비스,
API·CLI 계약을 설계한다.

## 폴더 구성

```text
<context>/
├── README.md
├── <context_id>_10-domain-model/
├── <context_id>_20-persistence/
├── <context_id>_30-service/
└── <context_id>_40-api/
```

- 템플릿 인덱스: [.template/README.md](.template/README.md)
- 예제 인덱스: [.examples/order/README.md](.examples/order/README.md)

`README.md`는 컨텍스트의 범위와 하위 문서 링크만 관리한다. 도메인 규칙,
스키마, 서비스 처리와 계약은 각각의 하위 폴더가 소유한다.
