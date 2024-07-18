#!/usr/bin/env -S bash -c '"$(dirname $(readlink -f "$0"))/../env.sh" pnpm exec tsx -- "$0" "$@"'

import { $, echo, updateArgv } from "zx"
import { inspect } from "node:util"

/*--- PARAMETERS -----------------------------------------------------*/

const sample_texts_with_pii = [
  extract_pii`
    Hello, my name is ${pii("David Johnson", "PERSON")} and I live in ${pii(
      "Maine",
      "LOCATION",
    )}.
    My credit card number is ${pii(
      "4095-2609-9393-4932",
      "CREDIT_CARD",
    )} and my crypto wallet id is ${pii(
      "16Yeky6GMjeNkAiNcBY7ZhrLoMSgg1BoyZ",
      "CRYPTO",
    )}.
    On ${pii("September 18", "DATE_TIME")} I visited ${pii(
      "microsoft.com",
      "URL",
    )} and sent an email to ${pii(
      "test@presidio.site",
      "EMAIL_ADDRESS",
    )}, from the IP ${pii("192.168.0.1", "IP_ADDRESS")}.
    My passport: ${pii("191280342", "US_PASSPORT")} and my phone number: ${pii(
      "(212) 555-1234",
      "PHONE_NUMBER",
    )}.
    This is a valid International Bank Account Number: ${pii(
      "IL150120690000003111111",
      "IBAN_CODE",
    )}.
    Can you please check the status on bank account ${pii(
      "954567876544",
      "US_BANK_NUMBER",
    )}?
    ${pii("Kate", "PERSON")}'s social security number is ${pii(
      "078-05-1126",
      "US_SSN",
    )}.
    Her driver license? it is ${pii("1234567A", "US_DRIVER_LICENSE")}.
  `,
]

/*--- PROGRAM --------------------------------------------------------*/

// The `env.sh` injects the original `CWD` location from which the script was
// executed as the first argument. Current cwd was changed by `env.sh` to the
// folder containing the current file, so here we are turning it back.
$.cwd = process.argv.at(2) as string
updateArgv(process.argv.slice(3))

await (async function main() {
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

    echo`${inspect(response)}`

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

function findOverlaps(data: PresidioResponse): Array<{
  first: RecognizerResult
  second: RecognizerResult
  intersection: Range
}> {
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

//</editor-fold>
