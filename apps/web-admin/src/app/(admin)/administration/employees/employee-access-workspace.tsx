"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useAdminSession } from "@/components/auth/admin-session-provider";
import type { EmployeeAccessDetail, EmployeeAccessList, EmployeeAccessSummary, EmployeeAccountStatus } from "@/lib/access/types";
import { AdminAccessApiError } from "@/lib/access/types";
import { adminAccessService } from "@/services/admin-access-service";

const DATE_TIME = new Intl.DateTimeFormat("vi-VN", { day: "2-digit", month: "2-digit", year: "numeric", hour: "2-digit", minute: "2-digit" });

function AccountStatus({ row }: { row: Pick<EmployeeAccessSummary, "status" | "statusLabel"> }) {
  return <span className={`admin-status account-${row.status.toLowerCase()}`}>{row.statusLabel}</span>;
}

function Modal({ title, subtitle, onClose, children }: { title: string; subtitle?: string; onClose: () => void; children: React.ReactNode }) {
  return <div className="admin-modal-backdrop" role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget) onClose(); }}><section className="admin-modal wide" role="dialog" aria-modal="true" aria-label={title}><header><div><h2>{title}</h2>{subtitle ? <p>{subtitle}</p> : null}</div><button type="button" onClick={onClose} aria-label="Đóng popup">×</button></header>{children}</section></div>;
}

export function EmployeeAccessWorkspace() {
  const { hasPermission } = useAdminSession();
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

  if (loading) return <div className="admin-page"><div className="admin-page-skeleton"><div /><div /><div /></div></div>;
  if (!data) return <div className="admin-page"><section className="admin-error-state"><span>!</span><h1>Chưa thể tải quản trị tài khoản</h1><p>{loadError}</p><button className="admin-primary-button" onClick={() => void load()}>Thử lại</button></section></div>;

  return <div className="admin-page">
    <header className="admin-page-header"><div><nav>Admin <span>›</span> Quản trị <span>›</span> Nhân viên</nav><h1>Tài khoản và quyền nhân viên</h1><p>Kiểm tra System Access, Role Assignment và quyền hiệu lực. Hồ sơ nhân sự vẫn do EPIC 02 quản lý.</p></div><div className="admin-page-actions"><button className="admin-secondary-button" type="button" onClick={() => void load()}>↻ Làm mới</button></div></header>

    <section className="admin-summary-grid"><article><span>Tổng tài khoản</span><strong>{data.items.length}</strong><small>Trong phạm vi quản trị hiện tại</small></article><article><span>Đang truy cập được</span><strong>{data.items.filter((item) => item.accessState === "ENABLED").length}</strong><small>Account và Role đều hiệu lực</small></article><article><span>Bị chặn bởi Account</span><strong>{data.items.filter((item) => item.accessState === "BLOCKED_ACCOUNT").length}</strong><small>Locked, disabled hoặc chưa kích hoạt</small></article><article><span>Không có Role hiệu lực</span><strong>{data.items.filter((item) => item.accessState === "NO_ACTIVE_ROLE").length}</strong><small>Không được vào vùng quản trị</small></article></section>

    <section className="admin-list-card"><div className="admin-status-tabs" role="tablist" aria-label="Lọc theo trạng thái tài khoản">{data.statusCounts.map((item) => <button className={status === item.status ? "active" : ""} type="button" role="tab" aria-selected={status === item.status} onClick={() => setStatus(item.status)} key={item.status}>{item.label}<b>{item.count}</b></button>)}</div>
      <div className={`admin-list-toolbar ${canReview ? "access-toolbar" : ""}`}><label className="admin-table-search"><span>⌕</span><input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Tìm mã nhân viên, tên, email hoặc định danh…" /></label><label><span className="sr-only">Phòng ban</span><select value={department} onChange={(event) => setDepartment(event.target.value)}><option value="ALL">Tất cả phòng ban</option>{data.departments.map((item) => <option value={item} key={item}>{item}</option>)}</select></label><label><span className="sr-only">Role</span><select value={role} onChange={(event) => setRole(event.target.value)}><option value="ALL">Tất cả Role</option>{data.roles.map((item) => <option value={item.roleCode} key={item.roleCode}>{item.roleName}</option>)}</select></label>{canReview ? <label><span className="sr-only">Permission</span><select value={permission} onChange={(event) => setPermission(event.target.value)}><option value="ALL">Tất cả Permission</option>{data.permissions.map((item) => <option value={item.permissionCode} key={item.permissionCode}>{item.permissionName}</option>)}</select></label> : null}<button className="admin-filter-reset" type="button" onClick={() => { setQuery(""); setDepartment("ALL"); setRole("ALL"); setPermission("ALL"); setStatus("ALL"); }}>Đặt lại</button></div>
      <div className="admin-table-meta"><span>Hiển thị <strong>{filtered.length}</strong> tài khoản</span><span>Quyền tính lúc {DATE_TIME.format(new Date(data.calculatedAt))}</span></div>
      <div className="admin-table-wrap"><table className="admin-table admin-access-table"><thead><tr><th>Nhân viên</th><th>System Access</th><th>Phòng ban</th><th>Trạng thái</th><th>Role hiệu lực</th><th>Khả năng truy cập</th><th>Cập nhật</th><th aria-label="Tùy chọn" /></tr></thead><tbody>{filtered.map((item) => <tr key={item.accountId}><td><strong>{item.displayName}</strong><small>{item.employeeCode} · {item.jobTitle}</small></td><td><strong>{item.loginIdentifier}</strong><small>{item.accountId}</small></td><td><span>{item.department}</span></td><td><AccountStatus row={item} /></td><td>{item.roles.filter((entry) => entry.status === "ACTIVE").length ? <div className="admin-role-chips">{item.roles.filter((entry) => entry.status === "ACTIVE").map((entry) => <span key={entry.roleCode}>{entry.roleName}</span>)}</div> : <span className="admin-unassigned">Chưa có Role hiệu lực</span>}</td><td><span className={`admin-access-state access-${item.accessState.toLowerCase()}`}>● {item.accessStateLabel}</span></td><td><span>{DATE_TIME.format(new Date(item.updatedAt))}</span></td><td className="admin-action-cell"><button type="button" aria-label={`Tùy chọn ${item.employeeCode}`} onClick={() => setMenuId((current) => current === item.accountId ? "" : item.accountId)}>•••</button>{menuId === item.accountId ? <div className="admin-row-menu">{canReview ? <button type="button" onClick={() => void openReview(item)}>Xem quyền hiệu lực</button> : <span>Không có quyền Access Review</span>}</div> : null}</td></tr>)}</tbody></table>{!filtered.length ? <div className="admin-empty"><span>⌕</span><h3>Không có tài khoản phù hợp</h3><p>Hãy thay đổi trạng thái, từ khóa hoặc bộ lọc hiện tại.</p></div> : null}</div>
      <footer className="admin-pagination"><span>Trang 1 / 1</span><div><button disabled>‹</button><button className="active">1</button><button disabled>›</button></div></footer>
    </section>

    {selected ? <Modal title={`Access Review · ${selected.employeeCode}`} subtitle={`${selected.displayName} · ${selected.department}`} onClose={() => setSelected(null)}>{detailLoading ? <div className="admin-modal-loading">Đang tính quyền hiệu lực…</div> : detail ? <AccessReviewContent detail={detail} /> : <div className="admin-inline-error">{detailError}</div>}</Modal> : null}
  </div>;
}

