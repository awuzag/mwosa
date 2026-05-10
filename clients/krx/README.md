# krx Client Workspace

`clients/krx` 는 KRX Data Marketplace OPEN API client 를 만들기 위한 작업 공간이다.

아직 Go client module 은 만들지 않았다. 현재는 KRX OPEN API 의 서비스별 이용신청
흐름을 다룰 보조 스크립트만 둔다. 실제 client 구현을 시작할 때 `go.mod`, typed
client, request/response model, fake HTTP tests 를 이 폴더에 추가한다.

## Scripts

- [scripts/apply-all-services.browser.js](scripts/apply-all-services.browser.js):
  로그인된 KRX OPEN API 브라우저 탭에서 전체 31개 서비스를 12개월로 일괄 신청하는
  브라우저 콘솔용 스크립트

## 일괄 신청 스크립트 사용법

1. Chrome 에서 KRX OPEN API 에 로그인한다.
2. `https://openapi.krx.co.kr/contents/OPP/INFO/service/OPPINFO004.cmd` 또는 같은
   origin 의 KRX OPEN API 페이지를 연다.
3. DevTools Console 또는 Sources > Snippets 에
   `scripts/apply-all-services.browser.js` 내용을 붙여 넣고 실행한다.
4. 확인 창에서 신청 범위와 기간을 확인한 뒤 진행한다.

스크립트 기본값:

- 신청 범위: `KRX 시리즈 일별시세정보` 부터 `ESG 지수` 까지 전체 31개 서비스
- 기간: `12M` 화면 옵션에 해당하는 `1Y`
- 신청 목적: `개인 연구`

실행 결과는 화면의 `<pre id="krx-apply-log">` 와 브라우저 콘솔에 함께 출력된다.
