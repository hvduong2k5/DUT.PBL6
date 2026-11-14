"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useEmployeeSession } from "@/components/auth/employee-session-provider";
import type { EmployeeAccessDetail, EmployeeAccessList, EmployeeAccessSummary, EmployeeAccountStatus, EmployeeRoleAssignment, InvitationCreated } from "@/features/employee/access/types";
import { AdminAccessApiError } from "@/features/employee/access/types";
import { adminAccessService } from "@/features/employee/access/service";

const DATE_TIME = new Intl.DateTimeFormat("vi-VN", { day: "2-digit", month: "2-digit", year: "numeric", hour: "2-digit", minute: "2-digit" });

function AccountStatus({ row }: { row: Pick<EmployeeAccessSummary, "status" | "statusLabel"> }) {
  return <span className={`admin-status account-${row.status.toLowerCase()}`}>{row.statusLabel}</span>;
}

function Modal({ title, subtitle, onClose, children }: { title: string; subtitle?: string; onClose: () => void; children: React.ReactNode }) {
  return <div className="admin-modal-backdrop" role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget) onClose(); }}><section className="admin-modal wide" role="dialog" aria-modal="true" aria-label={title}><header><div><h2>{title}</h2>{subtitle ? <p>{subtitle}</p> : null}</div><button type="button" onClick={onClose} aria-label="Đóng popup">×</button></header>{children}</section></div>;
}

