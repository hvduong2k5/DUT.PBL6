import axios from "axios";

export const API_BASE_URL = (process.env.NEXT_PUBLIC_API_URL || "http://localhost:8016/api/v1").replace(/\/$/u, "");

export const apiClient = axios.create({
  baseURL: API_BASE_URL,
  withCredentials: true,
  headers: {
    "Content-Type": "application/json"
  }
});

export function unwrapApiData<T>(payload: unknown): T | undefined {
  if (!payload || typeof payload !== "object" || Array.isArray(payload)) return undefined;
  const source = payload as Record<string, unknown>;
  if (source.success === false) return undefined;
  return ("data" in source ? source.data : source) as T;
}
