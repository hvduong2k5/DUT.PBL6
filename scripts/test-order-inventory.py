"""Run the real Order adapter against Inventory over TLS and disposable PostgreSQL.

Requires Docker, Python 3 and Go >=1.25 (Inventory toolchain). No deployment secrets.
"""

import argparse
import json
import os
from pathlib import Path
import secrets
import subprocess
import tempfile
import time
import uuid


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--go", default="go")
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[1]
    inventory = root / "services/inventory-service"
    order = root / "services/order-service"
    name = "codex-order-inventory-" + uuid.uuid4().hex[:10]
    password = secrets.token_urlsafe(24)
    env = os.environ.copy()
    env.update(POSTGRES_PASSWORD=password, PEER_TOKEN=secrets.token_urlsafe(32))

    def run(command, cwd=None, **kwargs):
        return subprocess.run(command, cwd=cwd, env=env, check=True, **kwargs)

    started = False
    provider = None
    with tempfile.TemporaryDirectory(prefix="order-inventory-") as folder:
        folder = Path(folder)
        stop = folder / "stop"
        ready = folder / "ready.json"
        result_file = folder / "result.json"
        env.update(
            PEER_CA_FILE=str(folder / "ca.pem"),
            PEER_READY_FILE=str(ready),
            PEER_STOP_FILE=str(stop),
            PEER_RESULT_FILE=str(result_file),
        )
        try:
            run(
                [
                    "docker",
                    "run",
                    "-d",
                    "--name",
                    name,
                    "-e",
                    "POSTGRES_PASSWORD",
                    "-e",
                    "POSTGRES_USER=peer",
                    "-e",
                    "POSTGRES_DB=peer_test",
                    "-p",
                    "127.0.0.1::5432",
                    "postgres:16-alpine",
                ],
                stdout=subprocess.DEVNULL,
            )
            started = True
            for _ in range(90):
                result = subprocess.run(
                    [
                        "docker",
                        "exec",
                        name,
                        "pg_isready",
                        "-U",
                        "peer",
                        "-d",
                        "peer_test",
                    ],
                    stdout=subprocess.DEVNULL,
                    stderr=subprocess.DEVNULL,
                )
                if result.returncode == 0:
                    break
                time.sleep(1)
            else:
                raise RuntimeError("Disposable PostgreSQL did not become ready")
            port = (
                subprocess.check_output(["docker", "port", name, "5432/tcp"], text=True)
                .strip()
                .rsplit(":", 1)[1]
            )
            env["INVENTORY_TEST_DATABASE_URL"] = (
                f"postgres://peer:{password}@127.0.0.1:{port}/peer_test?sslmode=disable"
            )
            env["PEER_ORDER_DATABASE_URL"] = env["INVENTORY_TEST_DATABASE_URL"]
            binary = folder / ("provider.exe" if os.name == "nt" else "provider")
            run([args.go, "build", "-o", str(binary), "./tests/peerfixture"], inventory)
            with (folder / "provider.log").open("w") as log:
                provider = subprocess.Popen(
                    [str(binary)],
                    cwd=inventory,
                    env=env,
                    stdout=log,
                    stderr=log,
                    creationflags=subprocess.CREATE_NO_WINDOW if os.name == "nt" else 0,
                )
                for _ in range(100):
                    if provider.poll() is not None:
                        log.flush()
                        detail = (
                            (folder / "provider.log")
                            .read_text()
                            .replace(password, "[redacted]")
                            .replace(env["PEER_TOKEN"], "[redacted]")
                        )
                        raise RuntimeError(
                            "Provider fixture exited before readiness: " + detail
                        )
                    if ready.exists():
                        break
                    time.sleep(0.1)
                else:
                    raise RuntimeError("Provider fixture did not become ready")
                env["ORDER_INVENTORY_TEST_ENDPOINT"] = json.loads(ready.read_text())[
                    "address"
                ]
                run(
                    [
                        args.go,
                        "test",
                        "-count=1",
                        "-v",
                        "./internal/paymentinventory",
                        "-run",
                        "^TestTerminalPeerFlow$",
                    ],
                    order,
                )
                run(
                    [
                        args.go,
                        "test",
                        "-count=1",
                        "-v",
                        "./internal/presentation/grpc",
                        "-run",
                        "^TestTerminalPostgresLifecycle$",
                    ],
                    inventory,
                )
                stop.write_text("stop")
                if provider.wait(timeout=10) != 0:
                    raise RuntimeError("Provider fixture cleanup failed")
                summary = json.loads(result_file.read_text())
                expected = {
                    "physical": 94,
                    "reserved": 0,
                    "batch_physical": 94,
                    "batch_reserved": 0,
                    "events": 3,
                }
                if summary != expected:
                    raise AssertionError(
                        f"Stock/outbox changed more than once: {summary}"
                    )
                print(
                    "PASS: stock 100 -> 94, no reserved stock left, exactly three commit outbox events despite replay/lost ACK"
                )
            print(
                "PASS: real Order adapter -> TLS gRPC Inventory -> PostgreSQL; no simulator"
            )
        finally:
            if provider and provider.poll() is None:
                stop.write_text("stop")
                try:
                    provider.wait(timeout=10)
                except subprocess.TimeoutExpired:
                    provider.terminate()
                    provider.wait(timeout=10)
            if started:
                subprocess.run(
                    ["docker", "rm", "-f", name],
                    stdout=subprocess.DEVNULL,
                    stderr=subprocess.DEVNULL,
                )


if __name__ == "__main__":
    main()
