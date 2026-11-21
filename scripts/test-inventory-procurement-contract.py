"""Validate the exact fixture consumed by Go against the canonical producer schema."""
import json
from pathlib import Path
from jsonschema import Draft7Validator, FormatChecker

root = Path(__file__).resolve().parents[1]
schema = json.loads((root / "packages/events/schemas/procurement/v1/goods_received.event.json").read_text(encoding="utf-8"))
fixture = json.loads((root / "services/inventory-service/internal/presentation/kafka/testdata/goods_received.json").read_text(encoding="utf-8"))
Draft7Validator.check_schema(schema)
Draft7Validator(schema, format_checker=FormatChecker()).validate(fixture)
assert fixture["data"]["supplier_id"] == "SUP-HTX-QUANG-THO"
assert fixture["id"] == "receipt-event-01"
assert "quantity_accepted" not in fixture["data"]
print("PASS: canonical GoodsReceived schema validates the Go consumer fixture")

import copy
validator=Draft7Validator(schema,format_checker=FormatChecker())
for unit,quantity,valid in [('kg',2.5,True),('GRAM',1,True),('UNIT',2.5,False),('KG',0.0101,False),('LB',2,False),('UNIT',2147483648,False)]:
    candidate=copy.deepcopy(fixture)
    candidate['data'].update(unit=unit,quantity_received=quantity)
    assert validator.is_valid(candidate)==valid,(unit,quantity,valid)
print('PASS: supported units, integer conversion and overflow contract cases')