export function EmployeeAccessWorkspace() {
  const { hasPermission } = useEmployeeSession();
  const [data, setData] = useState<EmployeeAccessList | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState("");
  const [status, setStatus] = useState<"ALL" | EmployeeAccountStatus>("ALL");
  const [query, setQuery] = useState("");
  const [department, setDepartment] = useState("ALL");
  const [role, setRole] = useState("ALL");
  const [permission, setPermission] = useState("ALL");
  const [menuId, setMenuId] = useState("");
  const [selected, setSelected] = useState<EmployeeAccessSummary | null>(null);
  const [detail, setDetail] = useState<EmployeeAccessDetail | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailError, setDetailError] = useState("");
  const canReview = hasPermission("ACCESS_REVIEW_VIEW");
  const canAssign = hasPermission("ROLE_ASSIGN");
  const canResetPassword = hasPermission("USER_PASSWORD_RESET");
  const canRevokeSessions = hasPermission("USER_SESSION_REVOKE");
  const canInvite = hasPermission("EMPLOYEE_INVITE");
  const canRevokeInvite = hasPermission("EMPLOYEE_INVITATION_REVOKE");
  const [notice, setNotice] = useState("");
  const [assignTarget, setAssignTarget] = useState<EmployeeAccessSummary | null>(null);
  const [passwordTarget, setPasswordTarget] = useState<EmployeeAccessSummary | null>(null);
  const [sessionsTarget, setSessionsTarget] = useState<EmployeeAccessSummary | null>(null);
  const [revokeTarget, setRevokeTarget] = useState<{ employee: EmployeeAccessSummary; assignment: EmployeeRoleAssignment } | null>(null);
  const [inviteOpen, setInviteOpen] = useState(false);

  const load = useCallback(async () => {
    setLoading(true); setLoadError("");
    try { setData(await adminAccessService.list()); }
    catch (cause) { setLoadError(cause instanceof AdminAccessApiError ? cause.message : "Không thể tải danh sách tài khoản nhân viên."); }
    finally { setLoading(false); }
  }, []);
  useEffect(() => { void Promise.resolve().then(load); }, [load]);

  const filtered = useMemo(() => {
    const needle = query.trim().toLocaleLowerCase("vi");
    return data?.items.filter((item) =>
      (status === "ALL" || item.status === status) &&
      (department === "ALL" || item.department === department) &&
      (role === "ALL" || item.roles.some((entry) => entry.roleCode === role && entry.status === "ACTIVE")) &&
      (permission === "ALL" || item.effectivePermissionCodes.some((code) => code === permission)) &&
      (!needle || [item.employeeCode, item.displayName, item.email, item.loginIdentifier].some((value) => value.toLocaleLowerCase("vi").includes(needle)))
    ) ?? [];
  }, [data, department, permission, query, role, status]);

  async function openReview(item: EmployeeAccessSummary) {
    setMenuId(""); setSelected(item); setDetail(null); setDetailError(""); setDetailLoading(true);
    try { setDetail(await adminAccessService.detail(item.employeeId)); }
    catch (cause) { setDetailError(cause instanceof AdminAccessApiError ? cause.message : "Không thể tải quyền hiệu lực của nhân viên."); }
    finally { setDetailLoading(false); }
  }

  async function afterEmployeeMutation(message: string, employeeId?: string) {
    setNotice(message); setAssignTarget(null); setPasswordTarget(null); setSessionsTarget(null); setRevokeTarget(null);
    await load();
    if (employeeId && selected?.employeeId === employeeId) {
      try { setDetail(await adminAccessService.detail(employeeId)); } catch { /* danh sách đã phản ánh thay đổi */ }
    }
  }

  if (loading) return <div className="admin-page"><div className="admin-page-skeleton"><div /><div /><div /></div></div>;
  if (!data) return <div className="admin-page"><section className="admin-error-state"><span>!</span><h1>Chưa thể tải quản trị tài khoản</h1><p>{loadError}</p><button className="admin-primary-button" onClick={() => void load()}>Thử lại</button></section></div>;

  return <div className="admin-page">
    <header className="admin-page-header"><div><nav>Admin <span>›</span> Quản trị <span>›</span> Nhân viên</nav><h1>Tài khoản và quyền nhân viên</h1><p>Kiểm tra System Access, Role Assignment và quyền hiệu lực. Hồ sơ nhân sự vẫn do EPIC 02 quản lý.</p></div><div className="admin-page-actions">{canInvite ? <button className="admin-primary-button" type="button" onClick={() => { setNotice(""); setInviteOpen(true); }}>✉ Mời nhân viên</button> : null}<button className="admin-secondary-button" type="button" onClick={() => void load()}>↻ Làm mới</button></div></header>

    {notice ? <p className="admin-section-note">✓ {notice}</p> : null}
    <p className="admin-section-note">Đây là cấu hình quyền hiện tại. Phiên đã đăng nhập có thể giữ quyền cũ đến khi token hết hạn.</p>
    <section className="admin-summary-grid"><article><span>Tổng tài khoản</span><strong>{data.items.length}</strong><small>Trong phạm vi quản trị hiện tại</small></article><article><span>Được cấp quyền hiện tại</span><strong>{data.items.filter((item) => item.accessState === "ENABLED").length}</strong><small>Account và Role đều hiệu lực</small></article><article><span>Bị chặn bởi Account</span><strong>{data.items.filter((item) => item.accessState === "BLOCKED_ACCOUNT").length}</strong><small>Locked, disabled hoặc chưa kích hoạt</small></article><article><span>Không có Role hiệu lực</span><strong>{data.items.filter((item) => item.accessState === "NO_ACTIVE_ROLE").length}</strong><small>Không có quyền cho token mới</small></article></section>

    <section className="admin-list-card"><div className="admin-status-tabs" role="tablist" aria-label="Lọc theo trạng thái tài khoản">{data.statusCounts.map((item) => <button className={status === item.status ? "active" : ""} type="button" role="tab" aria-selected={status === item.status} onClick={() => setStatus(item.status)} key={item.status}>{item.label}<b>{item.count}</b></button>)}</div>
      <div className={`admin-list-toolbar ${canReview ? "access-toolbar" : ""}`}><label className="admin-table-search"><span>⌕</span><input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Tìm mã nhân viên, tên, email hoặc định danh…" /></label><label><span className="sr-only">Phòng ban</span><select value={department} onChange={(event) => setDepartment(event.target.value)}><option value="ALL">Tất cả phòng ban</option>{data.departments.map((item) => <option value={item} key={item}>{item}</option>)}</select></label><label><span className="sr-only">Role</span><select value={role} onChange={(event) => setRole(event.target.value)}><option value="ALL">Tất cả Role</option>{data.roles.map((item) => <option value={item.roleCode} key={item.roleCode}>{item.roleName}</option>)}</select></label>{canReview ? <label><span className="sr-only">Permission</span><select value={permission} onChange={(event) => setPermission(event.target.value)}><option value="ALL">Tất cả Permission</option>{data.permissions.map((item) => <option value={item.permissionCode} key={item.permissionCode}>{item.permissionName}</option>)}</select></label> : null}<button className="admin-filter-reset" type="button" onClick={() => { setQuery(""); setDepartment("ALL"); setRole("ALL"); setPermission("ALL"); setStatus("ALL"); }}>Đặt lại</button></div>
      <div className="admin-table-meta"><span>Hiển thị <strong>{filtered.length}</strong> tài khoản</span><span>Quyền tính lúc {DATE_TIME.format(new Date(data.calculatedAt))}</span></div>
      <div className="admin-table-wrap"><table className="admin-table admin-access-table"><thead><tr><th>Nhân viên</th><th>System Access</th><th>Phòng ban</th><th>Trạng thái</th><th>Role hiệu lực</th><th>Khả năng truy cập</th><th>Cập nhật</th><th aria-label="Tùy chọn" /></tr></thead><tbody>{filtered.map((item) => <tr key={item.accountId}><td><strong>{item.displayName}</strong><small>{item.employeeCode} · {item.jobTitle}</small></td><td><strong>{item.loginIdentifier}</strong><small>{item.accountId}</small></td><td><span>{item.department}</span></td><td><AccountStatus row={item} /></td><td>{item.roles.filter((entry) => entry.status === "ACTIVE").length ? <div className="admin-role-chips">{item.roles.filter((entry) => entry.status === "ACTIVE").map((entry) => <span key={entry.roleCode}>{entry.roleName}</span>)}</div> : <span className="admin-unassigned">Chưa có Role hiệu lực</span>}</td><td><span className={`admin-access-state access-${item.accessState.toLowerCase()}`}>● {item.accessStateLabel}</span></td><td><span>{DATE_TIME.format(new Date(item.updatedAt))}</span></td><td className="admin-action-cell"><button type="button" aria-label={`Tùy chọn ${item.employeeCode}`} onClick={() => setMenuId((current) => current === item.accountId ? "" : item.accountId)}>•••</button>{menuId === item.accountId ? <div className="admin-row-menu">{canReview ? <button type="button" onClick={() => void openReview(item)}>Xem quyền hiệu lực</button> : <span>Không có quyền Access Review</span>}{canAssign ? <button type="button" onClick={() => { setMenuId(""); setNotice(""); setAssignTarget(item); }}>Gán Role</button> : null}{canResetPassword ? <button type="button" onClick={() => { setMenuId(""); setPasswordTarget(item); }}>Đặt lại mật khẩu</button> : null}{canRevokeSessions ? <button type="button" onClick={() => { setMenuId(""); setSessionsTarget(item); }}>Thu hồi phiên đăng nhập</button> : null}</div> : null}</td></tr>)}</tbody></table>{!filtered.length ? <div className="admin-empty"><span>⌕</span><h3>Không có tài khoản phù hợp</h3><p>Hãy thay đổi trạng thái, từ khóa hoặc bộ lọc hiện tại.</p></div> : null}</div>
      <footer className="admin-pagination"><span>Trang 1 / 1</span><div><button disabled>‹</button><button className="active">1</button><button disabled>›</button></div></footer>
    </section>

    {selected ? <Modal title={`Access Review · ${selected.employeeCode}`} subtitle={`${selected.displayName} · ${selected.department}`} onClose={() => setSelected(null)}>{detailLoading ? <div className="admin-modal-loading">Đang tính quyền hiệu lực…</div> : detail ? <AccessReviewContent detail={detail} canAssign={canAssign} canResetPassword={canResetPassword} canRevokeSessions={canRevokeSessions} onAssignRole={() => { if (selected) { setNotice(""); setAssignTarget(selected); } }} onResetPassword={() => { if (selected) setPasswordTarget(selected); }} onRevokeSessions={() => { if (selected) setSessionsTarget(selected); }} onRevokeAssignment={(assignment) => { if (selected) setRevokeTarget({ employee: selected, assignment }); }} /> : <div className="admin-inline-error">{detailError}</div>}</Modal> : null}
    {assignTarget ? <AssignRoleModal employee={assignTarget} roles={data.roles} onClose={() => setAssignTarget(null)} onSubmit={async (input) => { await adminAccessService.assignRole(assignTarget.employeeId, input); await afterEmployeeMutation(`Đã gán Role ${input.roleCode} cho ${assignTarget.displayName}. Quyền có hiệu lực ở lần đăng nhập/refresh token kế tiếp.`, assignTarget.employeeId); }} /> : null}
    {passwordTarget ? <PasswordResetModal employee={passwordTarget} onClose={() => setPasswordTarget(null)} onSubmit={async (input) => { await adminAccessService.resetPassword(passwordTarget.employeeId, input); await afterEmployeeMutation(`Đã đặt lại mật khẩu cho ${passwordTarget.displayName}.`, passwordTarget.employeeId); }} /> : null}
    {sessionsTarget ? <ConfirmActionModal title={`Thu hồi phiên · ${sessionsTarget.displayName}`} body="Mọi phiên đăng nhập hiện tại của nhân viên này sẽ bị đăng xuất ngay. Nhân viên phải đăng nhập lại để tiếp tục làm việc." confirmLabel="Thu hồi phiên" onClose={() => setSessionsTarget(null)} onConfirm={async () => { await adminAccessService.revokeSessions(sessionsTarget.employeeId); await afterEmployeeMutation(`Đã thu hồi mọi phiên đăng nhập của ${sessionsTarget.displayName}.`, sessionsTarget.employeeId); }} /> : null}
    {revokeTarget ? <ConfirmActionModal title={`Thu hồi Role · ${revokeTarget.assignment.roleName}`} body={`Thu hồi Role ${revokeTarget.assignment.roleCode} (${revokeTarget.assignment.scopeLabel}) khỏi ${revokeTarget.employee.displayName}? Quyền tương ứng sẽ bị gỡ ở lần đăng nhập/refresh token kế tiếp.`} confirmLabel="Thu hồi Role" onClose={() => setRevokeTarget(null)} onConfirm={async () => { await adminAccessService.revokeAssignment(revokeTarget.employee.employeeId, revokeTarget.assignment.assignmentId); await afterEmployeeMutation(`Đã thu hồi Role ${revokeTarget.assignment.roleCode} khỏi ${revokeTarget.employee.displayName}.`, revokeTarget.employee.employeeId); }} /> : null}
    {inviteOpen ? <InviteEmployeeModal canRevoke={canRevokeInvite} onClose={() => setInviteOpen(false)} /> : null}
  </div>;
}

