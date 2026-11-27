import { procurementRoute } from "./routes";
import { UUID_PATTERN, type RecordData } from "./forms";

type StoragePort = Pick<Storage, "getItem" | "setItem" | "removeItem">;
type Pending = { key: string; fingerprint: string };
export class CommandError extends Error {
  constructor(message: string, public status = 0, public pending = false, public requestId?: string, public code?: string) { super(message); }
}
function canonical(value: unknown): unknown {
  if (Array.isArray(value)) return value.map(canonical);
  if (value && typeof value === "object") return Object.fromEntries(Object.entries(value).sort(([a], [b]) => a.localeCompare(b)).map(([key, item]) => [key, canonical(item)]));
  return value;
}
export function commandStorageKey(actor: string, method: string, path: string) { return `procurement:command:v1:${actor}:${method}:${path}`; }

/** Persist only the key and a digest, never business data or authentication.
 * An uncertain attempt cannot silently become a new command after reload. */
export function createCommandRunner(storage: StoragePort, actor: string, fetcher: typeof fetch = fetch) {
  let busy = false;
  return async (method: string, path: string, payload: RecordData): Promise<RecordData> => {
    if (busy) throw new CommandError("Yêu cầu đang được xử lý.");
    if (!actor || !["POST", "PUT", "PATCH"].includes(method) || !procurementRoute(method, path)) throw new CommandError("Thao tác không được phép.");
    busy = true;
    try {
      const body = JSON.stringify(canonical(payload));
      const bytes = new TextEncoder().encode(body);
      if (bytes.length > 65536) throw new CommandError("Dữ liệu vượt giới hạn 64 KiB. Hãy giảm số dòng hoặc nội dung ghi chú.");
      const fingerprint = [...new Uint8Array(await crypto.subtle.digest("SHA-256", bytes))].map((n) => n.toString(16).padStart(2, "0")).join("");
      const slot = commandStorageKey(actor, method, path);
      let previous: Pending | undefined;
      try {
        const raw = storage.getItem(slot);
        if (raw !== null) {
          previous = JSON.parse(raw) as Pending;
          if (!previous || !/^[!-~]{16,128}$/u.test(previous.key) || !/^[0-9a-f]{64}$/u.test(previous.fingerprint)) throw new Error();
        }
      } catch { throw new CommandError("Không đọc được nhật ký retry trong phiên trình duyệt. Không gửi yêu cầu để tránh tạo trùng.", 0, true); }
      if (previous && previous.fingerprint !== fingerprint) throw new CommandError("Có yêu cầu trước chưa rõ kết quả. Hãy nhập lại đúng dữ liệu cũ hoặc liên hệ đối soát; không gửi dữ liệu mới bằng khóa khác.", 0, true, undefined, "RETRY_PAYLOAD_MISMATCH");
      const entry = previous ?? { key: `procurement-${crypto.randomUUID()}`, fingerprint };
      try { storage.setItem(slot, JSON.stringify(entry)); }
      catch { throw new CommandError("Không lưu được khóa retry. Cho phép session storage trước khi gửi yêu cầu."); }
      const clearOwnEntry = () => {
        const raw = storage.getItem(slot);
        if (raw === null) return;
        const current = JSON.parse(raw) as Pending;
        if (current.key === entry.key && current.fingerprint === entry.fingerprint) storage.removeItem(slot);
      };
      let response: Response;
      try {
        response = await fetcher(`/api/admin/procurement/${path}`, { method, headers: { "Content-Type": "application/json", "Idempotency-Key": entry.key }, body, credentials: "same-origin", cache: "no-store", redirect: "error", signal: AbortSignal.timeout(15000) });
      } catch { throw new CommandError("Chưa biết yêu cầu đã được ghi nhận hay chưa. Thử lại đúng dữ liệu với cùng khóa; không tạo lại đơn.", 0, true); }
      let data: RecordData;
      try {
        const parsed: unknown = await response.json();
        if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) throw new Error();
        data = parsed as RecordData;
      } catch { throw new CommandError("Phản hồi không hợp lệ; giữ khóa cũ để thử lại an toàn.", response.status, true); }
      if (!response.ok) {
        const uncertain = response.status >= 500 || Boolean(previous) || response.status === 408 || response.status === 429;
        if (!uncertain) { try { clearOwnEntry(); } catch { /* Retaining the key is safer than fabricating a new one. */ } }
        let message = response.status === 401 ? "Phiên đăng nhập hết hạn. Đăng nhập lại rồi thử lại." : response.status === 403 ? "Thiếu quyền hoặc phạm vi kho phù hợp." : response.status === 409 ? "Dữ liệu/phiên bản hoặc khóa retry xung đột. Tải lại dữ liệu trước khi thao tác mới." : "Yêu cầu chưa thành công. Kiểm tra dữ liệu và thông tin lỗi.";
        if ([400, 409, 422].includes(response.status) && typeof data.message === "string") message += " " + data.message.slice(0, 2000);
        throw new CommandError(message, response.status, uncertain, typeof data.requestId === "string" ? data.requestId : undefined, typeof data.code === "string" ? data.code : undefined);
      }
      if (![200, 201].includes(response.status) || typeof data.id !== "string" || !UUID_PATTERN.test(data.id)) throw new CommandError("Phản hồi thành công chưa có ID/trạng thái tài nguyên hợp lệ. Giữ khóa cũ để đối soát hoặc thử lại.", response.status, true);
      try { clearOwnEntry(); }
      catch { throw new CommandError("Máy chủ đã ghi nhận nhưng chưa xóa được nhật ký retry. Chưa tạo yêu cầu mới; hãy đối chiếu bằng ID kết quả.", response.status, true, undefined, "RETRY_JOURNAL_CLEANUP_FAILED"); }
      return data;
    } finally { busy = false; }
  };
}
