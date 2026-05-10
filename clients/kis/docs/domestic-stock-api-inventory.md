# KIS Domestic Stock OpenAPI Inventory

이 문서는 KIS Developers 포털에서 수집한 국내주식 API 문서 인벤토리다.
상세 요청/응답 필드와 예시는 `domestic-stock-openapi-catalog.json` 에 보관한다.

- 수집 시각: `2026-05-10T06:52:27Z`
- 원천: `https://apiportal.koreainvestment.com/apiservice-category`
- 국내주식 API 수: `186`
- OAuth 포함 총 API 수: `189`

## Collection Summary

| scope | collection | count | methods | realtime tryitout |
| --- | --- | ---: | --- | ---: |
| `sdk_dependency` | OAuth인증 | 3 | `POST:3` | 0 |
| `domestic_stock` | [국내주식] 주문/계좌 | 23 | `GET:18, POST:5` | 0 |
| `domestic_stock` | [국내주식] 기본시세 | 21 | `GET:21` | 0 |
| `domestic_stock` | [국내주식] ELW 시세 | 22 | `GET:22` | 0 |
| `domestic_stock` | [국내주식] 업종/기타 | 14 | `GET:14` | 0 |
| `domestic_stock` | [국내주식] 종목정보 | 26 | `GET:26` | 0 |
| `domestic_stock` | [국내주식] 시세분석 | 29 | `GET:29` | 0 |
| `domestic_stock` | [국내주식] 순위분석 | 22 | `GET:22` | 0 |
| `domestic_stock` | [국내주식] 실시간시세 | 29 | `POST:29` | 29 |

## OAuth인증

| name | method | access_url | real_tr_id | virtual_tr_id |
| --- | --- | --- | --- | --- |
| 접근토큰발급(P)[인증-001] | `POST` | `/oauth2/tokenP` | `` | `` |
| 접근토큰폐기(P)[인증-002] | `POST` | `/oauth2/revokeP` | `` | `` |
| 실시간 (웹소켓) 접속키 발급[실시간-000] | `POST` | `/oauth2/Approval` | `` | `` |

## [국내주식] 주문/계좌

| name | method | access_url | real_tr_id | virtual_tr_id |
| --- | --- | --- | --- | --- |
| 주식주문(현금)[v1_국내주식-001] | `POST` | `/uapi/domestic-stock/v1/trading/order-cash` | `(매도) TTTC0011U (매수) TTTC0012U` | `(매도) VTTC0011U (매수) VTTC0012U` |
| 주식주문(신용)[v1_국내주식-002] | `POST` | `/uapi/domestic-stock/v1/trading/order-credit` | `(매도) TTTC0051U (매수) TTTC0052U` | `모의투자 미지원` |
| 주식주문(정정취소)[v1_국내주식-003] | `POST` | `/uapi/domestic-stock/v1/trading/order-rvsecncl` | `TTTC0013U` | `VTTC0013U` |
| 주식정정취소가능주문조회[v1_국내주식-004] | `GET` | `/uapi/domestic-stock/v1/trading/inquire-psbl-rvsecncl` | `TTTC0084R` | `모의투자 미지원` |
| 주식일별주문체결조회[v1_국내주식-005] | `GET` | `/uapi/domestic-stock/v1/trading/inquire-daily-ccld` | `(3개월이내) TTTC0081R (3개월이전) CTSC9215R` | `(3개월이내) VTTC0081R (3개월이전) VTSC9215R` |
| 주식잔고조회[v1_국내주식-006] | `GET` | `/uapi/domestic-stock/v1/trading/inquire-balance` | `TTTC8434R` | `VTTC8434R` |
| 매수가능조회[v1_국내주식-007] | `GET` | `/uapi/domestic-stock/v1/trading/inquire-psbl-order` | `TTTC8908R` | `VTTC8908R` |
| 매도가능수량조회 [국내주식-165] | `GET` | `/uapi/domestic-stock/v1/trading/inquire-psbl-sell` | `TTTC8408R` | `모의투자 미지원` |
| 신용매수가능조회[v1_국내주식-042] | `GET` | `/uapi/domestic-stock/v1/trading/inquire-credit-psamount` | `TTTC8909R` | `모의투자 미지원` |
| 주식예약주문[v1_국내주식-017] | `POST` | `/uapi/domestic-stock/v1/trading/order-resv` | `CTSC0008U` | `모의투자 미지원` |
| 주식예약주문정정취소[v1_국내주식-018,019] | `POST` | `/uapi/domestic-stock/v1/trading/order-resv-rvsecncl` | `(예약취소) CTSC0009U (예약정정) CTSC0013U` | `모의투자 미지원` |
| 주식예약주문조회[v1_국내주식-020] | `GET` | `/uapi/domestic-stock/v1/trading/order-resv-ccnl` | `CTSC0004R` | `모의투자 미지원` |
| 퇴직연금 체결기준잔고[v1_국내주식-032] | `GET` | `/uapi/domestic-stock/v1/trading/pension/inquire-present-balance` | `TTTC2202R` | `모의투자 미지원` |
| 퇴직연금 미체결내역[v1_국내주식-033] | `GET` | `/uapi/domestic-stock/v1/trading/pension/inquire-daily-ccld` | `TTTC2201R(기존 KRX만 가능), TTTC2210R (KRX,NXT/SOR)` | `모의투자 미지원` |
| 퇴직연금 매수가능조회[v1_국내주식-034] | `GET` | `/uapi/domestic-stock/v1/trading/pension/inquire-psbl-order` | `TTTC0503R` | `모의투자 미지원` |
| 퇴직연금 예수금조회[v1_국내주식-035] | `GET` | `/uapi/domestic-stock/v1/trading/pension/inquire-deposit` | `TTTC0506R` | `모의투자 미지원` |
| 퇴직연금 잔고조회[v1_국내주식-036] | `GET` | `/uapi/domestic-stock/v1/trading/pension/inquire-balance` | `TTTC2208R` | `모의투자 미지원` |
| 주식잔고조회_실현손익[v1_국내주식-041] | `GET` | `/uapi/domestic-stock/v1/trading/inquire-balance-rlz-pl` | `TTTC8494R` | `모의투자 미지원` |
| 투자계좌자산현황조회[v1_국내주식-048] | `GET` | `/uapi/domestic-stock/v1/trading/inquire-account-balance` | `CTRP6548R` | `모의투자 미지원` |
| 기간별손익일별합산조회[v1_국내주식-052] | `GET` | `/uapi/domestic-stock/v1/trading/inquire-period-profit` | `TTTC8708R` | `모의투자 미지원` |
| 기간별매매손익현황조회[v1_국내주식-060] | `GET` | `/uapi/domestic-stock/v1/trading/inquire-period-trade-profit` | `TTTC8715R` | `모의투자 미지원` |
| 주식통합증거금 현황 [국내주식-191] | `GET` | `/uapi/domestic-stock/v1/trading/intgr-margin` | `TTTC0869R` | `모의투자 미지원` |
| 기간별계좌권리현황조회 [국내주식-211] | `GET` | `/uapi/domestic-stock/v1/trading/period-rights` | `CTRGA011R` | `모의투자 미지원` |

