"""REST API server for analyzer."""

import logging
import os
import pprint
import re
from logging.config import fileConfig
from pathlib import Path
from typing import Tuple, List, Dict, Any, AnyStr

from flask import Flask, request, jsonify, Response
from werkzeug.exceptions import HTTPException

from presidio_analyzer.analyzer_engine import AnalyzerEngine
from presidio_analyzer.batch_analyzer_engine import BatchAnalyzerEngine
from presidio_analyzer.analyzer_request import AnalyzerRequest
from presidio_anonymizer import BatchAnonymizerEngine

DEFAULT_PORT = "3000"

LOGGING_CONF_FILE = "logging.ini"

WELCOME_MESSAGE = r"""
 _______  _______  _______  _______ _________ ______  _________ _______
(  ____ )(  ____ )(  ____ \(  ____ \\__   __/(  __  \ \__   __/(  ___  )
| (    )|| (    )|| (    \/| (    \/   ) (   | (  \  )   ) (   | (   ) |
| (____)|| (____)|| (__    | (_____    | |   | |   ) |   | |   | |   | |
|  _____)|     __)|  __)   (_____  )   | |   | |   | |   | |   | |   | |
| (      | (\ (   | (            ) |   | |   | |   ) |   | |   | |   | |
| )      | ) \ \__| (____/\/\____) |___) (___| (__/  )___) (___| (___) |
|/       |/   \__/(_______/\_______)\_______/(______/ \_______/(_______)
"""


class Server:
    """HTTP Server for calling Presidio Analyzer."""

    def __init__(self):
        fileConfig(Path(Path(__file__).parent, LOGGING_CONF_FILE))
        self.logger = logging.getLogger("presidio-analyzer")
        self.logger.setLevel(os.environ.get("LOG_LEVEL", self.logger.level))
        self.app = Flask(__name__)
        self.logger.info("Starting analyzer engine")
        self.engine = AnalyzerEngine()
        self.batch_analyzer = BatchAnalyzerEngine(analyzer_engine=self.engine)
        self.batch_anonymizer = BatchAnonymizerEngine()
        self.logger.info(WELCOME_MESSAGE)

        @self.app.route("/health")
        def health() -> str:
            """Return basic health probe result."""
            return "Presidio Analyzer service is up"

        @self.app.route("/analyze", methods=["POST"])
        def analyze() -> Tuple[Response, int]:
            """Execute the analyzer function."""
            # Parse the request params
            try:
                req_data = AnalyzerRequest(request.get_json())
                if not req_data.text:
                    raise Exception("No text provided")

                if not req_data.language:
                    raise Exception("No language provided")

                recognizer_result_list = self.engine.analyze(
                    text=req_data.text,
                    language=req_data.language,
                    correlation_id=req_data.correlation_id,
                    score_threshold=req_data.score_threshold,
                    entities=req_data.entities,
                    return_decision_process=req_data.return_decision_process,
                    ad_hoc_recognizers=req_data.ad_hoc_recognizers,
                    context=req_data.context,
                )

                return jsonify(recognizer_result_list), 200
            except TypeError as te:
                error_msg = (
                    f"Failed to parse /analyze request "
                    f"for AnalyzerEngine.analyze(). {te.args[0]}"
                )
                self.logger.error(error_msg)
                return jsonify(error=error_msg), 400

            except Exception as e:
                self.logger.error(
                    f"A fatal error occurred during execution of "
                    f"AnalyzerEngine.analyze(). {e}"
                )
                return jsonify(error=e.args[0]), 500

        @self.app.route("/recognizers", methods=["GET"])
        def recognizers() -> Tuple[Response, int]:
            """Return a list of supported recognizers."""
            language = request.args.get("language")
            try:
                recognizers_list = self.engine.get_recognizers(language)
                names = [o.name for o in recognizers_list]
                return jsonify(names), 200
            except Exception as e:
                self.logger.error(
                    f"A fatal error occurred during execution of "
                    f"AnalyzerEngine.get_recognizers(). {e}"
                )
                return jsonify(error=e.args[0]), 500

        @self.app.route("/supportedentities", methods=["GET"])
        def supported_entities() -> Tuple[Response, int]:
            """Return a list of supported entities."""
            language = request.args.get("language")
            try:
                entities_list = self.engine.get_supported_entities(language)
                return jsonify(entities_list), 200
            except Exception as e:
                self.logger.error(
                    f"A fatal error occurred during execution of "
                    f"AnalyzerEngine.supported_entities(). {e}"
                )
                return jsonify(error=e.args[0]), 500

        @self.app.errorhandler(HTTPException)
        def http_exception(e):
            return jsonify(error=e.description), e.code

        @self.app.route("/batchanalyze", methods=["POST"])
        def batch_analyze() -> Tuple[Response, int]:
            """Execute the batch analyzer function."""
            # Parse the request params
            try:
                request_obj = request.get_json()
                print(request_obj["json_to_analyze"], type(request_obj))
                if not request_obj["json_to_analyze"]:
                    raise Exception(
                        "Please set a JSON field named 'json_to_analyze' in the body, with the JSON object "
                        "to analyze."
                    )

                # Note that this function implementation already adds the key as additional 'context'
                # for the decision (see batch_analyzer_engine.py line 96)
                recognizer_result_list = self.batch_analyzer.analyze_dict(
                    input_dict=request_obj["json_to_analyze"], language="en"
                )
                print(recognizer_result_list)

                anonymizer_results = self.batch_anonymizer.anonymize_dict(
                    recognizer_result_list
                )
                pprint.pprint(anonymizer_results)
                class_substring_pattern = re.compile(r"<([^>]*)>")
                unique_pii_list = recursive_find_pattern(
                    anonymizer_results, class_substring_pattern
                )
                unique_valid_pii_list = [
                    pii for pii in unique_pii_list if pii in data_items_set
                ]

                return jsonify(unique_valid_pii_list), 200
            except TypeError as te:
                error_msg = (
                    f"Failed to parse /batchanalyze request "
                    f"for AnalyzerEngine.analyze(). {te.args[0]}"
                )
                self.logger.error(error_msg)
                return jsonify(error=error_msg), 400

            except Exception as e:
                self.logger.error(
                    f"A fatal error occurred during execution of "
                    f"BatchAnalyzer.analyze_dict(). {e}"
                )
                return jsonify(error=e.args[0]), 500


