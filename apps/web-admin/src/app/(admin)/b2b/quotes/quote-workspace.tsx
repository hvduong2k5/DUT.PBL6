"use client";

import { FormEvent, useCallback, useEffect, useMemo, useState } from "react";
import { useSearchParams } from "next/navigation";
import { useAdminSession } from "@/components/auth/admin-session-provider";
import type { AdminB2BAction, AdminB2BActionInput, AdminB2BList, AdminB2BQuoteVersionHistory, AdminB2BRequestDetail, AdminB2BRequestSummary, AdminB2BStatus } from "@/lib/b2b/types";
import { AdminB2BApiError } from "@/lib/b2b/types";
import { adminB2BService } from "@/services/admin-b2b-service";

const VND = new Intl.NumberFormat("vi-VN", { style: "currency", currency: "VND", maximumFractionDigits: 0 });
const DATE = new Intl.DateTimeFormat("vi-VN", { day: "2-digit", month: "2-digit", year: "numeric" });
const DATE_TIME = new Intl.DateTimeFormat("vi-VN", { day: "2-digit", month: "2-digit", year: "numeric", hour: "2-digit", minute: "2-digit" });
type ModalAction = "VIEW" | "VERSION_HISTORY" | "ASSIGN" | "REQUEST_INFO" | "CREATE_DRAFT" | "ISSUE" | "WITHDRAW" | "REJECT" | "CONVERT_ORDER";

const ACTION_LABELS: Record<ModalAction, string> = { VIEW: "Xem chi tiết", VERSION_HISTORY: "Lịch sử phiên bản", ASSIGN: "Phân công phụ trách", REQUEST_INFO: "Yêu cầu bổ sung", CREATE_DRAFT: "Lập phiên bản báo giá", ISSUE: "Phát hành báo giá", WITHDRAW: "Thu hồi báo giá", REJECT: "Từ chối yêu cầu", CONVERT_ORDER: "Chuyển thành đơn hàng" };
const STATUS_ACTIONS: Record<AdminB2BStatus, ModalAction[]> = {
  REQUESTED: ["VIEW", "ASSIGN", "REQUEST_INFO", "CREATE_DRAFT", "REJECT"],
  NEEDS_INFO: ["VIEW", "ASSIGN", "CREATE_DRAFT", "REJECT"],
  DRAFT: ["VIEW", "VERSION_HISTORY", "ASSIGN", "REQUEST_INFO", "CREATE_DRAFT", "ISSUE", "REJECT"],
  SENT: ["VIEW", "VERSION_HISTORY", "ASSIGN", "CREATE_DRAFT", "WITHDRAW"],
  ACCEPTED: ["VIEW", "VERSION_HISTORY", "CONVERT_ORDER"],
  REJECTED: ["VIEW"],
  EXPIRED: ["VIEW", "VERSION_HISTORY", "CREATE_DRAFT"],
  WITHDRAWN: ["VIEW", "VERSION_HISTORY", "CREATE_DRAFT"],
  CONVERTED: ["VIEW", "VERSION_HISTORY"]
};

function StatusBadge({ row }: { row: Pick<AdminB2BRequestSummary, "status" | "statusLabel"> }) { return <span className={`admin-status status-${row.status.toLowerCase()}`}>{row.statusLabel}</span>; }

function Modal({ title, subtitle, onClose, children, wide = false }: { title: string; subtitle?: string; onClose: () => void; children: React.ReactNode; wide?: boolean }) {
  return <div className="admin-modal-backdrop" role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget) onClose(); }}><section className={`admin-modal ${wide ? "wide" : ""}`} role="dialog" aria-modal="true" aria-label={title}><header><div><h2>{title}</h2>{subtitle ? <p>{subtitle}</p> : null}</div><button type="button" onClick={onClose} aria-label="Đóng popup">×</button></header>{children}</section></div>;
}