## [국내주식] 기본시세

| name | method | access_url | real_tr_id | virtual_tr_id |
| --- | --- | --- | --- | --- |
| 주식현재가 시세[v1_국내주식-008] | `GET` | `/uapi/domestic-stock/v1/quotations/inquire-price` | `FHKST01010100` | `FHKST01010100` |
| 주식현재가 시세2[v1_국내주식-054] | `GET` | `/uapi/domestic-stock/v1/quotations/inquire-price-2` | `FHPST01010000` | `모의투자 미지원` |
| 주식현재가 체결[v1_국내주식-009] | `GET` | `/uapi/domestic-stock/v1/quotations/inquire-ccnl` | `FHKST01010300` | `FHKST01010300` |
| 주식현재가 일자별[v1_국내주식-010] | `GET` | `/uapi/domestic-stock/v1/quotations/inquire-daily-price` | `FHKST01010400` | `FHKST01010400` |
| 주식현재가 호가/예상체결[v1_국내주식-011] | `GET` | `/uapi/domestic-stock/v1/quotations/inquire-asking-price-exp-ccn` | `FHKST01010200` | `FHKST01010200` |
| 주식현재가 투자자[v1_국내주식-012] | `GET` | `/uapi/domestic-stock/v1/quotations/inquire-investor` | `FHKST01010900` | `FHKST01010900` |
| 주식현재가 회원사[v1_국내주식-013] | `GET` | `/uapi/domestic-stock/v1/quotations/inquire-member` | `FHKST01010600` | `FHKST01010600` |
| 국내주식기간별시세(일/주/월/년)[v1_국내주식-016] | `GET` | `/uapi/domestic-stock/v1/quotations/inquire-daily-itemchartprice` | `FHKST03010100` | `FHKST03010100` |
| 주식당일분봉조회[v1_국내주식-022] | `GET` | `/uapi/domestic-stock/v1/quotations/inquire-time-itemchartprice` | `FHKST03010200` | `FHKST03010200` |
| 주식일별분봉조회 [국내주식-213] | `GET` | `/uapi/domestic-stock/v1/quotations/inquire-time-dailychartprice` | `FHKST03010230` | `모의투자 미지원` |
| 주식현재가 당일시간대별체결[v1_국내주식-023] | `GET` | `/uapi/domestic-stock/v1/quotations/inquire-time-itemconclusion` | `FHPST01060000` | `FHPST01060000` |
| 주식현재가 시간외일자별주가[v1_국내주식-026] | `GET` | `/uapi/domestic-stock/v1/quotations/inquire-daily-overtimeprice` | `FHPST02320000` | `FHPST02320000` |
| 주식현재가 시간외시간별체결[v1_국내주식-025] | `GET` | `/uapi/domestic-stock/v1/quotations/inquire-time-overtimeconclusion` | `FHPST02310000` | `FHPST02310000` |
| 국내주식 시간외현재가[국내주식-076] | `GET` | `/uapi/domestic-stock/v1/quotations/inquire-overtime-price` | `FHPST02300000` | `모의투자 미지원` |
| 국내주식 시간외호가[국내주식-077] | `GET` | `/uapi/domestic-stock/v1/quotations/inquire-overtime-asking-price` | `FHPST02300400` | `모의투자 미지원` |
| 국내주식 장마감 예상체결가[국내주식-120] | `GET` | `/uapi/domestic-stock/v1/quotations/exp-closing-price` | `FHKST117300C0` | `모의투자 미지원` |
| ETF/ETN 현재가[v1_국내주식-068] | `GET` | `/uapi/etfetn/v1/quotations/inquire-price` | `FHPST02400000` | `모의투자 미지원` |
| ETF 구성종목시세[국내주식-073] | `GET` | `/uapi/etfetn/v1/quotations/inquire-component-stock-price` | `FHKST121600C0` | `모의투자 미지원` |
| NAV 비교추이(종목)[v1_국내주식-069] | `GET` | `/uapi/etfetn/v1/quotations/nav-comparison-trend` | `FHPST02440000` | `모의투자 미지원` |
| NAV 비교추이(일)[v1_국내주식-071] | `GET` | `/uapi/etfetn/v1/quotations/nav-comparison-daily-trend` | `FHPST02440200` | `모의투자 미지원` |
| NAV 비교추이(분)[v1_국내주식-070] | `GET` | `/uapi/etfetn/v1/quotations/nav-comparison-time-trend` | `FHPST02440100` | `모의투자 미지원` |

