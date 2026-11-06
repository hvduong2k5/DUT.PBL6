"""Compose the MVP public target from tracked provider contracts; no network access."""
import copy
from pathlib import Path
import yaml

ROOT = Path(__file__).resolve().parents[3]
SPECS = ROOT / 'docs/03_api_specs'
OUTPUT = SPECS / 'mvp-customer.openapi.yaml'


def openapi30(value):
    if isinstance(value, list):
        return [openapi30(v) for v in value]
    if not isinstance(value, dict):
        return value
    value = {k: openapi30(v) for k, v in value.items()}
    if isinstance(value.get('type'), list):
        types = value['type']
        nonnull = [t for t in types if t != 'null']
        if len(nonnull) != 1:
            raise ValueError('Explicit union conversion required: ' + str(types))
        value['type'] = nonnull[0]
        if 'null' in types:
            value['nullable'] = True
    if 'const' in value:
        value['enum'] = [value.pop('const')]
    return value


def namespace(value, prefix):
    if isinstance(value, dict):
        return {k: (v.replace('#/components/', '#/components/', 1).rsplit('/', 1)[0] + '/' + prefix + v.rsplit('/', 1)[1]
                    if k == '$ref' and v.startswith('#/components/') else namespace(v, prefix)) for k, v in value.items()}
    if isinstance(value, list):
        return [namespace(v, prefix) for v in value]
    return value


