import axios from "axios";
import { v4 as uuidv4 } from "uuid";
import { apiClient, unwrapApiData } from "@/lib/api";
import { AcceptedResult, AuthApiError, RecoveryVerification } from "@/lib/auth/types";

export interface CustomerRegistrationInput {
  firstName: string;
  lastName: string;
  email: string;
  rawPassword: string;
}

function toAuthApiError(error: unknown): AuthApiError {
  if (axios.isAxiosError(error)) {
    const body = error.response?.data && typeof error.response.data === "object" ? error.response.data : {};
    return new AuthApiError(error.response?.status ?? 503, body);
  }
  return new AuthApiError(503, { code: "IDENTITY_UNAVAILABLE", message: "Không thể kết nối dịch vụ xác thực." });
}

export async function registerCustomer(formData: CustomerRegistrationInput) {
  const idempotencyKey = uuidv4();
  try {
    const response = await apiClient.post(
      "/public/customer-registrations",
      {
        ...formData,
        idempotencyKey,
        requestFingerprint: formData.email.trim().toLocaleLowerCase()
      },
      { headers: { "Idempotency-Key": idempotencyKey } }
    );
    return unwrapApiData<unknown>(response.data) ?? response.data;
  } catch (error) {
    throw toAuthApiError(error);
  }
}

export async function changePassword(currentPassword: string, newPassword: string) {
  try {
    const response = await apiClient.put("/users/me/password", { currentPassword, newPassword });
    return unwrapApiData<unknown>(response.data) ?? response.data;
  } catch (error) {
    throw toAuthApiError(error);
  }
}

type LegacyRequestOptions = Omit<RequestInit, "body"> & { body?: Record<string, unknown> };

async function legacyRequest<T>(path: string, options: LegacyRequestOptions = {}): Promise<T> {
  const headers = new Headers(options.headers);
  if (options.body) headers.set("Content-Type", "application/json");
  const response = await fetch(`/api/auth${path}`, {
    ...options,
    headers,
    body: options.body ? JSON.stringify(options.body) : undefined,
    credentials: "include",
    cache: "no-store"
  });
  if (response.status === 204) return undefined as T;
  const payload = (response.headers.get("content-type") ?? "").includes("application/json") ? await response.json() : {};
  if (!response.ok) throw new AuthApiError(response.status, payload);
  return payload as T;
}

// Existing recovery screens remain available until Identity publishes their replacement contract.
export const authService = {
  requestRecovery(email: string) {
    return legacyRequest<AcceptedResult>("/recovery-requests", { method: "POST", body: { email } });
  },
  verifyRecovery(recoveryToken: string) {
    return legacyRequest<RecoveryVerification>("/recovery-requests/verify", { method: "POST", body: { recoveryToken } });
  },
  resetPassword(resetProof: string, newPassword: string) {
    return legacyRequest<void>("/recovery-requests/reset", { method: "POST", body: { resetProof, newPassword } });
  }
};
