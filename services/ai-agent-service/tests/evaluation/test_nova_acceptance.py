"""Golden-set evaluation — Suite B (live-model behavioural regression).

The same 30 cases and the same shared assertions as Suite A, but the
app is built from the environment (Settings.load()) so the LLM is the
real production model (Bedrock Nova Pro) and embeddings are Titan V2,
against a real Postgres. Fixture backends stay on (Catalog/Order are
not live yet); auth uses the same fake mapping as Suite A.

This suite does NOT block merges and is not run in CI: model behaviour
is a regression signal, tracked in docs ch. 12. It runs only when
explicitly enabled (it costs real model calls):

    AI_NOVA_ACCEPTANCE=1 pytest tests/evaluation/test_nova_acceptance.py

First run (2026-11-10, pre-calibration assertions): 25/30 — see docs
ch. 12 for the failure taxonomy and the resulting ADR-01..04.
"""
import dataclasses
import json
import os
import pathlib
import sys

import pytest
from fastapi.testclient import TestClient

sys.path.insert(0, str(pathlib.Path(__file__).parent.parent))
sys.path.insert(0, str(pathlib.Path(__file__).parent))

from assertions import check_case  # noqa: E402
from test_shopping_flow import chat, create_conversation  # noqa: E402

from ai_agent_service.config import Settings  # noqa: E402
from ai_agent_service.main import create_app  # noqa: E402
from ai_agent_service.platform.auth import Principal  # noqa: E402

pytestmark = pytest.mark.skipif(
    os.environ.get("AI_NOVA_ACCEPTANCE") != "1",
    reason="Suite B is opt-in: set AI_NOVA_ACCEPTANCE=1 (real model calls, real cost)",
)

CASES = json.loads((pathlib.Path(__file__).parent / "golden_cases.json").read_text("utf-8"))


@pytest.fixture(scope="module")
def client():
    settings = dataclasses.replace(Settings.load(), fixtures_mode=True)
    assert settings.llm_provider == "bedrock", settings.llm_provider
    assert settings.embedding_provider == "titan", settings.embedding_provider
    app = create_app(settings)

    def fake_auth(header):
        if header and header.startswith("Bearer "):
            return Principal(customer_id=header[7:].strip(), is_guest=False)
        return Principal(customer_id=None, is_guest=True)

    app.state.authenticator = fake_auth
    with TestClient(app) as test_client:
        yield test_client


@pytest.mark.parametrize("case", CASES, ids=[c["id"] for c in CASES])
def test_golden_case(client, case):
    headers = (
        {"Authorization": f"Bearer {case['auth']}"} if case["auth"] != "guest" else {}
    )
    conversation_id = create_conversation(client, headers)
    events: list[dict] = []
    for turn in case["turns"]:
        events = chat(client, conversation_id, turn, headers)

    failures = check_case(events, case)
    assert not failures, "\n".join(failures)
