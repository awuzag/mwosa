# KRX Monthly Aggregate Fixture

`krx-stock-daily-2026-06.zip`은 KRX OPEN API의 `stk_bydd_trd`와
`ksq_bydd_trd` 응답을 2026-06-01부터 2026-06-30까지 수집한 통합 테스트용
fixture다.

- 거래일: 21일
- snapshot: 42개
- 원본 row: 58,144개
- dataset SHA-256:
  `726ffedab13468cf7c018c239e93de827d3c7d0d321a9664755821fab1b77165`

ZIP 내부에는 `manifest.json`과 날짜/API별 JSON이 들어 있다. 테스트는 ZIP을
직접 읽어 각 파일 checksum과 전체 dataset checksum을 확인한 뒤,
`provider_raw_snapshots`에 unordered MongoDB bulk upsert한다. MongoDB data
directory나 dump 파일은 저장하지 않는다.

월간 집계는 정렬과 계산에 원화 단위 `total_traded_amount`를 사용한다. 마지막
`derived` jq 단계에서 `total_traded_amount_100m`을 추가하고 table에는
`월간 거래대금(억)`으로 표시한다. JSON에는 원화값과 억 단위 파생값이 함께
남는다. 다른 단위나 비율도 실행 엔진에 전용 옵션을 추가하지 않고 같은 파생
단계에서 확장한다.

## 재수집

기존 archive는 기본적으로 덮어쓰지 않는다. 명시적으로 교체할 때만
`KRX_FIXTURE_OVERWRITE=true`를 사용한다.

```bash
MWOSA_CONFIG="$HOME/.config/mwosa/config.json" \
KRX_FIXTURE_FROM=2026-06-01 \
KRX_FIXTURE_TO=2026-06-30 \
KRX_FIXTURE_OVERWRITE=true \
  task fixture:krx:collect
```

## 통합 테스트

```bash
task test:integration:krx-fixture
```

이 archive는 테스트 재현용이다. 외부 배포나 공개 저장소 반영 전에는 KRX 원본
데이터 재배포 조건을 별도로 확인한다.
