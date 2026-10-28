"""Offline contract gates: schemas, code generation, wire compatibility and examples.

--record-baselines is an explicit maintainer operation using pinned Git objects.
Normal verification cannot update compatibility baselines.
"""
import argparse
import copy
import hashlib
import importlib
import importlib.metadata
import json
import re
from pathlib import Path
import subprocess
import sys
import tempfile

import grpc_tools
from google.api import annotations_pb2
from google.protobuf.descriptor_pb2 import FileDescriptorSet
from jsonschema import Draft7Validator, FormatChecker
from jsonschema.validators import validator_for
from openapi_spec_validator import validate_spec
from openapi_schema_validator import OAS30Validator
import yaml

sys.dont_write_bytecode = True
HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[2]
PROTO = ROOT / 'packages/proto'
sys.path.insert(0, str(HERE))
from build_public_contract import build  # noqa: E402


def check(condition, message):
    if not condition:
        raise AssertionError(message)


def baseline_json(value):
    # Keep each field descriptor tuple on one line so generated baselines remain reviewable.
    text = json.dumps(value, indent=2)
    def compact(match):
        values = json.loads(match.group(0))
        return json.dumps(values) if all(not isinstance(v, (dict, list)) for v in values) else match.group(0)
    return re.sub(r'\[\n[^\[\]{}]*?\n\s*\]', compact, text) + '\n'


def run(args, **kwargs):
    return subprocess.run(args, check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, **kwargs)


def compile_proto(source, generated=None, go_plugin=None, go_grpc_plugin=None):
    names = sorted(p.relative_to(source).as_posix() for p in source.rglob('*.proto'))
    check(names, 'No protobuf files')
    with tempfile.TemporaryDirectory() as temporary:
        output = Path(temporary) / 'descriptors.pb'
        args = [sys.executable, '-m', 'grpc_tools.protoc', '-I' + str(source),
                '-I' + str(Path(grpc_tools.__file__).parent / '_proto'),
                '-I' + str(Path(annotations_pb2.__file__).parents[2]),
                '--include_imports', '--descriptor_set_out=' + str(output)]
        if generated:
            generated.mkdir(parents=True, exist_ok=True)
            args += ['--python_out=' + str(generated), '--grpc_python_out=' + str(generated)]
            if go_plugin:
                go_output = generated.parent / 'go'
                go_output.mkdir(exist_ok=True)
                args += ['--plugin=protoc-gen-go=' + str(go_plugin), '--go_out=' + str(go_output), '--go_opt=module=github.com/omamx/proto']
                if go_grpc_plugin:
                    args += ['--plugin=protoc-gen-go-grpc=' + str(go_grpc_plugin), '--go-grpc_out=' + str(go_output), '--go-grpc_opt=module=github.com/omamx/proto']
        result = run(args + names, cwd=source)
        if result.stderr:
            print(result.stderr.decode('utf-8'), file=sys.stderr, end='')
        return output.read_bytes(), names


def wire_manifest(raw):
    descriptor = FileDescriptorSet.FromString(raw)
    result = {'files': {}, 'messages': {}, 'enums': {}, 'services': {}}

    def enum(value, prefix):
        result['enums'][prefix + value.name] = {v.name: v.number for v in value.value}

    def message(value, prefix):
        name = prefix + value.name
        result['messages'][name] = {'fields': {str(f.number): [f.name, f.json_name, f.type, f.type_name, f.label,
            f.proto3_optional, value.oneof_decl[f.oneof_index].name if f.HasField('oneof_index') else None]
            for f in value.field}, 'reserved': [[v.start, v.end] for v in value.reserved_range],
            'reserved_names': list(value.reserved_name)}
        for v in value.enum_type:
            enum(v, name + '.')
        for v in value.nested_type:
            message(v, name + '.')

    for file in descriptor.file:
        if file.name.startswith('google/'):
            continue
        result['files'][file.name] = {'package': file.package, 'options': {
            key: getattr(file.options, key) for key in ['go_package', 'java_package', 'java_multiple_files', 'java_outer_classname']
            if file.options.HasField(key)}}
        for value in file.message_type:
            message(value, file.package + '.')
        for value in file.enum_type:
            enum(value, file.package + '.')
        for value in file.service:
            result['services'][file.package + '.' + value.name] = {
                m.name: [m.input_type, m.output_type, m.client_streaming, m.server_streaming] for m in value.method}
    return result


