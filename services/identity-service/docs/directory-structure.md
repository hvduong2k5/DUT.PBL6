# Cấu trúc thư mục chốt — identity-service

## 1. Quyết định kiến trúc
- Runtime: Java + Spring Boot.
- Database: PostgreSQL, schema migration bằng Flyway.
- Data access: MyBatis Mapper interface + XML SQL, không dùng JPA/Hibernate.
- Identity Provider: Keycloak. Keycloak sở hữu credential, đăng nhập, session, social identities, refresh token và ký JWT.
- identity-service sở hữu orchestration đăng ký/provisioning, mapping principal nội bộ, IAM nghiệp vụ riêng, quyết định quyền đặc biệt, audit và outbox event.
- Guest session và guest_order_grant thuộc Cart/Checkout và Order, không thuộc Identity.
- Không tạo bảng password hash, refresh token, login session hoặc recovery token trong identity_db khi Keycloak đang sở hữu các chức năng này.

## 2. Cây thư mục chuẩn
```text
services/identity-service/
├── README.md
├── build.gradle
├── settings.gradle
├── gradle.properties
├── gradlew
├── gradlew.bat
├── docs/
│   ├── architecture.md
│   ├── database-design.md
│   ├── implementation-plan.md
│   ├── api-contract-decisions.md
│   └── runbook.md
├── deploy/
│   └── keycloak/
│       ├── README.md
│       └── realm-export.template.json
└── src/
    ├── main/
    │   ├── java/com/hvduong/identity/
    │   │   ├── IdentityApplication.java
    │   │   ├── api/
    │   │   │   ├── registration/
    │   │   │   │   ├── RegistrationController.java
    │   │   │   │   └── dto/
    │   │   │   ├── account/
    │   │   │   │   ├── AccountLifecycleController.java
    │   │   │   │   └── dto/
    │   │   │   └── admin/
    │   │   │       ├── EmployeeInvitationController.java
    │   │   │       ├── IamRoleController.java
    │   │   │       ├── IamAssignmentController.java
    │   │   │       └── dto/
    │   │   ├── application/
    │   │   │   ├── registration/
    │   │   │   │   ├── RegisterPrincipalUseCase.java
    │   │   │   │   └── RegistrationOrchestrator.java
    │   │   │   ├── account/
    │   │   │   │   └── AccountLifecycleService.java
    │   │   │   ├── iam/
    │   │   │   │   ├── RoleManagementService.java
    │   │   │   │   └── RoleAssignmentService.java
    │   │   │   ├── authorization/
    │   │   │   │   └── SpecializedPermissionService.java
    │   │   │   └── audit/
    │   │   │       └── AuditService.java
    │   │   ├── domain/
    │   │   │   ├── principal/
    │   │   │   │   ├── IdentityPrincipal.java
    │   │   │   │   └── PrincipalStatus.java
    │   │   │   ├── iam/
    │   │   │   │   ├── IamRole.java
    │   │   │   │   ├── IamPermission.java
    │   │   │   │   └── IamRoleAssignment.java
    │   │   │   ├── authorization/
    │   │   │   │   ├── SpecializedPermissionPolicy.java
    │   │   │   │   └── AuthorizationDecision.java
    │   │   │   └── event/
    │   │   │       └── IdentityDomainEvent.java
    │   │   ├── infrastructure/
    │   │   │   ├── keycloak/
    │   │   │   │   ├── KeycloakAdminClient.java
    │   │   │   │   ├── KeycloakPrincipalMapper.java
    │   │   │   │   └── KeycloakExceptionTranslator.java
    │   │   │   ├── persistence/
    │   │   │   │   └── mybatis/
    │   │   │   │       ├── mapper/
    │   │   │   │       │   ├── IdentityPrincipalMapper.java
    │   │   │   │       │   ├── RegistrationOperationMapper.java
    │   │   │   │       │   ├── EmployeeInvitationMapper.java
    │   │   │   │       │   ├── IamRoleMapper.java
    │   │   │   │       │   ├── IamPermissionMapper.java
    │   │   │   │       │   ├── IamRoleAssignmentMapper.java
    │   │   │   │       │   ├── SpecializedPermissionPolicyMapper.java
    │   │   │   │       │   ├── AuthorizationDecisionMapper.java
    │   │   │   │       │   ├── IdentityAuditLogMapper.java
    │   │   │   │       │   └── OutboxEventMapper.java
    │   │   │   │       ├── row/
    │   │   │   │       │   └── IdentityPrincipalRow.java
    │   │   │   │       └── typehandler/
    │   │   │   ├── repository/
    │   │   │   │   ├── IdentityPrincipalRepository.java
    │   │   │   │   └── IamRepository.java
    │   │   │   ├── outbox/
    │   │   │   │   ├── OutboxEventWriter.java
    │   │   │   │   └── OutboxPublisher.java
    │   │   │   ├── kafka/
    │   │   │   │   ├── event/
    │   │   │   │   │   └── IdentityEventEnvelope.java
    │   │   │   │   └── IdentityEventPublisher.java
    │   │   │   └── grpc/
    │   │   │       ├── IdentityPermissionGrpcService.java
    │   │   │       └── GrpcAuditContextMapper.java
    │   │   ├── security/
    │   │   │   ├── SecurityConfig.java
    │   │   │   └── TrustedRequestContext.java
    │   │   ├── config/
    │   │   │   ├── KeycloakConfig.java
    │   │   │   ├── MyBatisConfig.java
    │   │   │   ├── KafkaConfig.java
    │   │   │   └── GrpcConfig.java
    │   │   └── common/
    │   │       ├── error/
    │   │       ├── idempotency/
    │   │       └── tracing/
    │   └── resources/
    │       ├── application.yml
    │       ├── application-local.yml
    │       ├── application-test.yml
    │       ├── db/migration/
    │       │   ├── V1__create_principals_and_provisioning.sql
    │       │   ├── V2__create_iam_role_permission_model.sql
    │       │   ├── V3__create_special_permission_policies.sql
    │       │   ├── V4__create_identity_audit_log.sql
    │       │   └── V5__create_outbox_events.sql
    │       └── mapper/
    │           ├── principal/IdentityPrincipalMapper.xml
    │           ├── registration/RegistrationOperationMapper.xml
    │           ├── iam/IamRoleMapper.xml
    │           ├── iam/IamPermissionMapper.xml
    │           ├── iam/IamRoleAssignmentMapper.xml
    │           ├── authorization/SpecializedPermissionPolicyMapper.xml
    │           ├── authorization/AuthorizationDecisionMapper.xml
    │           ├── audit/IdentityAuditLogMapper.xml
    │           └── outbox/OutboxEventMapper.xml
    └── test/
        ├── java/com/hvduong/identity/
        │   ├── application/registration/
        │   ├── application/iam/
        │   ├── application/authorization/
        │   ├── infrastructure/keycloak/
        │   ├── infrastructure/persistence/mybatis/
        │   ├── infrastructure/outbox/
        │   ├── infrastructure/grpc/
        │   └── security/
        └── resources/
            └── application-test.yml
```

