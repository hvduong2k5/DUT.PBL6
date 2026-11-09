from __future__ import annotations

from fastapi import FastAPI
from fastapi.responses import JSONResponse, Response

from . import __version__, db
from .config import Settings
from .observability import metrics_response, setup_logging


def create_app(settings: Settings | None = None) -> FastAPI:
    setup_logging()
    settings = settings or Settings.load()
    app = FastAPI(title="ai-agent-service", version=__version__)
    app.state.settings = settings

    @app.get("/health")
    async def health() -> dict:
        return {"status": "ok", "service": "ai-agent-service", "version": __version__}

    @app.get("/health/ready")
    async def ready() -> JSONResponse:
        up = db.ping(settings.database_url)
        body = {"status": "ready" if up else "not_ready", "db": "up" if up else "down"}
        return JSONResponse(body, status_code=200 if up else 503)

    @app.get("/metrics")
    async def metrics() -> Response:
        content, content_type = metrics_response()
        return Response(content, media_type=content_type)

    return app


app = create_app()