export function B2BQuoteWorkspace() {
  const searchParams = useSearchParams();
  const scenario = searchParams?.get("mockScenario") ?? undefined;
  const { hasPermission } = useAdminSession();
  const [data, setData] = useState<AdminB2BList | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState("");
  const [status, setStatus] = useState<"ALL" | AdminB2BStatus>("ALL");
  const [query, setQuery] = useState("");
  const [owner, setOwner] = useState("ALL");
  const [sla, setSla] = useState("ALL");
  const [menuId, setMenuId] = useState("");
  const [modal, setModal] = useState<{ action: ModalAction; row: AdminB2BRequestSummary } | null>(null);
  const [detail, setDetail] = useState<AdminB2BRequestDetail | null>(null);
  const [history, setHistory] = useState<AdminB2BQuoteVersionHistory | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [actionError, setActionError] = useState("");
  const [actionConflict, setActionConflict] = useState(false);
  const [actionPending, setActionPending] = useState(false);
  const [toast, setToast] = useState("");

  const load = useCallback(async () => {
    setLoading(true); setLoadError("");
    try { setData(await adminB2BService.list(scenario)); }
    catch (cause) { setLoadError(cause instanceof AdminB2BApiError ? cause.message : "Không thể tải hàng đợi B2B."); }
    finally { setLoading(false); }
  }, [scenario]);
  useEffect(() => { void Promise.resolve().then(load); }, [load]);

  const filtered = useMemo(() => {
    const needle = query.trim().toLocaleLowerCase("vi");
    return data?.items.filter((row) => (status === "ALL" || row.status === status) && (owner === "ALL" || (owner === "UNASSIGNED" ? !row.owner : row.owner?.employeeId === owner)) && (sla === "ALL" || row.slaState === sla) && (!needle || [row.requestNumber, row.companyName, row.taxCode].some((value) => value.toLocaleLowerCase("vi").includes(needle)))) ?? [];
  }, [data, owner, query, sla, status]);

  const openModal = useCallback(async (action: ModalAction, row: AdminB2BRequestSummary) => {
    setMenuId(""); setModal({ action, row }); setDetail(null); setHistory(null); setActionError(""); setActionConflict(false);
    if (["VIEW", "CREATE_DRAFT", "ISSUE"].includes(action)) {
      setDetailLoading(true);
      try { setDetail(await adminB2BService.detail(row.requestId, scenario)); }
      catch (cause) { setActionError(cause instanceof AdminB2BApiError ? cause.message : "Không thể tải chi tiết yêu cầu."); }
      finally { setDetailLoading(false); }
    }
    if (action === "VERSION_HISTORY" && row.currentQuoteVersion) {
      setDetailLoading(true);
      try { setHistory(await adminB2BService.versions(row.requestId, row.currentQuoteVersion, row.status, scenario)); }
      catch (cause) { setActionError(cause instanceof AdminB2BApiError ? cause.message : "Không thể tải lịch sử phiên bản."); }
      finally { setDetailLoading(false); }
    }
  }, [scenario]);

  async function act(input: Omit<AdminB2BActionInput, "idempotencyKey" | "expectedRevision">) {
    if (!modal) return;
    setActionPending(true); setActionError("");
    try {
      const result = await adminB2BService.act(modal.row.requestId, { ...input, expectedRevision: modal.row.revision, idempotencyKey: `admin-b2b-${globalThis.crypto.randomUUID()}` }, scenario);
      setData((current) => {
        if (!current) return current;
        const previous = current.items.find((row) => row.requestId === result.requestId);
        const nextStatus = input.action === "ASSIGN" ? previous?.status : result.status;
        const statusCounts = previous && nextStatus && previous.status !== nextStatus ? current.statusCounts.map((item) => item.status === previous.status ? { ...item, count: Math.max(0, item.count - 1) } : item.status === nextStatus ? { ...item, count: item.count + 1 } : item) : current.statusCounts;
        return { ...current, statusCounts, items: current.items.map((row) => row.requestId === result.requestId ? { ...row, ...(input.action === "ASSIGN" ? { owner: current.owners.find((item) => item.employeeId === input.assigneeId) ?? row.owner } : { status: result.status, statusLabel: result.statusLabel }), currentQuoteVersion: input.action === "CREATE_DRAFT" ? Math.max(result.quoteVersion ?? 0, (row.currentQuoteVersion ?? 0) + 1) : row.currentQuoteVersion, revision: result.revision } : row) };
      });
      setToast(result.orderNumber ? `${result.message} Mã đơn: ${result.orderNumber}.` : result.message); setModal(null);
      globalThis.setTimeout(() => setToast(""), 4200);
    } catch (cause) { setActionConflict(cause instanceof AdminB2BApiError && cause.code === "B2B_CONCURRENCY_CONFLICT"); setActionError(cause instanceof AdminB2BApiError ? cause.message : "Không thể thực hiện thao tác."); }
    finally { setActionPending(false); }
  }

  function firstDraftable() { const row = data?.items.find((item) => item.status === "REQUESTED" || item.status === "NEEDS_INFO"); if (row) void openModal("CREATE_DRAFT", row); else setToast("Không có yêu cầu phù hợp để lập báo giá mới."); }
  const allowed = (action: ModalAction) => {
    if (action === "VIEW" || action === "VERSION_HISTORY") return true;
    const required = { ASSIGN: "B2B_REQUEST_ASSIGN", REQUEST_INFO: "B2B_REQUEST_INFO_REQUEST", CREATE_DRAFT: "B2B_QUOTE_DRAFT", ISSUE: "B2B_QUOTE_ISSUE", WITHDRAW: "B2B_QUOTE_WITHDRAW", REJECT: "B2B_QUOTE_REJECT", CONVERT_ORDER: "B2B_ORDER_CONVERT" } as const;
    return hasPermission(required[action as Exclude<AdminB2BAction, "VIEW">]);
  };

  if (loading) return <div className="admin-page"><div className="admin-page-skeleton"><div /><div /><div /></div></div>;
  if (!data) return <div className="admin-page"><section className="admin-error-state"><span>!</span><h1>Chưa thể tải B2B Quote Workspace</h1><p>{loadError}</p><button className="admin-primary-button" onClick={() => void load()}>Thử lại</button></section></div>;

  return <div className="admin-page">
    <header className="admin-page-header"><div><nav>Admin <span>›</span> Kinh doanh <span>›</span> B2B</nav><h1>Quản lý yêu cầu báo giá B2B</h1><p>Thẩm định yêu cầu doanh nghiệp, quản lý phiên bản và phát hành báo giá đúng thẩm quyền.</p></div><div className="admin-page-actions"><button className="admin-secondary-button" type="button" onClick={() => void load()}>↻ Làm mới</button>{hasPermission("B2B_QUOTE_DRAFT") ? <button className="admin-primary-button" type="button" onClick={firstDraftable}>＋ Tạo báo giá</button> : null}</div></header>

    <section className="admin-summary-grid"><article><span>Yêu cầu đang mở</span><strong>{data.items.filter((item) => !["ACCEPTED", "REJECTED", "EXPIRED", "WITHDRAWN", "CONVERTED"].includes(item.status)).length}</strong><small>Cập nhật {DATE_TIME.format(new Date(data.updatedAt))}</small></article><article><span>Cần phản hồi</span><strong>{data.items.filter((item) => item.status === "REQUESTED").length}</strong><small>Chưa có báo giá nháp</small></article><article><span>SLA rủi ro/quá hạn</span><strong>{data.items.filter((item) => item.slaState !== "ON_TRACK").length}</strong><small>Cần ưu tiên xử lý</small></article><article><span>Giá trị ước tính</span><strong>{VND.format(data.items.reduce((sum, item) => sum + (item.estimatedValueVnd ?? 0), 0))}</strong><small>Không phải doanh thu đã xác nhận</small></article></section>

    <section className="admin-list-card"><div className="admin-status-tabs" role="tablist" aria-label="Lọc theo trạng thái">{data.statusCounts.map((item) => <button className={status === item.status ? "active" : ""} type="button" role="tab" aria-selected={status === item.status} onClick={() => setStatus(item.status)} key={item.status}>{item.label}<b>{item.count}</b></button>)}</div>
      <div className="admin-list-toolbar"><label className="admin-table-search"><span>⌕</span><input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Tìm mã yêu cầu, doanh nghiệp, mã số thuế…" /></label><label><span className="sr-only">Người phụ trách</span><select value={owner} onChange={(event) => setOwner(event.target.value)}><option value="ALL">Tất cả người phụ trách</option><option value="UNASSIGNED">Chưa phân công</option>{data.owners.map((item) => <option value={item.employeeId} key={item.employeeId}>{item.displayName}</option>)}</select></label><label><span className="sr-only">Trạng thái SLA</span><select value={sla} onChange={(event) => setSla(event.target.value)}><option value="ALL">Tất cả SLA</option><option value="ON_TRACK">Đúng hạn</option><option value="AT_RISK">Sắp quá hạn</option><option value="OVERDUE">Quá hạn</option></select></label><button className="admin-filter-reset" type="button" onClick={() => { setQuery(""); setOwner("ALL"); setSla("ALL"); setStatus("ALL"); }}>Đặt lại</button></div>
      <div className="admin-table-meta"><span>Hiển thị <strong>{filtered.length}</strong> yêu cầu</span><span>Dữ liệu theo scope quyền hiện tại</span></div>
      <div className="admin-table-wrap"><table className="admin-table"><thead><tr><th>Yêu cầu</th><th>Doanh nghiệp</th><th>Thời gian</th><th>Quy mô</th><th>Trạng thái</th><th>SLA</th><th>Phụ trách</th><th>Phiên bản</th><th aria-label="Tùy chọn" /></tr></thead><tbody>{filtered.map((row) => <tr key={row.requestId}><td><strong>{row.requestNumber}</strong><small>{row.requestId}</small></td><td><strong>{row.companyName}</strong><small>MST {row.taxCode}</small></td><td><span>{DATE_TIME.format(new Date(row.submittedAt))}</span></td><td><strong>{row.totalQuantity} set</strong><small>{row.lineCount} dòng SKU</small></td><td><StatusBadge row={row} /></td><td><span className={`admin-sla sla-${row.slaState.toLowerCase()}`}>● {row.slaLabel}</span></td><td>{row.owner ? <span>{row.owner.displayName}</span> : <span className="admin-unassigned">Chưa phân công</span>}</td><td>{row.currentQuoteVersion ? <><strong>V{row.currentQuoteVersion}</strong>{row.quoteExpiresAt ? <small>Đến {DATE.format(new Date(row.quoteExpiresAt))}</small> : null}</> : <span>—</span>}</td><td className="admin-action-cell"><button type="button" aria-label={`Tùy chọn ${row.requestNumber}`} onClick={() => setMenuId((current) => current === row.requestId ? "" : row.requestId)}>•••</button>{menuId === row.requestId ? <div className="admin-row-menu">{STATUS_ACTIONS[row.status].filter((action) => action === "VIEW" || allowed(action)).map((action) => <button type="button" className={action === "REJECT" || action === "WITHDRAW" ? "danger" : ""} key={action} onClick={() => void openModal(action, row)}>{ACTION_LABELS[action]}</button>)}</div> : null}</td></tr>)}</tbody></table>{!filtered.length ? <div className="admin-empty"><span>⌕</span><h3>Không có yêu cầu phù hợp</h3><p>Hãy thay đổi trạng thái, từ khóa hoặc bộ lọc hiện tại.</p></div> : null}</div>
      <footer className="admin-pagination"><span>Trang 1 / 1</span><div><button disabled>‹</button><button className="active">1</button><button disabled>›</button></div></footer>
    </section>

    {toast ? <div className="admin-toast" role="status">✓ {toast}</div> : null}
    {modal ? <ActionModal modal={modal} detail={detail} history={history} detailLoading={detailLoading} owners={data.owners} error={actionError} conflict={actionConflict} pending={actionPending} onClose={() => setModal(null)} onReload={() => { setModal(null); void load(); }} onAct={act} /> : null}
  </div>;
}

