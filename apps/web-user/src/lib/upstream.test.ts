import { afterEach, describe, expect, it, vi } from "vitest";
import { customerCommerceUpstreamUrl, gatewayUpstreamUrl } from "./upstream";

afterEach(() => {
  vi.unstubAllEnvs();
});

describe("gatewayUpstreamUrl", () => {
  it("prefers an explicit API_GATEWAY_UPSTREAM_URL over every other source", () => {
    vi.stubEnv("WEB_AUTH_MODE", "oidc");
    vi.stubEnv("API_GATEWAY_UPSTREAM_URL", "https://gateway.test/api/v1/");
    vi.stubEnv("IDENTITY_UPSTREAM_URL", "https://identity.test/api/v1");
    vi.stubEnv("CUSTOMER_CORE_UPSTREAM_URL", "https://gateway.test/api/v1/customer-api");
    expect(gatewayUpstreamUrl()).toBe("https://gateway.test/api/v1");
  });

  it("uses IDENTITY_UPSTREAM_URL in identity-bff mode, as the catalog and registration routes always have", () => {
    vi.stubEnv("WEB_AUTH_MODE", "identity-bff");
    vi.stubEnv("IDENTITY_UPSTREAM_URL", "https://gateway.test/api/v1");
    vi.stubEnv("CUSTOMER_CORE_UPSTREAM_URL", "https://gateway.test/api/v1/customer-api");
    expect(gatewayUpstreamUrl()).toBe("https://gateway.test/api/v1");
  });

  it("derives the gateway root from the customer-api mount in oidc mode (the deployed F-12 shape)", () => {
    vi.stubEnv("WEB_AUTH_MODE", "oidc");
    vi.stubEnv("CUSTOMER_CORE_UPSTREAM_URL", "https://api.example.test/api/v1/customer-api");
    expect(gatewayUpstreamUrl()).toBe("https://api.example.test/api/v1");
  });

  it("passes a root-shaped CUSTOMER_CORE_UPSTREAM_URL through unchanged (local mock)", () => {
    vi.stubEnv("WEB_AUTH_MODE", "legacy");
    vi.stubEnv("CUSTOMER_CORE_UPSTREAM_URL", "http://127.0.0.1:4010/api/v1");
    expect(gatewayUpstreamUrl()).toBe("http://127.0.0.1:4010/api/v1");
  });

  it("falls back to the local mock default when nothing is configured", () => {
    vi.stubEnv("WEB_AUTH_MODE", "legacy");
    expect(gatewayUpstreamUrl()).toBe("http://127.0.0.1:4010/api/v1");
  });
});

describe("customerCommerceUpstreamUrl", () => {
  it("stays on the identity BFF mount in identity-bff mode, where the session id is translated", () => {
    vi.stubEnv("WEB_AUTH_MODE", "identity-bff");
    vi.stubEnv("IDENTITY_UPSTREAM_URL", "https://gateway.test/api/v1");
    vi.stubEnv("CUSTOMER_CORE_UPSTREAM_URL", "https://gateway.test/api/v1/customer-api");
    expect(customerCommerceUpstreamUrl()).toBe("https://gateway.test/api/v1/customer-api");
  });

  it("uses the gateway root in oidc mode, where services verify the JWT themselves (the deployed shape)", () => {
    vi.stubEnv("WEB_AUTH_MODE", "oidc");
    vi.stubEnv("CUSTOMER_CORE_UPSTREAM_URL", "https://api.example.test/api/v1/customer-api");
    expect(customerCommerceUpstreamUrl()).toBe("https://api.example.test/api/v1");
  });
});
