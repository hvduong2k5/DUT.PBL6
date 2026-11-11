import json
import os
from dataclasses import dataclass
from pathlib import Path
from uuid import UUID

TOPIC = "order.events.v3"
COHORT = "order-v3"
SCHEMA_ROOT = Path(__file__).resolve().parents[4] / "packages/events/schemas/order/v3"
MIGRATION_ROOT = Path(__file__).resolve().parents[2] / "migrations"


def setting(name: str, default: str = "") -> str:
    value = os.getenv(f"ANALYTICS_{name}", default)
    path = os.getenv(f"ANALYTICS_{name}_FILE")
    return Path(path).read_text(encoding="utf-8").strip() if path else value


def positive(name: str, default: int) -> int:
    value = int(setting(name, str(default)))
    if value <= 0:
        raise ValueError(f"ANALYTICS_{name} must be positive")
    return value


@dataclass(frozen=True)
class Settings:
    postgres_dsn: str
    clickhouse_host: str
    clickhouse_port: int
    clickhouse_user: str
    clickhouse_password: str
    clickhouse_secure: bool
    brokers: str
    kafka_options: dict
    pipeline: str
    scope: str
    schema_dir: Path
    migrations: Path
    build_interval: int
    max_fact_rows: int
    freshness_seconds: int
    supply_enabled: bool = False

    @classmethod
    def load(cls):
        pipeline = setting("PIPELINE", "order-v3-dev")
        options = json.loads(setting("KAFKA_OPTIONS", "{}"))
        allowed = {
            "security.protocol",
            "sasl.mechanism",
            "sasl.username",
            "sasl.password",
            "ssl.ca.location",
            "ssl.endpoint.identification.algorithm",
        }
        if not isinstance(options, dict) or options.keys() - allowed:
            raise ValueError("Invalid Kafka connection options")
        return cls(
            setting("POSTGRES_DSN"),
            setting("CLICKHOUSE_HOST", "clickhouse"),
            positive("CLICKHOUSE_PORT", 8123),
            setting("CLICKHOUSE_USER", "analytics"),
            setting("CLICKHOUSE_PASSWORD"),
            setting("CLICKHOUSE_SECURE") == "1",
            setting("KAFKA_BROKERS", "kafka:9092"),
            options,
            pipeline,
            setting("SCOPE", "order-v3"),
            Path(setting("SCHEMA_DIR", str(SCHEMA_ROOT))),
            Path(setting("MIGRATIONS", str(MIGRATION_ROOT))),
            positive("BUILD_INTERVAL", 60),
            positive("MAX_FACT_ROWS", 500_000),
            positive("FRESHNESS_SECONDS", 86_400),
            setting("SUPPLY_ENABLED") == "1",
        )

    def kafka(self):
        return {"bootstrap.servers": self.brokers, **self.kafka_options}


@dataclass(frozen=True)
class AuthSettings:
    issuer: str
    public_key: str
    identity_url: str
    pilot_ids: frozenset[str]
    audience: str = "omama-api"
    session_origin: str = ""

    @classmethod
    def load(cls):
        result = cls(
            setting("JWT_ISSUER"),
            setting("JWT_PUBLIC_KEY"),
            setting("IDENTITY_URL"),
            frozenset(
                str(UUID(item.strip())) for item in setting("PILOT_IDS").split(",") if item.strip()
            ),
            setting("JWT_AUDIENCE", "omama-api"),
            setting("SESSION_ORIGIN"),
        )
        if not all((result.issuer, result.public_key, result.identity_url, result.pilot_ids)):
            raise ValueError("JWT issuer/key, Identity URL and pilot IDs are required for the API")
        return result