function ActionModal({ modal, detail, history, detailLoading, owners, error, conflict, pending, onClose, onReload, onAct }: { modal: { action: ModalAction; row: AdminB2BRequestSummary }; detail: AdminB2BRequestDetail | null; history: AdminB2BQuoteVersionHistory | null; detailLoading: boolean; owners: AdminB2BList["owners"]; error: string; conflict: boolean; pending: boolean; onClose: () => void; onReload: () => void; onAct: (input: Omit<AdminB2BActionInput, "idempotencyKey" | "expectedRevision">) => Promise<void> }) {
  const [reason, setReason] = useState("");
  const [assigneeId, setAssigneeId] = useState(modal.row.owner?.employeeId ?? owners[0]?.employeeId ?? "");
  const [discountPercent, setDiscountPercent] = useState(detail?.quote?.discountPercent ?? 18);
  const [customizationVnd, setCustomizationVnd] = useState(detail?.quote?.customizationVnd ?? 0);
  const [shippingVnd, setShippingVnd] = useState(detail?.quote?.shippingVnd ?? 0);
  const [vatPercent, setVatPercent] = useState(detail?.quote?.vatPercent ?? 8);
  const [expiresAt, setExpiresAt] = useState("");
  const [terms, setTerms] = useState("Đặt cọc 30% khi ký điện tử; phần còn lại thanh toán sau nghiệm thu theo hợp đồng.");
  const [invoiceSnapshotConfirmed, setInvoiceSnapshotConfirmed] = useState(false);
  const [availabilityCheckAcknowledged, setAvailabilityCheckAcknowledged] = useState(false);
  const subtotal = detail?.items.reduce((sum, item) => sum + item.catalogPriceVnd * item.quantity, 0) ?? 0;
  const grandTotal = (subtotal * (1 - discountPercent / 100) + customizationVnd + shippingVnd) * (1 + vatPercent / 100);
  function submit(event: FormEvent) { event.preventDefault(); if (modal.action === "ASSIGN") void onAct({ action: "ASSIGN", assigneeId }); else if (modal.action === "REQUEST_INFO") void onAct({ action: "REQUEST_INFO", reason }); else if (modal.action === "WITHDRAW") void onAct({ action: "WITHDRAW", reason }); else if (modal.action === "REJECT") void onAct({ action: "REJECT", reason }); else if (modal.action === "ISSUE") void onAct({ action: "ISSUE", reason: reason || "Phát hành báo giá sau khi đã rà soát đầy đủ." }); else if (modal.action === "CONVERT_ORDER") void onAct({ action: "CONVERT_ORDER", invoiceSnapshotConfirmed, availabilityCheckAcknowledged }); else if (modal.action === "CREATE_DRAFT") void onAct({ action: "CREATE_DRAFT", quote: { discountPercent, customizationVnd, shippingVnd, vatPercent, expiresAt, terms } }); }
  if (modal.action === "VIEW") return <Modal title={`Chi tiết ${modal.row.requestNumber}`} subtitle={modal.row.companyName} onClose={onClose} wide>{detailLoading ? <div className="admin-modal-loading">Đang tải chi tiết…</div> : detail ? <DetailContent detail={detail} /> : <div className="admin-inline-error">{error}</div>}</Modal>;
  if (modal.action === "VERSION_HISTORY") return <Modal title={`Lịch sử ${modal.row.requestNumber}`} subtitle={`${modal.row.companyName} · Snapshot theo từng Quote Version`} onClose={onClose} wide>{detailLoading ? <div className="admin-modal-loading">Đang tải lịch sử phiên bản…</div> : history ? <VersionHistoryContent history={history} /> : <div className="admin-inline-error">{error}</div>}</Modal>;
  return <Modal title={ACTION_LABELS[modal.action]} subtitle={`${modal.row.requestNumber} · ${modal.row.companyName}`} onClose={onClose} wide={modal.action === "CREATE_DRAFT"}><form className="admin-action-form" onSubmit={submit}>
    {detailLoading ? <div className="admin-modal-loading">Đang tải dữ liệu…</div> : null}
    {modal.action === "ASSIGN" ? <label><span>Nhân viên phụ trách *</span><select value={assigneeId} onChange={(event) => setAssigneeId(event.target.value)}>{owners.map((item) => <option value={item.employeeId} key={item.employeeId}>{item.displayName}</option>)}</select><small>Phân công thay đổi owner xử lý, không tự thay đổi trạng thái nghiệp vụ.</small></label> : null}
    {modal.action === "REQUEST_INFO" || modal.action === "WITHDRAW" || modal.action === "REJECT" ? <label><span>{modal.action === "REJECT" ? "Lý do từ chối" : modal.action === "WITHDRAW" ? "Lý do thu hồi báo giá" : "Thông tin cần doanh nghiệp bổ sung"} *</span><textarea value={reason} onChange={(event) => setReason(event.target.value)} placeholder="Nhập nội dung rõ ràng để lưu lịch sử và gửi cho doanh nghiệp…" /><small>Tối thiểu 10 ký tự. Ghi chú nội bộ không được đưa vào nội dung này.</small></label> : null}
    {modal.action === "ISSUE" ? <div className="admin-confirm-box"><strong>Phát hành Quote Version {detail?.quote?.version ?? modal.row.currentQuoteVersion ?? "hiện tại"}?</strong><p>Phiên bản đã phát hành trở thành snapshot bất biến và Customer có thể xem/chấp thuận đúng phiên bản này.</p><label className="admin-checkbox"><input type="checkbox" required /> Tôi đã kiểm tra giá, thời hạn, thuế, tệp và điều khoản.</label></div> : null}
    {modal.action === "WITHDRAW" ? <div className="admin-confirm-box danger"><strong>Thu hồi Quote Version {modal.row.currentQuoteVersion ?? "hiện tại"}?</strong><p>Khách hàng sẽ không thể chấp thuận phiên bản này. Lịch sử và snapshot đã phát hành vẫn được giữ để truy vết.</p></div> : null}
    {modal.action === "CONVERT_ORDER" ? <div className="admin-confirm-box"><strong>Chuyển Quote Version {modal.row.currentQuoteVersion ?? "đã chấp thuận"} thành Order B2B?</strong><p>Order mới giữ snapshot thương mại và không tự chuyển sang trạng thái đã thanh toán.</p><label className="admin-checkbox"><input type="checkbox" required checked={availabilityCheckAcknowledged} onChange={(event) => setAvailabilityCheckAcknowledged(event.target.checked)} /> Tôi đã kiểm tra lại khả năng đáp ứng/tồn kho theo chính sách.</label><label className="admin-checkbox"><input type="checkbox" required checked={invoiceSnapshotConfirmed} onChange={(event) => setInvoiceSnapshotConfirmed(event.target.checked)} /> Tôi xác nhận thông tin pháp lý và hóa đơn sẽ được chốt thành snapshot của Order.</label></div> : null}
    {modal.action === "CREATE_DRAFT" && detail ? <div className="admin-quote-builder"><div className="admin-builder-context"><strong>{detail.items.length} dòng sản phẩm · {detail.totalQuantity} set</strong><span>Giá catalog tạm tính: {VND.format(subtotal)}</span></div><div className="admin-form-grid"><label><span>Chiết khấu (%)</span><input type="number" min="0" max="100" value={discountPercent} onChange={(event) => setDiscountPercent(Number(event.target.value))} /></label><label><span>Phí tùy biến</span><input type="number" min="0" value={customizationVnd} onChange={(event) => setCustomizationVnd(Number(event.target.value))} /></label><label><span>Phí vận chuyển</span><input type="number" min="0" value={shippingVnd} onChange={(event) => setShippingVnd(Number(event.target.value))} /></label><label><span>VAT (%)</span><input type="number" min="0" max="100" value={vatPercent} onChange={(event) => setVatPercent(Number(event.target.value))} /></label><label><span>Hiệu lực đến *</span><input type="date" required value={expiresAt} onChange={(event) => setExpiresAt(event.target.value)} /></label></div><label><span>Điều khoản thương mại *</span><textarea value={terms} onChange={(event) => setTerms(event.target.value)} /></label><div className="admin-builder-total"><span>Tổng giá trị dự kiến</span><strong>{VND.format(grandTotal)}</strong><small>Preview này chỉ là tính toán UI; backend phải tính và xác nhận lại.</small></div></div> : null}
    {error ? <div className={`admin-inline-error ${conflict ? "conflict" : ""}`} role="alert"><span>{error}</span>{conflict ? <button type="button" onClick={onReload}>Tải dữ liệu mới để đối chiếu</button> : null}</div> : null}<footer><button className="admin-secondary-button" type="button" onClick={onClose}>Hủy</button><button className={`admin-primary-button ${modal.action === "REJECT" || modal.action === "WITHDRAW" ? "danger" : ""}`} disabled={pending || detailLoading || conflict}>{pending ? "Đang xử lý…" : ACTION_LABELS[modal.action]}</button></footer>
  </form></Modal>;
}

