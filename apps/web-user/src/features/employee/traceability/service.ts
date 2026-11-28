import { TraceApiError, type TraceBatchContext, type TraceBinding, type TraceContent, type TraceProfile, type TraceProfilePage } from "./types";

async function request<T>(path: string, method = "GET", body?: unknown, key?: string): Promise<T> {
  const response = await fetch(`/api/admin/trace/${path}`, { method, cache: "no-store", credentials: "include", headers: { Accept: "application/json", ...(body ? { "Content-Type": "application/json", "Idempotency-Key": key! } : {}) }, body: body ? JSON.stringify(body) : undefined });
  const payload = await response.json();
  if (!response.ok) {
    const message = response.status === 409 ? "Hồ sơ hoặc nguồn lô đã thay đổi. Tải lại; nếu nguồn thay đổi sau duyệt, hãy lập bản sửa đổi để duyệt lại."
      : response.status === 403 ? "Bạn không có quyền thao tác hồ sơ trong kho này."
      : response.status === 401 ? "Phiên đăng nhập đã hết hạn. Vui lòng đăng nhập lại."
      : response.status >= 500 ? "Chưa thể kết nối hồ sơ hoặc dịch vụ nguồn. Bạn có thể thử lại; phiếu nhập kho vẫn giữ nguyên."
      : payload.message || "Thông tin hồ sơ chưa hợp lệ. Hãy kiểm tra chứng nhận và các trường bắt buộc.";
    throw new TraceApiError(response.status, payload.code ?? "TRACEABILITY_ERROR", message);
  }
  return payload as T;
}
export const traceabilityService = {
  list: (cursor = "") => request<TraceProfilePage>(`records?page_size=20${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`),
  get: (id: string) => request<TraceProfile>(`records/${id}`),
  context: (batch: string, sku: string) => request<TraceBatchContext>(`batches/${encodeURIComponent(batch)}/context?sku_code=${encodeURIComponent(sku)}`),
  lookup: (binding: TraceBinding) => request<{ items: Array<{ profile_id: string }>; total: number }>(`records/by-batch?${new URLSearchParams(binding as unknown as Record<string, string>)}`),
  create: (binding: TraceBinding, content: TraceContent, key: string) => request<TraceProfile>("records", "POST", { ...binding, content }, key),
  edit: (id: string, revision: number, expected_version: number, content: TraceContent, key: string) => request<TraceProfile>(`records/${id}/revisions/${revision}`, "PATCH", { expected_version, content }, key),
  action: (id: string, revision: number, action: string, expected_version: number, reason: string, key: string) => request<TraceProfile>(`records/${id}/${action === "suspend" || action === "archive" ? action : `revisions/${revision}/${action}`}`, "POST", { expected_version, ...(reason ? { reason } : {}) }, key),
  revision: (id: string, from_revision: number, expected_version: number, reason: string, key: string) => request<TraceProfile>(`records/${id}/revisions`, "POST", { from_revision, expected_version, ...(reason ? { reason } : {}) }, key),
};
