"""Customer identity for the AI service.

Guests get an anonymous session. Logged-in customers are verified with the
same RS256/JWKS pattern Order and Profile use; personalization additionally
requires the customer's recorded consent (see conversations store).
"""
from __future__ import annotations

from dataclasses import dataclass

import httpx
import jwt
from jwt import PyJWKClient

from ..config import Settings


class AuthError(Exception):
    def __init__(self, code: str, status_code: int = 401):
        super().__init__(code)
        self.code = code
        self.status_code = status_code


@dataclass
class Principal:
    customer_id: str | None  # None for guests
    is_guest: bool


class CustomerVerifier:
    def __init__(self, settings: Settings):
        self.settings = settings
        self._jwks: PyJWKClient | None = None
        self._identity = (
            httpx.Client(base_url=settings.identity_base_url.rstrip("/"), timeout=1.8)
            if settings.identity_base_url
            else None
        )

    def _client(self) -> PyJWKClient:
        if not self.settings.jwks_url:
            raise AuthError("auth_not_configured")
        if self._jwks is None:
            self._jwks = PyJWKClient(self.settings.jwks_url, timeout=2)
        return self._jwks

    def close(self) -> None:
        if self._identity is not None:
            self._identity.close()

    def verify(self, token: str) -> Principal:
        try:
            signing_key = self._client().get_signing_key_from_jwt(token)
            claims = jwt.decode(
                token,
                signing_key.key,
                algorithms=["RS256"],
                audience=self.settings.jwt_audience or None,
                issuer=self.settings.jwt_issuer or None,
                options={"require": ["exp", "sub"]},
            )
        except AuthError:
            raise
        except jwt.PyJWTError as exc:
            raise AuthError("invalid_token") from exc
        customer_id = claims.get("user_id")
        if not customer_id:
            raise AuthError("invalid_token")
        if claims.get("service") or claims.get("permission_grants"):
            raise AuthError("customer_access_forbidden", 403)
        if self.settings.identity_base_url:
            self._verify_customer_state(token, claims, str(customer_id))
        return Principal(customer_id=str(customer_id), is_guest=False)

    def _verify_customer_state(self, token: str, claims: dict, customer_id: str) -> None:
        try:
            if self._identity is None:
                raise AuthError("auth_not_configured")
            response = self._identity.get(
                "/api/v1/users/me",
                headers={"Authorization": f"Bearer {token}"},
            )
        except httpx.HTTPError as exc:
            raise AuthError("auth_not_configured") from exc
        if response.status_code == 403:
            raise AuthError("customer_access_forbidden", 403)
        if response.status_code != 200:
            raise AuthError("invalid_token")
        try:
            envelope = response.json()
        except ValueError as exc:
            raise AuthError("invalid_token") from exc
        data = envelope.get("data", envelope) if isinstance(envelope, dict) else {}
        if data.get("principalType") != "CUSTOMER" or data.get("provisioningStatus") != "ACTIVE":
            raise AuthError("customer_access_forbidden", 403)
        if (
            str(data.get("principalId", "")) != customer_id
            or str(data.get("idpSubject", "")) != str(claims.get("sub", ""))
        ):
            raise AuthError("invalid_token")


def authenticate(authorization: str | None, settings: Settings) -> Principal:
    """Resolve the caller: Bearer JWT -> customer; no header -> guest."""
    if authorization:
        if not authorization.startswith("Bearer "):
            raise AuthError("invalid_token")
        return CustomerVerifier(settings).verify(authorization[7:])
    return Principal(customer_id=None, is_guest=True)
