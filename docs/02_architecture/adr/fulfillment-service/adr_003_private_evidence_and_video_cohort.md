# ADR-003 — Private evidence và video-required integrated-v2

Status: Proposed. Date:30/10/2026. Owner đề xuất: Product + Fulfillment + Shipping + Platform.

## Context

EPIC10 để video phase2/theo policy, nhưng schema và consumer Shipping v2 hiện yêu cầu nonempty video key, checksum và proof fields. Video upload trực tiếp có thể bị overwrite trước URL hết hạn; MIME/checksum browser không đáng tin. Care có media module nhưng không phải owner packing evidence.

## Decision

Domain hỗ trợ policy OPTIONAL/VIDEO_REQUIRED; cohort integrated-v2 pin VIDEO_REQUIRED. Optional/no-video shipping không bật cho tới contract/consumer mới được ratify. Video verify actual size/type/checksum/scan, promote hoặc pin immutable final object; event chứa key của final object. Metadata/ownership/retention thuộc Fulfillment. Bucket private; URL GET purpose-bound ≤15 phút sau ABAC+local access audit.

## Alternatives

Empty key/fake checksum để qua schema sai business evidence và consumer reject. Dùng public URL phá NFR09. Dùng Care DB/bucket credentials chung vi phạm ownership. Verify mutable object key rồi upload overwrite tạo evidence substitution. Reject.

## Consequences

MVP integrated cần thêm media infrastructure/UX dù EPIC core có thể demo no-video riêng. Product/Shipping phải sign-off scope này. Presigned GET revocation không tức thời; TTL giới hạn hoặc authenticated proxy khi yêu cầu cao hơn. FUL-T18–25 kiểm upload/overwrite/ACL/cleanup/audit.
