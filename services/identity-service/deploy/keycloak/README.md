# Keycloak Deployment

This directory contains the configurations and templates for setting up Keycloak for the Identity Service.

## Components
- `realm-export.template.json`: A baseline template to import the `omama` realm into Keycloak. It configures:
  - Frontend client (`frontend-web`) with PKCE enabled.
  - Backend admin client (`identity-service-admin`) with `client_credentials` grant type (Service Account).
  - Common roles (`CUSTOMER`, `EMPLOYEE`, `IDENTITY_ADMIN`).
  - Required audiences (`omama-api`) via `audience-scope`.
  - Default Admin User (`admin@omama.vn` / `admin`).

## Setup
1. Start Keycloak: `docker run -p 8080:8080 -e KEYCLOAK_ADMIN=admin -e KEYCLOAK_ADMIN_PASSWORD=admin quay.io/keycloak/keycloak:24.0.1 start-dev`
2. Import the `realm-export.template.json` in the Keycloak admin console or via the import volume script.
3. Generate a client secret for `identity-service-admin` and place it in the Identity Service's environment variable `KEYCLOAK_CLIENT_SECRET`.
4. Make sure to assign the `realm-management` client roles (like `manage-users`) to the `identity-service-admin` service account so it can manage principals.
