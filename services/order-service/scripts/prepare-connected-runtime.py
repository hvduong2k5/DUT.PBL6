"""Prepare private TEST configuration and import secrets into a Docker volume.

Requires existing Sandbox api-token.txt, webhook-secret.txt and the canonical
bank-accounts-private.json response. Never prints credentials/account details.
Repeated runs retain connection IDs, keys and coverage start. No provider writes.
"""
import argparse
import base64
from datetime import datetime, timedelta, timezone
import io
import json
import os
from pathlib import Path
import secrets
import subprocess
import tarfile
import uuid


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--directory', required=True, type=Path)
    args = parser.parse_args()
    directory = args.directory.resolve(strict=True)
    repo = Path(__file__).resolve().parents[3]
    if directory.is_relative_to(repo):
        raise SystemExit('Private configuration must be outside the repository')
    for name, minimum in [('api-token.txt', 64), ('webhook-secret.txt', 32)]:
        if len((directory / name).read_bytes().strip()) < minimum:
            raise SystemExit('Missing or invalid private ' + name)
    settings_file = directory / 'runtime-settings.json'
    receiver_file = directory / 'runtime-receiver.json'
    initialized = settings_file.exists()
    if not settings_file.exists():
        envelope = json.loads((directory / 'bank-accounts-private.json').read_text(encoding='utf-8-sig'))
        accounts = envelope['data']
        if not isinstance(accounts, list) or len(accounts) != 1:
            raise SystemExit('Require exactly one bank account or prepare reviewed private runtime settings manually')
        account = accounts[0]
        bank = str(uuid.UUID(account['id']))
        receiver = dict(bank_account_uuid=bank, bank_code=account['bank_code'], bank_name=account['bank_full_name'],
                        account_number=account['account_number'], account_holder=account['account_holder_name'], mode='BANK_ACCOUNT', currency='VND')
        receiver_file.write_text(json.dumps(receiver, ensure_ascii=False), encoding='utf-8')
        settings = {name: str(uuid.uuid4()) for name in ['ORDER_MERCHANT_ID', 'SEPAY_CONNECTION_ID', 'SEPAY_RECEIVER_ID']}
        settings.update({name: secrets.token_hex(32) for name in ['ORDER_INTERNAL_TOKEN', 'ORDER_JWT_SECRET', 'CALLBACK_SECRET']})
        settings.update(ORDER_ENCRYPTION_KEY=base64.b64encode(secrets.token_bytes(32)).decode(),
                        SEPAY_MERCHANT_IDENTITY='sandbox-bank:' + bank, SEPAY_EGRESS_SCOPE='sandbox-local-shared-egress',
                        SEPAY_BOOTSTRAP_AT=(datetime.now(timezone.utc)-timedelta(minutes=1)).isoformat().replace('+00:00', 'Z'))
        settings_file.write_text(json.dumps(settings), encoding='utf-8')
    settings = json.loads(settings_file.read_text(encoding='utf-8'))
    # Refuse half-initialized settings instead of silently rotating an identity.
    if not receiver_file.is_file():
        raise SystemExit('Existing runtime settings require the original receiver file')
    key_file = directory / 'runtime-guest-key.pem'
    if not key_file.exists():
        if initialized:
            raise SystemExit('Original Guest signing key missing; restore it instead of silently rotating it')
        from cryptography.hazmat.primitives.asymmetric import ec
        from cryptography.hazmat.primitives import serialization
        key = ec.generate_private_key(ec.SECP256R1())
        key_file.write_bytes(key.private_bytes(serialization.Encoding.PEM, serialization.PrivateFormat.PKCS8, serialization.NoEncryption()))
    env_file = directory / 'runtime.env'
    if any('\n' in str(value) or '\r' in str(value) for value in settings.values()):
        raise SystemExit('Invalid multiline setting')
    env_file.write_text(''.join(name + '=' + str(value) + '\n' for name, value in settings.items()), encoding='utf-8')
    names = ['api-token.txt', 'webhook-secret.txt', 'runtime-receiver.json', 'runtime-guest-key.pem']
    if (directory / 'runtime-tunnel-token.txt').exists():
        names.append('runtime-tunnel-token.txt')
    archive = io.BytesIO()
    with tarfile.open(fileobj=archive, mode='w') as tar:
        for name in names:
            data = (directory / name).read_bytes()
            info = tarfile.TarInfo(name)
            info.uid = info.gid = 10001
            info.mode = 0o400
            info.size = len(data)
            tar.addfile(info, io.BytesIO(data))
    subprocess.run(['docker', 'volume', 'create', 'pbl-order-connected-secrets'], check=True, stdout=subprocess.DEVNULL)
    subprocess.run(['docker', 'run', '--rm', '-i', '-v', 'pbl-order-connected-secrets:/run/secrets', 'alpine:3.20',
                    'tar', '-xpf', '-', '-C', '/run/secrets'], input=archive.getvalue(), check=True)
    print(json.dumps({'status': 'READY', 'private_env_file': str(env_file), 'secret_files_imported': len(names),
                      'provider_mutations': False, 'identities_preserved': True}))


if __name__ == '__main__':
    main()
