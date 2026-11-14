from __future__ import annotations

import asyncio
import logging
import re
import uuid
from contextlib import asynccontextmanager
from datetime import UTC, datetime

from fastapi import FastAPI, HTTPException, Request
from fastapi.exceptions import RequestValidationError
from fastapi.responses import JSONResponse, Response

from . import __version__, db
from .api_errors import ApiError, SAFE_MESSAGES
from .config import Settings, read_secret_file
from .modules.recommendation.service import RecommendationService
from .modules.shopping_advisor.routes import build_router
from .modules.shopping_advisor.seed import seed_knowledge
from .modules.shopping_advisor.tools import Backends, build_registry
from .modules.shopping_advisor.workflow import ShoppingAdvisor
from .observability import metrics_response, setup_logging
from .platform.audit import NullRunRecorder, PostgresRunRecorder
from .platform.conversations import (
    ConversationStore,
    MemoryConversationStore,
    PostgresConversationStore,
)
from .platform.auth import CustomerVerifier
from .platform.eligibility import ProductEligibility
from .platform.gateway import ToolGateway
from .platform.knowledge import (
    KnowledgeStore,
    MemoryKnowledgeStore,
    PostgresKnowledgeStore,
)
from .providers import build_embedding_provider, build_llm_provider

logger = logging.getLogger(__name__)


def build_backends(settings: Settings) -> Backends:
    if settings.fixtures_mode:
        from .integrations.fixtures import (
            FixtureAnalytics,
            FixtureCatalog,
            FixtureInventory,
            FixtureOrders,
            FixtureShipping,
        )

        catalog = FixtureCatalog()
        return Backends(
            catalog=catalog,
            inventory=FixtureInventory(),
            orders=FixtureOrders(),
            shipping=FixtureShipping(),
            analytics=FixtureAnalytics(catalog),
        )
    from .integrations.http_clients import (
        HttpAnalytics,
        HttpCatalog,
        HttpInventory,
        HttpOrders,
        HttpShipping,
    )

    catalog = HttpCatalog(settings.catalog_base_url)
    return Backends(
        catalog=catalog,
        inventory=HttpInventory(catalog),
        orders=HttpOrders(settings.order_base_url),
        shipping=HttpShipping(settings.shipping_base_url),
        analytics=HttpAnalytics(settings.analytics_base_url),
    )


def build_stores(settings: Settings, embedder) -> tuple[ConversationStore, KnowledgeStore]:
    if settings.fixtures_mode:
        return MemoryConversationStore(), MemoryKnowledgeStore(embedder)
    return (
        PostgresConversationStore(settings.database_url),
        PostgresKnowledgeStore(
            settings.database_url, embedder, settings.embedding_model
        ),
    )


