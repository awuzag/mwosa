def n:
  if . == null or . == "" then 0
  else (tostring | gsub(","; "") | tonumber? // 0)
  end;

def pct($start; $end):
  (($end / $start - 1) * 10000 | round) / 100;

def tail($count):
  if length > $count then .[(length - $count):] else . end;

def avg_amount($rows; $count):
  ($rows | map(.traded_amount | n) | tail($count)) as $amounts
  | if ($amounts | length) == 0 then 0
    else (($amounts | add) / ($amounts | length) | floor)
    end;

map(select(.security_type == "etf"))
| sort_by(.symbol, .trading_date)
| group_by(.symbol)
| map(
    sort_by(.trading_date) as $rows
    | ($rows[0]) as $first
    | ($rows[-1]) as $latest
    | ($first.opening_price | n) as $first_open
    | ($first.closing_price | n) as $first_close
    | ($latest.closing_price | n) as $latest_close
    | select($first.trading_date >= "2025-01-01")
    | select($first_open > 0 and $first_close > 0 and $latest_close > 0)
    | {
        symbol: $latest.symbol,
        name: $latest.name,
        first_date: $first.trading_date,
        first_open: $first_open,
        first_close: $first_close,
        latest_date: $latest.trading_date,
        latest_close: $latest_close,
        return_from_first_open_pct: pct($first_open; $latest_close),
        return_from_first_close_pct: pct($first_close; $latest_close),
        latest_traded_amount: ($latest.traded_amount | n),
        avg_traded_amount_5d: avg_amount($rows; 5),
        avg_traded_amount_20d: avg_amount($rows; 20)
      }
  )
| sort_by(.return_from_first_open_pct)
| reverse