function VersionHistoryContent({ history }: { history: AdminB2BQuoteVersionHistory }) {
  const [selectedVersion, setSelectedVersion] = useState(history.items[0]?.version ?? 0);
  const selected = history.items.find((item) => item.version === selectedVersion) ?? history.items[0];
  if (!selected) return <div className="admin-empty"><span>▤</span><h3>Chưa có phiên bản báo giá</h3><p>Hãy lập bản nháp đầu tiên từ menu hành động của yêu cầu.</p></div>;
  return <div className="admin-version-history">
    <aside className="admin-version-list"><header><strong>{history.items.length} phiên bản</strong><small>Phiên bản mới nhất ở trên</small></header>{history.items.map((item) => <button type="button" className={selected.version === item.version ? "active" : ""} onClick={() => setSelectedVersion(item.version)} key={item.quoteId}><span><b>Version {item.version}</b><em className={`version-status status-${item.status.toLowerCase()}`}>{item.statusLabel}</em></span><small>{item.issuedAt ? `Phát hành ${DATE_TIME.format(new Date(item.issuedAt))}` : `Tạo ${DATE_TIME.format(new Date(item.createdAt))}`}</small><small>{item.supersedesVersion ? `Thay thế Version ${item.supersedesVersion}` : "Phiên bản đầu tiên"}</small></button>)}</aside>
    <article className="admin-version-preview">
      <header><div><span className="eyebrow">Customer Quote Preview</span><h3>{selected.quoteNumber} · Version {selected.version}</h3><p>{selected.organization.legalName} · MST {selected.organization.taxCode}</p></div><div className="admin-snapshot-state"><em className={`version-status status-${selected.status.toLowerCase()}`}>{selected.statusLabel}</em><b>{selected.immutable ? "🔒 Snapshot bất biến" : "✎ Bản nháp có thể chỉnh sửa"}</b></div></header>
      <div className="admin-version-meta"><span><small>Người tạo</small><b>{selected.createdBy}</b></span><span><small>Phát hành</small><b>{selected.issuedAt ? DATE_TIME.format(new Date(selected.issuedAt)) : "Chưa phát hành"}</b></span><span><small>Hiệu lực đến</small><b>{DATE.format(new Date(selected.expiresAt))}</b></span><span><small>Đại diện</small><b>{selected.contact.displayName}</b></span></div>
      <section><h4>Dòng báo giá</h4><table><thead><tr><th>SKU/Sản phẩm</th><th>Quy cách</th><th>SL</th><th>Đơn giá B2B</th><th>Thành tiền</th></tr></thead><tbody>{selected.items.map((item) => <tr key={item.skuId}><td><strong>{item.name}</strong><small>{item.skuId}</small></td><td>{item.variant}</td><td>{item.quantity}</td><td>{VND.format(item.unitPriceVnd)}</td><td>{VND.format(item.lineTotalVnd)}</td></tr>)}</tbody></table></section>
      <div className="admin-version-columns"><section><h4>Điều khoản đã chốt</h4>{selected.terms.map((term) => <div className="admin-version-term" key={term.title}><b>{term.title}</b><p>{term.description}</p></div>)}<div className="admin-payment-summary"><span>Thanh toán</span><b>{selected.payment.bankName} · {selected.payment.accountNumberMasked}</b><small>{selected.payment.transferContent}</small></div></section><section className="admin-version-totals"><h4>Quyết toán phiên bản</h4><div><span>Tiền hàng</span><b>{VND.format(selected.totals.merchandiseVnd)}</b></div><div><span>Chiết khấu</span><b>−{VND.format(selected.totals.discountVnd)}</b></div><div><span>Tùy biến</span><b>{VND.format(selected.totals.customizationVnd)}</b></div><div><span>VAT</span><b>{VND.format(selected.totals.vatVnd)}</b></div><strong><small>Tổng giá trị</small>{VND.format(selected.totals.grandTotalVnd)}</strong><div><span>Đặt cọc {selected.totals.depositPercent}%</span><b>{VND.format(selected.totals.depositVnd)}</b></div><div><span>Còn lại</span><b>{VND.format(selected.totals.remainingVnd)}</b></div></section></div>
      <footer>{selected.immutable ? "Phiên bản đã phát hành chỉ được đọc; thay đổi giá hoặc điều khoản phải tạo Quote Version mới." : "Preview bản nháp chưa được hiển thị cho Customer và có thể thay đổi trước khi phát hành."}</footer>
    </article>
  </div>;
}

