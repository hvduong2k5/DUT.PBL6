import time
from datetime import UTC, datetime
from threading import Lock
from uuid import UUID

import httpx
import jwt
import psycopg
from fastapi import APIRouter, Request
from fastapi.responses import JSONResponse
from prometheus_client import Counter, Histogram
from pydantic import TypeAdapter, ValidationError
from starlette.concurrency import run_in_threadpool

from ..config import AuthSettings, positive
from .models import Behavior, Code, Config, ConsentWrite
from .store import Error, Store

REQUESTS = Counter(
    "analytics_recommendation_requests", "Recommendation results", ["placement", "strategy"]
)
QUERY_SECONDS = Histogram("analytics_recommendation_query_seconds", "Recommendation query time")
EVENTS = Counter("analytics_recommendation_events", "Accepted recommendation feedback", ["result"])
CODE = TypeAdapter(Code)


class CustomerAuth:
    def __init__(self, settings):
        self.settings = settings
        self.client = httpx.Client(timeout=0.5, follow_redirects=False)

    def authenticate(self, request):
        bearer = request.headers.get("authorization", "")
        cookie = request.cookies.get("OMAMA_SESSION", "")
        if not bearer and not cookie:
            raise Error(401, "UNAUTHENTICATED")
        headers, claims = {}, None
        if bearer:
            if not bearer.startswith("Bearer ") or len(bearer) > 8192:
                raise Error(401, "UNAUTHENTICATED")
            try:
                claims = jwt.decode(
                    bearer[7:],
                    self.settings.public_key,
                    algorithms=["RS256"],
                    audience=self.settings.audience,
                    issuer=self.settings.issuer,
                    options={"require": ["exp", "iss", "sub", "aud", "user_id"]},
                )
                UUID(claims["user_id"])
                if claims.get("service"):
                    raise ValueError
            except (jwt.PyJWTError, ValueError, TypeError) as exc:
                raise Error(401, "UNAUTHENTICATED") from exc
            headers["Authorization"] = bearer
        else:
            try:
                if str(UUID(cookie)) != cookie:
                    raise ValueError
            except (ValueError, TypeError) as exc:
                raise Error(401, "UNAUTHENTICATED") from exc
            if request.method != "GET" and (
                not self.settings.session_origin
                or request.headers.get("origin") != self.settings.session_origin
            ):
                raise Error(403, "INVALID_ORIGIN")
            headers["Cookie"] = "OMAMA_SESSION=" + cookie
            if self.settings.session_origin:
                headers["Origin"] = self.settings.session_origin
        try:
            with self.client.stream(
                "GET", self.settings.identity_url.rstrip("/") + "/api/v1/users/me", headers=headers
            ) as response:
                if response.status_code == 401:
                    raise Error(401, "UNAUTHENTICATED")
                if response.status_code == 403:
                    raise Error(403, "CUSTOMER_REQUIRED")
                if response.status_code != 200:
                    raise Error(503, "IDENTITY_UNAVAILABLE")
                body = bytearray()
                for chunk in response.iter_bytes():
                    body.extend(chunk)
                    if len(body) > 65536:
                        raise Error(503, "IDENTITY_UNAVAILABLE")
                import json

                envelope = json.loads(body)
                principal = envelope["data"]
                principal_id = str(UUID(principal["principalId"]))
                if envelope.get("success") is not True:
                    raise ValueError
                if claims and (
                    principal_id != claims["user_id"] or principal["idpSubject"] != claims["sub"]
                ):
                    raise ValueError
                if (
                    principal["principalType"] != "CUSTOMER"
                    or principal["provisioningStatus"] != "ACTIVE"
                ):
                    raise Error(403, "CUSTOMER_REQUIRED")
                return principal_id
        except (httpx.HTTPError, ValueError, KeyError, TypeError) as exc:
            raise Error(503, "IDENTITY_UNAVAILABLE") from exc


class Limiter:
    def __init__(self):
        self.lock = Lock()
        self.counts = {}

    def admit(self, key, budget):
        minute = int(time.monotonic()) // 60
        with self.lock:
            if len(self.counts) > 4096:
                self.counts = {k: v for k, v in self.counts.items() if v[0] == minute}
                if len(self.counts) > 4096:
                    raise Error(429, "RATE_LIMITED")
            before, count = self.counts.get(key, (minute, 0))
            count = count if before == minute else 0
            if count >= budget:
                raise Error(429, "RATE_LIMITED")
            self.counts[key] = (minute, count + 1)


async def read_body(request, model):
    if request.headers.get("content-type", "").split(";")[0] != "application/json":
        raise Error(415, "JSON_REQUIRED")
    body = bytearray()
    async for chunk in request.stream():
        body.extend(chunk)
        if len(body) > 4096:
            raise Error(413, "BODY_TOO_LARGE")
    try:
        return model.model_validate_json(body)
    except ValidationError as exc:
        raise Error(400, "INVALID_RECOMMENDATION_BODY") from exc


