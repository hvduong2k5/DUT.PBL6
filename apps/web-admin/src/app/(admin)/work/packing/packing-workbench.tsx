"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useAdminSession } from "@/components/auth/admin-session-provider";
import type { PackingChecklistState, PackingTaskDetail, PackingTaskList, PackingTaskSummary } from "@/lib/packing/types";
import { AdminPackingApiError } from "@/lib/packing/types";
import { adminPackingService } from "@/services/admin-packing-service";

const DATE_TIME = new Intl.DateTimeFormat("vi-VN", { dateStyle: "short", timeStyle: "short", timeZone: "Asia/Ho_Chi_Minh" });

function Status({ code, label, prefix = "packing" }: { code: string; label: string; prefix?: string }) {
  return <span className={`admin-status ${prefix}-${code.toLocaleLowerCase()}`}>{label}</span>;
}

function messageOf(error: unknown) {
  return error instanceof AdminPackingApiError ? error.message : "Không thể xử lý Workbench đóng gói.";
}

function ready(detail: PackingTaskDetail) {
  const missing: string[] = [];
  if (!detail.prerequisites.every((item) => item.ready)) missing.push("Điều kiện đầu vào chưa sẵn sàng");
  if (!detail.pickLines.every((line) => line.allocations.length > 0 && line.allocations.every((allocation) => allocation.eligible))) missing.push("Batch phân bổ chưa hợp lệ");
  const incomplete = detail.checklist.filter((item) => item.required && item.state !== "PASSED");
  if (incomplete.length) missing.push(`${incomplete.length} bước checklist bắt buộc chưa đạt`);
  if (detail.status !== "IN_PROGRESS") missing.push("Task không ở trạng thái đang đóng gói");
  return { ready: missing.length === 0, label: missing.length ? "Chưa đủ điều kiện hoàn tất" : "Đủ điều kiện hoàn tất", missingConditions: missing };
}

