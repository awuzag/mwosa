# impressive-krx-candidates Aggregate

KIS `inquire-daily-itemchartprice` raw 호출, 로컬 `instruments`,
`valuation_snapshots`, MongoDB aggregation, jq shaping, table layout을 한 파일에
묶은 Aggregate 예시다.

현재 예시는 MVP 검증용 spec이다.

```bash
mwosa validate aggregate examples/aggregate/impressive-krx-candidates/impressive-krx-candidates.aggregate.yaml
mwosa update aggregate impressive-krx-candidates --file examples/aggregate/impressive-krx-candidates/impressive-krx-candidates.aggregate.yaml
mwosa inspect aggregate-plan impressive-krx-candidates --param limit=10
```

`provider_raw` stage는 저장된 snapshot 조회가 아니라 live KIS API 호출이다.
KIS credential이 없는 환경에서는 provider 연동 smoke가 skip되어야 하며, key나
token 값은 출력하지 않는다.
