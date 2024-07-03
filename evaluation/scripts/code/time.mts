import { Instant, ZonedDateTime, ZoneId, Duration } from "@js-joda/core"

import { dropLeftWhile, dropRightWhile } from "./common.mjs"

export function format_zoned_timestamp(ts: ZonedDateTime): string {
  return ts.toOffsetDateTime().toString()
}

export function current_timestamp() {
  return format_zoned_timestamp(ZonedDateTime.now().withNano(0))
}

export function current_micro_timestamp() {
  // if a version of node does not have 'performance' global variable, try
  // 'microtime' package: https://www.npmjs.com/package/microtime
  const nowMicros = (performance.now() + performance.timeOrigin) * 1000

  return ZonedDateTime.ofInstant(
    Instant.ofEpochMicro(nowMicros),
    ZoneId.systemDefault(),
  )
}

export function show_duration(dur: Duration) {
  if (dur.seconds() === 0) {
    return "0s"
  }

  const duration_value = dur.abs()

  const h = duration_value.toHours()
  const m = duration_value.minusHours(h).toMinutes()
  const s = duration_value.minusHours(h).minusMinutes(m).seconds()

  const sign = dur.isNegative() ? "-" : ""
  const values = dropRightWhile(
    dropLeftWhile(
      [
        [h, "h"],
        [m, "m"],
        [s, "s"],
      ],
      ([v]) => v === 0,
    ),
    ([v]) => v === 0,
  ).reduce((acc, [value, unit]) => acc + value + unit, "")

  return sign + values
}

export function parse_duration(duration: string): number {
  if (/\d+s/.test(duration)) {
    return parseInt(duration.slice(0, -1))
  }

  throw new Error(`Unknown duration: "${duration}".`)
}
