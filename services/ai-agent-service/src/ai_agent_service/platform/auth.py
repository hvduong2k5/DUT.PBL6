"""Customer identity for the AI service.

Guests get an anonymous session. Logged-in customers are verified with the
same RS256/JWKS pattern Order and Profile use; personalization additionally
requires the customer's recorded consent (see conversations store).
"""
from __future__ import annotations

from dataclasses import dataclass

import jwt
from jwt import PyJWKClient

from ..config import Settings


class AuthError(Exception):
    def __init__(self, code: str):
        super().__init__(code)
        self.code = code


@dataclass
class Principal:
    customer_id: str | None  # None for guests
    is_guest: bool


class CustomerVerifier:
    def __init__(self, settings: Settings):
        self.settings = settings
        self._jwks: PyJWKClient | None = None

    def _client(self) -> PyJWKClient:
        if not self.settings.jwks_url:
            raise AuthError("auth_not_configured")
        if self._jwks is None:
            self._jwks = PyJWKClient(self.settings.jwks_url)
        return self._jwks

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
        return Principal(customer_id=str(customer_id), is_guest=False)


def authenticate(authorization: str | None, settings: Settings) -> Principal:
    """Resolve the caller: Bearer JWT -> customer; no header -> guest."""
    if authorization:
        if not authorization.startswith("Bearer "):
            raise AuthError("invalid_token")
        return CustomerVerifier(settings).verify(authorization[7:])
    return Principal(customer_id=None, is_guest=True)