## [국내주식] ELW 시세

| name | method | access_url | real_tr_id | virtual_tr_id |
| --- | --- | --- | --- | --- |
| ELW 현재가 시세[v1_국내주식-014] | `GET` | `/uapi/domestic-stock/v1/quotations/inquire-elw-price` | `FHKEW15010000` | `FHKEW15010000` |
| ELW 신규상장종목 [국내주식-181] | `GET` | `/uapi/elw/v1/quotations/newly-listed` | `FHKEW154800C0` | `모의투자 미지원` |
| ELW 민감도 순위[국내주식-170] | `GET` | `/uapi/elw/v1/ranking/sensitivity` | `FHPEW02850000` | `모의투자 미지원` |
| ELW 기초자산별 종목시세 [국내주식-186] | `GET` | `/uapi/elw/v1/quotations/udrl-asset-price` | `FHKEW154101C0` | `모의투자 미지원` |
| ELW 종목검색 [국내주식-166] | `GET` | `/uapi/elw/v1/quotations/cond-search` | `FHKEW15100000` | `모의투자 미지원` |
| ELW 당일급변종목[국내주식-171] | `GET` | `/uapi/elw/v1/ranking/quick-change` | `FHPEW02870000` | `모의투자 미지원` |
| ELW 기초자산 목록조회 [국내주식-185] | `GET` | `/uapi/elw/v1/quotations/udrl-asset-list` | `FHKEW154100C0` | `모의투자 미지원` |
| ELW 비교대상종목조회 [국내주식-183] | `GET` | `/uapi/elw/v1/quotations/compare-stocks` | `FHKEW151701C0` | `모의투자 미지원` |
| ELW LP매매추이 [국내주식-182] | `GET` | `/uapi/elw/v1/quotations/lp-trade-trend` | `FHPEW03760000` | `` |
| ELW 투자지표추이(체결) [국내주식-172] | `GET` | `/uapi/elw/v1/quotations/indicator-trend-ccnl` | `FHPEW02740100` | `모의투자 미지원` |
| ELW 투자지표추이(분별) [국내주식-174] | `GET` | `/uapi/elw/v1/quotations/indicator-trend-minute` | `FHPEW02740300` | `모의투자 미지원` |
| ELW 투자지표추이(일별) [국내주식-173] | `GET` | `/uapi/elw/v1/quotations/indicator-trend-daily` | `FHPEW02740200` | `모의투자 미지원` |
| ELW 변동성 추이(틱) [국내주식-180] | `GET` | `/uapi/elw/v1/quotations/volatility-trend-tick` | `FHPEW02840400` | `모의투자 미지원` |
| ELW 변동성추이(체결) [국내주식-177] | `GET` | `/uapi/elw/v1/quotations/volatility-trend-ccnl` | `FHPEW02840100` | `모의투자 미지원` |
| ELW 변동성 추이(일별) [국내주식-178] | `GET` | `/uapi/elw/v1/quotations/volatility-trend-daily` | `FHPEW02840200` | `모의투자 미지원` |
| ELW 민감도 추이(체결) [국내주식-175] | `GET` | `/uapi/elw/v1/quotations/sensitivity-trend-ccnl` | `FHPEW02830100` | `모의투자 미지원` |
| ELW 변동성 추이(분별) [국내주식-179] | `GET` | `/uapi/elw/v1/quotations/volatility-trend-minute` | `FHPEW02840300` | `모의투자 미지원` |
| ELW 민감도 추이(일별) [국내주식-176] | `GET` | `/uapi/elw/v1/quotations/sensitivity-trend-daily` | `FHPEW02830200` | `모의투자 미지원` |
| ELW 만기예정/만기종목 [국내주식-184] | `GET` | `/uapi/elw/v1/quotations/expiration-stocks` | `FHKEW154700C0` | `모의투자 미지원` |
| ELW 지표순위[국내주식-169] | `GET` | `/uapi/elw/v1/ranking/indicator` | `FHPEW02790000` | `모의투자 미지원` |
| ELW 상승률순위[국내주식-167] | `GET` | `/uapi/elw/v1/ranking/updown-rate` | `FHPEW02770000` | `모의투자 미지원` |
| ELW 거래량순위[국내주식-168] | `GET` | `/uapi/elw/v1/ranking/volume-rank` | `FHPEW02780000` | `모의투자 미지원` |

