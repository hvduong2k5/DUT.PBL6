"""Real Web OIDC/BFF -> Kong -> Identity/Profile/Order integration.

Docker, Python 3, Node/npm, Go, Java 21 and the Identity Gradle wrapper are
required. All data/credentials belong to disposable fixtures. No bank/Catalog.
Use --keep-running to inspect the same stack in a browser after acceptance.
"""

import argparse
import base64
import hashlib
import html
import http.cookiejar
import json
import os
from pathlib import Path
import re
import secrets
import shutil
import signal
import socket
import subprocess
import time
import urllib.error
import urllib.parse
import urllib.request
import uuid

ROOT = Path(__file__).resolve().parents[1]


def port():
    with socket.socket() as listener:
        listener.bind(("127.0.0.1", 0))
        return listener.getsockname()[1]


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *args, **kwargs):
        return None


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--go", default="go")
    parser.add_argument("--java", default="java")
    parser.add_argument("--gradle")
    parser.add_argument("--keep-running", action="store_true")
    parser.add_argument(
        "--output-dir", default=str(ROOT / ".system_generated/web-identity")
    )
    args = parser.parse_args()
    folder = Path(args.output_dir).resolve()
    folder.mkdir(parents=True, exist_ok=True)
    for transient in ["stop", "ready.json", "result.json", "failure.json"]:
        (folder / transient).unlink(missing_ok=True)
    suffix = uuid.uuid4().hex[:10]
    names, processes, logs = [], [], []
    env = os.environ.copy()
    password = secrets.token_urlsafe(24)
    admin_password = secrets.token_urlsafe(32)
    env.update(POSTGRES_PASSWORD=password, KEYCLOAK_ADMIN_PASSWORD=admin_password)
    ports = {
        key: port()
        for key in [
            "db",
            "redis",
            "keycloak",
            "identity",
            "identity_grpc",
            "profile",
            "profile_grpc",
            "order",
            "gateway",
            "gateway_admin",
            "web",
        ]
    }
    base = f"http://127.0.0.1:{ports['keycloak']}"
    realm = "order-auth-test-" + suffix
    issuer = base + "/realms/" + realm
    web = f"http://localhost:{ports['web']}"
    gateway = f"http://127.0.0.1:{ports['gateway']}"
    account_password = "Synthetic-Web-Test-2026-" + suffix
    accounts = {
        key: {
            "principal": str(uuid.uuid4()),
            "customer": str(uuid.uuid4()),
            "email": f"web-{key.lower()}-{suffix}@example.test",
        }
        for key in ["A", "B"]
    }

    def run(command, cwd=ROOT, **kwargs):
        return subprocess.run(
            [str(x) for x in command], cwd=cwd, env=env, check=True, **kwargs
        )

    def start(command, name, cwd=ROOT):
        log = (folder / (name + ".log")).open("w", encoding="utf-8")
        logs.append(log)
        process = subprocess.Popen(
            [str(x) for x in command],
            cwd=cwd,
            env=env,
            stdout=log,
            stderr=log,
            creationflags=(
                (subprocess.CREATE_NO_WINDOW | subprocess.CREATE_NEW_PROCESS_GROUP)
                if os.name == "nt"
                else 0
            ),
            start_new_session=os.name != "nt",
        )
        processes.append(process)
        return process

    def container(name, image, extra):
        names.append(name)
        run(
            ["docker", "run", "-d", "--name", name, *extra, image],
            stdout=subprocess.DEVNULL,
        )

    def request(
        url, data=None, token=None, method=None, opener=None, headers=None, form=False
    ):
        actual = dict(headers or {})
        if token:
            actual["Authorization"] = "Bearer " + token
        if data is not None:
            actual["Content-Type"] = (
                "application/x-www-form-urlencoded" if form else "application/json"
            )
            data = (urllib.parse.urlencode(data) if form else json.dumps(data)).encode()
        req = urllib.request.Request(url, data=data, headers=actual, method=method)
        try:
            response = (opener.open if opener else urllib.request.urlopen)(
                req, timeout=10
            )
        except urllib.error.HTTPError as error:
            response = error
        with response:
            raw = response.read()
            body = (
                json.loads(raw)
                if raw
                and "application/json" in response.headers.get("Content-Type", "")
                else raw.decode(errors="replace")
            )
            return response.status, body, response.headers

    def wait(url, expected=200, seconds=90):
        deadline = time.monotonic() + seconds
        while time.monotonic() < deadline:
            try:
                if request(url)[0] == expected:
                    return
            except (OSError, ValueError):
                pass
            time.sleep(0.25)
        raise RuntimeError("Fixture not ready: " + urllib.parse.urlparse(url).path)

    db_name = "web_identity"
    db_container = "codex-web-db-" + suffix

    def sql(query):
        return run(
            [
                "docker",
                "exec",
                "-i",
                db_container,
                "psql",
                "-U",
                "peer",
                "-d",
                db_name,
                "-v",
                "ON_ERROR_STOP=1",
            ],
            input=query,
            text=True,
            encoding="utf-8",
            capture_output=True,
        ).stdout

    try:
        container(
            db_container,
            "postgres:16-alpine",
            [
                "-e",
                "POSTGRES_PASSWORD",
                "-e",
                "POSTGRES_USER=peer",
                "-e",
                f"POSTGRES_DB={db_name}",
                "-p",
                f"127.0.0.1:{ports['db']}:5432",
            ],
        )
        for _ in range(120):
            check = subprocess.run(
                [
                    "docker",
                    "exec",
                    db_container,
                    "pg_isready",
                    "-U",
                    "peer",
                    "-d",
                    db_name,
                ],
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
            )
            if check.returncode == 0:
                break
            time.sleep(0.25)
        sql(
            "CREATE SCHEMA identity; CREATE SCHEMA identity_test; CREATE SCHEMA profile;"
        )
        container(
            "codex-web-redis-" + suffix,
            "redis:7-alpine",
            ["-p", f"127.0.0.1:{ports['redis']}:6379"],
        )
        template = json.loads(
            (
                ROOT
                / "services/identity-service/deploy/keycloak/realm-export.template.json"
            ).read_text(encoding="utf-8")
        )
        template.update(id=realm, realm=realm, users=[], sslRequired="none")
        for client in template["clients"]:
            if client["clientId"] == "frontend-web":
                client.update(
                    redirectUris=[web + "/api/auth/oidc/callback"],
                    webOrigins=[web],
                    standardFlowEnabled=True,
                    directAccessGrantsEnabled=False,
                )
                client.setdefault("attributes", {})["post.logout.redirect.uris"] = (
                    web + "/api/auth/oidc/logged-out"
                )
        realm_file = folder / "realm.json"
        realm_file.write_text(json.dumps(template), encoding="utf-8")
        kc_name = "codex-web-keycloak-" + suffix
        names.append(kc_name)
        run(
            [
                "docker",
                "run",
                "-d",
                "--name",
                kc_name,
                "-p",
                f"127.0.0.1:{ports['keycloak']}:8080",
                "-e",
                "KEYCLOAK_ADMIN=admin",
                "-e",
                "KEYCLOAK_ADMIN_PASSWORD",
                "-v",
                str(realm_file) + ":/opt/keycloak/data/import/realm.json:ro",
                "quay.io/keycloak/keycloak:24.0.5",
                "start-dev",
                "--import-realm",
            ],
            stdout=subprocess.DEVNULL,
        )
        wait(issuer + "/.well-known/openid-configuration", seconds=150)
        status, admin, _ = request(
            base + "/realms/master/protocol/openid-connect/token",
            {
                "grant_type": "password",
                "client_id": "admin-cli",
                "username": "admin",
                "password": admin_password,
            },
            form=True,
        )
        assert status == 200, "Fixture administrator unavailable"
        admin_token = admin["access_token"]
        status, profile, _ = request(
            base + "/admin/realms/" + realm + "/users/profile", token=admin_token
        )
        assert status == 200
        profile["attributes"].append(
            json.loads(
                (
                    ROOT
                    / "services/identity-service/deploy/keycloak/order-user-profile-attribute.json"
                ).read_text()
            )
        )
        assert (
            request(
                base + "/admin/realms/" + realm + "/users/profile",
                profile,
                admin_token,
                "PUT",
            )[0]
            == 200
        )
        status, roles, _ = request(
            base + "/admin/realms/" + realm + "/roles/CUSTOMER", token=admin_token
        )
        assert status == 200
        for account in accounts.values():
            status, _, headers = request(
                base + "/admin/realms/" + realm + "/users",
                {
                    "username": account["email"],
                    "email": account["email"],
                    "firstName": "Synthetic",
                    "lastName": "Customer",
                    "enabled": True,
                    "emailVerified": True,
                    "attributes": {"app_user_id": [account["principal"]]},
                    "credentials": [
                        {
                            "type": "password",
                            "value": account_password,
                            "temporary": False,
                        }
                    ],
                },
                admin_token,
                "POST",
            )
            assert status == 201
            account["subject"] = headers["Location"].rsplit("/", 1)[1]
            assert (
                request(
                    base
                    + "/admin/realms/"
                    + realm
                    + "/users/"
                    + account["subject"]
                    + "/role-mappings/realm",
                    [roles],
                    admin_token,
                    "POST",
                )[0]
                == 204
            )

        env.update(
            DB_HOST="127.0.0.1",
            DB_PORT=str(ports["db"]),
            DB_NAME=db_name,
            DB_USERNAME="peer",
            DB_PASSWORD=password,
            SPRING_DATASOURCE_URL=f"jdbc:postgresql://127.0.0.1:{ports['db']}/{db_name}?currentSchema=identity_test",
            GRPC_SERVER_PORT=str(ports["identity_grpc"]),
            SPRING_SECURITY_OAUTH2_RESOURCESERVER_JWT_ISSUER_URI=issuer,
            SPRING_SECURITY_OAUTH2_RESOURCESERVER_JWT_JWK_SET_URI=issuer
            + "/protocol/openid-connect/certs",
        )
        gradle = (
            args.gradle
            or shutil.which("gradle")
            or str(
                ROOT
                / "services/identity-service"
                / ("gradlew.bat" if os.name == "nt" else "gradlew")
            )
        )
        with (folder / "identity-build.log").open("w") as log:
            run(
                (
                    ["sh", gradle]
                    if os.name != "nt" and gradle.endswith("gradlew")
                    else [gradle]
                )
                + ["--no-daemon", "test", "bootJar"],
                ROOT / "services/identity-service",
                stdout=log,
                stderr=log,
            )
        env.update(
            SPRING_DATASOURCE_URL=f"jdbc:postgresql://127.0.0.1:{ports['db']}/{db_name}?currentSchema=identity",
            SERVER_PORT=str(ports["identity"]),
            KEYCLOAK_SERVER_URL=base,
            KEYCLOAK_REALM=realm,
            KEYCLOAK_CLIENT_SECRET="unused-test-client",
            SPRING_KAFKA_BOOTSTRAP_SERVERS="127.0.0.1:1",
        )
        jar = next(
            p
            for p in (ROOT / "services/identity-service/build/libs").glob("*.jar")
            if "plain" not in p.name
        )
        start([args.java, "-jar", jar], "identity")
        identity_url = f"http://127.0.0.1:{ports['identity']}"
        wait(identity_url + "/v3/api-docs")
        for account in accounts.values():
            sql(
                f"INSERT INTO identity.identity_principals(principal_id,idp_issuer,idp_subject,principal_type,provisioning_status) VALUES('{account['principal']}','{issuer}','{account['subject']}','CUSTOMER','ACTIVE');"
            )

        sql(
            "SET search_path TO profile,public;"
            + (
                ROOT / "services/profile-service/migrations/000001_init_schema.up.sql"
            ).read_text(encoding="utf-8")
        )
        sql(
            "SET search_path TO profile,public;"
            + (
                ROOT
                / "services/profile-service/migrations/000003_profile_integrity.up.sql"
            ).read_text(encoding="utf-8")
        )
        for index, account in enumerate(accounts.values()):
            sql(
                f"INSERT INTO profile.customer_profiles(id,user_id,full_name,phone_number,email,status) VALUES('{account['customer']}','{account['principal']}','Synthetic Customer {index}','090000000{index}','{account['email']}','ACTIVE');"
            )
        internal_token = secrets.token_urlsafe(32)
        env.update(
            DATABASE_URL=f"postgres://peer:{password}@127.0.0.1:{ports['db']}/{db_name}?sslmode=disable&search_path=profile",
            REDIS_ADDR=f"127.0.0.1:{ports['redis']}",
            HTTP_PORT=str(ports["profile"]),
            GRPC_PORT=str(ports["profile_grpc"]),
            PROFILE_INTERNAL_TOKEN=internal_token,
            PROFILE_AUTH_MODE="keycloak",
            PROFILE_OIDC_ISSUER=issuer,
            PROFILE_OIDC_AUDIENCE="omama-api",
            PROFILE_OIDC_ALLOW_LOCAL_HTTP="true",
            USE_VAULT_MOCK_DEV="true",
            KAFKA_BROKERS="127.0.0.1:1",
        )
        extension = ".exe" if os.name == "nt" else ""
        profile_binary = folder / ("profile" + extension)
        run(
            [args.go, "build", "-o", profile_binary, "./cmd/server"],
            ROOT / "services/profile-service",
        )
        start([profile_binary], "profile")
        profile_url = f"http://127.0.0.1:{ports['profile']}"
        wait(profile_url + "/livez")
        env.update(
            WEB_TEST_ISSUER=issuer,
            WEB_TEST_PROFILE_URL=profile_url,
            WEB_TEST_INTERNAL_TOKEN=internal_token,
            WEB_TEST_REDIS=env["REDIS_ADDR"],
            WEB_TEST_ORDER_PORT=str(ports["order"]),
        )
        for key, account in accounts.items():
            env["WEB_TEST_PRINCIPAL_" + key] = account["principal"]
        order_binary = folder / ("order" + extension)
        run(
            [args.go, "build", "-o", order_binary, "./tests/webidentityfixture"],
            ROOT / "services/order-service",
        )
        start([order_binary], "order")
        wait(f"http://127.0.0.1:{ports['order']}/api/v1/cart", 401)

        # Change only upstream host/port in a test copy, preserving real routes/plugins.
        kong = (ROOT / "services/api-gateway/config/kong.yml").read_text(
            encoding="utf-8"
        )
        for old, key in [
            ("http://identity-service:8016", "identity"),
            ("http://omamx-profile-service:8080", "profile"),
            ("http://order-service:8004", "order"),
        ]:
            kong = kong.replace(old, f"http://host.docker.internal:{ports[key]}")
        kong_file = folder / "kong.yml"
        kong_file.write_text(kong, encoding="utf-8")
        container(
            "codex-web-gateway-" + suffix,
            "kong:3.9",
            [
                "--add-host",
                "host.docker.internal:host-gateway",
                "-p",
                f"127.0.0.1:{ports['gateway']}:8000",
                "-p",
                f"127.0.0.1:{ports['gateway_admin']}:8001",
                "-e",
                "KONG_DATABASE=off",
                "-e",
                "KONG_DECLARATIVE_CONFIG=/etc/kong/kong.yml",
                "-e",
                "KONG_ADMIN_LISTEN=0.0.0.0:8001",
                "-e",
                "KONG_LOG_LEVEL=warn",
                "-v",
                str(kong_file) + ":/etc/kong/kong.yml:ro",
            ],
        )
        wait(f"http://127.0.0.1:{ports['gateway_admin']}/status")
        env.update(
            WEB_AUTH_MODE="oidc",
            WEB_PUBLIC_URL=web,
            WEB_OIDC_ISSUER=issuer,
            WEB_OIDC_CLIENT_ID="frontend-web",
            WEB_OIDC_API_AUDIENCE="omama-api",
            WEB_OIDC_COOKIE_SECRET=secrets.token_urlsafe(48),
            WEB_OIDC_ALLOW_LOCAL_HTTP="true",
            CUSTOMER_CORE_UPSTREAM_URL=gateway + "/api/v1",
            CUSTOMER_EXTENSIONS_UPSTREAM_URL=gateway + "/api/v1",
        )
        start(
            [
                shutil.which("node"),
                ROOT / "apps/web-user/node_modules/next/dist/bin/next",
                "dev",
                "--hostname",
                "0.0.0.0",
                "--port",
                str(ports["web"]),
            ],
            "web",
            ROOT / "apps/web-user",
        )
        wait(web + "/login")
        ready = {
            "web": web,
            "gateway": gateway,
            "identity": identity_url,
            "profile": profile_url,
            "issuer": issuer,
            "accounts": accounts,
            "password": account_password,
        }
        (folder / "ready.json").write_text(
            json.dumps(ready, indent=2), encoding="utf-8"
        )

        for key, account in accounts.items():
            jar = http.cookiejar.CookieJar()
            opener = urllib.request.build_opener(
                urllib.request.HTTPCookieProcessor(jar), NoRedirect()
            )
            status, _, headers = request(
                web + "/api/auth/oidc/login?returnUrl=%2Fcart", opener=opener
            )
            assert status in [302, 307], "Web did not start OIDC"
            authorize = headers["Location"]
            query = urllib.parse.parse_qs(urllib.parse.urlparse(authorize).query)
            assert query["code_challenge_method"] == ["S256"]
            status, page, authorize_headers = request(authorize, opener=opener)
            for _ in range(4):
                if status not in [302, 303, 307]:
                    break
                target = urllib.parse.urljoin(authorize, authorize_headers["Location"])
                assert (
                    urllib.parse.urlparse(target).netloc
                    == urllib.parse.urlparse(base).netloc
                ), "Unexpected login redirect origin; OAuth error=" + str(
                    urllib.parse.parse_qs(urllib.parse.urlparse(target).query).get(
                        "error"
                    )
                )
                status, page, authorize_headers = request(target, opener=opener)
            if status != 200:
                (folder / "authorization-error.html").write_text(
                    str(page), encoding="utf-8"
                )
                raise AssertionError(
                    f"Keycloak authorization page HTTP {status}; see private authorization-error.html"
                )
            form = re.search(r'<form\b[^>]*id="kc-form-login"[^>]*>', page)
            assert form, "No real Keycloak login form"
            action = html.unescape(re.search(r'action="([^"]+)"', form[0])[1])
            status, _, headers = request(
                action,
                {"username": account["email"], "password": account_password},
                form=True,
                opener=opener,
            )
            assert status in [302, 303], "Keycloak did not issue authorization code"
            status, _, headers = request(headers["Location"], opener=opener)
            assert (
                status in [302, 307] and headers["Location"] == web + "/cart"
            ), "OIDC callback failed"
            assert all(
                cookie.has_nonstandard_attr("HttpOnly")
                for cookie in jar
                if cookie.name.startswith("oma_")
            ), "Token cookie is script readable"
            status, session, _ = request(web + "/api/auth/me", opener=opener)
            assert (
                status == 200
                and session["customer"]["id"] == account["customer"]
                and session["principal"]["principalId"] == account["principal"]
            ), "Session did not bind Identity to Profile"
            assert not any(
                field in json.dumps(session)
                for field in ["access_token", "refresh_token", "id_token"]
            ), "Session exposes tokens"
            status, caps, _ = request(web + "/api/capabilities", opener=opener)
            assert status == 200 and "CART_MANAGE" in caps["capabilities"]
            status, cart, _ = request(
                web + "/api/cart",
                opener=opener,
                headers={
                    "X-User-ID": accounts["B" if key == "A" else "A"]["principal"],
                    "X-User-Role": "ADMIN",
                },
            )
            assert (
                status == 200 and cart["items"][0]["sku_code"] == "PRIVATE-" + key
            ), "Cross-owner Cart/header spoofing"
            status, refresh, _ = request(
                web + "/api/auth/refresh", {}, opener=opener, headers={"Origin": web}
            )
            assert status == 200 and refresh["refreshed"] is True, "Refresh failed"
            status, profile, headers = request(
                web + "/api/customer/profile", opener=opener
            )
            assert (
                status == 200 and profile["id"] == account["customer"]
            ), "Profile route failed"
            access = next(c.value for c in jar if c.name == "oma_access_token")
            status, owner_profile, _ = request(
                gateway + "/api/v1/profile",
                token=access,
                headers={
                    "X-User-ID": accounts["B" if key == "A" else "A"]["principal"],
                    "X-Internal-Token": internal_token,
                    "X-User-Role": "ADMIN",
                },
            )
            assert (
                status == 200 and owner_profile["id"] == account["customer"]
            ), "Gateway/Profile accepted a forged owner header"
            sql(
                f"UPDATE profile.customer_profiles SET status='SUSPENDED' WHERE id='{account['customer']}';"
            )
            assert (
                request(web + "/api/cart", opener=opener)[0] == 403
            ), "Inactive Customer retained Cart capability"
            sql(
                f"UPDATE profile.customer_profiles SET status='ACTIVE' WHERE id='{account['customer']}';"
            )
            sql(
                f"UPDATE identity.identity_principals SET provisioning_status='SUSPENDED' WHERE principal_id='{account['principal']}';"
            )
            assert (
                request(web + "/api/auth/me", opener=opener)[0] == 403
            ), "Suspended Identity principal retained Web session"
            sql(
                f"UPDATE identity.identity_principals SET provisioning_status='ACTIVE' WHERE principal_id='{account['principal']}';"
            )
            assert request(web + "/api/cart", opener=opener)[0] == 200
            if key == "A":
                sql(
                    f"UPDATE profile.customer_profiles SET user_id='{uuid.uuid4()}' WHERE id='{account['customer']}';"
                )
                status, unmapped, _ = request(web + "/api/auth/me", opener=opener)
                assert status == 200 and unmapped["customerAccess"] is False
                assert (
                    request(web + "/api/cart", opener=opener)[0] == 403
                ), "Missing Customer mapping granted Cart access"
                sql(
                    f"UPDATE profile.customer_profiles SET user_id='{account['principal']}' WHERE id='{account['customer']}';"
                )
            status, _, _ = request(
                web + "/api/auth/refresh",
                {},
                opener=opener,
                headers={"Origin": "https://attacker.invalid"},
            )
            assert status == 403, "Cross-origin mutation accepted"
            print(
                "PASS: real OIDC login/session, verified Customer, capabilities, Profile, isolated Cart and refresh for account "
                + key,
                flush=True,
            )

            status, logout, _ = request(
                web + "/api/auth/logout", {}, opener=opener, headers={"Origin": web}
            )
            assert status == 200 and logout.get(
                "logoutUrl"
            ), "Logout did not clear session/start provider logout"
            for _ in range(4):
                status, page, headers = request(logout["logoutUrl"], opener=opener)
                if status not in [302, 303, 307]:
                    break
                logout["logoutUrl"] = headers["Location"]
            assert (
                request(web + "/api/auth/me", opener=opener)[0] == 401
            ), "Logout retained local session"

        assert request(web + "/api/cart")[0] == 401
        assert (
            request(
                gateway + "/api/v1/profile",
                headers={
                    "X-User-ID": accounts["A"]["principal"],
                    "X-User-Role": "ADMIN",
                    "X-Internal-Token": internal_token,
                },
            )[0]
            == 401
        ), "Public Gateway forwarded internal identity credential"
        assert request(web + "/api/auth/oidc/callback?state=wrong&code=wrong")[0] == 401
        result = {
            "status": "PASS",
            "head": subprocess.check_output(
                ["git", "rev-parse", "HEAD"], cwd=ROOT, text=True
            ).strip(),
            "scope": "real Web OIDC -> Keycloak -> Kong -> Identity/PostgreSQL -> Profile/PostgreSQL -> Order handler/Redis; seeded cart, no Catalog/checkout/payment",
            "accounts": 2,
        }
        (folder / "result.json").write_text(json.dumps(result, indent=2))
        print(
            "PASS: no browser identity-header bypass; no token JSON exposure; missing session and invalid callback rejected",
            flush=True,
        )
        if args.keep_running:
            print(
                "READY: fixture retained for browser check; configuration in ignored ready.json",
                flush=True,
            )
            while not (folder / "stop").exists():
                time.sleep(0.5)
    except Exception as error:
        (folder / "failure.json").write_text(
            json.dumps({"type": type(error).__name__, "message": str(error)}),
            encoding="utf-8",
        )
        if args.keep_running:
            print(
                "FAIL: "
                + type(error).__name__
                + "; fixture retained for diagnostics until stop file exists",
                flush=True,
            )
            while not (folder / "stop").exists():
                time.sleep(0.5)
        raise
    finally:
        for process in reversed(processes):
            if process.poll() is None:
                if os.name == "nt":
                    subprocess.run(
                        ["taskkill", "/PID", str(process.pid), "/T", "/F"],
                        stdout=subprocess.DEVNULL,
                        stderr=subprocess.DEVNULL,
                    )
                else:
                    try:
                        os.killpg(process.pid, signal.SIGTERM)
                    except ProcessLookupError:
                        pass
                try:
                    process.wait(timeout=15)
                except subprocess.TimeoutExpired:
                    if os.name != "nt":
                        os.killpg(process.pid, signal.SIGKILL)
                    else:
                        process.kill()
                    process.wait(timeout=10)
        for log in logs:
            log.close()
        for name in reversed(names):
            subprocess.run(
                ["docker", "rm", "-f", name],
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
            )


if __name__ == "__main__":
    main()
