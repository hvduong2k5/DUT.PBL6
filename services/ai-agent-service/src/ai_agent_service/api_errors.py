from __future__ import annotations

from dataclasses import dataclass


@dataclass
class ApiError(Exception):
    status_code: int
    code: str
    message: str

    def __post_init__(self) -> None:
        super().__init__(self.message)


SAFE_MESSAGES = {
    "auth_not_configured": "Authentication is temporarily unavailable.",
    "conversation_not_found": "Conversation was not found.",
    "customer_access_forbidden": "This operation is available to customers only.",
    "guest_token_required": "A valid guest conversation token is required.",
    "guest_token_secret_not_configured": "Guest session security is not configured.",
    "idempotency_key_required": "A valid Idempotency-Key header is required.",
    "idempotency_conflict": "The Idempotency-Key was already used for a different request.",
    "internal_auth_not_configured": "Internal authentication is not configured.",
    "internal_error": "An unexpected error occurred.",
    "invalid_internal_token": "A valid internal credential is required.",
    "invalid_placement": "The recommendation placement is invalid.",
    "invalid_request": "The request is invalid.",
    "invalid_signal": "The feedback signal is invalid.",
    "invalid_token": "The access token is invalid.",
    "login_required": "Authentication is required.",
    "product_id_required": "product_id is required for this placement.",
}


def api_error(status_code: int, code: str) -> ApiError:
    return ApiError(status_code, code, SAFE_MESSAGES.get(code, "The request could not be processed."))
