import json
from dataclasses import dataclass, field
from pathlib import Path
from typing import Annotated, Literal
from uuid import UUID

from pydantic import AwareDatetime, BaseModel, ConfigDict, Field, StringConstraints, TypeAdapter

from ..config import positive, setting

Code = Annotated[str, StringConstraints(pattern=r"^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$")]
Count = Annotated[int, Field(strict=True, ge=0, le=2_147_483_647)]
Version = Annotated[int, Field(strict=True, ge=1)]


class Model(BaseModel):
    model_config = ConfigDict(extra="forbid", strict=True)


class Window(Model):
    from_: AwareDatetime = Field(alias="from")
    to: AwareDatetime


class Manifest(Model):
    schemaVersion: Literal["1"]
    transport: Literal["FILE_JSONL"]
    dataset: Literal["catalog", "sellability", "completed_baskets"]
    source: Literal["catalog-service", "inventory-service", "order-service"]
    snapshotId: UUID
    sourceRevision: Version
    scopeId: Code
    channel: Literal["D2C_WEB"]
    warehouseId: UUID
    sourceAsOf: AwareDatetime
    publishedAt: AwareDatetime
    expiresAt: AwareDatetime
    window: Window | None
    coverageStatus: Literal["COMPLETE"]
    policyVersion: Annotated[str, StringConstraints(min_length=1, max_length=128)]
    coverageEvidenceRef: Annotated[str, StringConstraints(min_length=1, max_length=256)]
    recordCount: Count
    pageCount: None
    dataFile: Literal["records.jsonl"]
    dataSha256: Annotated[str, StringConstraints(pattern="^[0-9a-f]{64}$")]


class Catalog(Model):
    recordKey: str
    skuCode: Code
    productId: UUID
    categoryId: UUID | None
    productVersion: Count
    skuVersion: Count
    priceVersion: Count | None
    publiclyVisible: bool
    skuActive: bool
    slug: Annotated[str, StringConstraints(max_length=500)]
    name: Annotated[str, StringConstraints(max_length=500)]
    thumbnailUrl: Annotated[str, StringConstraints(max_length=2048)]
    priceVnd: Annotated[int, Field(strict=True, ge=1, le=1_000_000_000_000)] | None
    weightGrams: Annotated[int, Field(strict=True, ge=1)] | None
    packagingType: Annotated[str, StringConstraints(max_length=100)] | None
    fallbackRank: Version | None
    sourceUpdatedAt: AwareDatetime


class Sellability(Model):
    recordKey: str
    skuCode: Code
    warehouseId: UUID
    skuActive: bool
    sellableUnits: Count
    eligibilityValidUntil: AwareDatetime
    observedAt: AwareDatetime


class BasketItem(Model):
    skuCode: Code
    quantityAtCompletion: Annotated[int, Field(strict=True, ge=1, le=999)]


class Basket(Model):
    recordKey: str
    orderId: UUID
    orderVersion: Version
    cohort: Literal["order-core-1"]
    channel: Literal["D2C_WEB"]
    warehouseId: UUID
    completedAt: AwareDatetime
    eligible: bool
    eligibilityReason: str | None
    customerPrincipalId: UUID | None
    identityMappingVersion: str | None
    items: Annotated[list[BasketItem], Field(min_length=1, max_length=100)]


class ConsentWrite(Model):
    enabled: bool
    policyVersion: Literal["personalization-v1"]


class Behavior(Model):
    eventId: UUID
    eventType: Literal["VIEW_ITEM", "ADD_TO_CART"]
    skuCode: Code
    occurredAt: AwareDatetime
    consentVersion: Version
    recommendationId: UUID | None = None


@dataclass(frozen=True)
class Config:
    dsn: str
    scope: str
    warehouse: str
    supply_warehouse: str
    subject_key: bytes
    key_version: str = "subject-v1"
    mapping_version: str = "scope-v1"
    identity_mapping_version: str = "identity-core-v1"
    export_root: Path = Path("/run/recommendation-exports")
    related: bool = True
    basket: bool = True
    personalized: bool = True
    catalog_age: int = 600
    inventory_age: int = 120
    order_age: int = 1800
    model_age: int = 86400
    interval: int = 60
    fallback_ranks: dict[str, int] = field(default_factory=dict)
    supply_scope: str = "order-v3"
    supply_pipeline: str = "order-v3-dev"

    @classmethod
    def load(cls):
        scope = json.loads(Path(setting("REC_SCOPE_CONFIG_FILE")).read_text(encoding="utf-8"))
        if scope.get("channel") != "D2C_WEB" or scope.get("orderCohort") != "order-core-1":
            raise ValueError("Recommendation requires an approved D2C/core scope")
        TypeAdapter(Code).validate_python(scope["scopeId"])
        UUID(scope["warehouseId"])
        key = setting("REC_SUBJECT_KEY").encode()
        if len(key) < 32 or not setting("REC_POSTGRES_DSN"):
            raise ValueError("Recommendation DB credential and subject key >=32 bytes required")
        fallback = setting("REC_FALLBACK_POLICY_FILE")
        ranks = json.loads(Path(fallback).read_text(encoding="utf-8")) if fallback else {}
        if any(not isinstance(k, str) or type(v) is not int or v < 1 for k, v in ranks.items()):
            raise ValueError("Fallback policy must map SKU codes to positive ranks")
        return cls(
            dsn=setting("REC_POSTGRES_DSN"),
            scope=scope["scopeId"],
            warehouse=str(UUID(scope["warehouseId"])),
            supply_warehouse=scope["supplyWarehouseId"],
            subject_key=key,
            key_version=setting("REC_KEY_VERSION", "subject-v1"),
            mapping_version=scope["mappingVersion"],
            identity_mapping_version=scope["identityMappingVersion"],
            export_root=Path(setting("REC_EXPORT_ROOT", "/run/recommendation-exports")),
            related=setting("REC_RELATED_ENABLED", "1") == "1",
            basket=setting("REC_BASKET_ENABLED", "1") == "1",
            personalized=setting("REC_PERSONALIZED_ENABLED", "1") == "1",
            catalog_age=positive("REC_CATALOG_MAX_AGE_SECONDS", 600),
            inventory_age=positive("REC_INVENTORY_MAX_AGE_SECONDS", 120),
            order_age=positive("REC_ORDER_MAX_AGE_SECONDS", 1800),
            model_age=positive("REC_MODEL_MAX_AGE_SECONDS", 86400),
            interval=positive("REC_SYNC_INTERVAL_SECONDS", 60),
            fallback_ranks=ranks,
            supply_scope=setting("SCOPE", "order-v3"),
            supply_pipeline=setting("PIPELINE", "order-v3-dev"),
        )
