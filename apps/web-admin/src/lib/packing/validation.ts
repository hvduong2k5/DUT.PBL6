import type { PackingChecklistUpdateInput, PackingTaskCompleteInput } from "./types";

const record = (value: unknown): value is Record<string, unknown> => Boolean(value) && typeof value === "object" && !Array.isArray(value);
const text = (value: unknown) => typeof value === "string" ? value.trim() : "";

export function parseAdminPackingPath(path: string[]) {
  if (path.length === 1 && path[0] === "tasks") return { kind: "task-list" as const };
  if (path.length === 2 && path[0] === "tasks" && path[1]) return { kind: "task-detail" as const, taskId: path[1] };
  if (path.length === 2 && path[0] === "workbench" && path[1] === "tasks") return { kind: "workbench-list" as const };
  if (path.length === 3 && path[0] === "workbench" && path[1] === "tasks" && path[2]) return { kind: "workbench-detail" as const, taskId: path[2] };
  if (path.length === 5 && path[0] === "workbench" && path[1] === "tasks" && path[2] && path[3] === "checklist" && path[4]) return { kind: "checklist-update" as const, taskId: path[2], itemId: path[4] };
  if (path.length === 4 && path[0] === "workbench" && path[1] === "tasks" && path[2] && path[3] === "complete") return { kind: "task-complete" as const, taskId: path[2] };
  return { kind: "invalid" as const };
}

function commonMutation(value: unknown) {
  if (!record(value)) return { source: undefined, errors: [{ field: "body", message: "Dữ liệu thao tác không hợp lệ." }] };
  const expectedRevision = Number(value.expectedRevision);
  const idempotencyKey = text(value.idempotencyKey);
  const errors: Array<{ field: string; message: string }> = [];
  if (!Number.isInteger(expectedRevision) || expectedRevision < 1) errors.push({ field: "expectedRevision", message: "Revision không hợp lệ; hãy tải lại công việc." });
  if (idempotencyKey.length < 16 || idempotencyKey.length > 160) errors.push({ field: "idempotencyKey", message: "Mã chống gửi trùng không hợp lệ." });
  return { source: value, expectedRevision, idempotencyKey, errors };
}

export function validatePackingChecklistUpdate(value: unknown): { ok: true; data: PackingChecklistUpdateInput } | { ok: false; errors: Array<{ field: string; message: string }> } {
  const parsed = commonMutation(value);
  if (!parsed.source) return { ok: false, errors: parsed.errors };
  const state = text(parsed.source.state) as PackingChecklistUpdateInput["state"];
  const note = text(parsed.source.note);
  if (state !== "PASSED" && state !== "FAILED") parsed.errors.push({ field: "state", message: "Kết quả checklist phải là Đạt hoặc Không đạt." });
  if (state === "FAILED" && note.length < 5) parsed.errors.push({ field: "note", message: "Khi không đạt, ghi chú cần ít nhất 5 ký tự." });
  if (note.length > 500) parsed.errors.push({ field: "note", message: "Ghi chú không được vượt quá 500 ký tự." });
  if (parsed.errors.length) return { ok: false, errors: parsed.errors };
  return { ok: true, data: { state, note: note || undefined, expectedRevision: parsed.expectedRevision!, idempotencyKey: parsed.idempotencyKey! } };
}

export function validatePackingTaskComplete(value: unknown): { ok: true; data: PackingTaskCompleteInput } | { ok: false; errors: Array<{ field: string; message: string }> } {
  const parsed = commonMutation(value);
  if (parsed.errors.length) return { ok: false, errors: parsed.errors };
  return { ok: true, data: { expectedRevision: parsed.expectedRevision!, idempotencyKey: parsed.idempotencyKey! } };
}
