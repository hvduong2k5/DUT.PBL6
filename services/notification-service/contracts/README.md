# Local foundation contract 0.1

`notification.openapi.json` describes the executable local N1 subset. Strict commands support ORDER_CONFIRMATION/BUYER/vi-VN only. New and duplicate responses expose actual `WAITING_CONTEXT`, not provider acceptance. Context references must be opaque; no contacts, OTPs or content in intake. Example input is in `examples/intake.json`.

`events/manifest.json` pins the Order v3 schema copied byte-for-byte from the audited base. Subscription is **disabled** and there is no Kafka consumer in N1. It is a contract fixture, not broker/deployment evidence. No shared package was modified.

Local Bearer maps to order-service with ORDER_CONFIRMATION permission; missing local token denies all protected endpoints. The executable refuses non-local mode and non-loopback listener. Real service identities/recipient resolution remain gated. Retry the same command/key on unknown HTTP/DB commit outcome.
