import subprocess
from typing import List


def pipe_processes(*programs: List[str], stdin=None) -> (str | None, str | None):
    # based on https://docs.python.org/3/library/subprocess.html#replacing-shell-pipeline

    if len(programs) == 0:
        return "", ""

    if len(programs) == 1:
        proc = subprocess.run(
            programs[0],
            stdin=stdin,
            capture_output=True,
            encoding="utf-8",
        )
        return proc.stdout, proc.stderr

    procs = []

    for i, program in enumerate(programs):
        procs.append(
            subprocess.Popen(
                program,
                stdin=stdin if i == 0 else procs[i - 1].stdout,
                stdout=subprocess.PIPE,
            )
        )
        if i != 0:
            procs[i - 1].stdout.close()

    [fin_stdout, fin_stderr] = procs[-1].communicate()
    return (
        fin_stdout.decode("utf-8") if fin_stdout is not None else None,
        fin_stderr.decode("utf-8") if fin_stderr is not None else None,
    )
