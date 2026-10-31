"""Disposable live Keycloak provisioning/PKCE and Order HTTP ownership test."""

import argparse
import atexit
import base64
import hashlib
import html
import http.cookiejar
import json
import os
import pathlib
import re
import secrets
import subprocess
import time
import urllib.error
import urllib.parse
import urllib.request
import uuid

parser = argparse.ArgumentParser(
    description="Isolated Keycloak PKCE -> Order HTTP live test"
)
parser.add_argument("--identity-repo", required=True, type=pathlib.Path)
parser.add_argument("--go", default="go")
parser.add_argument("--gradle", default="gradle")
args = parser.parse_args()
root = pathlib.Path(__file__).resolve().parents[3]
identity = args.identity_repo.resolve()
env = os.environ.copy()
admin = secrets.token_urlsafe(30)
password = secrets.token_urlsafe(25)
base = "http://127.0.0.1:18085"
realm = "order-auth-test-" + uuid.uuid4().hex[:8]
env.update(
    KEYCLOAK_ADMIN_PASSWORD=admin,
    KEYCLOAK_TEST_ADMIN_PASSWORD=admin,
    KEYCLOAK_TEST_CUSTOMER_PASSWORD=password,
    KEYCLOAK_TEST_URL=base,
    KEYCLOAK_TEST_REALM=realm,
    KEYCLOAK_TEST_ISSUER=base + "/realms/" + realm,
    KEYCLOAK_TEST_REDIS="127.0.0.1:16389",
)


def run(args, cwd=None):
    subprocess.run(args, cwd=cwd, env=env, check=True)


containers = []


def cleanup():
    for name in reversed(containers):
        subprocess.run(
            ["docker", "rm", "-f", name],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )


atexit.register(cleanup)
for name, image, port, extra in [
    (
        "codex-auth-kc-" + realm,
        "quay.io/keycloak/keycloak:24.0.5",
        "127.0.0.1:18085:8080",
        ["-e", "KEYCLOAK_ADMIN=admin", "-e", "KEYCLOAK_ADMIN_PASSWORD"],
    ),
    ("codex-auth-redis-" + realm, "redis:7-alpine", "127.0.0.1:16389:6379", []),
]:
    command = ["docker", "run", "-d", "--name", name, "-p", port] + extra + [image]
    if "keycloak" in image:
        command.append("start-dev")
    run(command)
    containers.append(name)


def call(path, data=None, token=None, method=None, form=False):
    headers = {"Accept": "application/json"}
    if token:
        headers["Authorization"] = "Bearer " + token
    if data is not None:
        headers["Content-Type"] = (
            "application/x-www-form-urlencoded" if form else "application/json"
        )
        data = (urllib.parse.urlencode(data) if form else json.dumps(data)).encode()
    req = urllib.request.Request(base + path, data=data, headers=headers, method=method)
    with urllib.request.urlopen(req, timeout=10) as r:
        b = r.read()
        return json.loads(b) if b else None


for _ in range(90):
    try:
        call("/realms/master")
        break
    except Exception:
        time.sleep(1)
else:
    raise RuntimeError("Keycloak not ready")


def admin_token():
    return call(
        "/realms/master/protocol/openid-connect/token",
        {
            "client_id": "admin-cli",
            "username": "admin",
            "password": admin,
            "grant_type": "password",
        },
        form=True,
    )["access_token"]


at = admin_token()
d = json.loads(
    (
        identity
        / "services/identity-service/deploy/keycloak/realm-export.template.json"
    ).read_text()
)
d["id"] = realm
d["realm"] = realm
d["users"] = []
d["sslRequired"] = "none"
for client in d["clients"]:
    if client["clientId"] == "frontend-web":
        client["redirectUris"] = ["http://127.0.0.1:18799/callback"]
        client["webOrigins"] = ["http://127.0.0.1:18799"]
        client["directAccessGrantsEnabled"] = False
call("/admin/realms", d, at)
profile = call("/admin/realms/" + realm + "/users/profile", token=at)
profile["attributes"].append(
    json.loads(
        (
            identity
            / "services/identity-service/deploy/keycloak/order-user-profile-attribute.json"
        ).read_text()
    )
)
call("/admin/realms/" + realm + "/users/profile", profile, at, "PUT")
for s in "AB":
    env["KEYCLOAK_TEST_PRINCIPAL_" + s] = str(uuid.uuid4())
    env["KEYCLOAK_TEST_EMAIL_" + s] = "codex-" + s.lower() + "@example.test"
print("Real Keycloak ready; provisioning through Identity Java adapter", flush=True)
run(
    [args.gradle, "--no-daemon", "test", "--tests", "*KeycloakLiveProvisioningTest"],
    identity / "services/identity-service",
)
# Admin access tokens are short-lived; compilation may outlast them.
at = admin_token()


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *a, **k):
        return None


