"""Golden-set evaluation — Suite A (deterministic contract).

~30 Vietnamese cases in 4 groups (Functional / Retrieval / Grounding /
Safety) run against the full API with the deterministic FakeProvider
and fixture backends. This suite BLOCKS merges: it verifies workflow
transitions, card contracts, price integrity, citations and safety
behaviour. Assertions are shared with Suite B (assertions.py) and
collect every failure, structure first.

Suite B (test_nova_acceptance.py) runs the same cases against the
live model as the production acceptance; see docs ch. 12.
"""
import json
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

CASES = json.loads((pathlib.Path(__file__).parent / "golden_cases.json").read_text("utf-8"))


@pytest.fixture(scope="module")
def client():
    app = create_app(Settings())

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