function AssignRoleModal({ employee, roles, onClose, onSubmit }: { employee: EmployeeAccessSummary; roles: Array<{ roleCode: string; roleName: string }>; onClose: () => void; onSubmit: (input: { roleCode: string; scopeType: "GLOBAL" | "WAREHOUSE" | "DEPARTMENT"; scopeId?: string; validUntil?: string; reason?: string }) => Promise<void> }) {
  const [roleCode, setRoleCode] = useState(roles[0]?.roleCode ?? "");
  const [scopeType, setScopeType] = useState<"GLOBAL" | "WAREHOUSE" | "DEPARTMENT">("GLOBAL");
  const [scopeId, setScopeId] = useState("");
  const [validUntil, setValidUntil] = useState("");
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  async function submit(event: React.FormEvent) {
    event.preventDefault(); setBusy(true); setError("");
    try {
      await onSubmit({ roleCode, scopeType, scopeId: scopeType === "GLOBAL" ? undefined : scopeId.trim(), validUntil: validUntil ? new Date(validUntil).toISOString() : undefined, reason: reason.trim() || undefined });
    } catch (cause) { setError(cause instanceof AdminAccessApiError ? cause.message : "Không thể gán Role."); setBusy(false); }
  }
  return <Modal title={`Gán Role · ${employee.displayName}`} subtitle={employee.email} onClose={onClose}><form className="admin-action-form" onSubmit={(event) => void submit(event)}>
    <label><span>Role *</span><select value={roleCode} onChange={(event) => setRoleCode(event.target.value)} required>{roles.map((role) => <option value={role.roleCode} key={role.roleCode}>{role.roleName} ({role.roleCode})</option>)}</select></label>
    <div className="admin-form-grid"><label><span>Phạm vi (Scope)</span><select value={scopeType} onChange={(event) => setScopeType(event.target.value as "GLOBAL" | "WAREHOUSE" | "DEPARTMENT")}><option value="GLOBAL">Toàn hệ thống (GLOBAL)</option><option value="WAREHOUSE">Theo kho (WAREHOUSE)</option><option value="DEPARTMENT">Theo phòng ban (DEPARTMENT)</option></select></label>
    <label><span>Mã phạm vi{scopeType === "GLOBAL" ? "" : " *"}</span><input value={scopeId} disabled={scopeType === "GLOBAL"} onChange={(event) => setScopeId(event.target.value)} placeholder={scopeType === "GLOBAL" ? "Không áp dụng" : scopeType === "WAREHOUSE" ? "VD: WH-HUE-01" : "VD: SALES"} required={scopeType !== "GLOBAL"} /></label>
    <label><span>Hiệu lực đến</span><input type="datetime-local" value={validUntil} onChange={(event) => setValidUntil(event.target.value)} /></label></div>
    <label><span>Lý do phân công</span><input value={reason} onChange={(event) => setReason(event.target.value)} placeholder="VD: Nhận nhiệm vụ kho Huế từ 11/2026" /></label>
    <p className="admin-section-note">Không thể tự gán Role cho chính mình — backend sẽ từ chối. Quyền mới chỉ vào token ở lần đăng nhập/refresh kế tiếp.</p>
    {error ? <div className="admin-inline-error">{error}</div> : null}
    <footer><button className="admin-secondary-button" type="button" onClick={onClose}>Hủy</button><button className="admin-primary-button" type="submit" disabled={busy}>{busy ? "Đang gán…" : "Gán Role"}</button></footer>
  </form></Modal>;
}