function DetailContent({ detail }: { detail: AdminB2BRequestDetail }) {
  return <div className="admin-detail"><div className="admin-detail-summary"><StatusBadge row={detail} /><span className={`admin-sla sla-${detail.slaState.toLowerCase()}`}>● {detail.slaLabel}</span><span>{detail.owner?.displayName ?? "Chưa phân công"}</span></div><div className="admin-detail-grid"><section><h3>Doanh nghiệp</h3><dl><div><dt>Pháp nhân</dt><dd>{detail.company.legalName}</dd></div><div><dt>Mã số thuế</dt><dd>{detail.company.taxCode}</dd></div><div><dt>Đại diện</dt><dd>{detail.requester.displayName} · {detail.requester.title}</dd></div><div><dt>Liên hệ</dt><dd>{detail.requester.phone}<br />{detail.requester.email}</dd></div><div><dt>Địa chỉ hóa đơn</dt><dd>{detail.company.invoiceAddress}</dd></div></dl></section><section><h3>Nhu cầu</h3><dl><div><dt>Mục đích</dt><dd>{detail.purposeLabel}</dd></div><div><dt>Ngày giao dự kiến</dt><dd>{DATE.format(new Date(detail.requestedDeliveryDate))}</dd></div><div><dt>Khu vực giao</dt><dd>{detail.deliveryLocation}</dd></div><div><dt>Ghi chú</dt><dd>{detail.notes || "Không có"}</dd></div></dl></section></div><section className="admin-detail-section"><h3>Dòng sản phẩm</h3><table><thead><tr><th>SKU/Sản phẩm</th><th>Quy cách</th><th>Số lượng</th><th>Giá catalog</th></tr></thead><tbody>{detail.items.map((item) => <tr key={item.skuId}><td><strong>{item.name}</strong><small>{item.skuId}</small></td><td>{item.variant}</td><td>{item.quantity} set</td><td>{VND.format(item.catalogPriceVnd)}</td></tr>)}</tbody></table></section><div className="admin-detail-grid"><section><h3>Tùy biến</h3><ul>{detail.customizations.map((item) => <li key={item}>✓ {item}</li>)}</ul></section><section><h3>Tệp doanh nghiệp</h3>{detail.files.map((file) => <div className="admin-file-row" key={file.fileId}><span>▧</span><div><strong>{file.fileName}</strong><small>{file.mediaType}</small></div><em>{file.scanStatus === "SAFE" ? "An toàn" : file.scanStatus}</em></div>)}</section></div><section className="admin-detail-section"><h3>Lịch sử hoạt động</h3><div className="admin-activity">{detail.activity.map((item) => <article key={`${item.occurredAt}-${item.description}`}><span /><div><strong>{item.description}</strong><small>{item.actorLabel} · {DATE_TIME.format(new Date(item.occurredAt))}</small></div></article>)}</div></section></div>;
}
