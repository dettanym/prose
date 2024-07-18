from dataclasses import dataclass
from typing import Any, Iterator, Set

from presidio_analyzer import DictAnalyzerResult, RecognizerResult


def convert_all_lists_to_dicts(data: Any) -> Any:
    """
    Recursively transforms all lists into dictionaries.
    :param data: Incoming data
    :return: Transformed data
    """

    if not isinstance(data, (list, dict)):
        return data

    if isinstance(data, dict):
        return {k: convert_all_lists_to_dicts(v) for k, v in data.items()}

    if isinstance(data, list):
        result = dict()

        for k, v in enumerate(data):
            # need to convert `int` indexes into strings, otherwise the
            # for-loop in the subsequent operation fails.
            result[str(k)] = convert_all_lists_to_dicts(v)

        return result

    raise TypeError("Unknown type of incoming data: " + str(type(data)))


def extract_recognizer_results(
    dict_results: Iterator[DictAnalyzerResult],
) -> Set[RecognizerResult]:
    """
    Extract `entity_type` fields from all nested `RecognizerResult` types within
    the incoming tree structure of `DictAnalyzerResult`.
    :param dict_results: Result of running `batch_analyzer.analyze_dict`
    function on JSON object
    :return: Set of entity types
    """

    todo = list(dict_results)
    final = set()

    while len(todo) > 0:
        # The type of `recognizer_results` is defined here:
        # https://github.com/microsoft/presidio/blob/3c7eb8909a3341f2597453fbcaba6184477aa464/presidio-analyzer/presidio_analyzer/dict_analyzer_result.py#L25
        recognizer_results = todo.pop().recognizer_results

        if isinstance(recognizer_results, list):
            # In this case `recognizer_results` has type
            # `Union[List[RecognizerResult], List[List[RecognizerResult]]]`.
            # Once we reach value of type `RecognizerResult`, we can extract
            # `entity_type` from it.
            for item in recognizer_results:
                if isinstance(item, RecognizerResult):
                    final.add(item)
                elif isinstance(item, list):
                    for rr in item:
                        if isinstance(rr, RecognizerResult):
                            final.add(rr)
                        else:
                            raise TypeError("Unknown type of result: " + str(type(rr)))
                else:
                    raise TypeError("Unknown type of result: " + str(type(item)))

        elif isinstance(recognizer_results, Iterator):
            todo.extend(recognizer_results)

        else:
            raise TypeError("Unknown type of result: " + str(type(recognizer_results)))

    return final


@dataclass
class RecognizerResultDict:
    entity_type: str
    start: int
    end: int
    score: float


def map_recognizer_result_to_dict(rr: RecognizerResult) -> RecognizerResultDict:
    return RecognizerResultDict(
        entity_type=rr.entity_type,
        start=rr.start,
        end=rr.end,
        score=rr.score,
    )