function PasswordResetModal({ employee, onClose, onSubmit }: { employee: EmployeeAccessSummary; onClose: () => void; onSubmit: (input: { newPassword: string; temporary: boolean }) => Promise<void> }) {
  const [newPassword, setNewPassword] = useState("");
  const [temporary, setTemporary] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  async function submit(event: React.FormEvent) {
    event.preventDefault(); setBusy(true); setError("");
    try { await onSubmit({ newPassword, temporary }); }
    catch (cause) { setError(cause instanceof AdminAccessApiError ? cause.message : "Không thể đặt lại mật khẩu."); setBusy(false); }
  }
  return <Modal title={`Đặt lại mật khẩu · ${employee.displayName}`} subtitle={employee.email} onClose={onClose}><form className="admin-action-form" onSubmit={(event) => void submit(event)}>
    <label><span>Mật khẩu mới *</span><input type="password" value={newPassword} onChange={(event) => setNewPassword(event.target.value)} placeholder="Ít nhất 8 ký tự" minLength={8} required autoComplete="new-password" /><small>Mật khẩu do quản trị viên đặt — hãy chuyển cho nhân viên qua kênh an toàn, không gửi kèm email công việc thường.</small></label>
    <label><span><input type="checkbox" checked={temporary} onChange={(event) => setTemporary(event.target.checked)} /> Mật khẩu tạm — bắt buộc đổi ở lần đăng nhập kế tiếp</span></label>
    {error ? <div className="admin-inline-error">{error}</div> : null}
    <footer><button className="admin-secondary-button" type="button" onClick={onClose}>Hủy</button><button className="admin-primary-button" type="submit" disabled={busy}>{busy ? "Đang đặt…" : "Đặt lại mật khẩu"}</button></footer>
  </form></Modal>;
}

