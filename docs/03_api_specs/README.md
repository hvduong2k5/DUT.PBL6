# API Specifications

Thư mục này chứa các API contract theo vòng đời `candidate → reviewed → approved`.

## Authentication MVP

| Artefact | Trạng thái | Mục đích |
| --- | --- | --- |
| `authentication-mvp.openapi.yaml` | `0.1.0-candidate` | OpenAPI đề xuất từ UI implementation và Mockoon. |
| `AUTHENTICATION_MVP_CONTRACT_REVIEW.md` | `PROPOSED / NOT APPROVED` | Context, implementation findings và câu hỏi cần review. |

## Customer Profile & Address Book MVP

| Artefact | Trạng thái | Mục đích |
| --- | --- | --- |
| `customer-profile-mvp.openapi.yaml` | `0.1.0-candidate` | OpenAPI đề xuất từ UI đã duyệt, implementation và Mockoon. |
| `CUSTOMER_PROFILE_MVP_CONTRACT_REVIEW.md` | `PROPOSED / NOT APPROVED` | Context, implementation findings và câu hỏi cần review. |

## Shopping Cart MVP

| Artefact | Trạng thái | Mục đích |
| --- | --- | --- |
| `shopping-cart-mvp.openapi.yaml` | `0.1.0-candidate` | OpenAPI đề xuất từ UI đã duyệt, implementation và Mockoon. |
| `SHOPPING_CART_MVP_CONTRACT_REVIEW.md` | `PROPOSED / NOT APPROVED` | Ranh giới Cart/Checkout, implementation findings và câu hỏi cần review. |

Validate bằng Redocly recommended rules:

```powershell
npx @redocly/cli lint .\docs\03_api_specs\authentication-mvp.openapi.yaml
npx @redocly/cli lint .\docs\03_api_specs\customer-profile-mvp.openapi.yaml
npx @redocly/cli lint .\docs\03_api_specs\shopping-cart-mvp.openapi.yaml
```

Candidate không được coi là production contract cho đến khi Backend, Architecture và Security phê duyệt. Mockoon chỉ được đồng bộ theo thay đổi contract sau review.
