"""REST API server for analyzer."""

import os

from opentelemetry import trace
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.instrumentation.flask import FlaskInstrumentor
from opentelemetry.sdk.resources import SERVICE_NAME, Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor

from .server import Server

DEFAULT_PORT = "3000"

if __name__ == "__main__":
    port = int(os.environ.get("PORT", DEFAULT_PORT))

    processor = BatchSpanProcessor(OTLPSpanExporter())

    provider = TracerProvider(
        resource=Resource.create(
            {
                SERVICE_NAME: "prose.presidio",
            }
        )
    )
    provider.add_span_processor(processor)

    trace.set_tracer_provider(provider)

    server = Server()
    FlaskInstrumentor().instrument_app(
        server.app,
        excluded_urls="/health",
    )

    server.app.run(host="0.0.0.0", port=port)