def create_app(settings: Settings | None = None) -> FastAPI:
    setup_logging()
    settings = settings or Settings.load()
    embedder = build_embedding_provider(settings)
    store, knowledge = build_stores(settings, embedder)
    backends = build_backends(settings)
    run_recorder = (
        NullRunRecorder() if settings.fixtures_mode else PostgresRunRecorder(settings.database_url)
    )

    async def record_tool(record: dict) -> None:
        await asyncio.to_thread(run_recorder.record_tool, record)

    gateway = ToolGateway(
        build_registry(backends),
        timeout_seconds=settings.tool_timeout_seconds,
        recorder=record_tool,
    )
    llm = build_llm_provider(settings)
    eligibility = ProductEligibility(
        concurrency=settings.eligibility_concurrency,
        deadline_seconds=settings.eligibility_deadline_seconds,
    )
    recommendations = RecommendationService(backends, gateway, eligibility)
    advisor = ShoppingAdvisor(
        llm, gateway, knowledge, settings, recommendations, eligibility, run_recorder
    )
    customer_verifier = CustomerVerifier(settings)

    @asynccontextmanager
    async def lifespan(app: FastAPI):
        cleanup_task = None
        if settings.fixtures_mode:
            from .integrations.fixtures import FixtureCatalog

            await seed_knowledge(knowledge, FixtureCatalog())
        elif settings.cleanup_interval_seconds > 0:
            async def cleanup_loop() -> None:
                while True:
                    await asyncio.sleep(settings.cleanup_interval_seconds)
                    try:
                        await asyncio.to_thread(store.cleanup_expired, datetime.now(UTC))
                    except Exception:
                        logger.exception("conversation retention cleanup failed")

            cleanup_task = asyncio.create_task(cleanup_loop())
        try:
            yield
        finally:
            if cleanup_task is not None:
                cleanup_task.cancel()
                try:
                    await cleanup_task
                except asyncio.CancelledError:
                    pass
            customer_verifier.close()

    app = FastAPI(title="ai-agent-service", version=__version__, lifespan=lifespan)
    app.state.settings = settings
    app.state.store = store
    app.state.knowledge = knowledge
    app.state.backends = backends
    app.state.feedback_records: list[dict] = []
    app.state.feedback_idempotency: dict[tuple[str, str], str] = {}
    app.state.customer_verifier = customer_verifier

    @app.middleware("http")
    async def request_id_middleware(request: Request, call_next):
        incoming = request.headers.get("x-request-id", "")
        request_id = (
            incoming
            if re.fullmatch(r"[A-Za-z0-9._:#-]{1,128}", incoming)
            else str(uuid.uuid4())
        )
        request.state.request_id = request_id
        response = await call_next(request)
        response.headers["X-Request-Id"] = request_id
        return response

    def error_response(request: Request, status_code: int, code: str, message: str):
        return JSONResponse(
            {"code": code, "message": message, "requestId": request.state.request_id},
            status_code=status_code,
            headers={"X-Request-Id": request.state.request_id},
        )

    @app.exception_handler(ApiError)
    async def handle_api_error(request: Request, exc: ApiError):
        return error_response(request, exc.status_code, exc.code, exc.message)

    @app.exception_handler(RequestValidationError)
    async def handle_validation_error(request: Request, exc: RequestValidationError):
        return error_response(request, 422, "invalid_request", SAFE_MESSAGES["invalid_request"])

    @app.exception_handler(HTTPException)
    async def handle_http_error(request: Request, exc: HTTPException):
        code = exc.detail if isinstance(exc.detail, str) else "invalid_request"
        message = SAFE_MESSAGES.get(code, "The request could not be processed.")
        return error_response(request, exc.status_code, code, message)

    @app.exception_handler(Exception)
    async def handle_unexpected_error(request: Request, exc: Exception):
        logger.exception("unhandled API error request_id=%s", request.state.request_id)
        return error_response(request, 500, "internal_error", SAFE_MESSAGES["internal_error"])

    @app.get("/health")
    async def health() -> dict:
        return {"status": "ok", "service": "ai-agent-service", "version": __version__}

    @app.get("/health/ready")
    async def ready() -> JSONResponse:
        db_up = True if settings.fixtures_mode else db.ping(settings.database_url)
        configured = settings.fixtures_mode or (
            settings.llm_provider != "fake"
            and settings.embedding_provider != "fake"
            and bool(settings.jwt_issuer and settings.jwt_audience and settings.jwks_url)
            and bool(settings.identity_base_url)
            and len(read_secret_file(settings.internal_token_file)) >= 32
            and len(read_secret_file(settings.guest_token_secret_file)) >= 32
        )
        ready_now = db_up and configured
        body = {
            "status": "ready" if ready_now else "not_ready",
            "db": "up" if db_up else "down",
            "configuration": "valid" if configured else "invalid",
        }
        return JSONResponse(body, status_code=200 if ready_now else 503)

    @app.get("/metrics")
    async def metrics() -> Response:
        content, content_type = metrics_response()
        return Response(content, media_type=content_type)

    app.include_router(build_router(advisor, recommendations, store, knowledge, settings))
    return app


app = create_app()
