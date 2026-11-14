from __future__ import annotations

import asyncio
import base64
import hashlib
import hmac
import inspect
import json
import uuid
from collections.abc import AsyncIterator

from fastapi import APIRouter, Request
from fastapi.responses import StreamingResponse
from pydantic import BaseModel, Field

from ...api_errors import api_error
from ...config import Settings, read_secret_file
from ...platform.auth import AuthError, Principal
from ...platform.conversations import ConversationStore
from ...platform.knowledge import KnowledgeStore
from ..recommendation.service import RecommendationService
from .workflow import ShoppingAdvisor


class CreateConversationResponse(BaseModel):
    conversation_id: str
    owner_type: str
    expires_at: str
    guest_token: str | None = None


class MessageRequest(BaseModel):
    content: str = Field(min_length=1, max_length=2000)


class FeedbackRequest(BaseModel):
    conversation_id: str = Field(min_length=1, max_length=128)
    product_id: str = Field(min_length=1, max_length=128)
    signal: str = Field(min_length=1, max_length=32)


class ConsentRequest(BaseModel):
    history_use: bool


class KnowledgeSyncRequest(BaseModel):
    title: str = Field(min_length=1, max_length=200)
    source: str = Field(default="api", min_length=1, max_length=100)
    text: str = Field(min_length=1, max_length=200_000)
    product_id: str | None = Field(default=None, max_length=128)
    visibility: str = Field(default="public", pattern="^(public|internal)$")


async def _principal(request: Request, settings: Settings) -> Principal:
    override = getattr(request.app.state, "authenticator", None)
    authorization = request.headers.get("authorization")
    try:
        if override is not None:
            principal = override(authorization)
            return await principal if inspect.isawaitable(principal) else principal
        if authorization:
            if not authorization.startswith("Bearer "):
                raise AuthError("invalid_token")
            return await asyncio.to_thread(
                request.app.state.customer_verifier.verify, authorization[7:]
            )
        return Principal(customer_id=None, is_guest=True)
    except AuthError as exc:
        status = 503 if exc.code == "auth_not_configured" else exc.status_code
        raise api_error(status, exc.code) from exc


async def _owned_conversation(request: Request, store: ConversationStore,
                              conversation_id: str, settings: Settings):
    conv = await asyncio.to_thread(store.get, conversation_id)
    principal = await _principal(request, settings)
    if conv is None:
        raise api_error(404, "conversation_not_found")
    if conv.owner_type == "customer" and conv.customer_id != principal.customer_id:
        raise api_error(404, "conversation_not_found")
    if conv.owner_type == "guest" and not principal.is_guest:
        raise api_error(404, "conversation_not_found")
    if conv.owner_type == "guest":
        supplied = request.headers.get("x-guest-conversation-token", "")
        supplied_hash = hashlib.sha256(supplied.encode("utf-8")).hexdigest()
        if not conv.guest_token_hash or not hmac.compare_digest(
            supplied_hash, conv.guest_token_hash
        ):
            raise api_error(401, "guest_token_required")
    return conv, principal


def _require_internal(request: Request, settings: Settings) -> None:
    expected = read_secret_file(settings.internal_token_file)
    if len(expected) < 32:
        raise api_error(503, "internal_auth_not_configured")
    authorization = request.headers.get("authorization", "")
    supplied = authorization[7:] if authorization.startswith("Bearer ") else ""
    if not supplied:
        supplied = request.headers.get("x-internal-token", "")
    if not hmac.compare_digest(supplied, expected):
        raise api_error(401, "invalid_internal_token")


def _idempotency_key(request: Request) -> str:
    key = request.headers.get("idempotency-key", "")
    try:
        return str(uuid.UUID(key))
    except (ValueError, AttributeError):
        raise api_error(400, "idempotency_key_required") from None


