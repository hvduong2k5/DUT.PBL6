import {
  AcceptedResult,
  AuthApiError,
  AuthenticatedSession,
  RegistrationConfirmation,
  RecoveryVerification
} from "@/lib/auth/types";
import { IDENTITY_API_BASE_URL } from "@/services/identity-service";

type RequestOptions = Omit<RequestInit, "body"> & {
  body?: Record<string, unknown>;
};

function getDevelopmentScenario(): string | undefined {
  if (process.env.NODE_ENV !== "development" || typeof window === "undefined") {
    return undefined;
  }

  const scenario = new URLSearchParams(window.location.search).get("mockScenario") ?? undefined;
  return scenario && /^[a-z0-9-]+$/u.test(scenario) ? scenario : undefined;
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const headers = new Headers(options.headers);
  const scenario = getDevelopmentScenario();

  if (options.body) {
    headers.set("Content-Type", "application/json");
  }
  if (scenario) {
    headers.set("X-Mock-Scenario", scenario);
  }

  const response = await fetch(`/api/auth${path}`, {
    ...options,
    headers,
    body: options.body ? JSON.stringify(options.body) : undefined,
    credentials: "include",
    cache: "no-store"
  });

  if (response.status === 204) {
    if (!response.ok) {
      throw new AuthApiError(response.status, { code: "UNKNOWN_ERROR", message: "Yêu cầu thất bại." });
    }
    return undefined as T;
  }

  const contentType = response.headers.get("content-type") ?? "";
  const payload = contentType.includes("application/json") ? await response.json() : {};

  if (!response.ok) {
    throw new AuthApiError(response.status, payload);
  }

  return payload as T;
}

export async function registerCustomer(input: { firstName: string; lastName: string; email: string; rawPassword: string }) {
  const idempotencyKey = crypto.randomUUID();
  const response = await fetch(`${IDENTITY_API_BASE_URL}/public/customer-registrations`, {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json", Accept: "application/json", "Idempotency-Key": idempotencyKey },
    body: JSON.stringify({ ...input, idempotencyKey, requestFingerprint: input.email.trim().toLocaleLowerCase() })
  });
  const payload = (response.headers.get("content-type") ?? "").includes("application/json") ? await response.json() : {};
  if (!response.ok) throw new AuthApiError(response.status, payload);
  return payload;
}

export const authService = {
  me() { return request<AuthenticatedSession>("/me"); },
  logout() { return request<{ logoutUrl?: string }>("/logout", { method: "POST" }); },
  refresh() { return request<{ refreshed: true }>("/refresh", { method: "POST" }); },
  register(input: { fullName: string; phoneNumber: string; email?: string; password: string }) { return request<AuthenticatedSession>("/register", { method: "POST", body: input }); },
  login(input: { phoneNumber: string; password: string; rememberMe: boolean }) { return request<AuthenticatedSession>("/login", { method: "POST", body: input }); },
  confirmRegistration(verificationToken: string) { return request<RegistrationConfirmation>("/registration-verifications/confirm", { method: "POST", body: { verificationToken } }); },
  resendRegistration(verificationId: string) { return request<AcceptedResult>(`/registration-verifications/${encodeURIComponent(verificationId)}/resend`, { method: "POST" }); },
  requestRecovery(email: string) {
    return request<AcceptedResult>("/recovery-requests", { method: "POST", body: { email } });
  },
  verifyRecovery(recoveryToken: string) {
    return request<RecoveryVerification>("/recovery-requests/verify", {
      method: "POST",
      body: { recoveryToken }
    });
  },
  resetPassword(resetProof: string, newPassword: string) {
    return request<void>("/recovery-requests/reset", {
      method: "POST",
      body: { resetProof, newPassword }
    });
  }
};
