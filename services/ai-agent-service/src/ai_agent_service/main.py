from __future__ import annotations

from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.responses import JSONResponse, Response

from . import __version__, db
from .config import Settings
from .modules.recommendation.service import RecommendationService
from .modules.shopping_advisor.routes import build_router
from .modules.shopping_advisor.seed import seed_knowledge
from .modules.shopping_advisor.tools import Backends, build_registry
from .modules.shopping_advisor.workflow import ShoppingAdvisor
from .observability import metrics_response, setup_logging
from .platform.conversations import (
    ConversationStore,
    MemoryConversationStore,
    PostgresConversationStore,
)
from .platform.eligibility import ProductEligibility
from .platform.gateway import ToolGateway
from .platform.knowledge import (
    KnowledgeStore,
    MemoryKnowledgeStore,
    PostgresKnowledgeStore,
)
from .providers import build_embedding_provider, build_llm_provider


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

    return Backends(
        catalog=HttpCatalog(settings.catalog_base_url),
        inventory=HttpInventory(settings.inventory_base_url),
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
    gateway = ToolGateway(build_registry(backends), timeout_seconds=settings.tool_timeout_seconds)
    llm = build_llm_provider(settings)
    eligibility = ProductEligibility(
        concurrency=settings.eligibility_concurrency,
        deadline_seconds=settings.eligibility_deadline_seconds,
    )
    recommendations = RecommendationService(backends, gateway, eligibility)
    advisor = ShoppingAdvisor(
        llm, gateway, knowledge, settings, recommendations, eligibility
    )

    @asynccontextmanager
    async def lifespan(app: FastAPI):
        if settings.fixtures_mode:
            from .integrations.fixtures import FixtureCatalog

            await seed_knowledge(knowledge, FixtureCatalog())
        yield

    app = FastAPI(title="ai-agent-service", version=__version__, lifespan=lifespan)
    app.state.settings = settings
    app.state.store = store
    app.state.knowledge = knowledge
    app.state.backends = backends
    app.state.feedback_records: list[dict] = []

    @app.get("/health")
    async def health() -> dict:
        return {"status": "ok", "service": "ai-agent-service", "version": __version__}

    @app.get("/health/ready")
    async def ready() -> JSONResponse:
        up = True if settings.fixtures_mode else db.ping(settings.database_url)
        body = {"status": "ready" if up else "not_ready", "db": "up" if up else "down"}
        return JSONResponse(body, status_code=200 if up else 503)

    @app.get("/metrics")
    async def metrics() -> Response:
        content, content_type = metrics_response()
        return Response(content, media_type=content_type)

    app.include_router(build_router(advisor, recommendations, store, knowledge, settings))
    return app


app = create_app()