def _save_feedback(
    database_url: str,
    conversation_id: str,
    customer_id: str | None,
    product_id: str,
    signal: str,
    idempotency_key: str,
    idempotency_owner: str,
    request_hash: str,
) -> bool:
    from ...db import connect

    with connect(database_url) as conn:
        row = conn.execute(
            "INSERT INTO recommendation_feedback "
            "(conversation_id, customer_id, product_id, signal, "
            "idempotency_key, idempotency_owner, request_hash) "
            "VALUES (%s, %s, %s, %s, %s, %s, %s) "
            "ON CONFLICT (idempotency_owner, idempotency_key) "
            "WHERE idempotency_owner IS NOT NULL AND idempotency_key IS NOT NULL "
            "DO UPDATE SET idempotency_key = EXCLUDED.idempotency_key "
            "WHERE recommendation_feedback.request_hash = EXCLUDED.request_hash "
            "RETURNING id",
            (
                conversation_id, customer_id, product_id, signal,
                idempotency_key, idempotency_owner, request_hash,
            ),
        ).fetchone()
    return row is not None


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
        principal = await _principal(request, settings)
        idempotency_key = _idempotency_key(request)
        guest_token = None
        if principal.is_guest:
            secret = read_secret_file(settings.guest_token_secret_file)
            if settings.fixtures_mode and not secret:
                secret = "fixture-only-guest-token-secret-32-bytes"
            if len(secret) < 32:
                raise api_error(503, "guest_token_secret_not_configured")
            digest = hmac.new(
                secret.encode("utf-8"), idempotency_key.encode("utf-8"), hashlib.sha256
            ).digest()
            guest_token = base64.urlsafe_b64encode(digest).decode("ascii").rstrip("=")
        guest_token_hash = (
            hashlib.sha256(guest_token.encode("utf-8")).hexdigest()
            if guest_token is not None
            else None
        )
        days = (
            settings.retention_guest_days
            if principal.is_guest
            else settings.retention_customer_days
        )
        conv = await asyncio.to_thread(
            store.create,
            owner_type="guest" if principal.is_guest else "customer",
            customer_id=principal.customer_id,
            retention_days=days,
            guest_token_hash=guest_token_hash,
            idempotency_key=idempotency_key,
            idempotency_owner=(
                "guest" if principal.is_guest else f"customer:{principal.customer_id}"
            ),
        )
        return CreateConversationResponse(
            conversation_id=conv.id,
            owner_type=conv.owner_type,
            expires_at=conv.expires_at.isoformat(),
            guest_token=guest_token,
        )

    @router.post("/api/v1/ai/shopping/conversations/{conversation_id}/messages")
    async def post_message(request: Request, conversation_id: str,
                           body: MessageRequest) -> StreamingResponse:
        conv, principal = await _owned_conversation(request, store, conversation_id, settings)
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
        conv, _ = await _owned_conversation(request, store, conversation_id, settings)
        slots, messages = await asyncio.gather(
            asyncio.to_thread(store.get_slots, conv.id),
            asyncio.to_thread(store.list_messages, conv.id),
        )
        return {
            "conversation_id": conv.id,
            "owner_type": conv.owner_type,
            "slots": slots,
            "messages": [
                {"role": m.role, "content": m.content}
                for m in messages
            ],
        }

    @router.post("/api/v1/ai/shopping/feedback")
    async def post_feedback(request: Request, body: FeedbackRequest) -> dict:
        idempotency_key = _idempotency_key(request)
        if body.signal not in ("helpful", "not_helpful", "clicked", "dismissed"):
            raise api_error(422, "invalid_signal")
        conv, principal = await _owned_conversation(
            request, store, body.conversation_id, settings
        )
        record = {
            "conversation_id": conv.id,
            "customer_id": principal.customer_id,
            "product_id": body.product_id,
            "signal": body.signal,
        }
        idempotency_owner = (
            f"customer:{principal.customer_id}"
            if principal.customer_id
            else f"guest:{conv.id}"
        )
        request_hash = hashlib.sha256(
            json.dumps(record, sort_keys=True, separators=(",", ":")).encode("utf-8")
        ).hexdigest()
        if settings.fixtures_mode:
            identity = (idempotency_owner, idempotency_key)
            previous_hash = request.app.state.feedback_idempotency.get(identity)
            if previous_hash and not hmac.compare_digest(previous_hash, request_hash):
                raise api_error(409, "idempotency_conflict")
            if previous_hash is None:
                request.app.state.feedback_records.append(record)
                request.app.state.feedback_idempotency[identity] = request_hash
        else:
            saved = await asyncio.to_thread(
                _save_feedback,
                settings.database_url,
                conv.id,
                principal.customer_id,
                body.product_id,
                body.signal,
                idempotency_key,
                idempotency_owner,
                request_hash,
            )
            if not saved:
                raise api_error(409, "idempotency_conflict")
        return {"status": "recorded"}

    @router.get("/api/v1/ai/recommendations")
    async def get_recommendations(request: Request, placement: str = "home",
                                  product_id: str | None = None,
                                  sku_codes: str = "") -> dict:
        await _principal(request, settings)
        if placement == "product_detail":
            if not product_id:
                raise api_error(422, "product_id_required")
            products = await recommendations.related(product_id)
        elif placement == "cart":
            products = await recommendations.basket(
                [s for s in sku_codes.split(",") if s]
            )
        elif placement == "home":
            products = await recommendations.home(request.headers.get("authorization"))
        else:
            raise api_error(422, "invalid_placement")
        return {"items": [p.public_dict() for p in products]}

    @router.get("/api/v1/ai/consent")
    async def get_consent(request: Request) -> dict:
        principal = await _principal(request, settings)
        if principal.is_guest:
            raise api_error(401, "login_required")
        allowed = await asyncio.to_thread(
            store.get_history_consent, principal.customer_id
        )
        return {"history_use": allowed}

    @router.put("/api/v1/ai/consent")
    async def put_consent(request: Request, body: ConsentRequest) -> dict:
        principal = await _principal(request, settings)
        if principal.is_guest:
            raise api_error(401, "login_required")
        await asyncio.to_thread(
            store.set_history_consent, principal.customer_id, body.history_use
        )
        return {"history_use": body.history_use}

    @router.post("/api/v1/ai/knowledge/products/sync")
    async def sync_knowledge(request: Request, body: KnowledgeSyncRequest) -> dict:
        _require_internal(request, settings)
        chunks = await knowledge.ingest(
            body.title, body.source, body.text,
            product_id=body.product_id, visibility=body.visibility,
        )
        return {"chunks": chunks}

    return router
