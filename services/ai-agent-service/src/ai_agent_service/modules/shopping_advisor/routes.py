from __future__ import annotations

import json
from collections.abc import AsyncIterator

from fastapi import APIRouter, HTTPException, Request
from fastapi.responses import StreamingResponse
from pydantic import BaseModel, Field

from ...config import Settings
from ...platform.auth import AuthError, Principal, authenticate
from ...platform.conversations import ConversationStore
from ...platform.knowledge import KnowledgeStore
from ..recommendation.service import RecommendationService
from .workflow import ShoppingAdvisor


class CreateConversationResponse(BaseModel):
    conversation_id: str
    owner_type: str
    expires_at: str


class MessageRequest(BaseModel):
    content: str = Field(min_length=1, max_length=2000)


class FeedbackRequest(BaseModel):
    conversation_id: str
    product_id: str
    signal: str  # helpful | not_helpful | clicked | dismissed


class ConsentRequest(BaseModel):
    history_use: bool


class KnowledgeSyncRequest(BaseModel):
    title: str
    source: str = "api"
    text: str
    product_id: str | None = None
    visibility: str = "public"


def _principal(request: Request, settings: Settings) -> Principal:
    override = getattr(request.app.state, "authenticator", None)
    authorization = request.headers.get("authorization")
    try:
        if override is not None:
            return override(authorization)
        return authenticate(authorization, settings)
    except AuthError as exc:
        raise HTTPException(status_code=401, detail=exc.code) from exc


def _owned_conversation(request: Request, store: ConversationStore,
                        conversation_id: str, settings: Settings):
    conv = store.get(conversation_id)
    principal = _principal(request, settings)
    if conv is None:
        raise HTTPException(status_code=404, detail="conversation_not_found")
    if conv.owner_type == "customer" and conv.customer_id != principal.customer_id:
        raise HTTPException(status_code=404, detail="conversation_not_found")
    if conv.owner_type == "guest" and not principal.is_guest:
        raise HTTPException(status_code=404, detail="conversation_not_found")
    return conv, principal


async def _sse(events: AsyncIterator[dict]) -> AsyncIterator[str]:
    async for event in events:
        yield f"data: {json.dumps(event, ensure_ascii=False)}\n\n"


def build_router(
    advisor: ShoppingAdvisor,
    recommendations: RecommendationService,
    store: ConversationStore,
    knowledge: KnowledgeStore,
    settings: Settings,
) -> APIRouter:
    router = APIRouter()

    @router.post("/api/v1/ai/shopping/conversations", status_code=201)
    async def create_conversation(request: Request) -> CreateConversationResponse:
        principal = _principal(request, settings)
        days = (
            settings.retention_guest_days
            if principal.is_guest
            else settings.retention_customer_days
        )
        conv = store.create(
            owner_type="guest" if principal.is_guest else "customer",
            customer_id=principal.customer_id,
            retention_days=days,
        )
        return CreateConversationResponse(
            conversation_id=conv.id,
            owner_type=conv.owner_type,
            expires_at=conv.expires_at.isoformat(),
        )

    @router.post("/api/v1/ai/shopping/conversations/{conversation_id}/messages")
    async def post_message(request: Request, conversation_id: str,
                           body: MessageRequest) -> StreamingResponse:
        conv, principal = _owned_conversation(request, store, conversation_id, settings)
        events = advisor.turn_events(
            store,
            conv,
            principal,
            body.content,
            authorization=request.headers.get("authorization"),
        )
        return StreamingResponse(
            _sse(events),
            media_type="text/event-stream",
            headers={"Cache-Control": "no-cache", "X-Accel-Buffering": "no"},
        )

    @router.get("/api/v1/ai/shopping/conversations/{conversation_id}")
    async def get_conversation(request: Request, conversation_id: str) -> dict:
        conv, _ = _owned_conversation(request, store, conversation_id, settings)
        return {
            "conversation_id": conv.id,
            "owner_type": conv.owner_type,
            "slots": store.get_slots(conv.id),
            "messages": [
                {"role": m.role, "content": m.content}
                for m in store.list_messages(conv.id)
            ],
        }

    @router.post("/api/v1/ai/shopping/feedback")
    async def post_feedback(request: Request, body: FeedbackRequest) -> dict:
        if body.signal not in ("helpful", "not_helpful", "clicked", "dismissed"):
            raise HTTPException(status_code=422, detail="invalid_signal")
        conv, principal = _owned_conversation(request, store, body.conversation_id, settings)
        record = {
            "conversation_id": conv.id,
            "customer_id": principal.customer_id,
            "product_id": body.product_id,
            "signal": body.signal,
        }
        if settings.fixtures_mode:
            request.app.state.feedback_records.append(record)
        else:
            from ...db import connect

            with connect(settings.database_url) as conn:
                conn.execute(
                    "INSERT INTO recommendation_feedback "
                    "(conversation_id, customer_id, product_id, signal) VALUES (%s, %s, %s, %s)",
                    (conv.id, principal.customer_id, body.product_id, body.signal),
                )
        return {"status": "recorded"}

    @router.get("/api/v1/ai/recommendations")
    async def get_recommendations(request: Request, placement: str = "home",
                                  product_id: str | None = None,
                                  sku_codes: str = "") -> dict:
        if placement == "product_detail":
            if not product_id:
                raise HTTPException(status_code=422, detail="product_id_required")
            products = await recommendations.related(product_id)
        elif placement == "cart":
            products = await recommendations.basket(
                [s for s in sku_codes.split(",") if s]
            )
        elif placement == "home":
            products = await recommendations.home(request.headers.get("authorization"))
        else:
            raise HTTPException(status_code=422, detail="invalid_placement")
        return {"items": [p.public_dict() for p in products]}

    @router.get("/api/v1/ai/consent")
    async def get_consent(request: Request) -> dict:
        principal = _principal(request, settings)
        if principal.is_guest:
            raise HTTPException(status_code=401, detail="login_required")
        return {"history_use": store.get_history_consent(principal.customer_id)}

    @router.put("/api/v1/ai/consent")
    async def put_consent(request: Request, body: ConsentRequest) -> dict:
        principal = _principal(request, settings)
        if principal.is_guest:
            raise HTTPException(status_code=401, detail="login_required")
        store.set_history_consent(principal.customer_id, body.history_use)
        return {"history_use": body.history_use}

    @router.post("/api/v1/ai/knowledge/products/sync")
    async def sync_knowledge(body: KnowledgeSyncRequest) -> dict:
        # Internal ingestion endpoint. Phase 2 will gate it with staff
        # permission grants when the internal modules land.
        chunks = await knowledge.ingest(
            body.title, body.source, body.text,
            product_id=body.product_id, visibility=body.visibility,
        )
        return {"chunks": chunks}

    return router