function ConfirmActionModal({ title, body, confirmLabel, onClose, onConfirm }: { title: string; body: string; confirmLabel: string; onClose: () => void; onConfirm: () => Promise<void> }) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  async function confirm() {
    setBusy(true); setError("");
    try { await onConfirm(); }
    catch (cause) { setError(cause instanceof AdminAccessApiError ? cause.message : "Không thể thực hiện thao tác."); setBusy(false); }
  }
  return <Modal title={title} onClose={onClose}><div className="admin-action-form">
    <p>{body}</p>
    {error ? <div className="admin-inline-error">{error}</div> : null}
    <footer><button className="admin-secondary-button" type="button" onClick={onClose}>Hủy</button><button className="admin-primary-button" type="button" disabled={busy} onClick={() => void confirm()}>{busy ? "Đang xử lý…" : confirmLabel}</button></footer>
  </div></Modal>;
}

function InviteEmployeeModal({ canRevoke, onClose }: { canRevoke: boolean; onClose: () => void }) {
  const [validDays, setValidDays] = useState(7);
  const [created, setCreated] = useState<InvitationCreated | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [revoked, setRevoked] = useState(false);
  async function submit(event: React.FormEvent) {
    event.preventDefault(); setBusy(true); setError("");
    try { setCreated(await adminAccessService.createInvitation(validDays)); }
    catch (cause) { setError(cause instanceof AdminAccessApiError ? cause.message : "Không thể tạo thư mời."); }
    finally { setBusy(false); }
  }
  async function revoke() {
    if (!created) return;
    setBusy(true); setError("");
    try { await adminAccessService.revokeInvitation(created.invitationId); setRevoked(true); }
    catch (cause) { setError(cause instanceof AdminAccessApiError ? cause.message : "Không thể thu hồi thư mời."); }
    finally { setBusy(false); }
  }
  return <Modal title="Mời nhân viên" subtitle="Tạo bản ghi thư mời trên Identity" onClose={onClose}>
    {created ? <div className="admin-action-form">
      <dl><div><dt>Mã thư mời</dt><dd>{created.invitationId}</dd></div><div><dt>Principal chờ kích hoạt</dt><dd>{created.principalId}</dd></div><div><dt>Trạng thái</dt><dd>{revoked ? "Đã thu hồi" : created.status}</dd></div><div><dt>Hết hạn</dt><dd>{DATE_TIME.format(new Date(created.expiresAt))}</dd></div></dl>
      <p className="admin-section-note">Giới hạn hiện tại của backend: Identity chưa phơi endpoint chấp nhận thư mời và realm chưa cấu hình email (F-46), nên người được mời chưa thể tự kích hoạt bằng mã này. Thư mời hiện đóng vai trò bản ghi theo dõi cho đến khi luồng chấp nhận được bổ sung.</p>
      {error ? <div className="admin-inline-error">{error}</div> : null}
      <footer>{canRevoke && !revoked ? <button className="admin-secondary-button" type="button" disabled={busy} onClick={() => void revoke()}>Thu hồi thư mời</button> : null}<button className="admin-primary-button" type="button" onClick={onClose}>Đóng</button></footer>
    </div> : <form className="admin-action-form" onSubmit={(event) => void submit(event)}>
      <label><span>Số ngày hiệu lực</span><input type="number" min={1} max={30} value={validDays} onChange={(event) => setValidDays(Number(event.target.value))} required /><small>Từ 1 đến 30 ngày, mặc định 7.</small></label>
      {error ? <div className="admin-inline-error">{error}</div> : null}
      <footer><button className="admin-secondary-button" type="button" onClick={onClose}>Hủy</button><button className="admin-primary-button" type="submit" disabled={busy}>{busy ? "Đang tạo…" : "Tạo thư mời"}</button></footer>
    </form>}
  </Modal>;
}

