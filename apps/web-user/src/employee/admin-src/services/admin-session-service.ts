import { parseAdminSession, type AdminSession } from "@admin/lib/auth/types";

export class AdminSessionError extends Error {
  constructor(readonly status: number) { super("Unable to load employee session."); }
}

export async function getAdminSession(): Promise<AdminSession> {
  const response = await fetch("/api/admin/session", { cache: "no-store", credentials: "include" });
  const payload: unknown = await response.json();
  if (!response.ok) throw new AdminSessionError(response.status);
  const session = parseAdminSession(payload);
  if (!session) throw new Error("Dữ liệu phân quyền nhân viên không hợp lệ.");
  return session;
}
