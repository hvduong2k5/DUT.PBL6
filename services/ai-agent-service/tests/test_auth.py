import time

import jwt as pyjwt
import pytest
from cryptography.hazmat.primitives.asymmetric import rsa

from ai_agent_service.config import Settings
from ai_agent_service.platform.auth import AuthError, CustomerVerifier, authenticate


def test_guest_without_header():
    principal = authenticate(None, Settings())
    assert principal.is_guest and principal.customer_id is None


def test_bearer_without_jwks_configured():
    with pytest.raises(AuthError) as exc:
        authenticate("Bearer abc", Settings())
    assert exc.value.code == "auth_not_configured"


def test_malformed_authorization_header():
    with pytest.raises(AuthError) as exc:
        authenticate("Token abc", Settings())
    assert exc.value.code == "invalid_token"


class _FakeKey:
    def __init__(self, key):
        self.key = key


class _FakeJwksClient:
    def __init__(self, public_key):
        self.public_key = public_key

    def get_signing_key_from_jwt(self, token):
        return _FakeKey(self.public_key)


def test_verify_extracts_user_id_claim():
    private_key = rsa.generate_private_key(public_exponent=65537, key_size=2048)
    settings = Settings(
        jwks_url="https://issuer.example/jwks", jwt_audience="ai-agent", jwt_issuer="iss"
    )
    verifier = CustomerVerifier(settings)
    verifier._jwks = _FakeJwksClient(private_key.public_key())
    token = pyjwt.encode(
        {
            "sub": "kc-subject-1",
            "user_id": "cust-42",
            "exp": int(time.time()) + 600,
            "aud": "ai-agent",
            "iss": "iss",
        },
        private_key,
        algorithm="RS256",
    )
    principal = verifier.verify(token)
    assert principal.customer_id == "cust-42"
    assert principal.is_guest is False


def test_verify_rejects_token_without_user_id():
    private_key = rsa.generate_private_key(public_exponent=65537, key_size=2048)
    settings = Settings(jwks_url="https://issuer.example/jwks")
    verifier = CustomerVerifier(settings)
    verifier._jwks = _FakeJwksClient(private_key.public_key())
    token = pyjwt.encode(
        {"sub": "svc", "exp": int(time.time()) + 600}, private_key, algorithm="RS256"
    )
    with pytest.raises(AuthError) as exc:
        verifier.verify(token)
    assert exc.value.code == "invalid_token"
