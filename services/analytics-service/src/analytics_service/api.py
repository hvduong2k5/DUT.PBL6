import logging
import re
import time
from datetime import UTC, date, datetime, timedelta
from threading import Lock
from uuid import UUID, uuid4

import httpx
import jwt
import psycopg
from clickhouse_connect.driver.exceptions import ClickHouseError
from fastapi import FastAPI, Request
from fastapi.exceptions import RequestValidationError
from fastapi.responses import JSONResponse, Response
from prometheus_client import CONTENT_TYPE_LATEST, generate_latest
from pydantic import BaseModel, ValidationError
from starlette.exceptions import HTTPException

from . import db, report, snapshots
from .config import AuthSettings, Settings
from .metrics import QUERY_SECONDS
from .migrate import clickhouse

log = logging.getLogger(__name__)
PREFIX = "/api/v1/admin/analytics"
BACKEND_ERRORS = (psycopg.Error, ClickHouseError, OSError, db.Conflict)
DATE_PATTERN = re.compile(r"[0-9]{4}-[0-9]{2}-[0-9]{2}")


class APIError(Exception):
    def __init__(self, status, code, message, retry_after=None):
        self.status = status
        self.code = code
        self.message = message
        self.retry_after = retry_after


def unavailable():
    return APIError(503, "ANALYTICS_TEMPORARILY_UNAVAILABLE", "Dashboard tạm thời gián đoạn.")


def unauthenticated():
    return APIError(401, "UNAUTHENTICATED", "Cần token nhân viên hợp lệ.")


def forbidden():
    return APIError(403, "FORBIDDEN", "Tài khoản chưa được cấp quyền Dashboard.")


class Role(BaseModel):
    roleCode: str
    scopeType: str


class Persona(BaseModel):
    principalId: str
    idpSubject: str
    principalType: str
    provisioningStatus: str
    roles: list[Role]


class IdentityResponse(BaseModel):
    success: bool
    data: Persona


class PilotAuth:
    def __init__(self, settings: AuthSettings):
        self.settings = settings
        self.client = httpx.Client(timeout=2, follow_redirects=False)

    def authorize(self, header):
        if not header.startswith("Bearer ") or len(header) > 8192:
            raise unauthenticated()
        try:
            claims = jwt.decode(
                header.removeprefix("Bearer "),
                self.settings.public_key,
                algorithms=["RS256"],
                issuer=self.settings.issuer,
                audience=self.settings.audience,
                leeway=5,
                options={"require": ["exp", "iss", "sub", "aud"]},
            )
            if claims.get("service"):
                raise unauthenticated()
            actor = str(UUID(claims.get("user_id", "")))
        except (jwt.PyJWTError, ValueError, TypeError, AttributeError) as exc:
            raise unauthenticated() from exc
        if actor not in self.settings.pilot_ids:
            raise forbidden()
        try:
            with self.client.stream(
                "GET",
                self.settings.identity_url.rstrip("/") + "/api/v1/users/me",
                headers={"Authorization": header},
            ) as response:
                if response.status_code == 401:
                    raise unauthenticated()
                if response.status_code == 403:
                    raise forbidden()
                if response.status_code != 200:
                    raise unavailable()
                content = bytearray()
                for chunk in response.iter_bytes():
                    content.extend(chunk)
                    if len(content) > 1_048_576:
                        raise unavailable()
                envelope = IdentityResponse.model_validate_json(content)
        except (httpx.HTTPError, ValidationError) as exc:
            raise unavailable() from exc
        if not envelope.success:
            raise unavailable()
        persona = envelope.data
        if (
            persona.principalId != actor
            or persona.idpSubject != claims["sub"]
            or persona.principalType != "EMPLOYEE"
            or persona.provisioningStatus != "ACTIVE"
            or not any(
                role.roleCode == "ADMIN" and role.scopeType == "GLOBAL" for role in persona.roles
            )
        ):
            raise forbidden()
        return actor


class RateLimit:
    def __init__(self):
        self.entries = {}
        self.lock = Lock()

    def admit(self, actor):
        now = int(time.monotonic())
        with self.lock:
            ends, count = self.entries.get(actor, (now + 60, 0))
            if now >= ends:
                ends, count = now + 60, 0
            if count >= 60:
                raise APIError(429, "RATE_LIMITED", "Đã vượt giới hạn truy vấn.", ends - now)
            self.entries[actor] = (ends, count + 1)


def date_range(start, end, compare=False):
    try:
        if not DATE_PATTERN.fullmatch(start) or not DATE_PATTERN.fullmatch(end):
            raise ValueError
        lower, upper = date.fromisoformat(start), date.fromisoformat(end)
        length = upper - lower
        if not 1 <= length.days <= 90:
            raise ValueError
        if compare:
            lower - length
    except (ValueError, OverflowError) as exc:
        raise APIError(400, "INVALID_DATE_RANGE", "Kỳ phải có từ 1 đến 90 ngày.") from exc
    return lower, upper