function AccessReviewContent({ detail }: { detail: EmployeeAccessDetail }) {
  return <div className="admin-access-review">
    <div className="admin-detail-summary"><AccountStatus row={detail} /><span className={`admin-access-state access-${detail.accessState.toLowerCase()}`}>● {detail.accessStateLabel}</span><span>Tính lúc {DATE_TIME.format(new Date(detail.calculatedAt))}</span></div>
    {detail.status === "LOCKED" && detail.lock ? <section className="admin-lock-notice"><strong>Account đang bị khóa</strong><p>{detail.lock.reason}</p><small>Hiệu lực {DATE_TIME.format(new Date(detail.lock.effectiveAt))} · {detail.lock.decidedByLabel}</small></section> : null}
    <div className="admin-detail-grid"><section><h3>System Access</h3><dl><div><dt>Account ID</dt><dd>{detail.accountId}</dd></div><div><dt>Định danh đăng nhập</dt><dd>{detail.loginIdentifier}</dd></div><div><dt>Xác minh email</dt><dd>{detail.identity.emailVerified ? "Đã xác minh" : "Chưa xác minh"}</dd></div><div><dt>Đăng nhập gần nhất</dt><dd>{detail.identity.lastAuthenticatedAt ? DATE_TIME.format(new Date(detail.identity.lastAuthenticatedAt)) : "Chưa có"}</dd></div><div><dt>Mật khẩu</dt><dd>Không được hiển thị hoặc đọc lại</dd></div></dl></section><section><h3>Tham chiếu Employee Profile</h3><dl><div><dt>Employee ID</dt><dd>{detail.profileReference.employeeId}</dd></div><div><dt>Mã nhân viên</dt><dd>{detail.profileReference.employeeCode}</dd></div><div><dt>Họ tên</dt><dd>{detail.displayName}</dd></div><div><dt>Phòng ban</dt><dd>{detail.department}</dd></div><div><dt>Chủ sở hữu dữ liệu</dt><dd>EPIC 02 · Employee Profile</dd></div></dl></section></div>
    <section><h3>Role Assignment</h3><div className="admin-assignment-list">{detail.roleAssignments.map((assignment) => <article key={assignment.assignmentId}><header><div><strong>{assignment.roleName}</strong><small>{assignment.roleCode} · {assignment.assignmentId}</small></div><em className={`assignment-${assignment.assignmentStatus.toLowerCase()}`}>{assignment.assignmentStatus}</em></header><dl><div><dt>Scope</dt><dd>{assignment.scopeLabel}</dd></div><div><dt>Hiệu lực</dt><dd>{DATE_TIME.format(new Date(assignment.effectiveFrom))}{assignment.effectiveUntil ? ` – ${DATE_TIME.format(new Date(assignment.effectiveUntil))}` : " – Không thời hạn"}</dd></div><div><dt>Người gán</dt><dd>{assignment.assignedByLabel}</dd></div></dl></article>)}</div></section>
    <section><h3>Permission hiệu lực</h3><p className="admin-section-note">Permission trùng từ nhiều Role chỉ xuất hiện một lần nhưng vẫn giữ đủ nguồn cấp.</p><table className="admin-permission-table"><thead><tr><th>Permission</th><th>Action / Resource</th><th>Scope</th><th>Nguồn Role</th></tr></thead><tbody>{detail.effectivePermissions.map((item) => <tr key={item.permissionCode}><td><strong>{item.permissionName}</strong><small>{item.permissionCode}</small></td><td>{item.action} / {item.resource}</td><td>{item.scopeLabels.join(", ")}</td><td>{item.sources.map((source) => source.roleName).join(", ")}</td></tr>)}</tbody></table>{!detail.effectivePermissions.length ? <p className="admin-file-empty">Không có Permission hiệu lực tại thời điểm tính.</p> : null}</section>
    <footer>Access Review mô tả cấu hình quyền hiện tại, không thay thế Audit Log. Điều kiện động của từng nghiệp vụ vẫn được module sở hữu kiểm tra lại.</footer>
  </div>;
}
