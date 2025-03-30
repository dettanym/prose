"""REST API server for analyzer."""

import logging
import os
from logging.config import fileConfig
from pathlib import Path
from typing import Tuple

from flask import Flask, Response, jsonify, request
from flask_caching import Cache
from presidio_analyzer.analyzer_engine import AnalyzerEngine
from presidio_analyzer.analyzer_request import AnalyzerRequest
from presidio_analyzer.batch_analyzer_engine import BatchAnalyzerEngine
from presidio_anonymizer import BatchAnonymizerEngine
from werkzeug.exceptions import HTTPException

from .helpers import convert_all_lists_to_dicts, extract_data_types_from_results

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

    def __init__(self, settings):
        fileConfig(Path(Path(__file__).parent, LOGGING_CONF_FILE))
        self.logger = logging.getLogger("presidio-analyzer")
        self.logger.setLevel(os.environ.get("LOG_LEVEL", self.logger.level))
        print("enable cache:  " + str(settings["enable_cache"]))
        self.cache = Cache(
            config=(
                {"CACHE_TYPE": "SimpleCache"}
                if settings["enable_cache"]
                else {"CACHE_TYPE": "NullCache", "CACHE_NO_NULL_WARNING": True}
            )
        )
        self.app = Flask(__name__)
        self.logger.info("Starting analyzer engine")
        self.engine = AnalyzerEngine()
        self.batch_analyzer = BatchAnalyzerEngine(analyzer_engine=self.engine)
        self.batch_anonymizer = BatchAnonymizerEngine()
        self.cache.init_app(self.app)
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

        def make_cache_key():
            return request.get_data(as_text=True)

        @self.app.route("/batchanalyze", methods=["POST"])
        @self.cache.cached(make_cache_key=make_cache_key)
        def batch_analyze() -> Tuple[Response, int]:
            """Execute the batch analyzer function."""
            # Parse the request params
            try:
                request_obj = request.get_json()
                if (
                    "json_to_analyze" not in request_obj
                    or request_obj["json_to_analyze"] is None
                ):
                    raise Exception(
                        "Please set a JSON field named 'json_to_analyze' in the body, with the JSON object "
                        "to analyze."
                    )

                # Note that this function implementation already adds the key as additional 'context'
                # for the decision (see batch_analyzer_engine.py line 96)
                recognizer_result_list = self.batch_analyzer.analyze_dict(
                    input_dict=convert_all_lists_to_dicts(
                        request_obj["json_to_analyze"]
                    ),
                    language="en",
                )

                unique_pii_list = extract_data_types_from_results(
                    recognizer_result_list
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