function AccessReviewContent({ detail, canAssign, canResetPassword, canRevokeSessions, onAssignRole, onResetPassword, onRevokeSessions, onRevokeAssignment }: { detail: EmployeeAccessDetail; canAssign: boolean; canResetPassword: boolean; canRevokeSessions: boolean; onAssignRole: () => void; onResetPassword: () => void; onRevokeSessions: () => void; onRevokeAssignment: (assignment: EmployeeRoleAssignment) => void }) {
  return <div className="admin-access-review">
    <div className="admin-detail-summary"><AccountStatus row={detail} /><span className={`admin-access-state access-${detail.accessState.toLowerCase()}`}>● {detail.accessStateLabel}</span><span>Tính lúc {DATE_TIME.format(new Date(detail.calculatedAt))}</span></div>
    {canAssign || canResetPassword || canRevokeSessions ? <div className="admin-detail-summary">{canAssign ? <button className="admin-secondary-button" type="button" onClick={onAssignRole}>Gán Role</button> : null}{canResetPassword ? <button className="admin-secondary-button" type="button" onClick={onResetPassword}>Đặt lại mật khẩu</button> : null}{canRevokeSessions ? <button className="admin-secondary-button" type="button" onClick={onRevokeSessions}>Thu hồi phiên đăng nhập</button> : null}</div> : null}
    {detail.status === "LOCKED" && detail.lock ? <section className="admin-lock-notice"><strong>Account đang bị khóa</strong><p>{detail.lock.reason}</p><small>Hiệu lực {DATE_TIME.format(new Date(detail.lock.effectiveAt))} · {detail.lock.decidedByLabel}</small></section> : null}
    <div className="admin-detail-grid"><section><h3>System Access</h3><dl><div><dt>Account ID</dt><dd>{detail.accountId}</dd></div><div><dt>Định danh đăng nhập</dt><dd>{detail.loginIdentifier}</dd></div><div><dt>Mật khẩu</dt><dd>Không được hiển thị hoặc đọc lại</dd></div></dl></section><section><h3>Tham chiếu Employee Profile</h3><dl><div><dt>Employee ID</dt><dd>{detail.profileReference.employeeId}</dd></div><div><dt>Mã nhân viên</dt><dd>{detail.profileReference.employeeCode}</dd></div><div><dt>Họ tên</dt><dd>{detail.displayName}</dd></div><div><dt>Phòng ban</dt><dd>{detail.department}</dd></div><div><dt>Chủ sở hữu dữ liệu</dt><dd>EPIC 02 · Employee Profile</dd></div></dl></section></div>
    <section><h3>Role Assignment</h3><div className="admin-assignment-list">{detail.roleAssignments.map((assignment) => <article key={assignment.assignmentId}><header><div><strong>{assignment.roleName}</strong><small>{assignment.roleCode} · {assignment.assignmentId}</small></div><em className={`assignment-${assignment.assignmentStatus.toLowerCase()}`}>{assignment.assignmentStatus}</em>{canAssign && (assignment.assignmentStatus === "ACTIVE" || assignment.assignmentStatus === "SCHEDULED") ? <button className="admin-secondary-button" type="button" onClick={() => onRevokeAssignment(assignment)}>Thu hồi</button> : null}</header><dl><div><dt>Scope</dt><dd>{assignment.scopeLabel}</dd></div><div><dt>Hiệu lực</dt><dd>{DATE_TIME.format(new Date(assignment.effectiveFrom))}{assignment.effectiveUntil ? ` – ${DATE_TIME.format(new Date(assignment.effectiveUntil))}` : " – Không thời hạn"}</dd></div><div><dt>Người gán</dt><dd>{assignment.assignedByLabel}</dd></div></dl></article>)}</div></section>
    <section><h3>Permission hiệu lực</h3><p className="admin-section-note">Permission trùng từ nhiều Role chỉ xuất hiện một lần nhưng vẫn giữ đủ nguồn cấp.</p><table className="admin-permission-table"><thead><tr><th>Permission</th><th>Action / Resource</th><th>Scope</th><th>Nguồn Role</th></tr></thead><tbody>{detail.effectivePermissions.map((item) => <tr key={item.permissionCode}><td><strong>{item.permissionName}</strong><small>{item.permissionCode}</small></td><td>{item.action} / {item.resource}</td><td>{item.scopeLabels.join(", ")}</td><td>{item.sources.map((source) => source.roleName).join(", ")}</td></tr>)}</tbody></table>{!detail.effectivePermissions.length ? <p className="admin-file-empty">Không có Permission hiệu lực tại thời điểm tính.</p> : null}</section>
    <footer>Access Review mô tả cấu hình quyền hiện tại, không thay thế Audit Log. Điều kiện động của từng nghiệp vụ vẫn được module sở hữu kiểm tra lại.</footer>
  </div>;
}
