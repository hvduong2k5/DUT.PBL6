"""Isolated HTTP Identity fixture with ephemeral signing keys for the Compose tests."""

from datetime import UTC, datetime, timedelta
from pathlib import Path

import jwt
import uvicorn
from cryptography.hazmat.primitives import serialization
from cryptography.hazmat.primitives.asymmetric import rsa
from fastapi import Cookie, FastAPI, Header
from fastapi.responses import JSONResponse

ISSUER = "http://identity-fixture:8016"
ADMIN = "913d2296-944c-4a75-b684-d1a0cf8fd4e8"
CUSTOMER = "27bc0f3a-69e5-4aa7-8644-3c7fcb7b3660"
WAREHOUSE = "fb3ac39b-8597-470e-90b3-0b72b01cf4a1"
IDS = {"admin": ADMIN, "customer": CUSTOMER, "warehouse": WAREHOUSE, "outage": ADMIN}
STAFF_SESSIONS = {
    "f47ac10b-58cc-4372-a567-0e02b2c3d479": "admin",
    "f47ac10b-58cc-4372-a567-0e02b2c3d478": "restricted",
    "f47ac10b-58cc-4372-a567-0e02b2c3d477": "customer",
}
KEY = rsa.generate_private_key(public_exponent=65537, key_size=2048)
PUBLIC = KEY.public_key().public_bytes(
    serialization.Encoding.PEM,
    serialization.PublicFormat.SubjectPublicKeyInfo,
)
app = FastAPI()


@app.get("/livez")
def live():
    return {"status": "ok"}


@app.get("/token/{kind}")
def token(kind: str):
    claims = {
        "sub": kind,
        "user_id": IDS[kind],
        "aud": "omama-api",
        "iss": ISSUER,
        "exp": datetime.now(UTC) + timedelta(minutes=5),
    }
    return {"token": jwt.encode(claims, KEY, algorithm="RS256")}


@app.get("/api/v1/public/auth/session")
def session(omama_session: str = Cookie("", alias="OMAMA_SESSION")):
    kind = STAFF_SESSIONS.get(omama_session)
    if kind is None:
        return {"success": True, "data": {"authenticated": False}}
    return {
        "success": True,
        "data": {
            "authenticated": True,
            "principalId": IDS.get(kind, ADMIN),
            "expiresAt": (datetime.now(UTC) + timedelta(minutes=5)).isoformat(),
        },
    }


@app.get("/api/v1/users/me")
def identity(
    authorization: str = Header(""),
    omama_session: str = Cookie("", alias="OMAMA_SESSION"),
):
    if authorization:
        claims = jwt.decode(
            authorization.removeprefix("Bearer "),
            PUBLIC,
            algorithms=["RS256"],
            issuer=ISSUER,
            audience="omama-api",
        )
        kind = claims["sub"]
    else:
        kind = STAFF_SESSIONS.get(omama_session)
    if kind is None:
        return JSONResponse(status_code=401, content={"success": False})
    if kind == "outage":
        return JSONResponse(status_code=503, content={"success": False})
    return {
        "success": True,
        "data": {
            "principalId": IDS.get(kind, ADMIN),
            "idpSubject": kind,
            "principalType": "CUSTOMER" if kind == "customer" else "EMPLOYEE",
            "provisioningStatus": "ACTIVE",
            "displayName": "Nhân viên pilot",
            "email": "pilot@example.test",
            "permissions": ["INVENTORY_MANAGEMENT_VIEW", "BATCH_VIEW"] if kind == "admin" else [],
            "roles": [
                {"roleCode": "ADMIN", "scopeType": "WAREHOUSE" if kind == "warehouse" else "GLOBAL"}
            ],
        },
    }


if __name__ == "__main__":
    Path("/fixture/public.pem").write_bytes(PUBLIC)
    uvicorn.run(app, host="0.0.0.0", port=8016, access_log=False)
