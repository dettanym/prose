#!/usr/bin/env -S bash -c '"$(dirname $(readlink -f "$0"))/../env.sh" pnpm exec tsx -- "$0" "$@"'

import { $, echo, updateArgv } from "zx"
import { inspect } from "node:util"

/*--- PARAMETERS -----------------------------------------------------*/

const sample_texts_with_pii = [
  // sentences from huggingface example: https://huggingface.co/spaces/presidio/presidio_demo
  extract_pii`Hello, my name is ${pii(
    "David Johnson",
    "PERSON",
  )} and I live in ${pii("Maine", "LOCATION")}.`,
  extract_pii`My credit card number is ${pii(
    "4095-2609-9393-4932",
    "CREDIT_CARD",
  )} and my crypto wallet id is ${pii(
    "16Yeky6GMjeNkAiNcBY7ZhrLoMSgg1BoyZ",
    "CRYPTO",
  )}.`,
  extract_pii`On ${pii("September 18", "DATE_TIME")} I visited ${pii(
    "microsoft.com",
    "URL",
  )} and sent an email to ${pii(
    "test@presidio.site",
    "EMAIL_ADDRESS",
  )}, from the IP ${pii("192.168.0.1", "IP_ADDRESS")}.`,
  extract_pii`My passport: ${pii(
    "191280342",
    "US_PASSPORT",
  )} and my phone number: ${pii("(212) 555-1234", "PHONE_NUMBER")}.`,
  extract_pii`This is a valid International Bank Account Number: ${pii(
    "IL150120690000003111111",
    "IBAN_CODE",
  )}.`,
  extract_pii`Can you please check the status on bank account ${pii(
    "954567876544",
    "US_BANK_NUMBER",
  )}?`,
  extract_pii`${pii("Kate", "PERSON")}'s social security number is ${pii(
    "078-05-1126",
    "US_SSN",
  )}.`,
  extract_pii`Her driver license? it is ${pii(
    "1234567A",
    "US_DRIVER_LICENSE",
  )}.`,
]

/*--- PROGRAM --------------------------------------------------------*/

// The `env.sh` injects the original `CWD` location from which the script was
// executed as the first argument. Current cwd was changed by `env.sh` to the
// folder containing the current file, so here we are turning it back.
$.cwd = process.argv.at(2) as string
updateArgv(process.argv.slice(3))

await (async function main() {
  validateData(sample_texts_with_pii)

  for (const { text: sample_text, expected_pii } of sample_texts_with_pii) {
    const response = (await fetch("http://localhost:3000/batchanalyze", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        entire_recognizer_result: true,
        analyze_params: { score_threshold: 0.1 },
        json_to_analyze: sample_text,
      }),
    }).then((_) => _.json())) as PresidioResponse

    echo`${inspect(findMatchingRanges(expected_pii, response))}`

    for (const { first, second, intersection } of findOverlaps(response)) {
      echo`
        these two overlap:
        ${inspect(first)}
        and
        ${inspect(second)}
      `
      echo`first:    "${sample_text.slice(first!.start, first!.end)}"`
      echo`second:   "${sample_text.slice(second!.start, second!.end)}"`
      echo`overlap:  "${sample_text.slice(
        intersection.start,
        intersection.end,
      )}"`
    }
  }
})()

//<editor-fold desc="--- HELPERS --------------------------------------------------------">

type Range = {
  readonly start: number
  readonly end: number
}

type BasePII = { readonly value: string; readonly entity_type: string }
type PII = BasePII & { readonly _tag: "pii" }
type ExpectedPII = BasePII & Range

function pii(value: PII["value"], entity_type: PII["entity_type"]): PII {
  return { _tag: "pii", value, entity_type }
}

function extract_pii(
  pieces: TemplateStringsArray,
  ...args: PII[]
): { text: string; expected_pii: ExpectedPII[] } {
  return pieces.reduce<{ text: string; expected_pii: ExpectedPII[] }>(
    ({ text, expected_pii }, piece, i) => {
      text += piece

      if (!args[i]) {
        return { text, expected_pii }
      }

      const { value, entity_type } = args[i]
      const start = text.length
      text += value
      const end = text.length

      expected_pii.push({ value, entity_type, start, end })

      return { text, expected_pii }
    },
    { text: "", expected_pii: [] },
  )
}

