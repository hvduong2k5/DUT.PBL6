"""Check actual connected-runtime response/event JSON against pinned contracts."""
import argparse
import importlib.util
import json
from pathlib import Path
import re
import jsonschema
import yaml


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--test-log', required=True, type=Path)
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[3]
    helper = importlib.util.spec_from_file_location('payment_evidence', Path(__file__).with_name('verify-payment-evidence.py'))
    module = importlib.util.module_from_spec(helper)
    helper.loader.exec_module(module)
    specs = root / 'docs/03_api_specs'
    checkout = module.nullable(yaml.safe_load((specs / 'order-cart-checkout.candidate.openapi.yaml').read_text(encoding='utf-8')))
    order = module.nullable(yaml.safe_load((specs / 'order-core.candidate.openapi.yaml').read_text(encoding='utf-8')))
    schemas = {'Quote': checkout, 'CheckoutOperation': checkout, 'OrderDetail': order}
    found, event_types = set(), set()
    responses = events = 0
    chunks = []
    for line in args.test_log.read_text(encoding='utf-8-sig').splitlines():
        try:
            event = json.loads(line)
        except ValueError:
            continue
        chunks.append(event.get('Output', ''))
    # go test -json may split one t.Log line across multiple output records.
    for output in ''.join(chunks).splitlines():
        match = re.search(r'CONNECTED_CONTRACT (\w+) (\{.*\})', output)
        if match:
            name, raw = match.groups()
            document = dict(schemas[name], **{'$ref': '#/components/schemas/' + name})
            jsonschema.Draft7Validator(document, format_checker=jsonschema.FormatChecker()).validate(json.loads(raw))
            found.add(name)
            responses += 1
        match = re.search(r'CONNECTED_EVENT (\{.*\})', output)
        if match:
            payload = json.loads(match.group(1))
            kind = payload['type']
            if kind.startswith('vn.omama.order.'):
                path = root / 'packages/events/schemas/order/v3' / (kind.split('.')[3] + '.event.json')
            else:
                path = specs / 'order-payment-events.candidate.schema.json'
            schema = json.loads(path.read_text(encoding='utf-8'))
            jsonschema.Draft7Validator(schema, format_checker=jsonschema.FormatChecker()).validate(payload)
            event_types.add(kind)
            events += 1
    if found != set(schemas):
        raise SystemExit('Missing responses: ' + str(set(schemas) - found))
    required = {'vn.omama.order.' + x + '.v3' for x in ('placed', 'paid', 'fulfillment_ready', 'completed')}
    if not required <= event_types:
        raise SystemExit('Missing events: ' + str(required - event_types))
    print(json.dumps({'validated_responses': responses, 'response_schemas': sorted(found), 'validated_events': events, 'event_types': sorted(event_types)}))


if __name__ == '__main__':
    main()
