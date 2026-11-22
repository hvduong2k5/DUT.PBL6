import json
from pathlib import Path

import yaml
from jsonschema import Draft202012Validator
from openapi_spec_validator import validate
from pydantic import TypeAdapter

from analytics_service.recommendation.models import (
    Basket,
    Behavior,
    Catalog,
    ConsentWrite,
    Manifest,
    Sellability,
)

CONTRACTS = Path("/app/contracts")
if not CONTRACTS.exists():
    CONTRACTS = Path(__file__).resolve().parents[3] / "docs/03_api_specs"


def test_recommendation_machine_contract_matches_runtime():
    path = CONTRACTS / "analytics-recommendation.v1.schema.json"
    schema = json.loads(path.read_text(encoding="utf-8"))
    Draft202012Validator.check_schema(schema)
    actual = TypeAdapter(
        Manifest | Catalog | Sellability | Basket | ConsentWrite | Behavior
    ).json_schema()
    actual["$schema"] = "https://json-schema.org/draft/2020-12/schema"
    assert schema == actual
    spec_path = CONTRACTS / "analytics-recommendation.candidate.openapi.yaml"
    validate(yaml.safe_load(spec_path.read_text(encoding="utf-8")), base_uri=spec_path.as_uri())