def create_app(settings=None, auth=None):
    settings = settings or Settings.load()
    auth = auth or PilotAuth(AuthSettings.load())
    app = FastAPI(docs_url=None, redoc_url=None, openapi_url=None)
    freshness = timedelta(seconds=settings.freshness_seconds)
    limiter = RateLimit()

    @app.middleware("http")
    async def headers(request, call_next):
        request.state.request_id = str(uuid4())
        started = time.monotonic()
        response = await call_next(request)
        response.headers["X-Request-ID"] = request.state.request_id
        if request.url.path.startswith(PREFIX):
            response.headers["Cache-Control"] = "private, no-store"
            QUERY_SECONDS.observe(time.monotonic() - started)
            log.info(
                "dashboard status=%s request_id=%s", response.status_code, request.state.request_id
            )
        return response

    @app.exception_handler(APIError)
    async def api_error(request, error):
        headers = {}
        if error.retry_after:
            headers["Retry-After"] = str(error.retry_after)
        if error.status == 401:
            headers["WWW-Authenticate"] = "Bearer"
        return JSONResponse(
            status_code=error.status,
            headers=headers,
            content={
                "code": error.code,
                "message": error.message,
                "requestId": request.state.request_id,
            },
        )

    @app.exception_handler(RequestValidationError)
    async def invalid_query(request, error):
        date_missing = any(item["loc"][-1] in ("from", "to") for item in error.errors())
        code = "INVALID_DATE_RANGE" if date_missing else "INVALID_QUERY"
        return await api_error(
            request, APIError(400, code, "Thiếu tham số hoặc tham số không hợp lệ.")
        )

    @app.exception_handler(HTTPException)
    async def http_error(request, error):
        if request.url.path.startswith(PREFIX):
            return await api_error(
                request, APIError(error.status_code, "INVALID_QUERY", "Route không hợp lệ.")
            )
        return JSONResponse(status_code=error.status_code, content={"detail": error.detail})

    def authorize(request):
        limiter.admit(auth.authorize(request.headers.get("Authorization", "")))

    def load(start, end, snapshot_id=None):
        try:
            with db.connect(settings) as conn:
                snapshot = db.get_generation(conn, settings.scope, settings.pipeline, snapshot_id)
            if snapshot is None:
                if snapshot_id:
                    raise APIError(
                        404, "SNAPSHOT_NOT_FOUND", "Tải lại Dashboard để lấy snapshot mới."
                    )
                raise unavailable()
            client = clickhouse(settings)
            try:
                approved = [
                    day
                    for text in snapshot["coverage"]["days"]
                    if start <= (day := date.fromisoformat(text)) < end
                ]
                if approved:
                    counts = snapshots.read_daily(
                        client,
                        snapshot["generation_id"],
                        min(approved),
                        max(approved) + report.DAY,
                    )
                    for day in approved:
                        expected = snapshot["coverage"]["days"][day.isoformat()].get("counts")
                        if expected is None or any(
                            counts.get((day, milestone), 0) != count
                            for milestone, count in expected.items()
                        ):
                            raise db.Conflict("Rollup differs from the certified counts")
                else:
                    counts = {}
            finally:
                client.close()
        except BACKEND_ERRORS as exc:
            log.warning("Serving store failure: %s", type(exc).__name__)
            raise unavailable() from exc
        return snapshot, counts

    @app.get("/livez")
    def live():
        return {"status": "ok"}

    @app.get("/readyz")
    def ready():
        try:
            with db.connect(settings) as conn:
                conn.execute("SELECT 1 FROM serving_generation LIMIT 1")
            client = clickhouse(settings)
            try:
                client.query("SELECT 1 FROM analytics_db.order_milestone_daily LIMIT 1")
            finally:
                client.close()
        except BACKEND_ERRORS as exc:
            raise unavailable() from exc
        return {"status": "ready"}

    @app.get("/metrics")
    def metrics():
        return Response(generate_latest(), media_type=CONTENT_TYPE_LATEST)

    @app.get(PREFIX + "/dashboard")
    def dashboard(request: Request):
        authorize(request)
        args = request.query_params
        compare = args.get("comparison")
        if compare not in (None, "previous_period"):
            raise APIError(400, "INVALID_QUERY", "Kỳ so sánh không hợp lệ.")
        start, end = date_range(args.get("from", ""), args.get("to", ""), compare)
        snapshot, counts = load(start - (end - start) if compare else start, end)
        return report.dashboard(snapshot, start, end, compare, counts, datetime.now(UTC), freshness)

    @app.get(PREFIX + "/metrics/{metric_id}/series")
    def series(request: Request, metric_id: str):
        authorize(request)
        args = request.query_params
        if metric_id not in snapshots.ORDER_METRICS:
            raise APIError(400, "INVALID_QUERY", "Metric không hỗ trợ chuỗi ngày.")
        try:
            snapshot_id = UUID(args.get("snapshotId", ""))
        except ValueError as exc:
            raise APIError(400, "INVALID_QUERY", "snapshotId không hợp lệ.") from exc
        start, end = date_range(args.get("from", ""), args.get("to", ""))
        snapshot, counts = load(start, end, snapshot_id)
        return report.series(snapshot, metric_id, start, end, counts, datetime.now(UTC), freshness)

    return app
