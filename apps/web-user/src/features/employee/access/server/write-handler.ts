import { NextRequest, NextResponse } from "next/server";
import { getTrustedAccessToken } from "@/lib/auth/server-credential";
import { parseAdminAccessWritePath } from "../validation";

/**
 * F-47: the administration workspaces gain real mutations. No service
 * implements a write projection for the console, so in OIDC mode the BFF
 * forwards each action to the identity admin APIs with the caller's staff
 * token; identity's own @PreAuthorize guards (ROLE_MANAGE, PERMISSION_MANAGE,
 * ROLE_ASSIGN, USER_PASSWORD_RESET, USER_SESSION_REVOKE, EMPLOYEE_INVITE)
 * stay the authority and their errors are propagated, not hidden.
 *
 * Deliberately NOT offered: toggling a role's isEnabled flag. The backend
 * routes PATCH isEnabled=false through its soft-delete end state (F-48), so
 * presenting it as a reversible disable would be a lie; the request is
 * rejected explicitly instead of being silently dropped.
 */

const failure = (status: number, code: string, message: string) =>
  NextResponse.json({ code, message, requestId: `BFF-ACCESS-WRITE-${crypto.randomUUID()}`, errors: [] }, { status, headers: { "Cache-Control": "no-store" } });

class UpstreamWriteError extends Error {
  constructor(readonly status: number, readonly code: string, message: string) {
    super(message);
  }
}

type Rec = Record<string, unknown>;
const rec = (v: unknown): Rec => (v && typeof v === "object" && !Array.isArray(v) ? (v as Rec) : {});
const str = (v: unknown): string => (typeof v === "string" ? v : "");

interface Client {
  request(method: string, path: string, body?: unknown): Promise<{ status: number; data: unknown }>;
  get(path: string): Promise<unknown>;
}

function makeClient(base: string, token: string): Client {
  async function request(method: string, path: string, body?: unknown) {
    let response: Response;
    try {
      response = await fetch(`${base}${path}`, {
        method,
        headers: { Accept: "application/json", "Content-Type": "application/json", Authorization: `Bearer ${token}` },
        body: body === undefined ? undefined : JSON.stringify(body),
        cache: "no-store",
        redirect: "manual",
        signal: AbortSignal.timeout(10000),
      });
    } catch {
      throw new UpstreamWriteError(503, "ADMIN_ACCESS_UNAVAILABLE", "Dịch vụ quản trị truy cập chưa sẵn sàng.");
    }
    const payload: unknown = await response.json().catch(() => null);
    const envelope = rec(payload);
    if (!response.ok) {
      throw new UpstreamWriteError(
        response.status,
        str(envelope.code) || "ADMIN_ACCESS_ERROR",
        str(envelope.message) || "Không thể xử lý yêu cầu quản trị truy cập.",
      );
    }
    return { status: response.status, data: "data" in envelope ? envelope.data : payload };
  }
  return {
    request,
    async get(path: string) {
      return (await request("GET", path)).data;
    },
  };
}

async function resolveRoleId(client: Client, roleCode: string): Promise<string> {
  const roles = await client.get("/admin/roles");
  const match = (Array.isArray(roles) ? roles : []).map((r) => rec(r)).find((r) => str(r.roleCode) === roleCode);
  if (!match || !str(match.roleId)) throw new UpstreamWriteError(404, "NOT_FOUND", "Không tìm thấy Role cần thao tác.");
  return str(match.roleId);
}

const ROLE_CODE = /^[A-Z][A-Z0-9_]{1,63}$/u;
const SCOPE_TYPES = new Set(["GLOBAL", "WAREHOUSE", "DEPARTMENT"]);

function invalid(message: string): UpstreamWriteError {
  return new UpstreamWriteError(400, "INVALID_ARGUMENT", message);
}

function textField(body: Rec, field: string, max: number): string {
  const value = str(body[field]).trim();
  if (!value || value.length > max) throw invalid(`Trường ${field} không hợp lệ.`);
  return value;
}

async function readBody(request: NextRequest): Promise<Rec> {
  const payload: unknown = await request.json().catch(() => null);
  if (!payload || typeof payload !== "object" || Array.isArray(payload)) throw invalid("Nội dung yêu cầu không hợp lệ.");
  return payload as Rec;
}

const ok = (status: number, data: unknown) => NextResponse.json(data, { status, headers: { "Cache-Control": "no-store" } });

