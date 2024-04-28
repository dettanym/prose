"""REST API server for analyzer."""

import os

from .server import Server

DEFAULT_PORT = "3000"

if __name__ == "__main__":
    port = int(os.environ.get("PORT", DEFAULT_PORT))
    server = Server()
    server.app.run(host="0.0.0.0", port=port)
