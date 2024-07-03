import { fs, path } from "zx"

export async function get_test_results_dir(
  PRJ_ROOT: string,
  application: string,
  hostname: string,
  timestamp: string,
) {
  const test_run_results_dir = path.join(
    PRJ_ROOT,
    "evaluation/vegeta",
    application,
    hostname,
    timestamp,
  )

  await fs.mkdirp(test_run_results_dir)

  return (...segments: string[]) => path.join(test_run_results_dir, ...segments)
}
