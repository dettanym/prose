#!/usr/bin/env -S bash -c '"$(dirname $(readlink -f "$0"))/../env.sh" pnpm exec tsx -- "$0" "$@"'

import { $, echo, updateArgv } from "zx"
import { inspect } from "node:util"

/*--- PARAMETERS -----------------------------------------------------*/

const sample_texts_with_pii = [
  `
Here are a few example sentences we currently support:

Hello, my name is David Johnson and I live in Maine.
My credit card number is 4095-2609-9393-4932 and my crypto wallet id is 16Yeky6GMjeNkAiNcBY7ZhrLoMSgg1BoyZ.

On September 18 I visited microsoft.com and sent an email to test@presidio.site,  from the IP 192.168.0.1.

My passport: 191280342 and my phone number: (212) 555-1234.

This is a valid International Bank Account Number: IL150120690000003111111 . Can you please check the status on bank account 954567876544?

Kate's social security number is 078-05-1126.  Her driver license? it is 1234567A.
`,
]

/*--- PROGRAM --------------------------------------------------------*/

// The `env.sh` injects the original `CWD` location from which the script was
// executed as the first argument. Current cwd was changed by `env.sh` to the
// folder containing the current file, so here we are turning it back.
$.cwd = process.argv.at(2) as string
updateArgv(process.argv.slice(3))

await (async function main() {
  for (const sample_text of sample_texts_with_pii) {
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