## [국내주식] 업종/기타

| name | method | access_url | real_tr_id | virtual_tr_id |
| --- | --- | --- | --- | --- |
| 국내업종 현재지수[v1_국내주식-063] | `GET` | `/uapi/domestic-stock/v1/quotations/inquire-index-price` | `FHPUP02100000` | `모의투자 미지원` |
| 국내업종 일자별지수[v1_국내주식-065] | `GET` | `/uapi/domestic-stock/v1/quotations/inquire-index-daily-price` | `FHPUP02120000` | `모의투자 미지원` |
| 국내업종 시간별지수(초)[국내주식-064] | `GET` | `/uapi/domestic-stock/v1/quotations/inquire-index-tickprice` | `FHPUP02110100` | `모의투자 미지원` |
| 국내업종 시간별지수(분)[국내주식-119] | `GET` | `/uapi/domestic-stock/v1/quotations/inquire-index-timeprice` | `FHPUP02110200` | `모의투자 미지원` |
| 업종 분봉조회[v1_국내주식-045] | `GET` | `/uapi/domestic-stock/v1/quotations/inquire-time-indexchartprice` | `FHKUP03500200` | `모의투자 미지원` |
| 국내주식업종기간별시세(일/주/월/년)[v1_국내주식-021] | `GET` | `/uapi/domestic-stock/v1/quotations/inquire-daily-indexchartprice` | `FHKUP03500100` | `FHKUP03500100` |
| 국내업종 구분별전체시세[v1_국내주식-066] | `GET` | `/uapi/domestic-stock/v1/quotations/inquire-index-category-price` | `FHPUP02140000` | `모의투자 미지원` |
| 국내주식 예상체결지수 추이[국내주식-121] | `GET` | `/uapi/domestic-stock/v1/quotations/exp-index-trend` | `FHPST01840000` | `모의투자 미지원` |
| 국내주식 예상체결 전체지수[국내주식-122] | `GET` | `/uapi/domestic-stock/v1/quotations/exp-total-index` | `FHKUP11750000` | `모의투자 미지원` |
| 변동성완화장치(VI) 현황 [v1_국내주식-055] | `GET` | `/uapi/domestic-stock/v1/quotations/inquire-vi-status` | `FHPST01390000` | `모의투자 미지원` |
| 금리 종합(국내채권/금리) [국내주식-155] | `GET` | `/uapi/domestic-stock/v1/quotations/comp-interest` | `FHPST07020000` | `모의투자 미지원` |
| 종합 시황/공시(제목) [국내주식-141] | `GET` | `/uapi/domestic-stock/v1/quotations/news-title` | `FHKST01011800` | `모의투자 미지원` |
| 국내휴장일조회[국내주식-040] | `GET` | `/uapi/domestic-stock/v1/quotations/chk-holiday` | `CTCA0903R` | `모의투자 미지원` |
| 국내선물 영업일조회 [국내주식-160] | `GET` | `/uapi/domestic-stock/v1/quotations/market-time` | `HHMCM000002C0` | `모의투자 미지원` |

## [국내주식] 종목정보