export async function handleLiveAdminAccessWrite(request: NextRequest, path: string[]) {
  const route = parseAdminAccessWritePath(path, request.method);
  if (route.kind === "invalid") return failure(404, "NOT_FOUND", "Đường dẫn quản trị không tồn tại.");
  const token = getTrustedAccessToken(request);
  if (!token) return failure(401, "AUTH_REQUIRED", "Vui lòng đăng nhập bằng tài khoản nhân viên.");
  const base = process.env.IDENTITY_UPSTREAM_URL?.replace(/\/$/u, "");
  if (!base) return failure(503, "ADMIN_ACCESS_UNAVAILABLE", "Dịch vụ quản trị truy cập chưa sẵn sàng.");
  const client = makeClient(base, token);
  try {
    switch (route.kind) {
      case "create-role": {
        const body = await readBody(request);
        const roleCode = textField(body, "roleCode", 64);
        if (!ROLE_CODE.test(roleCode)) throw invalid("Mã Role phải viết hoa, bắt đầu bằng chữ cái và chỉ gồm chữ, số, dấu gạch dưới.");
        const roleName = textField(body, "roleName", 120);
        const description = str(body.description).trim().slice(0, 500);
        const result = await client.request("POST", "/admin/roles", { roleCode, roleName, description });
        return ok(result.status, { roleId: str(result.data) });
      }
      case "update-role": {
        const body = await readBody(request);
        if ("isEnabled" in body) throw invalid("Vô hiệu hóa Role chưa được hỗ trợ: backend hiện biến thao tác này thành xóa mềm không khôi phục được (F-48).");
        const patch: Rec = {};
        if ("roleName" in body) patch.roleName = textField(body, "roleName", 120);
        if ("description" in body) patch.description = str(body.description).trim().slice(0, 500);
        if (!Object.keys(patch).length) throw invalid("Không có trường nào để cập nhật.");
        const roleId = await resolveRoleId(client, route.roleCode);
        const result = await client.request("PATCH", `/admin/roles/${encodeURIComponent(roleId)}`, patch);
        return ok(result.status, { updated: true });
      }
      case "delete-role": {
        const roleId = await resolveRoleId(client, route.roleCode);
        const result = await client.request("DELETE", `/admin/roles/${encodeURIComponent(roleId)}`);
        return ok(result.status, { deleted: true });
      }
      case "replace-role-permissions": {
        const body = await readBody(request);
        const raw = Array.isArray(body.permissions) ? body.permissions : null;
        if (!raw) throw invalid("Danh sách Permission không hợp lệ.");
        const permissions = [...new Set(raw.map((c) => str(c).trim()).filter(Boolean))];
        if (permissions.length > 200 || permissions.some((c) => !ROLE_CODE.test(c))) throw invalid("Danh sách Permission chứa mã không hợp lệ.");
        const roleId = await resolveRoleId(client, route.roleCode);
        const result = await client.request("PUT", `/admin/roles/${encodeURIComponent(roleId)}/permissions`, { permissions });
        return ok(result.status, { updated: true, permissionCount: permissions.length });
      }
      case "assign-role": {
        const body = await readBody(request);
        const roleCode = textField(body, "roleCode", 64);
        const scopeType = str(body.scopeType).trim() || "GLOBAL";
        if (!SCOPE_TYPES.has(scopeType)) throw invalid("Loại phạm vi không hợp lệ.");
        const scopeId = str(body.scopeId).trim();
        if (scopeType !== "GLOBAL" && !scopeId) throw invalid("Phạm vi này cần mã phạm vi (scopeId).");
        const roleId = await resolveRoleId(client, roleCode);
        const payload: Rec = { principalId: route.employeeId, roleId, scopeType, scopeId: scopeType === "GLOBAL" ? null : scopeId };
        if (str(body.validFrom)) payload.validFrom = str(body.validFrom);
        if (str(body.validUntil)) payload.validUntil = str(body.validUntil);
        if (str(body.reason).trim()) payload.reason = str(body.reason).trim().slice(0, 500);
        const result = await client.request("POST", "/admin/role-assignments", payload);
        return ok(result.status, { assignmentId: str(result.data) });
      }
      case "revoke-assignment": {
        const result = await client.request("DELETE", `/admin/role-assignments/${encodeURIComponent(route.assignmentId)}`);
        return ok(result.status, { revoked: true });
      }
      case "reset-password": {
        const body = await readBody(request);
        const newPassword = typeof body.newPassword === "string" ? body.newPassword : "";
        if (newPassword.length < 8) throw invalid("Mật khẩu mới phải có ít nhất 8 ký tự.");
        const result = await client.request("POST", `/admin/users/${encodeURIComponent(route.employeeId)}/password/reset`, {
          newPassword,
          temporary: body.temporary !== false,
        });
        return ok(result.status, { reset: true });
      }
      case "revoke-sessions": {
        const result = await client.request("POST", `/admin/users/${encodeURIComponent(route.employeeId)}/sessions/revoke`);
        return ok(result.status, { revoked: true });
      }
      case "create-invitation": {
        const body: Rec = await readBody(request).catch((): Rec => ({}));
        const validDays = typeof body.validDays === "number" && Number.isFinite(body.validDays) ? Math.trunc(body.validDays) : 7;
        if (validDays < 1 || validDays > 30) throw invalid("Số ngày hiệu lực của thư mời phải từ 1 đến 30.");
        const result = await client.request("POST", "/admin/employee-invitations", { validDays });
        const data = rec(result.data);
        return ok(result.status, {
          invitationId: str(data.invitationId),
          principalId: str(data.principalId),
          status: str(data.status),
          expiresAt: str(data.expiresAt),
        });
      }
      case "revoke-invitation": {
        const result = await client.request("DELETE", `/admin/employee-invitations/${encodeURIComponent(route.invitationId)}`);
        return ok(result.status, { revoked: true });
      }
    }
  } catch (cause) {
    if (cause instanceof UpstreamWriteError) {
      if (cause.status === 401) return failure(401, "AUTH_REQUIRED", "Phiên đăng nhập không hợp lệ hoặc đã hết hạn.");
      if (cause.status === 403) return failure(403, cause.code || "PERMISSION_FORBIDDEN", cause.message || "Bạn chưa được cấp quyền thực hiện thao tác này.");
      if (cause.status === 404) return failure(404, cause.code || "NOT_FOUND", cause.message || "Không tìm thấy dữ liệu cần thao tác.");
      if (cause.status === 400 || cause.status === 409 || cause.status === 422) return failure(cause.status, cause.code, cause.message);
      return failure(503, "ADMIN_ACCESS_UNAVAILABLE", "Dịch vụ quản trị truy cập chưa sẵn sàng.");
    }
    return failure(503, "ADMIN_ACCESS_UNAVAILABLE", "Dịch vụ quản trị truy cập chưa sẵn sàng.");
  }
}
