import {
  AddressInput,
  AddressList,
  CustomerAddress,
  CustomerApiError,
  CustomerProfile,
  ProfileInput
} from "@/lib/customer/types";

type RequestOptions = Omit<RequestInit, "body"> & { body?: object };

function getDevelopmentScenario(): string | undefined {
  if (process.env.NODE_ENV !== "development" || typeof window === "undefined") return undefined;
  const scenario = new URLSearchParams(window.location.search).get("mockScenario") ?? undefined;
  return scenario && /^[a-z0-9-]+$/u.test(scenario) ? scenario : undefined;
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const headers = new Headers(options.headers);
  const scenario = getDevelopmentScenario();
  if (options.body) headers.set("Content-Type", "application/json");
  if (scenario) headers.set("X-Mock-Scenario", scenario);

  const response = await fetch(`/api/customer${path}`, {
    ...options,
    headers,
    body: options.body ? JSON.stringify(options.body) : undefined,
    credentials: "include",
    cache: "no-store"
  });
  const contentType = response.headers.get("content-type") ?? "";
  const payload = contentType.includes("application/json") ? await response.json() : {};
  if (!response.ok) throw new CustomerApiError(response.status, payload);
  return payload as T;
}

export const customerService = {
  getProfile() {
    return request<CustomerProfile>("/profile");
  },
  updateProfile(input: ProfileInput, version: number) {
    return request<CustomerProfile>("/profile", {
      method: "PATCH",
      headers: { "If-Match": `\"profile-v${version}\"` },
      body: input
    });
  },
  getAddresses() {
    return request<AddressList>("/addresses");
  },
  createAddress(input: AddressInput) {
    return request<CustomerAddress>("/addresses", { method: "POST", body: input });
  },
  updateAddress(id: string, input: AddressInput, version: number) {
    return request<CustomerAddress>(`/addresses/${encodeURIComponent(id)}`, {
      method: "PATCH",
      headers: { "If-Match": `\"address-v${version}\"` },
      body: input
    });
  },
  deleteAddress(id: string, version: number) {
    return request<AddressList>(`/addresses/${encodeURIComponent(id)}`, {
      method: "DELETE",
      headers: { "If-Match": `\"address-v${version}\"` }
    });
  },
  setDefaultAddress(id: string, version: number) {
    return request<AddressList>(`/addresses/${encodeURIComponent(id)}/default`, {
      method: "PUT",
      headers: { "If-Match": `\"address-v${version}\"` }
    });
  }
};
