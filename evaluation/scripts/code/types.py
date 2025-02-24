from dataclasses import dataclass


@dataclass
class SummaryResultFile:
    filename: str


@dataclass
class RawResultsFile:
    filename: str


Result_File = SummaryResultFile | RawResultsFile