def compatibility(old, new, excluded_packages=()):
    count = 0
    for name, file in old['files'].items():
        if file['package'] in excluded_packages:
            continue
        check(name in new['files'], 'Removed proto file: ' + name)
        check(file['package'] == new['files'][name]['package'], 'Changed package: ' + name)
        for key, value in file['options'].items():
            check(new['files'][name]['options'].get(key) == value, 'Changed existing language option: ' + name + ':' + key)
    for name, message in old['messages'].items():
        if any(name.startswith(p + '.') for p in excluded_packages):
            continue
        check(name in new['messages'], 'Removed message: ' + name)
        current = new['messages'][name]
        for tag, field in message['fields'].items():
            check(current['fields'].get(tag) == field, 'Changed field/tag/presence: ' + name + ':' + tag)
            count += 1
        for start, end in message['reserved']:
            check(all(not start <= int(tag) < end for tag in current['fields']), 'Reused reserved field: ' + name)
            check(any(a <= start and b >= end for a, b in current['reserved']), 'Removed reserved range: ' + name)
        check(set(message['reserved_names']) <= set(current['reserved_names']), 'Removed reserved name: ' + name)
    for name, values in old['enums'].items():
        if any(name.startswith(p + '.') for p in excluded_packages):
            continue
        check(name in new['enums'], 'Removed enum: ' + name)
        check(all(new['enums'][name].get(k) == v for k, v in values.items()), 'Changed enum number: ' + name)
    for name, methods in old['services'].items():
        if any(name.startswith(p + '.') for p in excluded_packages):
            continue
        check(name in new['services'], 'Removed service: ' + name)
        check(all(new['services'][name].get(k) == v for k, v in methods.items()), 'Removed/changed RPC: ' + name)
    return count


