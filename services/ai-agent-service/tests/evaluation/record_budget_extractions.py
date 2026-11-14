"""Record Nova Pro's raw extract_needs output for the adversarial
budget cases (docs ch. 14). MANUAL TOOL — not a test, never run in CI.

Each case costs exactly one Bedrock extraction call. The recordings
are committed (budget_recordings.json) so the gate variants can be
compared offline, deterministically, at zero marginal model cost.
Re-record when the model or the extraction spec changes.

Usage (inside the test image, with the service .env):
    python tests/evaluation/record_budget_extractions.py [output_dir]
"""
import asyncio
import hashlib
import json
import pathlib
import sys
from datetime import datetime, timezone

from ai_agent_service.config import Settings  # noqa: E402
from ai_agent_service.modules.shopping_advisor.workflow import EXTRACT_SPEC  # noqa: E402
from ai_agent_service.providers.llm import BedrockProvider, LLMRequest, Message  # noqa: E402

PROMPT_VERSION = "extract-spec-v2-2026-11-10"
SCHEMA_VERSION = "needs-v1"

HERE = pathlib.Path(__file__).resolve().parent


async def main() -> int:
    out_dir = pathlib.Path(sys.argv[1]) if len(sys.argv) > 1 else HERE
    cases = json.loads((HERE / "budget_cases.json").read_text("utf-8"))
    settings = Settings.load()
    assert settings.llm_provider == "bedrock", settings.llm_provider
    provider = BedrockProvider(settings)
    recordings = []
    for case in cases:
        joined = "\n".join(case["turns"])
        response = await provider.generate(
            LLMRequest(
                messages=[Message(role="user", content=joined)],
                tools=[EXTRACT_SPEC],
            )
        )
        arguments = response.tool_calls[0].arguments if response.tool_calls else {}
        recordings.append(
            {
                "case_id": case["id"],
                "conversation_hash": hashlib.sha256(joined.encode("utf-8")).hexdigest(),
                "raw_arguments": arguments,
            }
        )
        print(f"{case['id']}: {arguments}", flush=True)
    payload = {
        "recorded_at": datetime.now(timezone.utc).isoformat(),
        "model_id": settings.model_id,
        "region": settings.aws_region,
        "prompt_version": PROMPT_VERSION,
        "schema_version": SCHEMA_VERSION,
        "recordings": recordings,
    }
    out_path = out_dir / "budget_recordings.json"
    out_path.write_text(json.dumps(payload, ensure_ascii=False, indent=1), encoding="utf-8")
    print(f"wrote {out_path} ({len(recordings)} cases)")
    return 0


if __name__ == "__main__":
    sys.exit(asyncio.run(main()))