function validateData(data: Array<ReturnType<typeof extract_pii>>) {
  const indexes = data
    .map((d, i) => (findOverlaps(d.expected_pii).length > 0 ? i : null))
    .filter((x) => x != null)
    .join(", ")

  if (indexes.length > 0) {
    throw new Error(
      // The algorithm searching for matches between expected data and response
      // from presidio does not account for the overlaps of PII objects in the
      // input data.
      `Incorrect input data: overlaps found at indices: ${indexes}`,
    )
  }
}

function hasOverlap(a: Range, b: Range): boolean {
  return a.start <= b.end && b.start <= a.end
}

function equalOverlap(a: Range, b: Range): boolean {
  return a.start === b.start && a.end === b.end
}

function nestedOverlap(a: Range, b: Range): Range | null {
  return b.start <= a.start && a.end <= b.end
    ? { start: a.start, end: a.end }
    : a.start <= b.start && b.end <= a.end
      ? { start: b.start, end: b.end }
      : null
}

type RecognizerResult = {
  entity_type: string
  score: number
} & Range
type PresidioResponse = Array<RecognizerResult>

function overlap(a: Range, b: Range): Range | null {
  const start = Math.max(a.start, b.start)
  const end = Math.min(a.end, b.end)

  return start <= end ? { start, end } : null
}

function findOverlaps<T extends Range>(
  data: Array<T>,
): Array<{ first: T; second: T; intersection: Range }> {
  const final = []

  for (let i = 0; i < data.length - 1; i++) {
    for (let j = i + 1; j < data.length; j++) {
      const intersection = overlap(data[i]!, data[j]!)
      if (intersection != null) {
        final.push({ first: data[i]!, second: data[j]!, intersection })
      }
    }
  }

  return final
}

type ExpectedRecognizedMatch = {
  readonly value: string
  readonly entity_type: string
  readonly score: number
} & Range
type OverlappedMatch = {
  readonly expected: ExpectedPII
  readonly found: RecognizerResult
  readonly overlap: Range
}

function findMatchingRanges(
  expected: Array<ExpectedPII>,
  found: Array<RecognizerResult>,
): {
  // true positive
  exact_matches: Array<ExpectedRecognizedMatch>
  // false negative
  missed_pii: Array<ExpectedPII>
  // false positive
  mismatched_entity_type: Array<ExpectedRecognizedMatch>
  // part of some other PII. false positive
  nested_pii: Array<OverlappedMatch>
  // reusing data from other PII, shouldn't happen. false positive
  overlapping_pii: Array<OverlappedMatch>
  // false positive
  extra_found_pii: Array<RecognizerResult>
} {
  const _expected: Array<ExpectedPII | null> = [...expected]
  const _found: Array<RecognizerResult | null> = [...found]

  const exact_matches = [] as Array<ExpectedRecognizedMatch>
  const mismatched_entity_type = [] as Array<ExpectedRecognizedMatch>
  const nested_pii = [] as Array<OverlappedMatch>
  const overlapping_pii = [] as Array<OverlappedMatch>

  for (let i = 0; i < _expected.length; i++) {
    const e = _expected[i]
    if (!e) continue

    for (let j = 0; j < _found.length; j++) {
      const f = _found[j]
      if (!f) continue

      if (equalOverlap(e, f)) {
        _expected[i] = null
        _found[j] = null

        if (e.entity_type === f.entity_type) {
          exact_matches.push({ ...e, score: f.score })
        } else {
          mismatched_entity_type.push({ ...f, value: e.value })
        }
      } else if (hasOverlap(e, f)) {
        _found[j] = null

        let o: Range | null
        if ((o = nestedOverlap(e, f))) {
          nested_pii.push({ expected: e, found: f, overlap: o })
        } else if ((o = overlap(e, f))) {
          overlapping_pii.push({ expected: e, found: f, overlap: o })
        } else {
          throw new Error("absurd")
        }
      } else {
      }
    }
  }

  return {
    exact_matches,
    missed_pii: _expected.filter((v) => v != null),
    mismatched_entity_type,
    nested_pii,
    overlapping_pii,
    extra_found_pii: _found.filter((v) => v != null),
  }
}
//</editor-fold>
