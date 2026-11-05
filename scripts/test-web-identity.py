"""Real Web OIDC/BFF -> Kong -> Identity/Profile/Order integration.

Docker, Python 3, Node/npm, Go, Java 21 and the Identity Gradle wrapper are
required. All data/credentials belong to disposable fixtures. Real Catalog and
Shipping quotes; payment statements and packing TEST, external carrier FAKE.
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
import sys

sys.dont_write_bytecode = True
from mvp_native_checks import native_identity
from mvp_customer_checks import profile_addresses
from mvp_customer_checks import checkout_shipping
import mvp_fulfillment_checks as fulfillment_checks

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
    parser.add_argument("--flutter", help="Run the native Dart repositories against the same live stack")
    parser.add_argument("--keep-running", action="store_true")
    parser.add_argument("--fulfillment", action="store_true", help="Use actual Fulfillment/media providers instead of TEST packing")
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
    env["MVP_GO"] = args.go
    fulfillment_network = None
    # The disposable fixture must never inherit a developer's external payment
    # or carrier configuration. Keep executable/toolchain environment intact.
    for key in list(env):
        if key.startswith(("SEPAY_", "SHIPPING_", "ORDER_SHIPPING_")):
            env.pop(key)
    password = secrets.token_urlsafe(24)
    admin_password = secrets.token_urlsafe(32)
    env.update(POSTGRES_PASSWORD=password, KEYCLOAK_ADMIN_PASSWORD=admin_password)
    ports = {
        key: port()
        for key in [
            "db",
            "redis",
            "keycloak",
            "catalog",
            "catalog_grpc",
            "identity",
            "identity_grpc",
            "profile",
            "profile_grpc",
            "order",
            "gateway",
            "gateway_admin",
            "web",
            "kafka",
            "shipping",
            "shipping_grpc",
            "control",
            "order_grpc", "fulfillment", "fulfillment_grpc", "storage", "scanner", "kafka_docker",
        ]
    }
    base = f"http://127.0.0.1:{ports['keycloak']}"
    realm = "order-auth-test-" + suffix
    issuer = base + "/realms/" + realm
    web = f"http://127.0.0.1:{ports['web']}"
    gateway = f"http://127.0.0.1:{ports['gateway']}"
    account_password = "Synthetic-Web-Test-2026-" + suffix
    accounts = {
        key: {
            "principal": str(uuid.uuid4()),
            "customer": str(uuid.uuid4()),
            "email": f"web-{key.lower()}-{suffix}@example.test",
        }
        for key in ["A", "B", "E"]
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

    last_gateway_request = 0.0

    def request(
        url, data=None, token=None, method=None, opener=None, headers=None, form=False
    ):
        nonlocal last_gateway_request
        # This is a functional acceptance test, not a load test. One BFF call
        # fans out to several Gateway requests, all sharing the fixture IP and
        # its real 300/minute budget. Leave headroom for async work and polling.
        if url.startswith((web + "/", gateway + "/")):
            time.sleep(max(0, 2.0 - (time.monotonic() - last_gateway_request)))
            last_gateway_request = time.monotonic()
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

    def sql(query, tuples=False):
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
                *(["-t", "-A"] if tuples else []),
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
                f"{'0.0.0.0' if args.fulfillment else '127.0.0.1'}:{ports['db']}:5432",
            ],
        )
        for _ in range(120):
            check = subprocess.run(
                [
                    "docker",
                    "exec",
                    db_container,
                    "pg_isready",
                    "-h",
                    "127.0.0.1",
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
        else:
            raise RuntimeError("PostgreSQL TCP listener did not become ready")
        sql(
            "CREATE SCHEMA identity; CREATE SCHEMA identity_test; CREATE SCHEMA profile; CREATE SCHEMA shipping; CREATE SCHEMA orders; CREATE SCHEMA catalog_test; CREATE SCHEMA catalog_live;"
        )
        container(
            "codex-web-redis-" + suffix,
            "redis:7-alpine",
            ["-p", f"127.0.0.1:{ports['redis']}:6379"],
        )
        kafka_name = "codex-web-kafka-" + suffix
        container(
            kafka_name,
            "apache/kafka@sha256:77e3df9054047a88b520d0cc46e16696d3b22022e1d580aeccd2632df6532837",
            [
                "-p",
                f"127.0.0.1:{ports['kafka']}:9094",
                "-p", f"{'0.0.0.0' if args.fulfillment else '127.0.0.1'}:{ports['kafka_docker']}:9095",
                "-e",
                "KAFKA_NODE_ID=1",
                "-e",
                "KAFKA_PROCESS_ROLES=broker,controller",
                "-e",
                "KAFKA_LISTENERS=PLAINTEXT://:9092,CONTROLLER://:9093,EXTERNAL://:9094,DOCKER://:9095",
                "-e",
                f"KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://localhost:9092,EXTERNAL://127.0.0.1:{ports['kafka']},DOCKER://host.docker.internal:{ports['kafka_docker']}",
                "-e",
                "KAFKA_LISTENER_SECURITY_PROTOCOL_MAP=CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT,EXTERNAL:PLAINTEXT,DOCKER:PLAINTEXT",
                "-e",
                "KAFKA_CONTROLLER_LISTENER_NAMES=CONTROLLER",
                "-e",
                "KAFKA_CONTROLLER_QUORUM_VOTERS=1@localhost:9093",
                "-e",
                "KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR=1",
                "-e",
                "KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR=1",
                "-e",
                "KAFKA_TRANSACTION_STATE_LOG_MIN_ISR=1",
            ],
        )
        for _ in range(60):
            ready = subprocess.run(
                [
                    "docker",
                    "exec",
                    kafka_name,
                    "/opt/kafka/bin/kafka-topics.sh",
                    "--bootstrap-server",
                    "localhost:9092",
                    "--list",
                ],
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
            )
            if ready.returncode == 0:
                break
            time.sleep(1)
        else:
            raise RuntimeError("Kafka not ready")
        # CLI needs a listener reachable both from native services and docker exec.
        for topic in [
            "identity.events.v1",
            "profile.events.v1",
            "shipping.events.v2",
            "shipping.events.v1",
            "order.events.v3",
            "order.sources.v1",
            "fulfillment.task.events.v1", "fulfillment.events.v2",
        ]:
            run(
                [
                    "docker",
                    "exec",
                    kafka_name,
                    "/opt/kafka/bin/kafka-topics.sh",
                    "--bootstrap-server",
                    "localhost:9092",
                    "--create",
                    "--if-not-exists",
                    "--topic",
                    topic,
                    "--partitions",
                    "1",
                    "--replication-factor",
                    "1",
                ],
                stdout=subprocess.DEVNULL,
            )
        template = json.loads(
            (
                ROOT
                / "services/identity-service/deploy/keycloak/realm-export.template.json"
            ).read_text(encoding="utf-8")
        )
        template.update(
            id=realm, realm=realm, users=[], sslRequired="none", verifyEmail=False,
            accessTokenLifespan=1200,
        )
        service_secret = secrets.token_urlsafe(32)
        for client in template["clients"]:
            if client["clientId"] == "identity-service-admin":
                client["secret"] = service_secret
            if client["clientId"] == "frontend-web":
                client.update(
                    redirectUris=[gateway + "/api/v1/public/auth/callback"],
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
                "-v",
                str(ROOT / "services/identity-service/deploy/keycloak/themes/omama")
                + ":/opt/keycloak/themes/omama:ro",
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
        admin_api = base + "/admin/realms/" + realm
        clients = request(
            admin_api + "/clients?clientId=identity-service-admin", token=admin_token
        )[1]
        service_account = request(
            admin_api + "/clients/" + clients[0]["id"] + "/service-account-user",
            token=admin_token,
        )[1]
        management = request(
            admin_api + "/clients?clientId=realm-management", token=admin_token
        )[1][0]
        admin_role = request(
            admin_api + "/clients/" + management["id"] + "/roles/realm-admin",
            token=admin_token,
        )[1]
        assert (
            request(
                admin_api
                + "/users/"
                + service_account["id"]
                + "/role-mappings/clients/"
                + management["id"],
                [admin_role],
                admin_token,
                "POST",
            )[0]
            == 204
        )
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
        # Boot the real Catalog and verify its migration/HTTP wiring before Order.
        catalog_url = f"http://127.0.0.1:{ports['catalog']}"
        env.update(SPRING_DATASOURCE_URL=f"jdbc:postgresql://127.0.0.1:{ports['db']}/{db_name}?currentSchema=catalog_live",
                   SERVER_PORT=str(ports['catalog']), GRPC_SERVER_PORT=str(ports['catalog_grpc']),
                   SPRING_KAFKA_LISTENER_AUTO_STARTUP="false", CATALOG_OUTBOX_ENABLED="false",
                   CATALOG_INTERNAL_TOKEN=env.get("WEB_TEST_INTERNAL_TOKEN", ""))
        catalog_token = secrets.token_urlsafe(32)
        env["CATALOG_INTERNAL_TOKEN"] = catalog_token
        with (folder / "catalog-build.log").open("w") as log:
            run((["sh", gradle] if os.name != "nt" and gradle.endswith("gradlew") else [gradle]) + ["--no-daemon", "test", "bootJar"], ROOT / "services/catalog-service", stdout=log, stderr=log)
        catalog_jar = next(p for p in (ROOT / "services/catalog-service/build/libs").glob("*.jar") if "plain" not in p.name)
        catalog_environment=env.copy()
        catalog_process=start([args.java, "-jar", catalog_jar], "catalog")
        wait(catalog_url + "/api/v1/products")
        sql("""SET search_path TO catalog_live,public;
            INSERT INTO categories(id,name,slug,description,image_url) VALUES('10000000-0000-0000-0000-000000000001','Synthetic TEST category','test-category','','');
            INSERT INTO products(id,category_id,name,slug,description,story,approval_status,created_by) VALUES('20000000-0000-0000-0000-000000000001','10000000-0000-0000-0000-000000000001','Synthetic TEST SKU','test-product','Synthetic fixture','Synthetic fixture','APPROVED','synthetic-fixture');
            INSERT INTO product_variants(id,product_id,sku_code,variant_name,weight_value) VALUES('30000000-0000-0000-0000-000000000001','20000000-0000-0000-0000-000000000001','PEER','Synthetic TEST SKU',500);
            INSERT INTO channel_prices(variant_id,channel,amount_units,created_by) VALUES('30000000-0000-0000-0000-000000000001','D2C_WEB',1000,'synthetic-fixture');
            """)
        env.update(SPRING_KAFKA_LISTENER_AUTO_STARTUP="true", CATALOG_OUTBOX_ENABLED="true")
        env.update(
            SPRING_DATASOURCE_URL=f"jdbc:postgresql://127.0.0.1:{ports['db']}/{db_name}?currentSchema=identity",
            GRPC_SERVER_PORT=str(ports["identity_grpc"]),
            SERVER_PORT=str(ports["identity"]),
            KEYCLOAK_SERVER_URL=base,
            KEYCLOAK_PUBLIC_URL=base,
            KEYCLOAK_SCOPE="openid",
            KEYCLOAK_MOBILE_REDIRECT_URI="omama://auth/callback",
            AUTH_REDIRECT_URI=gateway + "/api/v1/public/auth/callback",
            FRONTEND_BASE_URL=web,
            COOKIE_SECURE="false",
            COOKIE_DOMAIN="",
            CUSTOMER_API_GATEWAY_URL=gateway,
            KEYCLOAK_REALM=realm,
            KEYCLOAK_CLIENT_SECRET=service_secret,
            SPRING_KAFKA_BOOTSTRAP_SERVERS=f"127.0.0.1:{ports['kafka']}",
            IDENTITY_REGISTRATION_FINGERPRINT_KEY=secrets.token_urlsafe(48),
        )
        jar = next(
            p
            for p in (ROOT / "services/identity-service/build/libs").glob("*.jar")
            if "plain" not in p.name
        )
        start([args.java, "-jar", jar], "identity")
        identity_url = f"http://127.0.0.1:{ports['identity']}"
        env["IDENTITY_URL"]=identity_url
        wait(identity_url + "/v3/api-docs")
        for account in accounts.values():
            sql(
                f"INSERT INTO identity.identity_principals(principal_id,idp_issuer,idp_subject,principal_type,provisioning_status) VALUES('{account['principal']}','{issuer}','{account['subject']}','{"EMPLOYEE" if account is accounts["E"] else "CUSTOMER"}','ACTIVE');"
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
        sql(
            "INSERT INTO profile.administrative_units(code,name,full_name,parent_code,level) VALUES('HUE-01','Synthetic TEST ward','Synthetic TEST ward','75','WARD');"
        )
        internal_token = catalog_token
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
            KAFKA_BROKERS=f"127.0.0.1:{ports['kafka']}",
        )
        extension = ".exe" if os.name == "nt" else ""
        profile_binary = folder / ("profile" + extension)
        run(
            [args.go, "build", "-o", profile_binary, "./cmd/server"],
            ROOT / "services/profile-service",
        )
        start([profile_binary], "profile")
        profile_url = f"http://127.0.0.1:{ports['profile']}"
        peer_ready = folder / "inventory-ready.json"
        peer_ready.unlink(missing_ok=True)
        env.update(
            INVENTORY_TEST_DATABASE_URL=f"postgres://peer:{password}@127.0.0.1:{ports['db']}/{db_name}?sslmode=disable",
            PEER_TOKEN=secrets.token_urlsafe(32),
            PEER_CA_FILE=str(folder / "inventory-ca.pem"),
            PEER_READY_FILE=str(peer_ready),
            PEER_STOP_FILE=str(folder / "inventory-stop"),
            PEER_RESULT_FILE=str(folder / "inventory-result.json"),
        )
        (folder / "inventory-stop").unlink(missing_ok=True)
        if args.fulfillment:
            env.update(PEER_ALLOW_DOCKER="true", PEER_PACKING_TOKEN=secrets.token_urlsafe(40))
            (folder / "inventory-reads").write_text(env["PEER_PACKING_TOKEN"])
        inventory_binary = folder / ("inventory" + extension)
        run(
            [args.go, "build", "-o", inventory_binary, "./tests/peerfixture"],
            ROOT / "services/inventory-service",
        )
        start([inventory_binary], "inventory", ROOT / "services/inventory-service")
        for _ in range(100):
            if peer_ready.exists():
                break
            time.sleep(0.1)
        else:
            raise RuntimeError("Inventory TLS provider not ready")
        peer_address = json.loads(peer_ready.read_text())["address"]
        if args.fulfillment:
            peer_address = "127.0.0.1:" + peer_address.rsplit(":",1)[1]
            # The fixture exports only a public CA certificate (no private key).
            # Its native owner writes mode 0600; the unprivileged Fulfillment
            # container must be able to read this public trust anchor on Linux.
            (folder / "inventory-ca.pem").chmod(0o644)
        (folder / "inventory-token").write_text(env["PEER_TOKEN"])
        # Revalidation must fail on a changed price, future price, or unapproved parent product.
        headers={"X-Internal-Token":catalog_token}
        selection={"items":[{"sku_code":"PEER","quantity":1}],"channel":"D2C_WEB"}
        assert request(catalog_url+"/api/v1/internal/catalog/snapshots",selection)[0]==401
        status,pinned,_=request(catalog_url+"/api/v1/internal/catalog/snapshots",selection,headers=headers)
        assert status==200 and pinned["subtotal"]==1000
        assert request(catalog_url+"/api/v1/internal/catalog/snapshots/"+pinned["snapshot_id"]+"/revalidate",method="POST",headers=headers)[1]["valid"]
        sql("UPDATE catalog_live.channel_prices SET amount_units=1001,version=version+1;")
        assert not request(catalog_url+"/api/v1/internal/catalog/snapshots/"+pinned["snapshot_id"]+"/revalidate",method="POST",headers=headers)[1]["valid"]
        sql("UPDATE catalog_live.channel_prices SET amount_units=1000,version=version+1,effective_from=NOW()+interval '1 day';")
        assert request(catalog_url+"/api/v1/internal/catalog/snapshots",selection,headers=headers)[0]==422
        sql("UPDATE catalog_live.channel_prices SET effective_from=NOW()-interval '1 day'; UPDATE catalog_live.products SET approval_status='DRAFT';")
        assert request(catalog_url+"/api/v1/internal/catalog/snapshots",selection,headers=headers)[0]==422
        sql("UPDATE catalog_live.products SET approval_status='APPROVED',version=version+1;")
        print("PASS: real Catalog rejects missing credentials, stale snapshots, future prices and unapproved products",flush=True)
        # Catalog reads live display availability from the authenticated TLS Inventory API.
        catalog_process.terminate()
        catalog_process.wait(timeout=20)
        caller_environment=env.copy()
        env.clear();env.update(catalog_environment)
        env.update(IDENTITY_URL=caller_environment["IDENTITY_URL"], CATALOG_INVENTORY_TARGET=peer_address, CATALOG_INVENTORY_TOKEN_FILE=str(folder / "inventory-token"), CATALOG_INVENTORY_CA_FILE=str(folder / "inventory-ca.pem"), CATALOG_INVENTORY_SERVER_NAME="localhost")
        start([args.java, "-jar", catalog_jar], "catalog")
        env.clear();env.update(caller_environment)
        wait(catalog_url + "/api/v1/products")
        shipping_token = secrets.token_urlsafe(32)
        shipping_webhook = secrets.token_urlsafe(32)
        env.update(
            SHIPPING_DATABASE_URL=f"postgres://peer:{password}@127.0.0.1:{ports['db']}/{db_name}?sslmode=disable&search_path=shipping",
            SHIPPING_HTTP_ADDRESS=f"127.0.0.1:{ports['shipping']}",
            SHIPPING_GRPC_ADDRESS=f"127.0.0.1:{ports['shipping_grpc']}",
            SHIPPING_INTERNAL_TOKEN=shipping_token,
            SHIPPING_FAKE_WEBHOOK_SECRET=shipping_webhook,
            SHIPPING_KAFKA_BROKERS=env["KAFKA_BROKERS"],
            SHIPPING_V2_ENABLED="true",
        )
        shipping_binary = folder / ("shipping" + extension)
        run(
            [args.go, "build", "-o", shipping_binary, "./cmd/server"],
            ROOT / "services/shipping-service",
        )
        shipping_process = start([shipping_binary], "shipping")
        shipping_url = f"http://127.0.0.1:{ports['shipping']}"
        wait(shipping_url + "/health/ready")
        (folder / "shipping-token").write_text(shipping_token)
        if args.fulfillment:
            fulfillment_network = "codex-web-fulfillment-"+suffix
            fulfillment_network, fulfillment_scanner = fulfillment_checks.prepare(ROOT,folder,ports,env,accounts,sql,run,container,request,issuer,None,db_name,password,suffix,peer_address)
            try:
                wait(f"http://127.0.0.1:{ports['fulfillment']}/health/ready")
            except RuntimeError:
                diagnostic=run(["docker","logs","codex-web-fulfillment-"+suffix],capture_output=True,text=True)
                (folder/"fulfillment-container.log").write_text(diagnostic.stdout+diagnostic.stderr,encoding="utf-8")
                raise
        env.update(
            WEB_TEST_CHECKOUT="true",
            WEB_TEST_REAL_CATALOG_URL=catalog_url,
            WEB_TEST_FIXTURE_DIR=str(folder),
            WEB_TEST_CONTROL_PORT=str(ports["control"]),
            WEB_TEST_ORDER_DATABASE=f"postgres://peer:{password}@127.0.0.1:{ports['db']}/{db_name}?sslmode=disable&search_path=orders",
            WEB_TEST_CATALOG_DATABASE=f"postgres://peer:{password}@127.0.0.1:{ports['db']}/{db_name}?sslmode=disable&search_path=catalog_test",
            SEPAY_INVENTORY_GRPC=peer_address,
            SEPAY_INVENTORY_PROFILE="terminal",
            SEPAY_INVENTORY_TOKEN_FILE=str(folder / "inventory-token"),
            SEPAY_INVENTORY_CA_FILE=env["PEER_CA_FILE"],
            SEPAY_INVENTORY_SERVER_NAME="localhost",
            ORDER_SHIPPING_GRPC=f"127.0.0.1:{ports['shipping_grpc']}",
            ORDER_SHIPPING_TOKEN_FILE=str(folder / "shipping-token"),
            ORDER_SHIPPING_ALLOW_LOCAL_INSECURE="true",
        )
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
            env["WEB_TEST_CUSTOMER_" + key] = account["customer"]
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
            ("http://catalog-service:8006", "catalog"),
            ("http://order-service:8004", "order"),
        ]:
            kong = kong.replace(old, f"http://host.docker.internal:{ports[key]}")
        if args.fulfillment:
            import yaml
            config=yaml.safe_load(kong)
            for service in config['services']:
                if service['name']=='fulfillment-service':service['url']=f"http://host.docker.internal:{ports['fulfillment']}"
            kong=yaml.safe_dump(config,sort_keys=False)
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
            WEB_AUTH_MODE="identity-bff",
            WEB_PUBLIC_URL=web,
            WEB_OIDC_ISSUER=issuer,
            WEB_OIDC_CLIENT_ID="frontend-web",
            WEB_OIDC_API_AUDIENCE="omama-api",
            WEB_OIDC_COOKIE_SECRET=secrets.token_urlsafe(48),
            WEB_OIDC_ALLOW_LOCAL_HTTP="true",
            IDENTITY_UPSTREAM_URL=gateway + "/api/v1",
            CUSTOMER_CORE_UPSTREAM_URL=gateway + "/api/v1/customer-api",
            CUSTOMER_EXTENSIONS_UPSTREAM_URL=gateway + "/api/v1/customer-api",
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
        status,listing,_=request(web+"/api/catalog/products")
        assert status==200 and listing["items"][0]["product_id"]=="20000000-0000-0000-0000-000000000001", ("Catalog Web list",status,listing)
        status,detail,_=request(web+"/api/catalog/products/test-product")
        assert status==200 and detail["variants"][0]["sku_code"]=="PEER" and detail["variants"][0]["stock_available"]==100, ("Catalog Web detail",status,detail)
        print("PASS: Web reads product, price, SKU and live Inventory availability from real Catalog",flush=True)
        for key in ["A","E"]:
            native_identity(request,gateway,issuer,profile_url,f"http://127.0.0.1:{ports['order']}",accounts[key],account_password,key=="E")
        print("PASS: native PKCE/JSON callback, app-bound state, refresh/logout and direct bearer Customer/Employee gates",flush=True)
        real_packing = None
        if args.fulfillment:
            staff_session=native_identity(request,gateway,issuer,profile_url,f"http://127.0.0.1:{ports['order']}",accounts['E'],account_password,True,retain=True)
            customer_session=native_identity(request,gateway,issuer,profile_url,f"http://127.0.0.1:{ports['order']}",accounts['A'],account_password,False,retain=True)
            fulfillment_checks.pack.storage_port=ports['storage']
            def shipping_control(online):
                nonlocal shipping_process
                if online:
                    shipping_process=start([shipping_binary],"shipping")
                    wait(shipping_url+"/health/ready")
                else:
                    shipping_process.terminate()
                    shipping_process.wait(timeout=20)
            def handoff(order_id):
                return json.loads(sql(f"SELECT json_build_object('hash',h.request_hash,'attempts',h.attempts,'shipment_id',h.shipment_id,'shipments',(SELECT count(*) FROM shipping.shipments s WHERE s.order_id=h.order_id::text)) FROM orders.fulfillment_shipping_handoffs h WHERE h.order_id='{uuid.UUID(order_id)}';",tuples=True) or "null")
            def real_packing(order_id, order_until):
                return fulfillment_checks.pack(request,gateway,staff_session['accessToken'],customer_session['accessToken'],folder,order_id,order_until,shipping_url,shipping_token,f"http://127.0.0.1:{ports['control']}",run,fulfillment_scanner,shipping_control,handoff)
        foreign_address = None
        registration = {
            "email": f"registered-{suffix}@example.test",
            "firstName": "New",
            "lastName": "Customer",
            "password": account_password,
        }
        registration_key = uuid.uuid4().hex
        registration_headers = {"Origin": web, "Idempotency-Key": registration_key}
        assert (
            request(
                web + "/api/auth/register",
                registration,
                headers={**registration_headers, "Origin": "https://attacker.invalid"},
            )[0]
            == 403
        )
        assert (
            request(
                web + "/api/auth/register",
                dict(registration, password="short"),
                headers={**registration_headers, "Idempotency-Key": uuid.uuid4().hex},
            )[0]
            == 422
        )
        sql(
            "CREATE SEQUENCE profile.registration_test_attempt; CREATE FUNCTION profile.registration_test_outage() RETURNS trigger AS $$ BEGIN PERFORM nextval('profile.registration_test_attempt'); RAISE EXCEPTION 'injected provisioning outage'; END; $$ LANGUAGE plpgsql; CREATE TRIGGER registration_test_outage BEFORE INSERT ON profile.customer_profiles FOR EACH ROW EXECUTE FUNCTION profile.registration_test_outage();"
        )
        status, registered, _ = request(
            web + "/api/auth/register", registration, headers=registration_headers
        )
        assert status == 202 and registered["status"] == "REGISTERED", (
            "registration",
            status,
            registered,
        )
        assert (
            request(
                web + "/api/auth/register", registration, headers=registration_headers
            )[1]["status"]
            == "REGISTERED"
        )
        assert (
            request(
                web + "/api/auth/register",
                dict(registration, password=account_password + "changed"),
                headers=registration_headers,
            )[0]
            == 409
        )
        principal = (
            sql(
                f"SELECT principal_id FROM identity.registration_operations WHERE idempotency_key='{registration_key}';"
            )
            .splitlines()[2]
            .strip()
        )
        event = (
            sql(
                f"SELECT payload::text FROM identity.outbox_events WHERE aggregate_id='{principal}';"
            )
            .splitlines()[2]
            .strip()
        )
        run(
            [
                "docker",
                "exec",
                "-i",
                kafka_name,
                "/opt/kafka/bin/kafka-console-producer.sh",
                "--bootstrap-server",
                "localhost:9092",
                "--topic",
                "identity.events.v1",
            ],
            input=event + "\n",
            text=True,
            encoding="utf-8",
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )
        for _ in range(120):
            attempts = int(
                sql(
                    "SELECT CASE WHEN is_called THEN last_value ELSE 0 END FROM profile.registration_test_attempt;"
                )
                .splitlines()[2]
                .strip()
            )
            if attempts >= 2:
                break
            time.sleep(0.25)
        assert attempts >= 2, "Provisioning outage was not exercised"
        assert "(0 rows)" in sql(
            f"SELECT id FROM profile.customer_profiles WHERE user_id='{principal}';"
        )
        sql("DROP TRIGGER registration_test_outage ON profile.customer_profiles;")
        for _ in range(160):
            rows = sql(
                f"SELECT id FROM profile.customer_profiles WHERE user_id='{principal}';"
            ).splitlines()
            if len(rows) > 2 and re.fullmatch(r"[0-9a-f-]{36}", rows[2].strip()):
                customer = rows[2].strip()
                break
            time.sleep(0.25)
        else:
            raise AssertionError("Identity outbox/Kafka did not provision Customer")
        assert customer != principal
        # Re-deliver exactly the persisted registration payload through Kafka.
        event = (
            sql(
                f"SELECT payload::text FROM identity.outbox_events WHERE aggregate_id='{principal}';"
            )
            .splitlines()[2]
            .strip()
        )
        run(
            [
                "docker",
                "exec",
                "-i",
                kafka_name,
                "/opt/kafka/bin/kafka-console-producer.sh",
                "--bootstrap-server",
                "localhost:9092",
                "--topic",
                "identity.events.v1",
            ],
            input=event + "\n",
            text=True,
            encoding="utf-8",
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )
        time.sleep(2)
        assert "(1 row)" in sql(
            f"SELECT id FROM profile.customer_profiles WHERE user_id='{principal}';"
        )
        assert "(1 row)" in sql(
            f"SELECT event_id FROM identity.outbox_events WHERE aggregate_id='{principal}' AND delivery_status='PUBLISHED';"
        )
        accounts["C"] = {
            "principal": principal,
            "customer": customer,
            "email": registration["email"],
        }
        print(
            "PASS: registration -> real Keycloak -> atomic Identity outbox -> real Kafka -> Customer; transient DB failure recovers without skipping offsets, retries/replay do not duplicate",
            flush=True,
        )
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
            status, _, headers = request(headers["Location"], opener=opener)
            assert status == 302, "Identity BFF did not initiate PKCE"
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
                if cookie.name in ["OMAMA_SESSION", "OMAMA_AUTH_STATE"]
            ), "Token cookie is script readable"
            assert any(c.name == "OMAMA_SESSION" for c in jar)
            cookie_credential = next(c.value for c in jar if c.name == "OMAMA_SESSION")
            status, public_session, _ = request(
                gateway + "/api/v1/public/auth/session", opener=opener
            )
            assert status == 200 and public_session["data"]["authenticated"]
            assert (
                public_session["data"]["sessionId"] != cookie_credential
            ), "Public JSON exposed the HttpOnly session credential"
            assert (
                request(
                    gateway + "/api/v1/public/auth/logout",
                    {},
                    opener=opener,
                    headers={"Origin": "https://attacker.invalid"},
                )[0]
                == 403
            )
            assert not any(
                c.name in ["oma_access_token", "oma_refresh_token", "oma_id_token"]
                for c in jar
            ), "Token cookie leaked from Identity BFF"
            status, session, _ = request(web + "/api/auth/me", opener=opener)
            if key == "E":
                assert status==200 and session["principal"]["principalType"]=="EMPLOYEE" and not session["customerAccess"] and session["customer"] is None
                for path in ["/api/cart", "/api/checkout/quote", "/api/orders", "/api/customer/addresses"]:
                    assert request(web+path, {} if path=="/api/checkout/quote" else None, opener=opener,headers={"Origin":web})[0]==403, ("Employee Web purchase gate",path)
                assert request(gateway+"/api/v1/customer-api/cart",opener=opener)[0]==403
                print("PASS: Employee with legacy CUSTOMER role and active Customer mapping is denied all Web purchase paths",flush=True)
                continue
            assert (
                status == 200
                and session["customer"]["id"] == account["customer"]
                and session["principal"]["principalId"] == account["principal"]
            ), "Session did not bind Identity to Profile"
            assert not any(
                field in json.dumps(session)
                for field in ["access_token", "refresh_token", "id_token"]
            ), "Session exposes tokens"
            if key == "C":
                assert session["customer"]["status"] == "PENDING_VERIFICATION"
                status, caps, _ = request(web + "/api/capabilities", opener=opener)
                assert status == 200 and caps["capabilities"] == ["PROFILE_MANAGE"]
                status, profile, _ = request(
                    web + "/api/customer/profile", opener=opener
                )
                assert status == 200 and profile["id"] == customer
                update = {
                    "full_name": "New Customer Updated",
                    "phone_number": "0905123456",
                    "email": registration["email"],
                    "version": profile["version"],
                }
                status, profile, _ = request(
                    web + "/api/customer/profile",
                    update,
                    method="PUT",
                    opener=opener,
                    headers={"Origin": web, "If-Match": f'"{profile["version"]}"'},
                )
                assert status == 200 and profile["status"] == "PENDING_VERIFICATION", (
                    "onboarding",
                    status,
                    profile,
                )
                assert request(web + "/api/cart", opener=opener)[0] == 403
                assert request(web + "/api/customer/addresses", opener=opener)[0] == 403
                assert (
                    request(gateway + "/api/v1/customer-api/cart", opener=opener)[0]
                    == 403
                ), "Direct Order API bypassed Customer verification"
                print(
                    "PASS: newly registered Customer can complete Profile; unverified purchase/address access remains blocked",
                    flush=True,
                )
                continue
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
            status, native_refresh, refresh_headers = request(
                gateway + "/api/v1/public/auth/tokens/refresh",
                {},
                opener=opener,
                headers={"Origin": web},
            )
            assert status == 200, (
                "Identity refresh",
                status,
                native_refresh,
                refresh_headers.get("WWW-Authenticate"),
            )
            status, native_session, session_headers = request(
                gateway + "/api/v1/public/auth/session", opener=opener
            )
            assert status == 200 and native_session["data"]["authenticated"], (
                "Identity session after refresh",
                status,
                native_session,
                session_headers.get("WWW-Authenticate"),
            )
            status, refresh, _ = request(
                web + "/api/auth/refresh", {}, opener=opener, headers={"Origin": web}
            )
            assert status == 200 and refresh["authenticated"] is True, (
                "Web refresh",
                status,
                refresh,
            )
            status, profile, headers = request(
                web + "/api/customer/profile", opener=opener
            )
            assert (
                status == 200 and profile["id"] == account["customer"]
            ), "Profile route failed"
            address = profile_addresses(request, web, opener, profile, foreign_address)
            if key == "A":
                foreign_address = address
            if key == "A":
                placed_order = checkout_shipping(
                    request,
                    web,
                    opener,
                    f"http://127.0.0.1:{ports['control']}",
                    shipping_url,
                    shipping_token,
                    shipping_webhook,
                    packing=real_packing,
                )
            elif key == "B":
                for path in [
                    "/api/orders/" + placed_order,
                    "/api/orders/" + placed_order + "/tracking",
                ]:
                    assert (
                        request(web + path, opener=opener)[0] == 404
                    ), "Cross-owner order access"
            status, owner_profile, _ = request(
                gateway + "/api/v1/customer-api/profile",
                opener=opener,
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
            assert status == 200 and logout["success"] is True, "Identity logout failed"
            assert (
                request(web + "/api/auth/me", opener=opener)[0] == 401
            ), "Logout retained local session"

        assert request(web + "/api/cart")[0] == 401
        for action, expected in [("LOGIN_SUCCESS", 3), ("LOGOUT_SUCCESS", 2)]:
            count = int(
                sql(
                    "SELECT count(*) FROM identity.identity_audit_log WHERE action_code='"
                    + action
                    + "' AND actor_kind='CUSTOMER';"
                )
                .splitlines()[2]
                .strip()
            )
            assert count >= expected, ("Identity audit missing", action, count)
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
        assert (
            request(gateway + "/api/v1/public/auth/callback?state=wrong&code=wrong")[0]
            == 400
        )
        inventory_schema = (
            sql("SELECT nspname FROM pg_namespace WHERE nspname LIKE 'peer_%';")
            .splitlines()[2]
            .strip()
        )
        assert re.fullmatch(
            r"peer_[a-f0-9-]{36}", inventory_schema
        ), "Unexpected Inventory fixture schema"
        stock = json.loads(
            sql(
                f"""SELECT json_build_object('physical',i.physical_qty,'reserved',i.reserved_qty,'batch_physical',b.physical_qty,'batch_reserved',b.reserved_qty,'events',(SELECT count(*) FROM "{inventory_schema}".outbox_events WHERE event_type='vn.omama.inventory.stock.committed.v1')) FROM "{inventory_schema}".inventory_items i JOIN "{inventory_schema}".batches b ON b.sku=i.sku WHERE i.sku='PEER';"""
            )
            .splitlines()[2]
            .strip()
        )
        assert stock == {
            "physical": 98,
            "reserved": 0,
            "batch_physical": 98,
            "batch_reserved": 0,
            "events": 1,
        }, ("Inventory did not reserve/release/commit exactly once", stock)
        assert "(2 rows)" in sql("SELECT id FROM orders.orders;")
        mobile_client_runtime="NOT_RUN"
        if args.flutter:
            session=native_identity(request,gateway,issuer,profile_url,f"http://127.0.0.1:{ports['order']}",accounts["B"],account_password,False,retain=True)
            mobile_config=folder/"mobile-live.json"
            mobile_config.write_text(json.dumps({"gateway":gateway,"issuer":issuer,"session":session}),encoding="utf-8")
            env["MVP_LIVE_CONFIG"]=str(mobile_config)
            with (folder/"mobile-live.log").open("w",encoding="utf-8") as log:
                run([args.flutter,"test","--no-pub","--dart-define=IDENTITY_ALLOW_HTTP=true","test/live_mvp_contract_test.dart"],ROOT/"apps/mobile-app",stdout=log,stderr=log)
            after=json.loads(sql(f"SELECT json_build_object('physical',physical_qty,'reserved',reserved_qty) FROM \"{inventory_schema}\".inventory_items WHERE sku='PEER';",tuples=True))
            assert after == {"physical":98,"reserved":0}, ("Native cancellation did not release Inventory",after)
            mobile_client_runtime="PASS"
            print("PASS: native Dart runtime -> real Catalog/Profile/cart/quote/checkout/replay/order/payment instructions/cancel -> Inventory released",flush=True)
        result = {
            "status": "PASS",
            "head": subprocess.check_output(
                ["git", "rev-parse", "HEAD"], cwd=ROOT, text=True
            ).strip(),
            "scope": "real Web/Keycloak/Kong/Identity/Profile/Catalog/Order/Inventory-TLS/Shipping-v2/PostgreSQL/Redis/Kafka; " + ("real Fulfillment/RustFS-versioned-video/ffprobe/ClamAV (fixture signature DB); " if args.fulfillment else "Fulfillment TEST; ") + "SePay statements TEST, carrier FAKE, no external money",
            "mobile_client_runtime": mobile_client_runtime,
            "accounts": 4,
            "inventory": stock,
            "flows": [
                "native-identity-pkce-state-refresh-logout-and-persona-gates",
                "real-catalog-public-snapshot-revalidation-and-inventory-display",
                "profile-addresses-ownership-CAS",
                "registration-outbox-kafka-provisioning-replay-transient-failure",
                "checkout-inventory-reserve-cancel-release-payment-commit",
                "shipping-revision-kafka-delivery-order-history",
                "native-app-bound-PKCE-refresh-logout-persona-gates",
                *( ["native-Dart-real-purchase-cancel-and-Inventory-release"] if args.flutter else [] ),
            ],
        }
        (folder / "result.json").write_text(json.dumps(result, indent=2))
        print(
            "PASS: no browser identity-header bypass; no Web token JSON exposure; missing session and invalid callback rejected",
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
            import traceback

            traceback.print_exc()
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
        (folder / "inventory-stop").write_text("stop")
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
        if fulfillment_network:
            subprocess.run(["docker","network","rm",fulfillment_network],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)


if __name__ == "__main__":
    main()
