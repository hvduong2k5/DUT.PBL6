import { formatStrongVersionEtag, requestPublicJson } from "@/lib/api-contract";
import type { Profile, ShippingAddress } from "@/lib/api-contract";
import type { CheckoutProvince } from "@/lib/checkout/types";
import {
  mapMvpAddress,
  mapMvpAddressList,
  mapMvpProfile,
  toMvpAddressUpdate,
  toMvpAddressWrite,
  toMvpProfileUpdate
} from "@/lib/customer/mvp";
import type { AddressInput, ProfileInput } from "@/lib/customer/types";

function developmentHeaders(): Headers {
  const headers = new Headers();
  if (process.env.NODE_ENV !== "development" || typeof window === "undefined") return headers;
  const scenario = new URLSearchParams(window.location.search).get("mockScenario") ?? undefined;
  if (scenario && /^[a-z0-9-]+$/u.test(scenario)) headers.set("X-Mock-Scenario", scenario);
  return headers;
}

export const customerService = {
  async getProfile() {
    const response = await requestPublicJson<Profile>("/api/customer/profile", {
      credentials: "include",
      headers: developmentHeaders()
    });
    return mapMvpProfile(response.data, response.etag);
  },

  async updateProfile(input: ProfileInput, version: number) {
    const response = await requestPublicJson<Profile>("/api/customer/profile", {
      method: "PUT",
      credentials: "include",
      headers: developmentHeaders(),
      ifMatch: formatStrongVersionEtag(version),
      body: toMvpProfileUpdate(input)
    });
    return mapMvpProfile(response.data, response.etag);
  },

  async getAddresses() {
    const response = await requestPublicJson<ShippingAddress[]>("/api/customer/addresses", {
      credentials: "include",
      headers: developmentHeaders()
    });
    return mapMvpAddressList(response.data);
  },

  async getAddress(id: string) {
    const response = await requestPublicJson<ShippingAddress>(`/api/customer/addresses/${encodeURIComponent(id)}`, {
      credentials: "include",
      headers: developmentHeaders()
    });
    return mapMvpAddress(response.data, response.etag);
  },

  async getLocations() {
    const response = await requestPublicJson<{ provinces: CheckoutProvince[] }>("/api/customer/locations", {
      credentials: "include",
      headers: developmentHeaders()
    });
    return response.data;
  },

  async createAddress(input: AddressInput) {
    const response = await requestPublicJson<ShippingAddress>("/api/customer/addresses", {
      method: "POST",
      credentials: "include",
      headers: developmentHeaders(),
      body: toMvpAddressWrite(input)
    });
    return mapMvpAddress(response.data, response.etag);
  },

  async updateAddress(id: string, input: AddressInput, version: number) {
    const response = await requestPublicJson<ShippingAddress>(`/api/customer/addresses/${encodeURIComponent(id)}`, {
      method: "PUT",
      credentials: "include",
      headers: developmentHeaders(),
      ifMatch: formatStrongVersionEtag(version),
      body: toMvpAddressUpdate(input)
    });
    return mapMvpAddress(response.data, response.etag);
  },

  async deleteAddress(id: string, version: number) {
    const response = await requestPublicJson<ShippingAddress[]>(`/api/customer/addresses/${encodeURIComponent(id)}`, {
      method: "DELETE",
      credentials: "include",
      headers: developmentHeaders(),
      ifMatch: formatStrongVersionEtag(version)
    });
    return mapMvpAddressList(response.data);
  },

  async setDefaultAddress(id: string, version: number) {
    const response = await requestPublicJson<ShippingAddress[]>(`/api/customer/addresses/${encodeURIComponent(id)}/default`, {
      method: "PUT",
      credentials: "include",
      headers: developmentHeaders(),
      ifMatch: formatStrongVersionEtag(version)
    });
    return mapMvpAddressList(response.data);
  },

  async patchDefaultAddress(id: string, version: number) {
    const response = await requestPublicJson<ShippingAddress[]>(`/api/customer/addresses/${encodeURIComponent(id)}/default`, {
      method: "PATCH",
      credentials: "include",
      headers: developmentHeaders(),
      ifMatch: formatStrongVersionEtag(version)
    });
    return mapMvpAddressList(response.data);
  }
};
