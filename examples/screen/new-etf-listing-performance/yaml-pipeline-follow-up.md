# YAML Pipeline Follow-Up

이 use case 는 현재 `yaml_pipeline` 만으로 동일하게 표현하기 어렵다.

필요한 계산은 symbol 별 전체 일봉을 그룹으로 보고 다음 값을 한 번에 만들어야 한다.

- 저장된 첫 일봉 날짜
- 첫 일봉 시가와 종가
- 최신 일봉 종가
- 첫 일봉 대비 최신 종가 수익률
- 최근 5일/20일 평균 거래대금

현재 pipeline 은 `source.daily_bars`, `transform.window_metrics`,
`transform.latest_per_symbol`, `filter.field`, `rank.by_field` 를 제공하지만,
"첫 row per symbol" 과 "첫 row + 최신 row 를 합친 listing proxy metric" 을
생성하는 selector 는 없다.

후속으로 추가할 만한 작은 selector 후보:

- `transform.first_per_symbol`: symbol 별 가장 이른 row 를 남긴다.
- `transform.listing_proxy_metrics`: daily bars 에서 `first_date`,
  `first_open`, `first_close`, `latest_date`, `latest_close`,
  `return_from_first_open_pct`, `return_from_first_close_pct` 를 만든다.

그 전까지 이 예제는 jq 기반 spec 으로 유지하는 편이 맞다. jq는 그룹핑, 첫 row와
최신 row 추출, 수익률 계산, flat CSV row 생성에 강하고, YAML pipeline 은 이미
계산된 후보군을 검증 가능한 selector/filter/rank 흐름으로 반복 실행할 때 더
적합하다.