| name | method | access_url | real_tr_id | virtual_tr_id |
| --- | --- | --- | --- | --- |
| 상품기본조회[v1_국내주식-029] | `GET` | `/uapi/domestic-stock/v1/quotations/search-info` | `CTPF1604R` | `모의투자 미지원` |
| 주식기본조회[v1_국내주식-067] | `GET` | `/uapi/domestic-stock/v1/quotations/search-stock-info` | `CTPF1002R` | `모의투자 미지원` |
| 국내주식 대차대조표[v1_국내주식-078] | `GET` | `/uapi/domestic-stock/v1/finance/balance-sheet` | `FHKST66430100` | `모의투자 미지원` |
| 국내주식 손익계산서[v1_국내주식-079] | `GET` | `/uapi/domestic-stock/v1/finance/income-statement` | `FHKST66430200` | `모의투자 미지원` |
| 국내주식 재무비율[v1_국내주식-080] | `GET` | `/uapi/domestic-stock/v1/finance/financial-ratio` | `FHKST66430300` | `모의투자 미지원` |
| 국내주식 수익성비율[v1_국내주식-081] | `GET` | `/uapi/domestic-stock/v1/finance/profit-ratio` | `FHKST66430400` | `모의투자 미지원` |
| 국내주식 기타주요비율[v1_국내주식-082] | `GET` | `/uapi/domestic-stock/v1/finance/other-major-ratios` | `FHKST66430500` | `모의투자 미지원` |
| 국내주식 안정성비율[v1_국내주식-083] | `GET` | `/uapi/domestic-stock/v1/finance/stability-ratio` | `FHKST66430600` | `모의투자 미지원` |
| 국내주식 성장성비율[v1_국내주식-085] | `GET` | `/uapi/domestic-stock/v1/finance/growth-ratio` | `FHKST66430800` | `모의투자 미지원` |
| 국내주식 당사 신용가능종목[국내주식-111] | `GET` | `/uapi/domestic-stock/v1/quotations/credit-by-company` | `FHPST04770000` | `모의투자 미지원` |
| 예탁원정보(배당일정)[국내주식-145] | `GET` | `/uapi/domestic-stock/v1/ksdinfo/dividend` | `HHKDB669102C0` | `모의투자 미지원` |
| 예탁원정보(주식매수청구일정)[국내주식-146] | `GET` | `/uapi/domestic-stock/v1/ksdinfo/purreq` | `HHKDB669103C0` | `모의투자 미지원` |
| 예탁원정보(합병/분할일정)[국내주식-147] | `GET` | `/uapi/domestic-stock/v1/ksdinfo/merger-split` | `HHKDB669104C0` | `모의투자 미지원` |
| 예탁원정보(액면교체일정)[국내주식-148] | `GET` | `/uapi/domestic-stock/v1/ksdinfo/rev-split` | `HHKDB669105C0` | `모의투자 미지원` |
| 예탁원정보(자본감소일정)[국내주식-149] | `GET` | `/uapi/domestic-stock/v1/ksdinfo/cap-dcrs` | `HHKDB669106C0` | `모의투자 미지원` |
| 예탁원정보(상장정보일정)[국내주식-150] | `GET` | `/uapi/domestic-stock/v1/ksdinfo/list-info` | `HHKDB669107C0` | `모의투자 미지원` |
| 예탁원정보(공모주청약일정)[국내주식-151] | `GET` | `/uapi/domestic-stock/v1/ksdinfo/pub-offer` | `HHKDB669108C0` | `모의투자 미지원` |
| 예탁원정보(실권주일정)[국내주식-152] | `GET` | `/uapi/domestic-stock/v1/ksdinfo/forfeit` | `HHKDB669109C0` | `모의투자 미지원` |
| 예탁원정보(의무예치일정)[국내주식-153] | `GET` | `/uapi/domestic-stock/v1/ksdinfo/mand-deposit` | `HHKDB669110C0` | `모의투자 미지원` |
| 예탁원정보(유상증자일정) [국내주식-143] | `GET` | `/uapi/domestic-stock/v1/ksdinfo/paidin-capin` | `HHKDB669100C0` | `모의투자 미지원` |
| 예탁원정보(무상증자일정) [국내주식-144] | `GET` | `/uapi/domestic-stock/v1/ksdinfo/bonus-issue` | `HHKDB669101C0` | `모의투자 미지원` |
| 예탁원정보(주주총회일정) [국내주식-154] | `GET` | `/uapi/domestic-stock/v1/ksdinfo/sharehld-meet` | `HHKDB669111C0` | `모의투자 미지원` |
| 국내주식 종목추정실적 [국내주식-187] | `GET` | `/uapi/domestic-stock/v1/quotations/estimate-perform` | `HHKST668300C0` | `모의투자 미지원` |
| 당사 대주가능 종목 [국내주식-195] | `GET` | `/uapi/domestic-stock/v1/quotations/lendable-by-company` | `CTSC2702R` | `모의투자 미지원` |
| 국내주식 종목투자의견 [국내주식-188] | `GET` | `/uapi/domestic-stock/v1/quotations/invest-opinion` | `FHKST663300C0` | `모의투자 미지원` |
| 국내주식 증권사별 투자의견 [국내주식-189] | `GET` | `/uapi/domestic-stock/v1/quotations/invest-opbysec` | `FHKST663400C0` | `모의투자 미지원` |

## [국내주식] 시세분석