def login(email):
    verifier = secrets.token_urlsafe(48)
    challenge = (
        base64.urlsafe_b64encode(hashlib.sha256(verifier.encode()).digest())
        .decode()
        .rstrip("=")
    )
    state = secrets.token_hex(10)
    params = {
        "client_id": "frontend-web",
        "redirect_uri": "http://127.0.0.1:18799/callback",
        "response_type": "code",
        "scope": "openid",
        "code_challenge_method": "S256",
        "code_challenge": challenge,
        "state": state,
    }
    opener = urllib.request.build_opener(
        urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()), NoRedirect()
    )
    with opener.open(
        base
        + "/realms/"
        + realm
        + "/protocol/openid-connect/auth?"
        + urllib.parse.urlencode(params)
    ) as r:
        page = r.read().decode()
    form = re.search(r'<form\b[^>]*id="kc-form-login"[^>]*>', page)
    assert form, "No login form"
    action = html.unescape(re.search(r'action="([^"]+)"', form[0])[1])
    try:
        with opener.open(
            urllib.request.Request(
                action,
                data=urllib.parse.urlencode(
                    {"username": email, "password": password}
                ).encode(),
            )
        ) as r:
            page = r.read().decode()
            raise RuntimeError(
                "Login did not return authorization code; required action or invalid credentials"
            )
    except urllib.error.HTTPError as e:
        if e.code not in (302, 303):
            raise
        loc = e.headers["Location"]
        e.close()
    query = urllib.parse.parse_qs(urllib.parse.urlparse(loc).query)
    assert query.get("state") == [state], "Wrong redirect state"
    return call(
        "/realms/" + realm + "/protocol/openid-connect/token",
        {
            "grant_type": "authorization_code",
            "client_id": "frontend-web",
            "code": query["code"][0],
            "redirect_uri": "http://127.0.0.1:18799/callback",
            "code_verifier": verifier,
        },
        form=True,
    )["access_token"]


for s in "AB":
    env["ORDER_TEST_TOKEN_" + s] = login(env["KEYCLOAK_TEST_EMAIL_" + s])
print(
    "PASS: real Authorization Code + PKCE tokens issued for A/B (password grant disabled)",
    flush=True,
)
# Give only this isolated fixture a normal Account API audience/role mapper.
# The original Order tokens above remain unchanged. A successful ordinary profile
# update is required first, so a generic 401/403 cannot masquerade as attribute protection.
client = call("/admin/realms/" + realm + "/clients?clientId=frontend-web", token=at)[0]
for mapper in [
    {
        "name": "test-account-audience",
        "protocolMapper": "oidc-audience-mapper",
        "config": {
            "included.custom.audience": "account",
            "access.token.claim": "true",
            "id.token.claim": "false",
        },
    },
    {
        "name": "test-account-roles",
        "protocolMapper": "oidc-usermodel-client-role-mapper",
        "config": {
            "usermodel.clientRoleMapping.clientId": "account",
            "claim.name": "resource_access.account.roles",
            "jsonType.label": "String",
            "multivalued": "true",
            "access.token.claim": "true",
            "id.token.claim": "false",
        },
    },
]:
    mapper["protocol"] = "openid-connect"
    call(
        "/admin/realms/"
        + realm
        + "/clients/"
        + client["id"]
        + "/protocol-mappers/models",
        mapper,
        at,
    )
account_token = login(env["KEYCLOAK_TEST_EMAIL_A"])
call("/realms/" + realm + "/account", token=account_token)
ordinary_profile = {
    "username": env["KEYCLOAK_TEST_EMAIL_A"],
    "firstName": "Integration",
    "lastName": "A",
    "email": env["KEYCLOAK_TEST_EMAIL_A"],
}
call("/realms/" + realm + "/account", ordinary_profile, account_token, "POST")
print("PASS: customer can use Account API for ordinary profile updates", flush=True)
try:
    call(
        "/realms/" + realm + "/account",
        {
            **ordinary_profile,
            "attributes": {"app_user_id": [env["KEYCLOAK_TEST_PRINCIPAL_B"]]},
        },
        account_token,
        "POST",
    )
except urllib.error.HTTPError as e:
    assert e.code in (400, 403), "Unexpected Account API status"
users = call(
    "/admin/realms/"
    + realm
    + "/users?username="
    + urllib.parse.quote(env["KEYCLOAK_TEST_EMAIL_A"]),
    token=at,
)
assert len(users) == 1 and users[0]["attributes"]["app_user_id"] == [
    env["KEYCLOAK_TEST_PRINCIPAL_A"]
], "Customer changed server-owned principal"
print("PASS: customer cannot overwrite app_user_id through Account API", flush=True)

clients = call("/admin/realms/" + realm + "/clients?clientId=frontend-web", token=at)
scopes = call("/admin/realms/" + realm + "/client-scopes", token=at)
scope = next(s for s in scopes if s["name"] == "order-customer")
call(
    "/admin/realms/"
    + realm
    + "/clients/"
    + clients[0]["id"]
    + "/default-client-scopes/"
    + scope["id"],
    token=at,
    method="DELETE",
)
env["KEYCLOAK_TEST_NO_ORDER_AUDIENCE_TOKEN"] = login(env["KEYCLOAK_TEST_EMAIL_A"])
call(
    "/admin/realms/"
    + realm
    + "/clients/"
    + clients[0]["id"]
    + "/default-client-scopes/"
    + scope["id"],
    token=at,
    method="PUT",
)
run(
    [
        args.go,
        "test",
        "-count=1",
        "-v",
        "./internal/transport",
        "-run",
        "^TestKeycloakLiveCart$",
    ],
    root / "services/order-service",
)
print(
    "PASS: Identity Java adapter -> real Keycloak PKCE -> Order HTTP -> real Redis",
    flush=True,
)
print(
    "Scope: real Identity provisioning adapter + Keycloak + Order HTTP + Redis; registration DB/Profile/checkout NOT_RUN",
    flush=True,
)