def record_baselines():
    sources = json.loads((HERE / 'sources.json').read_bytes())['sources']
    for name in ['dev', 'order', 'catalog', 'shipping']:
        sha = sources[name]['sha']
        with tempfile.TemporaryDirectory() as temporary:
            source = Path(temporary)
            paths = run(['git', 'ls-tree', '-r', '--name-only', sha, '--', 'packages/proto'], cwd=ROOT).stdout.decode().splitlines()
            hashes = {}
            for path in paths:
                if not path.endswith('.proto'):
                    continue
                data = run(['git', 'show', sha + ':' + path], cwd=ROOT).stdout
                target = source / Path(path).relative_to('packages/proto')
                target.parent.mkdir(parents=True, exist_ok=True)
                target.write_bytes(data)
                hashes[path] = hashlib.sha256(data).hexdigest()
            raw, _ = compile_proto(source)
            baseline = {'source_sha': sha, 'source_file_sha256': hashes, 'wire': wire_manifest(raw)}
            path = HERE / 'baselines' / (name + '.json')
            path.parent.mkdir(exist_ok=True)
            path.write_text(baseline_json(baseline), encoding='utf-8')
            print('Recorded pinned baseline:', name, sha)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--record-baselines', action='store_true')
    parser.add_argument('--go-plugin', type=Path)
    parser.add_argument('--go-grpc-plugin', type=Path)
    parser.add_argument('--output', type=Path, default=ROOT / '.system_generated/mvp-contracts/verification.json')
    options = parser.parse_args()
    if options.record_baselines:
        record_baselines()
        return
    result = {'release': 'mvp-2026-10-26', 'scope': 'static contract/codegen compatibility only; no live service or E2E acceptance'}
    public = yaml.safe_load((ROOT / 'docs/03_api_specs/mvp-customer.openapi.yaml').read_bytes())
    check(public == build(), 'Generated public contract is stale; run build_public_contract.py')
    spec_count = 0
    snapshots = ROOT / 'docs/03_api_specs/frontend-contracts'
    snapshot_hashes = json.loads((snapshots / 'snapshot-hashes.json').read_bytes())
    for name, hashes in snapshot_hashes['files'].items():
        check(hashlib.sha256((snapshots / name).read_bytes()).hexdigest() == hashes['git_normalized_sha256'],
              'Historical frontend reference changed: ' + name)
    for path in sorted((ROOT / 'docs/03_api_specs').rglob('*.yaml')):
        document = yaml.safe_load(path.read_bytes())
        if not isinstance(document, dict) or 'openapi' not in document:
            continue
        try:
            validate_spec(document, base_uri=path.resolve().as_uri())
        except Exception as exc:
            raise AssertionError('Invalid OpenAPI: ' + str(path.relative_to(ROOT)) + ': ' + str(exc)[:220]) from exc
        spec_count += 1
    result['openapi'] = {'validated_specs': spec_count, 'generated_public_contract': 'current'}
    schema_count = 0
    for directory in ['packages/events/schemas', 'packages/events/profiles']:
        for path in (ROOT / directory).rglob('*.json'):
            schema = json.loads(path.read_bytes())
            validator_for(schema).check_schema(schema)
            schema_count += 1
    examples = negatives = 0
    ingress = Draft7Validator(json.loads((ROOT / 'packages/events/profiles/mvp/order-source.event.json').read_bytes()), format_checker=FormatChecker())
    accepted = {'signature': '0' * 64, 'id': 'packing-accepted-example', 'source': 'fulfillment',
                'order_id': '11111111-1111-4111-8111-111111111111',
                'resource_id': '22222222-2222-4222-8222-222222222222',
                'version': 1, 'generation': 1, 'type': 'PACKING_ACCEPTED'}
    ingress.validate(accepted)
    check(not ingress.is_valid({**accepted, 'type': 'FULFILLMENT_ACCEPTED'}),
          'Ingress accepts a name Order core cannot process')
    examples += 1
    negatives += 1
    pairs = [(p, ROOT / 'packages/events/examples/order/v3' / p.name.replace('.event', ''))
             for p in (ROOT / 'packages/events/schemas/order/v3').glob('*.json')]
    pairs += [(ROOT / 'packages/events/schemas/fulfillment/v2/packing_completed.event.json', ROOT / 'packages/events/examples/fulfillment/v2/packing_completed.json'),
              (ROOT / 'packages/events/schemas/shipping/v2/shipment_delivered.event.json', ROOT / 'packages/events/examples/shipping/v2/shipment_delivered.json')]
    for path, example in pairs:
        validator = Draft7Validator(json.loads(path.read_bytes()), format_checker=FormatChecker())
        payload = json.loads(example.read_bytes())
        validator.validate(payload)
        examples += 1
        bad = copy.deepcopy(payload)
        if '/order/v3/' in path.as_posix():
            bad['data'].pop('ordinal')
        elif 'packing_completed' in path.name:
            bad['data'].pop('package_id')
        else:
            bad['data'].pop('resource_version')
        check(not validator.is_valid(bad), 'Schema accepts missing required source evidence: ' + path.name)
        negatives += 1
    # Preserve the legacy packing wire contract exactly, including its old required set.
    legacy = json.loads((ROOT / 'packages/events/schemas/fulfillment/v1/packing_completed.event.json').read_bytes())
    check('package_id' not in legacy['properties']['data']['required'], 'Breaking v1 packing required fields')
    sources = json.loads((HERE / 'sources.json').read_bytes())
    for path, expected in sources['preserved_event_baselines'].items():
        check(hashlib.sha256((ROOT / path).read_bytes().replace(b'\r\n', b'\n')).hexdigest() == expected,
              'Modified frozen event baseline: ' + path)
    result['events'] = {'schemas_checked': schema_count, 'valid_examples': examples, 'invalid_examples_rejected': negatives}
    with tempfile.TemporaryDirectory() as temporary:
        generated = Path(temporary) / 'python'
        raw, names = compile_proto(PROTO, generated, options.go_plugin, options.go_grpc_plugin)
        current = wire_manifest(raw)
        for service, methods in {
            'omamx.order.v1.OrderService': ['GetShippingSnapshot'],
            'omamx.fulfillment.v1.FulfillmentService': ['GetTaskRevision'],
            'omamx.shipping.v2.ShippingService': ['CalculateShippingFee', 'GetShipmentRevision'],
        }.items():
            check(set(methods) <= set(current['services'].get(service, {})), 'Missing MVP source proof RPC: ' + service)
        # The proto folder named profile must not resolve to Python's stdlib profile.py.
        for directory in [p for p in generated.rglob('*') if p.is_dir()]:
            (directory / '__init__.py').touch()
        modules = [n.removesuffix('.proto').replace('/', '.') for n in names]
        run([sys.executable, '-I', '-c',
             'import sys,importlib,json; sys.dont_write_bytecode=True; sys.path.insert(0,sys.argv[1]); '
             '[(importlib.import_module(n+"_pb2"),importlib.import_module(n+"_pb2_grpc")) for n in json.loads(sys.argv[2])]',
             str(generated), json.dumps(modules)])
        result['protobuf'] = {'compiled_files': len(names), 'python_codegen_and_import': 'PASS', 'baselines': {}}
        if options.go_plugin:
            go_files = sorted((generated.parent / 'go').rglob('*.pb.go'))
            check(len(go_files) >= len(names), 'Missing Go code generation output')
            packages = {}
            for path in go_files:
                package = re.search(r'^package (\w+)', path.read_text(encoding='utf-8'), re.M).group(1)
                packages.setdefault(path.parent, set()).add(package)
            check(all(len(values) == 1 for values in packages.values()), 'Generated Go folder mixes package names')
            result['protobuf']['go_codegen'] = {'status': 'PASS', 'generated_files': len(go_files),
                'go_grpc': 'PASS' if options.go_grpc_plugin else 'NOT_RUN'}
        for path in sorted((HERE / 'baselines').glob('*.json')):
            baseline = json.loads(path.read_bytes())
            excluded = ('omamx.shipping.v1',) if path.stem == 'order' else ()
            count = compatibility(baseline['wire'], current, excluded)
            result['protobuf']['baselines'][path.stem] = {'source_sha': baseline['source_sha'], 'preserved_fields': count,
                'exception': 'Unimplemented Order 3.2 Shipping v1 quote proposal migrated explicitly to v2; provider v1 retained.' if excluded else None}
        check(len(result['protobuf']['baselines']) == 4, 'Expected four pinned wire baselines')
        # Prove the guard rejects both a changed field and a removed RPC.
        dev = json.loads((HERE / 'baselines/dev.json').read_bytes())['wire']
        for category in ['field', 'rpc']:
            bad = copy.deepcopy(current)
            if category == 'field':
                bad['messages']['omamx.common.v1.Money']['fields'].pop('1')
            else:
                bad['services']['omamx.inventory.v1.InventoryService'].pop('ReserveStock')
            try:
                compatibility(dev, bad)
            except AssertionError:
                negatives += 1
            else:
                raise AssertionError('Compatibility guard failed to reject ' + category)
        registry = yaml.safe_load((HERE / 'registry.yaml').read_bytes())
        for entry in registry['contracts']:
            check((ROOT / entry['artifact']).is_file(), 'Missing registered contract: ' + entry['id'])
            if 'rpc' in entry:
                service, method = entry['rpc'].rsplit('.', 1)
                check(method in current['services'].get(service, {}), 'Missing registered RPC: ' + entry['rpc'])
        # Write descriptors outside Git for other contract tooling.
        descriptor = ROOT / '.system_generated/mvp-contracts/descriptor.pb'
        descriptor.parent.mkdir(parents=True, exist_ok=True)
        descriptor.write_bytes(raw)
    response = public['paths']['/api/v1/checkout']['post']['responses']
    check('202' in response and '201' in response, 'Checkout acceptance and placed operation must both be documented')
    check('post' in public['paths']['/api/v1/checkout/quote'], 'Missing quote workflow')
    check('get' in public['paths']['/api/v1/checkout-operations/{id}'], 'Missing operation polling')
    def expand(value):
        if isinstance(value, list):
            return [expand(v) for v in value]
        if not isinstance(value, dict):
            return value
        if '$ref' in value:
            target = public
            for part in value['$ref'][2:].split('/'):
                target = target[part]
            return expand(target)
        return {k: expand(v) for k, v in value.items()}
    checkout = public['paths']['/api/v1/checkout']['post']
    request_validator = OAS30Validator(expand(checkout['requestBody']['content']['application/json']['schema']))
    request_validator.validate({'quote_id': '11111111-1111-4111-8111-111111111111', 'cart_revision': 1, 'payment_method': 'VIETQR'})
    check(not request_validator.is_valid({'channel': 'D2C_WEB', 'shipping_address': {}, 'items': [], 'payment_method': 'VIETQR'}),
          'Public target still accepts legacy direct checkout without quote/revision')
    negatives += 1
    profile_update = public['paths']['/api/v1/profile']['put']['requestBody']['content']['application/json']['schema']
    check('email' not in expand(profile_update).get('properties', {}), 'Email writes leaked into Profile public target')
    result['registry'] = {'contracts_checked': len(registry['contracts']), 'runtime_acceptance': 'PENDING'}
    result['negative_guards'] = negatives
    result['tool_versions'] = {p: importlib.metadata.version(p) for p in ['grpcio-tools', 'protobuf', 'googleapis-common-protos', 'openapi-spec-validator', 'PyYAML', 'jsonschema']}
    result['status'] = 'PASS'
    options.output.parent.mkdir(parents=True, exist_ok=True)
    options.output.write_text(json.dumps(result, indent=2) + '\n', encoding='utf-8')
    print(json.dumps(result, indent=2))


if __name__ == '__main__':
    main()
