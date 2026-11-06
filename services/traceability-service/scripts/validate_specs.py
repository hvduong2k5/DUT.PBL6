"""Validate service contracts; --runtime also checks local synthetic HTTP data."""
import argparse
import copy
import json
from pathlib import Path
from urllib.parse import urlsplit
from urllib.request import HTTPRedirectHandler, Request, build_opener

import yaml
from jsonschema import Draft7Validator, FormatChecker
from openapi_schema_validator import OAS30Validator
from openapi_spec_validator import validate_spec

SERVICE = Path(__file__).resolve().parents[1]
REPOSITORY = SERVICE.parents[1]


class RejectRedirects(HTTPRedirectHandler):
    """A local validation endpoint cannot redirect a bearer token elsewhere."""

    def redirect_request(self, request, response, code, message, headers, new_url):
        return None


def read_runtime_profile(url, token):
    request = Request(url, headers={'Authorization': 'Bearer ' + token})
    with build_opener(RejectRedirects()).open(request, timeout=5) as response:
        return json.load(response)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--runtime', action='store_true')
    parser.add_argument('--api', default='http://127.0.0.1:8003')
    args = parser.parse_args()
    document = yaml.safe_load((REPOSITORY / 'docs/03_api_specs/traceability-service.openapi.yaml').read_text(encoding='utf-8'))
    validate_spec(document)
    print('PASS: complete OpenAPI document')
    schema = json.loads((SERVICE / 'contracts/events/traceability.event.schema.json').read_text(encoding='utf-8'))
    Draft7Validator.check_schema(schema)
    validator = Draft7Validator(schema, format_checker=FormatChecker())
    sample = json.loads((REPOSITORY / 'docs/04_testing/traceability-service/artifacts/published.event.json').read_text(encoding='utf-8'))
    validator.validate(sample)
    for event_type, status in [('profile.published', 'ACTIVE'), ('profile.suspended', 'SUSPENDED'), ('profile.archived', 'ARCHIVED'), ('certification.status.changed', 'ACTIVE')]:
        event = copy.deepcopy(sample)
        event['type'] = 'vn.omama.traceability.' + event_type + '.v1'
        event['data']['status'] = status
        if event_type.startswith('certification.'):
            event['data']['eligible'] = False
            event['data']['reason_code'] = 'CERTIFICATE_EXPIRED'
        validator.validate(event)
        invalid = copy.deepcopy(event)
        invalid['data']['internal_evidence'] = 'must never be distributed'
        assert list(validator.iter_errors(invalid)), 'schema accepted an internal evidence field'
    print('PASS: actual broker sample + four synthetic event variants and privacy rejection')
    if not args.runtime:
        return
    api = urlsplit(args.api)
    if api.scheme != 'http' or api.hostname not in {'127.0.0.1', 'localhost', '::1'} or api.username or api.password or api.query or api.fragment or api.path not in {'', '/'}:
        raise ValueError('runtime validator accepts only an explicit loopback HTTP origin')

    def validate_instance(instance, fragment):
        root = copy.deepcopy(fragment)
        root['components'] = document['components']
        OAS30Validator(root, format_checker=OAS30Validator.FORMAT_CHECKER).validate(instance)

    seed = json.loads((SERVICE / '.sandbox/create.json').read_text(encoding='utf-8'))
    validate_instance(seed, document['paths']['/api/v1/admin/trace/records']['post']['requestBody']['content']['application/json']['schema'])
    report = json.loads((SERVICE / '.sandbox/smoke-report.json').read_text(encoding='utf-8'))
    token = (SERVICE / '.sandbox/editor.jwt').read_text().strip()
    profile = read_runtime_profile(args.api.rstrip('/') + '/api/v1/admin/trace/records/' + report['profile_id'], token)
    validate_instance(profile, {'$ref': '#/components/schemas/Profile'})
    print('PASS: synthetic create request + actual redacted admin response')


if __name__ == '__main__':
    main()
