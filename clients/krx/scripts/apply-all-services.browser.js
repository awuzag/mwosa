(() => {
  "use strict";

  const CONFIG = {
    duration: "1Y", // KRX UI label: 12M
    purpose: "개인 연구",
    delayMs: 350,
    endpoint: "/contents/OPP/USES/service/OPPUSES001_S2D2.cmd",
  };

  const services = [
    {
      category: "지수",
      name: "KRX 시리즈 일별시세정보",
      apiId: "krx_dd_trd",
      boId: "SsgXTEspyJESKvyXZtCU",
    },
    {
      category: "지수",
      name: "KOSPI 시리즈 일별시세정보",
      apiId: "kospi_dd_trd",
      boId: "EREKZauXnMmxyIlqzeDN",
    },
    {
      category: "지수",
      name: "KOSDAQ 시리즈 일별시세정보",
      apiId: "kosdaq_dd_trd",
      boId: "nimebcamqFNIPNcRrHoO",
    },
    {
      category: "지수",
      name: "채권지수 시세정보",
      apiId: "bon_dd_trd",
      boId: "vMxIKCtPBUeRytCqkoFv",
    },
    {
      category: "지수",
      name: "파생상품지수 시세정보",
      apiId: "drvprod_dd_trd",
      boId: "rPBjbLtScMwmSXWDOYPd",
    },
    {
      category: "주식",
      name: "유가증권 일별매매정보",
      apiId: "stk_bydd_trd",
      boId: "JvJFzlAENzZlPBDNGAWC",
    },
    {
      category: "주식",
      name: "코스닥 일별매매정보",
      apiId: "ksq_bydd_trd",
      boId: "hZjGpkllgCBCWqeTsYFj",
    },
    {
      category: "주식",
      name: "코넥스 일별매매정보",
      apiId: "knx_bydd_trd",
      boId: "HSiRvxGSYnvaKuAuqpqp",
    },
    {
      category: "주식",
      name: "신주인수권증권 일별매매정보",
      apiId: "sw_bydd_trd",
      boId: "erXKnEAzTqcGnkcoSdGA",
    },
    {
      category: "주식",
      name: "신주인수권증서 일별매매정보",
      apiId: "sr_bydd_trd",
      boId: "YieGrzzJtKhbaNLuKmhz",
    },
    {
      category: "주식",
      name: "유가증권 종목기본정보",
      apiId: "stk_isu_base_info",
      boId: "PiwgMdTwmsenXhmqqxuj",
    },
    {
      category: "주식",
      name: "코스닥 종목기본정보",
      apiId: "ksq_isu_base_info",
      boId: "CifLHplnUFMgpHIMMPXs",
    },
    {
      category: "주식",
      name: "코넥스 종목기본정보",
      apiId: "knx_isu_base_info",
      boId: "COgTLqgmGlqyJvaEFNIc",
    },
    {
      category: "증권상품",
      name: "ETF 일별매매정보",
      apiId: "etf_bydd_trd",
      boId: "nrEpCLaZpoLCTzPUMxuF",
    },
    {
      category: "증권상품",
      name: "ETN 일별매매정보",
      apiId: "etn_bydd_trd",
      boId: "VujebrcOsZQMybnUuwLk",
    },
    {
      category: "증권상품",
      name: "ELW 일별매매정보",
      apiId: "elw_bydd_trd",
      boId: "brBhSEuDCUNpmfsCslfM",
    },
    {
      category: "채권",
      name: "국채전문유통시장 일별매매정보",
      apiId: "kts_bydd_trd",
      boId: "CEnOyORzHgXWpdbUfWyf",
    },
    {
      category: "채권",
      name: "일반채권시장 일별매매정보",
      apiId: "bnd_bydd_trd",
      boId: "JfStBNhXISpVVfBHgspT",
    },
    {
      category: "채권",
      name: "소액채권시장 일별매매정보",
      apiId: "smb_bydd_trd",
      boId: "yrTTOsXuYzHprbWLuYzd",
    },
    {
      category: "파생상품",
      name: "선물 일별매매정보 (주식선물外)",
      apiId: "fut_bydd_trd",
      boId: "ilaVYOabbaicHbKTsqga",
    },
    {
      category: "파생상품",
      name: "주식선물(유가) 일별매매정보",
      apiId: "eqsfu_stk_bydd_trd",
      boId: "JzVvQnspImpuqtZlFWpJ",
    },
    {
      category: "파생상품",
      name: "주식선물(코스닥) 일별매매정보",
      apiId: "eqkfu_ksq_bydd_trd",
      boId: "henfdJADfLTCUCBWIRCj",
    },
    {
      category: "파생상품",
      name: "옵션 일별매매정보 (주식옵션外)",
      apiId: "opt_bydd_trd",
      boId: "AoTvuFpukvuBsfypkZbq",
    },
    {
      category: "파생상품",
      name: "주식옵션(유가) 일별매매정보",
      apiId: "eqsop_bydd_trd",
      boId: "fwWKgzbevDVtAoECgkpA",
    },
    {
      category: "파생상품",
      name: "주식옵션(코스닥) 일별매매정보",
      apiId: "eqkop_bydd_trd",
      boId: "AFNbHSizSPnEssZoUqiS",
    },
    {
      category: "일반상품",
      name: "석유시장 일별매매정보",
      apiId: "oil_bydd_trd",
      boId: "rTvrZvAFKfcaLPOggJtW",
    },
    {
      category: "일반상품",
      name: "금시장 일별매매정보",
      apiId: "gold_bydd_trd",
      boId: "sxveSnWzWNzWxQASsgEG",
    },
    {
      category: "일반상품",
      name: "배출권 시장 일별매매정보",
      apiId: "ets_bydd_trd",
      boId: "IZiYdcgRQFMeENJPEMKG",
    },
    {
      category: "ESG",
      name: "ESG 증권상품",
      apiId: "esg_etp_info",
      boId: "dpRoGGhdnfSZSrMFtUCz",
    },
    {
      category: "ESG",
      name: "사회책임투자채권 정보",
      apiId: "sri_bond_info",
      boId: "MwsSXzVIceQhMSJUeCdp",
    },
    {
      category: "ESG",
      name: "ESG 지수",
      apiId: "esg_index_info",
      boId: "WgFYvEvsseQMARfMVZCq",
    },
  ];

  const codeLabels = {
    CD001: "정상",
    CDE01: "승인 대기 중",
    CDE02: "이미 이용 중",
  };

  const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

  const ensureKrxOrigin = () => {
    if (location.origin !== "https://openapi.krx.co.kr") {
      throw new Error(
        "이 스크립트는 https://openapi.krx.co.kr 에 로그인한 탭에서 실행해야 합니다.",
      );
    }
  };

  const makeLogElement = () => {
    document.body.innerHTML = "";
    const log = document.createElement("pre");
    log.id = "krx-apply-log";
    log.style.whiteSpace = "pre-wrap";
    log.style.padding = "16px";
    log.style.font = "13px/1.5 ui-monospace, SFMono-Regular, Menlo, Consolas, monospace";
    document.body.appendChild(log);
    return log;
  };

  const appendLog = (log, message) => {
    log.textContent += `${message}\n`;
    console.log(message);
  };

  const applyService = async (service) => {
    const body = new URLSearchParams({
      boId: service.boId,
      useApplPd: CONFIG.duration,
      applPurps: CONFIG.purpose,
    });

    const response = await fetch(CONFIG.endpoint, {
      method: "POST",
      credentials: "same-origin",
      headers: {
        "Content-Type": "application/x-www-form-urlencoded; charset=UTF-8",
        "X-Requested-With": "XMLHttpRequest",
      },
      body,
    });

    const text = await response.text();
    let parsed;
    try {
      parsed = JSON.parse(text);
    } catch {
      parsed = null;
    }

    const result = parsed?.result ?? {};
    const code = result._error_code ?? `HTTP_${response.status}`;
    const message = result._error_message ?? text.replace(/\s+/g, " ").slice(0, 160);

    return {
      ...service,
      httpStatus: response.status,
      code,
      status: codeLabels[code] ?? "확인 필요",
      message,
    };
  };

  const summarize = (results) => {
    return results.reduce((acc, result) => {
      acc[result.status] = (acc[result.status] ?? 0) + 1;
      return acc;
    }, {});
  };

  const run = async () => {
    ensureKrxOrigin();

    const ok = window.confirm(
      [
        "KRX OPEN API 전체 31개 서비스를 이용신청합니다.",
        "",
        "범위: KRX 시리즈 일별시세정보 ~ ESG 지수",
        "기간: 12M (useApplPd=1Y)",
        `신청 목적: ${CONFIG.purpose}`,
        "",
        "실제 KRX 계정에 이용신청이 생성됩니다. 진행할까요?",
      ].join("\n"),
    );

    if (!ok) {
      return;
    }

    const log = makeLogElement();
    const results = [];
    appendLog(log, `KRX OPEN API 일괄 신청 시작: ${services.length}개`);
    appendLog(log, `기간=${CONFIG.duration}, 신청목적=${CONFIG.purpose}`);
    appendLog(log, "");

    for (let index = 0; index < services.length; index += 1) {
      const service = services[index];
      try {
        const result = await applyService(service);
        results.push(result);
        appendLog(
          log,
          [
            String(index + 1).padStart(2, "0"),
            service.category,
            service.name,
            service.apiId,
            result.code,
            result.status,
            result.message,
          ].join(" | "),
        );
      } catch (error) {
        const result = {
          ...service,
          httpStatus: 0,
          code: "FETCH_ERROR",
          status: "요청 실패",
          message: error instanceof Error ? error.message : String(error),
        };
        results.push(result);
        appendLog(
          log,
          [
            String(index + 1).padStart(2, "0"),
            service.category,
            service.name,
            service.apiId,
            result.code,
            result.status,
            result.message,
          ].join(" | "),
        );
      }

      await sleep(CONFIG.delayMs);
    }

    appendLog(log, "");
    appendLog(log, `요약: ${JSON.stringify(summarize(results))}`);
    appendLog(log, "완료");
    console.table(results);
  };

  run().catch((error) => {
    console.error(error);
    window.alert(error instanceof Error ? error.message : String(error));
  });
})();
