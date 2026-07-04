const fixture = "priority-candidate-e2e";
const asOfDate = "2026-07-01";

db.aggregate_priority_candidates.deleteMany({ fixture, as_of_date: asOfDate });
db.aggregate_priority_valuation_snapshots.deleteMany({ fixture, as_of_date: asOfDate });
db.aggregate_live_symbols.deleteMany({ fixture });

db.aggregate_priority_candidates.insertMany([
  {
    fixture,
    as_of_date: asOfDate,
    symbol: "005930",
    name: "삼성전자",
    change_pct: 2.34,
    traded_amount: NumberLong("123400000000"),
    relative_volume_20d: 1.8,
    high_52w_pct: 87.4,
    close_position_pct: 72.1,
    rsi_14: 61.2,
    adx_14: 24.7,
    atr_pct_14: 2.8,
    trend: "상승",
    label: "거래대금 확대",
  },
  {
    fixture,
    as_of_date: asOfDate,
    symbol: "000660",
    name: "SK하이닉스",
    change_pct: 3.21,
    traded_amount: NumberLong("210000000000"),
    relative_volume_20d: 2.4,
    high_52w_pct: 95.2,
    close_position_pct: 88.6,
    rsi_14: 68.5,
    adx_14: 31.2,
    atr_pct_14: 3.1,
    trend: "강세",
    label: "52주 고점 근접",
  },
]);

db.aggregate_priority_valuation_snapshots.insertMany([
  {
    fixture,
    as_of_date: asOfDate,
    symbol: "005930",
    market_cap_minor: NumberLong("420000000000000"),
  },
  {
    fixture,
    as_of_date: asOfDate,
    symbol: "000660",
    market_cap_minor: NumberLong("180000000000000"),
  },
]);

db.aggregate_live_symbols.insertMany([
  {
    fixture,
    market_key: "krx",
    security_type: "stock",
    symbol: "005930",
    name: "삼성전자",
  },
  {
    fixture,
    market_key: "krx",
    security_type: "stock",
    symbol: "000660",
    name: "SK하이닉스",
  },
]);

printjson({
  status: "ok",
  fixture,
  as_of_date: asOfDate,
  candidate_count: db.aggregate_priority_candidates.countDocuments({ fixture, as_of_date: asOfDate }),
  valuation_count: db.aggregate_priority_valuation_snapshots.countDocuments({ fixture, as_of_date: asOfDate }),
  live_symbol_count: db.aggregate_live_symbols.countDocuments({ fixture }),
});
