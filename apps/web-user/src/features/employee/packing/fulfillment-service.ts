import { FulfillmentError, type FulfillmentPage, type FulfillmentHistoryPage, type FulfillmentTask, type PackingMedia, type PendingPackingCommand } from "./fulfillment-types";

const messages: Record<string, string> = {
  VERSION_MISMATCH: "Dữ liệu đã thay đổi. Hãy tải lại task và kiểm tra trước khi thao tác tiếp.",
  FORBIDDEN: "Bạn không có quyền thao tác tại kho này.",
  FULFILLMENT_ADMIN_REQUIRED: "Tài khoản cần quyền ADMIN tại kho hoặc toàn hệ thống để đóng gói.",
  TASK_NOT_ASSIGNED: "Task chưa được giao cho bạn hoặc chưa bắt đầu.",
  TASK_ALREADY_ASSIGNED: "Task đã được người khác nhận. Hãy tải lại danh sách.",
  MEDIA_NOT_VERIFIED: "Video chưa được xác minh. Hãy chờ hoặc kiểm tra lại trạng thái video.",
  FULFILLMENT_UPSTREAM_UNAVAILABLE: "Chưa kết nối được dịch vụ đóng gói. Kết quả thao tác có thể chưa rõ; giữ nguyên yêu cầu khi thử lại.",
  AUTH_REQUIRED: "Phiên đăng nhập đã hết hạn. Vui lòng đăng nhập lại.",
};
async function read<T>(response: Response): Promise<T> {
  const payload = await response.json().catch(() => ({}));
  if (!response.ok) throw new FulfillmentError(response.status, payload.code ?? "UPSTREAM_ERROR", messages[payload.code] ?? `Không thể hoàn tất thao tác (${payload.code ?? response.status}).`);
  return payload as T;
}
export async function fulfillmentRequest<T>(path: string, init: RequestInit = {}): Promise<T> {
  try { return await read<T>(await fetch(`/api/admin/fulfillment/${path}`, { ...init, credentials: "same-origin", cache: "no-store" })); }
  catch (error) {
    if (error instanceof FulfillmentError || (error instanceof DOMException && error.name === "AbortError")) throw error;
    throw new FulfillmentError(503, "FULFILLMENT_UPSTREAM_UNAVAILABLE", messages.FULFILLMENT_UPSTREAM_UNAVAILABLE);
  }
}
export const fulfillmentService = {
  queue(query: URLSearchParams) { return fulfillmentRequest<FulfillmentPage>(`tasks?${query}`); },
  task(id: string) { return fulfillmentRequest<FulfillmentTask>(`tasks/${id}`); },
  history(id: string, cursor?: string) { return fulfillmentRequest<FulfillmentHistoryPage>(`tasks/${id}/history?${new URLSearchParams({ limit: "25", ...(cursor ? { cursor } : {}) })}`); },
  command(command: PendingPackingCommand) { return fulfillmentRequest<unknown>(`tasks/${command.taskId}/${command.action}`, {
    method: command.method, headers: { "Content-Type": "application/json", "If-Match": `"${command.version}"`, "Idempotency-Key": command.key }, body: JSON.stringify(command.body),
  }); },
  media(id: string, signal?: AbortSignal) { return fulfillmentRequest<PackingMedia>(`media/${id}`, { signal }); },
  completeMedia(id: string, key: string) { return fulfillmentRequest<PackingMedia>(`media/${id}/complete`, { method: "POST", headers: { "Content-Type": "application/json", "Idempotency-Key": key }, body: "{}" }); },
  upload(task: FulfillmentTask, file: File, hash: string, key: string) { return fulfillmentRequest<PackingMedia>(`tasks/${task.task_id}/video`, {
    method: "POST", headers: { "Content-Type": file.type, "X-Video-Size": String(file.size), "X-Video-Sha256": hash, "X-Package-ID": task.package.package_id, "Idempotency-Key": key }, body: file,
  }); },
};
