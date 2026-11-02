import { parseAdminSession, type AdminSession } from "@/lib/auth/types";

export async function getAdminSession(): Promise<AdminSession> {
  const response = await fetch("/api/admin/session", { cache: "no-store", credentials: "include" });
  const payload: unknown = await response.json();
  if (!response.ok) throw new Error("Không thể tải phiên nhân viên.");
  const session = parseAdminSession(payload);
  if (!session) throw new Error("Dữ liệu phân quyền nhân viên không hợp lệ.");
  return session;
}