def build():
    sources = {name: yaml.safe_load((SPECS / path).read_bytes()) for name, path in {
        'Order': 'order-service.openapi.yaml',
        'Payment': 'order-payment-sepay.candidate.openapi.yaml',
        'Profile': 'profile-service.openapi.yaml',
        'Frontend': 'frontend-contracts/customer-stable.openapi.yaml',
        'D2C': 'openapi_d2c.yaml',
    }.items()}
    spec = {'openapi': '3.0.3', 'info': {
        'title': 'MVP Customer API - integration target', 'version': '1.0.0-integration',
        'description': 'Git integration target mvp-2026-10-26. Provider and consumer implementation remains gated by packages/contracts/mvp/registry.yaml. Async checkout uses quote/revision and CheckoutOperation. Monetary values in the Order flow are integer VND. Original provider specs and frontend references are preserved.'},
        'servers': [{'url': 'https://api.omama.example', 'description': 'Contract placeholder; no live deployment claimed'}],
        'security': [{'bearerAuth': []}], 'paths': {}, 'components': {
            'schemas': {}, 'parameters': {}, 'responses': {},
            'securitySchemes': {'bearerAuth': {'type': 'http', 'scheme': 'bearer', 'bearerFormat': 'JWT',
                'description': 'Verified issuer/audience/signature/expiry and ownership. Identity contract must be implemented; no client identity headers are trusted.'}}},
        'x-contract-status': 'integration-target', 'x-release': 'mvp-2026-10-26'}
    for name in ['Order', 'Payment', 'Profile', 'Frontend', 'D2C']:
        source = sources[name]
        for kind in ['schemas', 'parameters', 'responses', 'securitySchemes']:
            for key, value in source.get('components', {}).get(kind, {}).items():
                spec['components'].setdefault(kind, {})[name + key] = namespace(copy.deepcopy(value), name)
    schemas = spec['components']['schemas']
    schemas['PublicError'] = {'type': 'object', 'additionalProperties': False,
        'required': ['code', 'message', 'requestId'], 'properties': {
            'code': {'type': 'string', 'minLength': 1}, 'message': {'type': 'string', 'minLength': 1},
            'requestId': {'type': 'string', 'minLength': 1}, 'retryable': {'type': 'boolean'},
            'operation_id': {'type': 'string', 'format': 'uuid'},
            'details': {'type': 'object', 'additionalProperties': True}}}
    for name in ['Order', 'Payment', 'Profile', 'Frontend', 'D2C']:
        for key, parameter in spec['components']['parameters'].items():
            if key.startswith(name) and parameter.get('in') == 'header' and parameter.get('name') in ('X-Idempotency-Key', 'Idempotency-Key'):
                parameter.update(name='Idempotency-Key', required=True, schema={'type': 'string', 'minLength': 16, 'maxLength': 128, 'pattern': '^[!-~]+$'})
    selected_order = [p for p in sources['Order']['paths'] if p.startswith('/api/v1/') and '/admin/' not in p and '/payments/' not in p]
    # Orders are the workflow authority. References keep their old direct-checkout shape.
    for name, selected in [('Order', selected_order), ('Payment', [p for p in sources['Payment']['paths'] if p.startswith('/api/v1/orders/')])]:
        for path in selected:
            spec['paths'][path] = namespace(copy.deepcopy(sources[name]['paths'][path]), name)
            for method, operation in spec['paths'][path].items():
                if method not in ('get', 'post', 'put', 'patch', 'delete'):
                    continue
                operation['operationId'] = name.lower() + '_' + operation.get('operationId', method + path.replace('/', '_'))
                security = operation.get('security', sources[name].get('security', [{'bearerAuth': []}]))
                known = sources[name].get('components', {}).get('securitySchemes', {})
                operation['security'] = [{(name + key if key in known else key): value for key, value in alternative.items()} for alternative in security]
                operation['x-provider-contract'] = 'order-service' if name == 'Order' else 'order-payment-sepay'
                operation['x-implementation-status'] = 'provider-or-adapter-verification-required'
    # Owner-scoped Profile public boundary, provider DTO/concurrency retained.
    for path, item in sources['Profile']['paths'].items():
        if path not in ('/profile', '/profile/addresses', '/profile/addresses/{id}', '/profile/addresses/{id}/default'):
            continue
        spec['paths']['/api/v1' + path] = namespace(copy.deepcopy(item), 'Profile')
        for method, operation in spec['paths']['/api/v1' + path].items():
            if method not in ('get', 'post', 'put', 'patch', 'delete'):
                continue
            operation['operationId'] = 'profile_' + operation['operationId']
            operation['security'] = [{'bearerAuth': []}]
            operation['parameters'] = [p for p in operation.get('parameters', []) if p.get('name') not in ('X-User-ID', 'X-User-Role')]
            operation['x-provider-contract'] = 'profile-service'
            operation['x-gateway-identity'] = 'Map verified subject to X-User-ID; add X-Internal-Token server-side. Strip client-supplied identity/trust headers.'
    # Catalog public DTO remains the established frontend shape; BFF maps proto JSON.
    for path in ('/api/v1/categories', '/api/v1/products', '/api/v1/products/search', '/api/v1/products/{id_or_slug}'):
        spec['paths'][path] = namespace(copy.deepcopy(sources['Frontend']['paths'][path]), 'Frontend')
        op = spec['paths'][path]['get']
        op['operationId'] = 'catalog_' + op['operationId']
        op['security'] = []
        op['x-provider-contract'] = 'hvduong.catalog.v1.CatalogPublicService'
        op['x-bff-mapping'] = 'products -> items, page_info -> pagination; Money converted without precision loss. See packages/contracts/mvp/MIGRATION.md.'
        # Preserve existing category/price/sort filtering instead of generic generated filters.
        legacy = sources['D2C']['paths'][path]['get']
        op['parameters'] = namespace(copy.deepcopy(legacy.get('parameters', [])), 'D2C')
    # One error boundary and idempotency spelling. Provider shapes stay unchanged.
    for path, item in spec['paths'].items():
        for method, operation in item.items():
            if method not in ('get', 'post', 'put', 'patch', 'delete'):
                continue
            for parameter in operation.get('parameters', []):
                if parameter.get('in') == 'header' and parameter.get('name') == 'X-Idempotency-Key':
                    parameter['name'] = 'Idempotency-Key'
                    parameter['x-provider-name'] = 'X-Idempotency-Key'
            operation['x-contract-status'] = 'integration-target'
            for status in list(operation['responses']):
                if str(status).startswith(('4', '5')):
                    operation['responses'][status] = {'description': 'Mapped public error; provider data is sanitized.',
                        'content': {'application/json': {'schema': {'$ref': '#/components/schemas/PublicError'}}}}
    # Imported component-only errors stay as references; published operations use PublicError.
    # Public Profile rejects email writes: identity owns email changes.
    update = spec['paths']['/api/v1/profile']['put'].get('requestBody', {})
    request_schema = update.get('content', {}).get('application/json', {}).get('schema', {})
    if '$ref' in request_schema:
        request_schema = schemas[request_schema['$ref'].rsplit('/', 1)[1]]
    request_schema.get('properties', {}).pop('email', None)
    # Publish only reachable components; unused staff/refund/source DTOs stay in provider specs.
    needed = set()
    def visit(value):
        if isinstance(value, dict):
            if '$ref' in value:
                ref = value['$ref']
                if ref.startswith('#/components/'):
                    _, _, kind, key = ref.split('/')
                    pair = (kind, key)
                    if pair not in needed:
                        needed.add(pair)
                        visit(spec['components'][kind][key])
            for key, child in value.items():
                if key == 'security':
                    for alternative in child:
                        needed.update(('securitySchemes', name) for name in alternative)
                elif key != '$ref':
                    visit(child)
        elif isinstance(value, list):
            for child in value:
                visit(child)
    visit({'paths': spec['paths'], 'security': spec['security']})
    spec['components'] = {kind: {key: value for key, value in entries.items() if (kind, key) in needed}
                          for kind, entries in spec['components'].items()}
    spec['components'] = {kind: entries for kind, entries in spec['components'].items() if entries}
    return openapi30(spec)


if __name__ == '__main__':
    OUTPUT.write_text(yaml.safe_dump(build(), sort_keys=False, allow_unicode=True, width=110), encoding='utf-8')
    print(OUTPUT)