| name | method | access_url | real_tr_id | virtual_tr_id |
| --- | --- | --- | --- | --- |
| 종목조건검색 목록조회[국내주식-038] | `GET` | `/uapi/domestic-stock/v1/quotations/psearch-title` | `HHKST03900300` | `모의투자 미지원` |
| 종목조건검색조회 [국내주식-039] | `GET` | `/uapi/domestic-stock/v1/quotations/psearch-result` | `HHKST03900400` | `모의투자 미지원` |
| 관심종목 그룹조회 [국내주식-204] | `GET` | `/uapi/domestic-stock/v1/quotations/intstock-grouplist` | `HHKCM113004C7` | `모의투자 미지원` |
| 관심종목(멀티종목) 시세조회 [국내주식-205] | `GET` | `/uapi/domestic-stock/v1/quotations/intstock-multprice` | `FHKST11300006` | `모의투자 미지원` |
| 관심종목 그룹별 종목조회 [국내주식-203] | `GET` | `/uapi/domestic-stock/v1/quotations/intstock-stocklist-by-group` | `HHKCM113004C6` | `모의투자 미지원` |
| 국내기관_외국인 매매종목가집계[국내주식-037] | `GET` | `/uapi/domestic-stock/v1/quotations/foreign-institution-total` | `FHPTJ04400000` | `모의투자 미지원` |
| 외국계 매매종목 가집계 [국내주식-161] | `GET` | `/uapi/domestic-stock/v1/quotations/frgnmem-trade-estimate` | `FHKST644100C0` | `모의투자 미지원` |
| 종목별 투자자매매동향(일별) | `GET` | `/uapi/domestic-stock/v1/quotations/investor-trade-by-stock-daily` | `FHPTJ04160001` | `모의투자 미지원` |
| 시장별 투자자매매동향(시세)[v1_국내주식-074] | `GET` | `/uapi/domestic-stock/v1/quotations/inquire-investor-time-by-market` | `FHPTJ04030000` | `모의투자 미지원` |
| 시장별 투자자매매동향(일별) [국내주식-075] | `GET` | `/uapi/domestic-stock/v1/quotations/inquire-investor-daily-by-market` | `FHPTJ04040000` | `모의투자 미지원` |
| 종목별 외국계 순매수추이 [국내주식-164] | `GET` | `/uapi/domestic-stock/v1/quotations/frgnmem-pchs-trend` | `FHKST644400C0` | `모의투자 미지원` |
| 회원사 실시간 매매동향(틱) [국내주식-163] | `GET` | `/uapi/domestic-stock/v1/quotations/frgnmem-trade-trend` | `FHPST04320000` | `모의투자 미지원` |
| 주식현재가 회원사 종목매매동향 [국내주식-197] | `GET` | `/uapi/domestic-stock/v1/quotations/inquire-member-daily` | `FHPST04540000` | `모의투자 미지원` |
| 종목별 프로그램매매추이(체결)[v1_국내주식-044] | `GET` | `/uapi/domestic-stock/v1/quotations/program-trade-by-stock` | `FHPPG04650101` | `모의투자 미지원` |
| 종목별 프로그램매매추이(일별) [국내주식-113] | `GET` | `/uapi/domestic-stock/v1/quotations/program-trade-by-stock-daily` | `FHPPG04650201` | `모의투자 미지원` |
| 종목별 외인기관 추정가집계[v1_국내주식-046] | `GET` | `/uapi/domestic-stock/v1/quotations/investor-trend-estimate` | `HHPTJ04160200` | `모의투자 미지원` |
| 종목별일별매수매도체결량 [v1_국내주식-056] | `GET` | `/uapi/domestic-stock/v1/quotations/inquire-daily-trade-volume` | `FHKST03010800` | `모의투자 미지원` |
| 프로그램매매 종합현황(시간) [국내주식-114] | `GET` | `/uapi/domestic-stock/v1/quotations/comp-program-trade-today` | `FHPPG04600101` | `모의투자 미지원` |
| 프로그램매매 종합현황(일별)[국내주식-115] | `GET` | `/uapi/domestic-stock/v1/quotations/comp-program-trade-daily` | `FHPPG04600001` | `모의투자 미지원` |
| 프로그램매매 투자자매매동향(당일) [국내주식-116] | `GET` | `/uapi/domestic-stock/v1/quotations/investor-program-trade-today` | `HHPPG046600C1` | `모의투자 미지원` |
| 국내주식 신용잔고 일별추이[국내주식-110] | `GET` | `/uapi/domestic-stock/v1/quotations/daily-credit-balance` | `FHPST04760000` | `모의투자 미지원` |
| 국내주식 예상체결가 추이[국내주식-118] | `GET` | `/uapi/domestic-stock/v1/quotations/exp-price-trend` | `FHPST01810000` | `모의투자 미지원` |
| 국내주식 공매도 일별추이[국내주식-134] | `GET` | `/uapi/domestic-stock/v1/quotations/daily-short-sale` | `FHPST04830000` | `모의투자 미지원` |
| 국내주식 시간외예상체결등락률 [국내주식-140] | `GET` | `/uapi/domestic-stock/v1/ranking/overtime-exp-trans-fluct` | `FHKST11860000` | `모의투자 미지원` |
| 국내주식 체결금액별 매매비중 [국내주식-192] | `GET` | `/uapi/domestic-stock/v1/quotations/tradprt-byamt` | `FHKST111900C0` | `모의투자 미지원` |
| 국내 증시자금 종합 [국내주식-193] | `GET` | `/uapi/domestic-stock/v1/quotations/mktfunds` | `FHKST649100C0` | `모의투자 미지원` |
| 종목별 일별 대차거래추이 [국내주식-135] | `GET` | `/uapi/domestic-stock/v1/quotations/daily-loan-trans` | `HHPST074500C0` | `모의투자 미지원` |
| 국내주식 상하한가 포착 [국내주식-190] | `GET` | `/uapi/domestic-stock/v1/quotations/capture-uplowprice` | `FHKST130000C0` | `모의투자 미지원` |
| 국내주식 매물대/거래비중 [국내주식-196] | `GET` | `/uapi/domestic-stock/v1/quotations/pbar-tratio` | `FHPST01130000` | `모의투자 미지원` |