def recursive_find_pattern(
    d: Dict[AnyStr, Any], pattern: re.Pattern[AnyStr]
) -> List[AnyStr]:
    def match_string(input_string: str):
        pattern_found = pattern.search(input_string)
        if pattern_found is None:
            return

        first_match = pattern_found.group(
            1
        )  # group(1) gets matching data type within <>
        if first_match is not None and first_match not in acc:
            acc.append(first_match)

    def recursive_switch_case(v):
        if isinstance(v, dict):
            dict_find_pattern_in_value(v)
        elif isinstance(v, list):
            list_find_pattern_in_value(v)
        elif isinstance(v, str):
            match_string(v)

    def list_find_pattern_in_value(input_list: list):
        for v in input_list:
            recursive_switch_case(v)

    def dict_find_pattern_in_value(input_dict: dict):
        for v in input_dict.values():
            recursive_switch_case(v)

    acc = []
    dict_find_pattern_in_value(d)
    return acc


data_items_set = [
    "CREDIT_CARD",
    "NRP",
    "US_ITIN",
    "PERSON",
    "US_BANK_NUMBER",
    "US_PASSPORT",
    "IP_ADDRESS",
    "US_DRIVER_LICENSE",
    "CRYPTO",
    "URL",
    "PHONE_NUMBER",
    "IBAN_CODE",
    "DATE_TIME",
    "LOCATION",
    "EMAIL_ADDRESS",
    "US_SSN",
]

if __name__ == "__main__":
    port = int(os.environ.get("PORT", DEFAULT_PORT))
    server = Server()
    server.app.run(host="0.0.0.0", port=port)