*Đây là cấu trúc đích. Chỉ tạo class/Mapper khi use case tương ứng sắp được triển khai; không cần sinh hàng loạt class rỗng chỉ để khớp cây thư mục.*

## 3. Trách nhiệm các lớp
- **api:** REST controller và request/response DTO; không chứa SQL.
- **application:** điều phối use case, transaction và port/cổng tích hợp.
- **domain:** entity, enum, policy và quy tắc IAM; không phụ thuộc Spring, MyBatis hoặc Keycloak SDK.
- **infrastructure/keycloak:** gọi Keycloak Admin API và ánh xạ lỗi; không lưu credential.
- **infrastructure/persistence/mybatis/mapper:** interface MyBatis và Row projection.
- **resources/mapper:** các XML SQL, chia theo aggregate/domain.
- **infrastructure/outbox và infrastructure/kafka:** lưu event cùng transaction nghiệp vụ, rồi publish sau commit; consumer downstream phải idempotent.
- **infrastructure/grpc:** chỉ các RPC thực sự được định nghĩa trong shared proto chuẩn.
- **security:** bảo vệ endpoint và xác thực trusted context; không tin các X-User-* header đi trực tiếp từ Internet.
- **config:** cấu hình framework/client; không đặt business logic ở đây.

## 4. Cấu hình MyBatis chuẩn
```yaml
mybatis:
  mapper-locations: classpath:mapper/**/*.xml
  configuration:
    map-underscore-to-camel-case: true
```
Trong `IdentityApplication`, dùng `@MapperScan("com.hvduong.identity.infrastructure.persistence.mybatis.mapper")`. Dùng XML cho truy vấn phức tạp và interface Mapper cho contract gọi SQL.

## 5. JWKS và gRPC
Khuyến nghị mặc định: Gateway đọc OIDC discovery/JWKS trực tiếp từ Keycloak. Không cần RPC lấy JWKS riêng trừ khi cần giữ tương thích với Gateway/contract cũ; nếu bỏ RPC thì cập nhật shared proto, tài liệu API và service boundary cùng một PR.
Giữ RPC CheckSpecializedPermission (dùng đúng tên thực trong shared proto) nếu Order/Finance cần quyết định quyền động. Không tự tạo ValidateToken hay CheckPermission nếu chúng không nằm trong proto chuẩn.

## 6. Những gì không lưu trong identity_db
Không tạo bảng password hash, refresh token, login session, social credential hoặc recovery token khi Keycloak sở hữu xác thực. Không lưu guest_session hoặc guest_order_grant; Cart/Checkout và Order sở hữu các cơ chế Guest. Profile sở hữu dữ liệu hồ sơ; Order sở hữu đơn hàng.

## 7. Điều kiện trước khi merge
1. Chuẩn hóa tên service identity-service trong thư mục, deployment, service discovery, logging và topic.
2. Chốt đăng nhập theo OIDC Authorization Code + PKCE; không tự nhận password rồi tự phát JWT.
3. Chốt sub/user_id, issuer, audience, token TTL và chính sách revoke/disable.
4. Đồng bộ OpenAPI, shared proto và Kafka event schema; không code dựa trên RPC chỉ xuất hiện trong tài liệu cũ.
5. Chạy Flyway trên PostgreSQL mới và test MyBatis Mapper bằng PostgreSQL integration tests.
6. Không commit client secret, private key hoặc credential vào YAML/realm export.

## 8. Ba nguyên tắc không thay đổi
1. Keycloak là nguồn xác thực chính. Không xây thêm hệ thống lưu password, refresh token và login session song song trong Identity DB.
2. MyBatis là lớp truy cập database. Mỗi Mapper interface có XML SQL tương ứng khi cần; Flyway quản lý schema. Không dùng JPA/Hibernate song song.
3. Identity không sở hữu dữ liệu nghiệp vụ của service khác. Profile giữ hồ sơ; Order giữ đơn hàng và quyền Guest theo đơn; Checkout/Cart quản lý phiên Guest. Identity chỉ giữ mapping principal, IAM nghiệp vụ riêng, chính sách quyền đặc biệt, audit và outbox.