## [국내주식] 순위분석

| name | method | access_url | real_tr_id | virtual_tr_id |
| --- | --- | --- | --- | --- |
| 거래량순위[v1_국내주식-047] | `GET` | `/uapi/domestic-stock/v1/quotations/volume-rank` | `FHPST01710000` | `모의투자 미지원` |
| 국내주식 등락률 순위[v1_국내주식-088] | `GET` | `/uapi/domestic-stock/v1/ranking/fluctuation` | `FHPST01700000` | `모의투자 미지원` |
| 국내주식 호가잔량 순위[국내주식-089] | `GET` | `/uapi/domestic-stock/v1/ranking/quote-balance` | `FHPST01720000` | `모의투자 미지원` |
| 국내주식 수익자산지표 순위[v1_국내주식-090] | `GET` | `/uapi/domestic-stock/v1/ranking/profit-asset-index` | `FHPST01730000` | `모의투자 미지원` |
| 국내주식 시가총액 상위[v1_국내주식-091] | `GET` | `/uapi/domestic-stock/v1/ranking/market-cap` | `FHPST01740000` | `모의투자 미지원` |
| 국내주식 재무비율 순위[v1_국내주식-092] | `GET` | `/uapi/domestic-stock/v1/ranking/finance-ratio` | `FHPST01750000` | `모의투자 미지원` |
| 국내주식 시간외잔량 순위[v1_국내주식-093] | `GET` | `/uapi/domestic-stock/v1/ranking/after-hour-balance` | `FHPST01760000` | `모의투자 미지원` |
| 국내주식 우선주/괴리율 상위[v1_국내주식-094] | `GET` | `/uapi/domestic-stock/v1/ranking/prefer-disparate-ratio` | `FHPST01770000` | `모의투자 미지원` |
| 국내주식 이격도 순위[v1_국내주식-095] | `GET` | `/uapi/domestic-stock/v1/ranking/disparity` | `FHPST01780000` | `모의투자 미지원` |
| 국내주식 시장가치 순위[v1_국내주식-096] | `GET` | `/uapi/domestic-stock/v1/ranking/market-value` | `FHPST01790000` | `모의투자 미지원` |
| 국내주식 체결강도 상위[v1_국내주식-101] | `GET` | `/uapi/domestic-stock/v1/ranking/volume-power` | `FHPST01680000` | `모의투자 미지원` |
| 국내주식 관심종목등록 상위[v1_국내주식-102] | `GET` | `/uapi/domestic-stock/v1/ranking/top-interest-stock` | `FHPST01800000` | `모의투자 미지원` |
| 국내주식 예상체결 상승/하락상위[v1_국내주식-103] | `GET` | `/uapi/domestic-stock/v1/ranking/exp-trans-updown` | `FHPST01820000` | `모의투자 미지원` |
| 국내주식 당사매매종목 상위[v1_국내주식-104] | `GET` | `/uapi/domestic-stock/v1/ranking/traded-by-company` | `FHPST01860000` | `모의투자 미지원` |
| 국내주식 신고/신저근접종목 상위[v1_국내주식-105] | `GET` | `/uapi/domestic-stock/v1/ranking/near-new-highlow` | `FHPST01870000` | `모의투자 미지원` |
| 국내주식 배당률 상위[국내주식-106] | `GET` | `/uapi/domestic-stock/v1/ranking/dividend-rate` | `HHKDB13470100` | `모의투자 미지원` |
| 국내주식 대량체결건수 상위[국내주식-107] | `GET` | `/uapi/domestic-stock/v1/ranking/bulk-trans-num` | `FHKST190900C0` | `모의투자 미지원` |
| 국내주식 신용잔고 상위[국내주식-109] | `GET` | `/uapi/domestic-stock/v1/ranking/credit-balance` | `FHKST17010000` | `모의투자 미지원` |
| 국내주식 공매도 상위종목[국내주식-133] | `GET` | `/uapi/domestic-stock/v1/ranking/short-sale` | `FHPST04820000` | `모의투자 미지원` |
| 국내주식 시간외등락율순위 [국내주식-138] | `GET` | `/uapi/domestic-stock/v1/ranking/overtime-fluctuation` | `FHPST02340000` | `모의투자 미지원` |
| 국내주식 시간외거래량순위 [국내주식-139] | `GET` | `/uapi/domestic-stock/v1/ranking/overtime-volume` | `FHPST02350000` | `모의투자 미지원` |
| HTS조회상위20종목 [국내주식-214] | `GET` | `/uapi/domestic-stock/v1/ranking/hts-top-view` | `HHMCM000100C0` | `모의투자 미지원` |