export function PackingWorkbench() {
  const { session, hasPermission } = useAdminSession();
  const [data, setData] = useState<PackingTaskList | null>(null);
  const [selectedId, setSelectedId] = useState("");
  const [detail, setDetail] = useState<PackingTaskDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [detailLoading, setDetailLoading] = useState(false);
  const [error, setError] = useState("");
  const [detailError, setDetailError] = useState("");
  const [savingItem, setSavingItem] = useState("");
  const [notes, setNotes] = useState<Record<string, string>>({});
  const [completeOpen, setCompleteOpen] = useState(false);
  const [completing, setCompleting] = useState(false);
  const [toast, setToast] = useState("");

  const openTask = useCallback(async (task: PackingTaskSummary) => {
    setSelectedId(task.taskId); setDetailLoading(true); setDetailError("");
    try { setDetail(await adminPackingService.assignedTask(task.taskId)); }
    catch (cause) { setDetail(null); setDetailError(messageOf(cause)); }
    finally { setDetailLoading(false); }
  }, []);

  const load = useCallback(async () => {
    setLoading(true); setError("");
    try {
      const result = await adminPackingService.assignedTasks();
      setData(result);
      const first = result.items.find((item) => item.status === "IN_PROGRESS") ?? result.items.find((item) => item.status !== "COMPLETED") ?? result.items[0];
      if (first) await openTask(first); else { setSelectedId(""); setDetail(null); }
    } catch (cause) { setError(messageOf(cause)); setData(null); }
    finally { setLoading(false); }
  }, [openTask]);

  useEffect(() => { void Promise.resolve().then(load); }, [load]);
  useEffect(() => { if (!toast) return; const timer = window.setTimeout(() => setToast(""), 3500); return () => window.clearTimeout(timer); }, [toast]);

  const metrics = useMemo(() => {
    const items = data?.items ?? [];
    return {
      total: items.filter((item) => item.status !== "COMPLETED").length,
      active: items.filter((item) => item.status === "IN_PROGRESS").length,
      urgent: items.filter((item) => item.status !== "COMPLETED" && (item.slaState === "AT_RISK" || item.slaState === "OVERDUE")).length,
      completed: items.filter((item) => item.status === "COMPLETED").length
    };
  }, [data]);

  async function updateChecklist(itemId: string, state: Exclude<PackingChecklistState, "PENDING">) {
    if (!detail) return;
    const note = (notes[itemId] ?? "").trim();
    if (state === "FAILED" && note.length < 5) { setDetailError("Hãy nhập ghi chú ít nhất 5 ký tự khi đánh dấu Không đạt."); return; }
    setSavingItem(itemId); setDetailError("");
    try {
      const result = await adminPackingService.updateChecklist(detail.taskId, itemId, { state, note: note || undefined, expectedRevision: detail.revision, idempotencyKey: crypto.randomUUID() });
      setDetail((current) => {
        if (!current) return current;
        const next = { ...current, revision: result.revision, checklist: current.checklist.map((item) => item.itemId === itemId ? result.item : item) };
        return { ...next, completionReadiness: ready(next) };
      });
      setToast(result.message);
    } catch (cause) { setDetailError(messageOf(cause)); }
    finally { setSavingItem(""); }
  }

  async function completeTask() {
    if (!detail) return;
    setCompleting(true); setDetailError("");
    try {
      const result = await adminPackingService.completeTask(detail.taskId, { expectedRevision: detail.revision, idempotencyKey: crypto.randomUUID() });
      setDetail((current) => current ? { ...current, status: result.status, statusLabel: result.statusLabel, updatedAt: result.completedAt, revision: result.revision, completionReadiness: { ready: false, label: "Task đã hoàn tất", missingConditions: [] }, history: [...current.history, { eventId: `complete-${result.revision}`, label: "Hoàn tất Packing Task", occurredAt: result.completedAt, actorLabel: result.completedBy, detail: "Kết quả đã được chuyển sang quy trình Order và Shipping." }] } : current);
      setData((current) => current ? { ...current, items: current.items.map((item) => item.taskId === result.taskId ? { ...item, status: result.status, statusLabel: result.statusLabel, slaState: "COMPLETE", slaLabel: "Đã hoàn tất", updatedAt: result.completedAt, revision: result.revision } : item) } : current);
      setCompleteOpen(false); setToast(result.message);
    } catch (cause) { setDetailError(messageOf(cause)); setCompleteOpen(false); }
    finally { setCompleting(false); }
  }

  if (loading) return <div className="admin-page"><section className="admin-page-skeleton"><div /><div /></section></div>;
  if (error) return <div className="admin-page"><section className="admin-error-state"><span>!</span><h1>Không thể mở Workbench đóng gói</h1><p>{error}</p><button className="admin-primary-button" onClick={() => void load()}>Thử lại</button></section></div>;

  return <div className="admin-page packing-workbench-page">
    <header className="admin-page-header"><div><nav>Admin <span>›</span> Công việc của tôi <span>›</span> Đóng gói</nav><h1>Workbench đóng gói</h1><p>Chỉ hiển thị Packing Task được phân công cho {session?.employee.displayName}. Thực hiện theo SKU, Batch và checklist của Epic 10.</p></div><div className="admin-page-actions"><button className="admin-secondary-button" type="button" onClick={() => void load()}>↻ Làm mới</button></div></header>
    <section className="admin-summary-grid"><article><span>Việc đang chờ</span><strong>{metrics.total}</strong><small>Task được phân công và chưa hoàn tất</small></article><article><span>Đang đóng gói</span><strong>{metrics.active}</strong><small>Task có thể tiếp tục thao tác</small></article><article><span>Cần ưu tiên</span><strong>{metrics.urgent}</strong><small>Sắp hoặc đã quá SLA</small></article><article><span>Đã hoàn tất</span><strong>{metrics.completed}</strong><small>Trong hàng đợi công việc hiện tại</small></article></section>
    <section className="packing-workbench-layout">
      <aside className="packing-work-queue"><header><div><h2>Việc được giao</h2><p>Sắp theo mức SLA do upstream cung cấp</p></div><b>{data?.items.length ?? 0}</b></header><div>{data?.items.map((task) => <button type="button" className={selectedId === task.taskId ? "active" : ""} onClick={() => void openTask(task)} key={task.taskId}><span><strong>{task.taskNumber}</strong><small>{task.orderNumber} · {task.lineCount} dòng / {task.totalQuantity} SP</small></span><span><Status code={task.status} label={task.statusLabel} /><small>{task.dueAt ? DATE_TIME.format(new Date(task.dueAt)) : "Không có hạn"}</small></span><Status code={task.slaState} label={task.slaLabel} prefix="sla" /></button>)}</div>{!data?.items.length ? <p className="packing-work-empty">Bạn chưa có Packing Task được phân công.</p> : null}</aside>
      <main className="packing-work-detail">
        {detailLoading ? <div className="admin-modal-loading">Đang tải hướng dẫn lấy hàng và checklist…</div> : detailError && !detail ? <div className="admin-inline-error">{detailError}</div> : detail ? <>
          <header className="packing-work-detail-header"><div><small>{detail.orderNumber}</small><h2>{detail.taskNumber}</h2><p>{detail.orderSourceLabel} · Revision {detail.revision}</p></div><div><Status code={detail.status} label={detail.statusLabel} /><Status code={detail.priority} label={detail.priorityLabel} prefix="priority" /><Status code={detail.slaState} label={detail.slaLabel} prefix="sla" /></div></header>
          {detail.blockedReason ? <section className="admin-packing-blocker"><strong>Task đang bị chặn</strong><p>{detail.blockedReason}</p></section> : null}
          <section><header className="admin-section-heading"><div><h3>1. Kiểm tra điều kiện đầu vào</h3><p>Workbench chỉ đọc trạng thái Payment, Order và phân bổ Inventory.</p></div></header><div className="admin-packing-prerequisites">{detail.prerequisites.map((item) => <article className={item.ready ? "ready" : "blocked"} key={item.code}><span>{item.ready ? "✓" : "!"}</span><div><strong>{item.label}</strong><small>{item.detail}</small></div></article>)}</div></section>
          <section><header className="admin-section-heading"><div><h3>2. Lấy đúng SKU và Batch</h3><p>Dữ liệu theo snapshot Order và phân bổ của Inventory; nhân viên không tự đổi Batch tại đây.</p></div></header><div className="admin-pick-list">{detail.pickLines.map((line) => <article key={line.lineId}><header><div><strong>{line.productName}</strong><small>{line.skuCode} · {line.skuLabel}</small></div><b>× {line.quantity}</b></header><div>{line.allocations.map((allocation) => <span className={allocation.eligible ? "eligible" : "ineligible"} key={allocation.allocationId}><strong>{allocation.batchCode}</strong><small>{allocation.quantity} sản phẩm · {allocation.locationLabel ?? "Chưa có vị trí"} · HSD {allocation.expiresAt ? DATE_TIME.format(new Date(allocation.expiresAt)) : "chưa có"}</small><em>{allocation.eligibilityLabel}</em></span>)}</div></article>)}</div></section>
          <section><header className="admin-section-heading"><div><h3>3. Xác nhận checklist</h3><p>Mỗi kết quả lưu người thực hiện và thời điểm. Không đạt phải có ghi chú sai lệch.</p></div><strong>{detail.checklist.filter((item) => item.state === "PASSED").length}/{detail.checklist.length} đạt</strong></header><div className="packing-work-checklist">{detail.checklist.map((item) => <article className={item.state.toLocaleLowerCase()} key={item.itemId}><div className="packing-work-check-title"><Status code={item.state} label={item.stateLabel} prefix="check" /><div><strong>{item.label}{item.required ? " *" : ""}</strong><small>{item.confirmedBy ? `${item.confirmedBy} · ${item.confirmedAt ? DATE_TIME.format(new Date(item.confirmedAt)) : ""}` : "Chưa xác nhận"}</small></div></div><textarea aria-label={`Ghi chú ${item.label}`} value={notes[item.itemId] ?? item.note ?? ""} onChange={(event) => setNotes((current) => ({ ...current, [item.itemId]: event.target.value }))} placeholder="Ghi chú nếu có; bắt buộc khi Không đạt" disabled={detail.status !== "IN_PROGRESS" || savingItem === item.itemId} /><div className="packing-work-check-actions"><button type="button" disabled={!hasPermission("PACKING_CHECKLIST_UPDATE") || detail.status !== "IN_PROGRESS" || Boolean(savingItem)} onClick={() => void updateChecklist(item.itemId, "FAILED")}>! Không đạt</button><button className="pass" type="button" disabled={!hasPermission("PACKING_CHECKLIST_UPDATE") || detail.status !== "IN_PROGRESS" || Boolean(savingItem)} onClick={() => void updateChecklist(item.itemId, "PASSED")}>{savingItem === item.itemId ? "Đang lưu…" : "✓ Đạt"}</button></div></article>)}</div></section>
          <section className={`packing-work-completion ${detail.completionReadiness.ready ? "ready" : "blocked"}`}><div><small>Bước cuối</small><h3>{detail.completionReadiness.label}</h3>{detail.completionReadiness.missingConditions.length ? <ul>{detail.completionReadiness.missingConditions.map((item) => <li key={item}>{item}</li>)}</ul> : <p>Tất cả điều kiện MVP đã đạt. Bằng chứng media thuộc Giai đoạn 2 và không được giả lập là đã upload trong luồng này.</p>}</div><button className="admin-primary-button" type="button" disabled={!hasPermission("PACKING_TASK_COMPLETE") || !detail.completionReadiness.ready || completing} onClick={() => setCompleteOpen(true)}>Hoàn tất đóng gói</button></section>
          {detailError ? <div className="admin-inline-error">{detailError}</div> : null}
        </> : <div className="admin-empty"><h3>Chọn một Packing Task</h3><p>Chi tiết thao tác sẽ hiển thị tại đây.</p></div>}
      </main>
    </section>
    {completeOpen && detail ? <div className="admin-modal-backdrop" role="presentation" onMouseDown={() => !completing && setCompleteOpen(false)}><section className="admin-modal packing-complete-modal" role="dialog" aria-modal="true" aria-labelledby="packing-complete-title" onMouseDown={(event) => event.stopPropagation()}><header><div><h2 id="packing-complete-title">Xác nhận hoàn tất đóng gói</h2><p>{detail.taskNumber} · {detail.orderNumber}</p></div><button type="button" disabled={completing} onClick={() => setCompleteOpen(false)}>×</button></header><div><p>Hệ thống sẽ ghi nhận người thao tác, chống gửi lặp và chuyển kết quả sang quy trình Order/Shipping. Packing không tự sửa tồn kho hoặc tạo Shipment.</p></div><footer><button className="admin-secondary-button" type="button" disabled={completing} onClick={() => setCompleteOpen(false)}>Quay lại kiểm tra</button><button className="admin-primary-button" type="button" disabled={completing} onClick={() => void completeTask()}>{completing ? "Đang hoàn tất…" : "Xác nhận hoàn tất"}</button></footer></section></div> : null}
    {toast ? <div className="admin-toast" role="status">{toast}</div> : null}
  </div>;
}
