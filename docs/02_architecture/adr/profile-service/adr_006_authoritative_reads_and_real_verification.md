# ADR-006 — Authoritative reads và kiểm chứng implementation thật

- Ngày: 08/10/2026
- Trạng thái: Implemented

## Bối cảnh

L1 không có TTL; Pub/Sub có thể mất invalidation. Contract test tự tạo HTTP handler giả, benchmark chỉ gọi mock RAM nhưng được diễn giải là gRPC SLA P99.

## Quyết định

Profile và checkout address đọc PostgreSQL trực tiếp. Cache helper bổ sung TTL/deep copy và best-effort invalidation, không là nguồn authoritative cho PII. Không tuyên bố SLA từ benchmark giả.

Sinh/register protobuf service thật, kiểm tra bằng gRPC client/network và PostgreSQL. Postman gọi backend Docker thật với assertions chính xác; bỏ simulation POST để giả gRPC/Kafka. TDD/integration dùng chính migration đã version-control. Kafka kiểm chứng event do worker thật phát, không chỉ validate request fixture.

## Hệ quả

Tăng DB load cho profile reads; cần đo tải thực trước khi bật cache lại. Benchmark PostgreSQL chỉ cho ns/op và allocations trong môi trường kiểm thử, không đại diện production P99. OTP integration, MS-18 consumer và production infrastructure vẫn cần kiểm chứng riêng.