## [국내주식] 실시간시세

| name | method | access_url | real_tr_id | virtual_tr_id |
| --- | --- | --- | --- | --- |
| 국내주식 실시간체결가 (KRX) [실시간-003] | `POST` | `/tryitout/H0STCNT0` | `H0STCNT0` | `H0STCNT0` |
| 국내주식 실시간호가 (KRX) [실시간-004] | `POST` | `/tryitout/H0STASP0` | `H0STASP0` | `H0STASP0` |
| 국내주식 실시간체결통보 [실시간-005] | `POST` | `/tryitout/H0STCNI0` | `H0STCNI0` | `H0STCNI9` |
| 국내주식 실시간예상체결 (KRX) [실시간-041] | `POST` | `/tryitout/H0STANC0` | `H0STANC0` | `모의투자 미지원` |
| 국내주식 실시간회원사 (KRX) [실시간-047] | `POST` | `/tryitout/H0STMBC0` | `H0STMBC0` | `모의투자 미지원` |
| 국내주식 실시간프로그램매매 (KRX) [실시간-048] | `POST` | `/tryitout/H0STPGM0` | `H0STPGM0` | `모의투자 미지원` |
| 국내주식 장운영정보 (KRX) [실시간-049] | `POST` | `/tryitout/H0STMKO0` | `H0STMKO0` | `모의투자 미지원` |
| 국내주식 시간외 실시간호가 (KRX) [실시간-025] | `POST` | `/tryitout/H0STOAA0` | `H0STOAA0` | `모의투자 미지원` |
| 국내주식 시간외 실시간체결가 (KRX) [실시간-042] | `POST` | `/tryitout/H0STOUP0` | `H0STOUP0` | `모의투자 미지원` |
| 국내주식 시간외 실시간예상체결 (KRX) [실시간-024] | `POST` | `/tryitout/H0STOAC0` | `H0STOAC0` | `모의투자 미지원` |
| 국내지수 실시간체결 [실시간-026] | `POST` | `/tryitout/H0UPCNT0` | `H0UPCNT0` | `모의투자 미지원` |
| 국내지수 실시간예상체결 [실시간-027] | `POST` | `/tryitout/H0UPANC0` | `H0UPANC0` | `모의투자 미지원` |
| 국내지수 실시간프로그램매매 [실시간-028] | `POST` | `/tryitout/H0UPPGM0` | `H0UPPGM0` | `모의투자 미지원` |
| ELW 실시간호가 [실시간-062] | `POST` | `/tryitout/H0EWASP0` | `H0EWASP0` | `모의투자 미지원` |
| ELW 실시간체결가 [실시간-061] | `POST` | `/tryitout/H0EWCNT0` | `H0EWCNT0` | `모의투자 미지원` |
| ELW 실시간예상체결 [실시간-063] | `POST` | `/tryitout/H0EWANC0` | `H0EWANC0` | `모의투자 미지원` |
| 국내ETF NAV추이 [실시간-051] | `POST` | `/tryitout/H0STNAV0` | `H0STNAV0` | `모의투자 미지원` |
| 국내주식 실시간체결가 (통합) | `POST` | `/tryitout/H0UNCNT0` | `H0UNCNT0` | `모의투자 미지원` |
| 국내주식 실시간호가 (통합) | `POST` | `/tryitout/H0UNASP0` | `H0UNASP0` | `모의투자 미지원` |
| 국내주식 실시간예상체결 (통합) | `POST` | `/tryitout/H0UNANC0` | `H0UNANC0` | `모의투자 미지원` |
| 국내주식 실시간회원사 (통합) | `POST` | `/tryitout/H0UNMBC0` | `H0UNMBC0` | `모의투자 미지원` |
| 국내주식 실시간프로그램매매 (통합) | `POST` | `/tryitout/H0UNPGM0` | `H0UNPGM0` | `모의투자 미지원` |
| 국내주식 장운영정보 (통합) | `POST` | `/tryitout/H0UNMKO0` | `H0UNMKO0` | `모의투자 미지원` |
| 국내주식 실시간체결가 (NXT) | `POST` | `/tryitout/H0NXCNT0` | `H0NXCNT0` | `모의투자 미지원` |
| 국내주식 실시간호가 (NXT) | `POST` | `/tryitout/H0NXASP0` | `H0NXASP0` | `모의투자 미지원` |
| 국내주식 실시간예상체결 (NXT) | `POST` | `/tryitout/H0NXANC0` | `H0NXANC0` | `모의투자 미지원` |
| 국내주식 실시간회원사 (NXT) | `POST` | `/tryitout/H0NXMBC0` | `H0NXMBC0` | `모의투자 미지원` |
| 국내주식 실시간프로그램매매 (NXT) | `POST` | `/tryitout/H0NXPGM0` | `H0NXPGM0` | `모의투자 미지원` |
| 국내주식 장운영정보 (NXT) | `POST` | `/tryitout/H0NXMKO0` | `H0NXMKO0` | `모의투자 미지원` |