def mount(app, config=None, auth=None):
    config = config or Config.load()
    auth = auth or CustomerAuth(AuthSettings.load())
    store = Store(config)
    limiter = Limiter()
    query_budget = positive("REC_QUERY_RATE_LIMIT", 60)
    router = APIRouter()
    app.state.recommendation_store = store

    @app.exception_handler(Error)
    async def error_response(request, exc):
        return JSONResponse(
            status_code=exc.status,
            headers={
                "Cache-Control": "private, no-store",
                **({"Retry-After": "60"} if exc.status == 429 else {}),
            },
            content={
                "code": exc.code,
                "message": "Chưa thể xử lý yêu cầu gợi ý này.",
                "requestId": getattr(request.state, "request_id", "recommendation"),
            },
        )

    @app.exception_handler(psycopg.Error)
    async def database_unavailable(request, _):
        return await error_response(request, Error(503, "RECOMMENDATION_TEMPORARILY_UNAVAILABLE"))

    def subject(request):
        principal = auth.authenticate(request)
        return store.subject(principal)

    @router.get("/api/v1/products/recommended")
    @QUERY_SECONDS.time()
    def recommended(request: Request):
        limiter.admit(("query", request.client.host if request.client else "local"), query_budget)
        query = request.query_params
        placement = query.get("placement")
        allowed = {
            "product_detail": {"placement", "productId", "skuCode", "limit"},
            "cart": {"placement", "skuCodes", "limit"},
            "home": {"placement", "limit"},
        }
        if placement not in allowed or set(query) - allowed[placement]:
            raise Error(400, "INVALID_RECOMMENDATION_QUERY")
        if any(len(query.getlist(k)) != 1 for k in query if k != "skuCodes"):
            raise Error(400, "INVALID_RECOMMENDATION_QUERY")
        try:
            raw_limit = query.get("limit", "6")
            if not raw_limit.isascii() or not raw_limit.isdecimal() or not 1 <= int(raw_limit) <= 6:
                raise ValueError
            context = {}
            if placement == "product_detail":
                context["productId"] = str(UUID(query["productId"]))
                if "skuCode" in query:
                    context["skuCode"] = CODE.validate_python(query["skuCode"])
            if placement == "cart":
                codes = query.getlist("skuCodes")
                if len(codes) > 20:
                    raise ValueError
                context["skuCodes"] = sorted({CODE.validate_python(s) for s in codes})
        except (ValueError, KeyError, ValidationError) as exc:
            raise Error(400, "INVALID_RECOMMENDATION_QUERY") from exc
        enabled = {
            "product_detail": config.related,
            "cart": config.basket,
            "home": config.personalized,
        }
        if not enabled[placement]:
            raise Error(503, "RECOMMENDATION_DISABLED")
        customer = None
        if placement == "home" and (
            request.headers.get("authorization") or request.cookies.get("OMAMA_SESSION")
        ):
            customer = subject(request)
            limiter.admit(("home", customer), query_budget)
        result = store.recommend(placement, context, int(raw_limit), customer, datetime.now(UTC))
        REQUESTS.labels(placement, result["strategy"]).inc()
        return JSONResponse(result, headers={"Cache-Control": "private, no-store"})

    @router.get("/api/v1/recommendations/consent")
    def get_consent(request: Request):
        actor = subject(request)
        limiter.admit(("consent_read", actor), 60)
        result = store.get_consent(actor)
        return JSONResponse(
            result,
            headers={"ETag": f'"{result["consentVersion"]}"', "Cache-Control": "private, no-store"},
        )

    @router.put("/api/v1/recommendations/consent")
    async def consent_write(request: Request):
        actor = await run_in_threadpool(subject, request)
        limiter.admit(("consent_write", actor), 10)
        version = request.headers.get("if-match", "")
        if not version:
            raise Error(428, "CONSENT_PRECONDITION_REQUIRED")
        try:
            if (
                len(version) > 24
                or not version.startswith('"')
                or not version.endswith('"')
                or not version[1:-1].isascii()
                or not version[1:-1].isdecimal()
            ):
                raise ValueError
            expected = int(version[1:-1])
            if expected < 0:
                raise ValueError
            key = UUID(request.headers.get("idempotency-key", ""))
        except ValueError as exc:
            raise Error(400, "INVALID_RECOMMENDATION_QUERY") from exc
        body = await read_body(request, ConsentWrite)
        result = await run_in_threadpool(
            store.set_consent, actor, body.model_dump(), expected, key, datetime.now(UTC)
        )
        return JSONResponse(
            result,
            headers={"ETag": f'"{result["consentVersion"]}"', "Cache-Control": "private, no-store"},
        )

    @router.post("/api/v1/recommendations/events")
    async def event(request: Request):
        actor = await run_in_threadpool(subject, request)
        limiter.admit(("event", actor), 120)
        body = await read_body(request, Behavior)
        result = await run_in_threadpool(store.accept_event, actor, body, datetime.now(UTC))
        EVENTS.labels(result).inc()
        return JSONResponse(
            {"status": result},
            status_code=202 if result == "ACCEPTED" else 200,
            headers={"Cache-Control": "private, no-store"},
        )

    app.include_router(router)
    return store
