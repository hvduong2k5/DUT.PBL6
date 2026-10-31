"use client";

import { useCallback, useEffect, useMemo, useState, type FormEvent } from "react";
import { useAdminSession } from "@/components/auth/admin-session-provider";
import type { InventoryOperationResult, WarehouseWorkbenchContext } from "@/lib/inventory/types";
import { AdminInventoryApiError } from "@/lib/inventory/types";
import { adminInventoryService } from "@/services/admin-inventory-service";

type Operation = "RECEIPT" | "ISSUE" | "ADJUSTMENT";
const DATE_TIME = new Intl.DateTimeFormat("vi-VN", { dateStyle: "short", timeStyle: "short", timeZone: "Asia/Ho_Chi_Minh" });
const DATE = new Intl.DateTimeFormat("vi-VN", { day: "2-digit", month: "2-digit", year: "numeric" });
const QUANTITY = new Intl.NumberFormat("vi-VN");

function messageOf(error: unknown) { return error instanceof AdminInventoryApiError ? error.message : "Không thể xử lý nghiệp vụ kho."; }

export function WarehouseWorkbench() {
  const { session, hasPermission } = useAdminSession();
  const [data, setData] = useState<WarehouseWorkbenchContext | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [operation, setOperation] = useState<Operation>("RECEIPT");
  const [skuId, setSkuId] = useState("");
  const [batchId, setBatchId] = useState("");
  const [batchMode, setBatchMode] = useState<"EXISTING" | "NEW">("EXISTING");
  const [batchCode, setBatchCode] = useState("");
  const [manufacturingDate, setManufacturingDate] = useState("");
  const [expiresAt, setExpiresAt] = useState("");
  const [quantity, setQuantity] = useState("1");
  const [reference, setReference] = useState("");
  const [purpose, setPurpose] = useState<"ORDER" | "APPROVED_NON_ORDER">("ORDER");
  const [adjustmentKind, setAdjustmentKind] = useState<"DAMAGE" | "LOSS" | "COUNT">("DAMAGE");
  const [reason, setReason] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [operationError, setOperationError] = useState("");
  const [result, setResult] = useState<InventoryOperationResult | null>(null);

  const load = useCallback(async () => {
    setLoading(true); setError("");
    try {
      const next = await adminInventoryService.workbench();
      setData(next);
      setSkuId(next.skus[0]?.skuId ?? "");
      setBatchId(next.skus[0]?.batches[0]?.batchId ?? "");
    } catch (cause) { setData(null); setError(messageOf(cause)); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { void Promise.resolve().then(load); }, [load]);
  const selectedSku = useMemo(() => data?.skus.find((item) => item.skuId === skuId) ?? null, [data, skuId]);
  const selectedBatch = selectedSku?.batches.find((item) => item.batchId === batchId) ?? null;
  const metrics = useMemo(() => ({
    available: data?.skus.reduce((sum, item) => sum + item.balance.available, 0) ?? 0,
    batches: data?.skus.reduce((sum, item) => sum + item.batches.length, 0) ?? 0,
    expiring: data?.skus.reduce((sum, item) => sum + item.batches.filter((batch) => batch.expiryState !== "SAFE").length, 0) ?? 0,
    pending: data?.pendingApprovalCount ?? 0
  }), [data]);
  const canSubmit = operation === "RECEIPT" ? hasPermission("INVENTORY_RECEIPT_CREATE") : operation === "ISSUE" ? hasPermission("INVENTORY_ISSUE_CREATE") : hasPermission("INVENTORY_ADJUSTMENT_CREATE");

  function changeSku(nextSkuId: string) { const nextSku = data?.skus.find((item) => item.skuId === nextSkuId); setSkuId(nextSkuId); setBatchId(nextSku?.batches[0]?.batchId ?? ""); setResult(null); setOperationError(""); }
  function changeOperation(next: Operation) { setOperation(next); setResult(null); setOperationError(""); setReason(""); setReference(""); setQuantity("1"); }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selectedSku) return;
    setSubmitting(true); setOperationError(""); setResult(null);
    try {
      const common = { skuId: selectedSku.skuId, expectedRevision: selectedSku.revision, idempotencyKey: crypto.randomUUID() };
      const operationResult = operation === "RECEIPT"
        ? await adminInventoryService.receive({ ...common, batchMode, batchId: batchMode === "EXISTING" ? batchId : undefined, batchCode: batchMode === "NEW" ? batchCode : undefined, manufacturingDate: batchMode === "NEW" ? manufacturingDate : undefined, expiresAt: batchMode === "NEW" ? expiresAt : undefined, quantity: Number(quantity), sourceReference: reference })
        : operation === "ISSUE"
          ? await adminInventoryService.issue({ ...common, batchId, purpose, quantity: Number(quantity), reference, reason: reason || undefined })
          : await adminInventoryService.adjust({ ...common, batchId, kind: adjustmentKind, quantity: adjustmentKind === "COUNT" ? undefined : Number(quantity), countedQuantity: adjustmentKind === "COUNT" ? Number(quantity) : undefined, reference, reason });
      setResult(operationResult);
      setReference(""); setReason(""); setQuantity("1");
    } catch (cause) { setOperationError(messageOf(cause)); }
    finally { setSubmitting(false); }
  }

  if (loading) return <div className="admin-page"><section className="admin-page-skeleton"><div /><div /></section></div>;
  if (!data) return <div className="admin-page"><section className="admin-error-state"><span>!</span><h1>Không thể mở Warehouse Workbench</h1><p>{error}</p><button className="admin-primary-button" onClick={() => void load()}>Thử lại</button></section></div>;

  return <div className="admin-page warehouse-workbench-page">
    <header className="admin-page-header"><div><nav>Admin <span>›</span> Công việc của tôi <span>›</span> Kho</nav><h1>Warehouse Operations Workbench</h1><p>Ghi nhận nhập, xuất, hỏng, thất thoát và kiểm kê bằng biến động có truy vết theo Epic 09.</p></div><div className="admin-page-actions"><button className="admin-secondary-button" type="button" onClick={() => void load()}>↻ Làm mới</button></div></header>
    <section className="admin-summary-grid"><article><span>Khả dụng</span><strong>{QUANTITY.format(metrics.available)}</strong><small>Giá trị do Inventory API cung cấp</small></article><article><span>Batch đang quản lý</span><strong>{metrics.batches}</strong><small>Trong phạm vi kho hiện tại</small></article><article><span>Batch cần chú ý HSD</span><strong>{metrics.expiring}</strong><small>Cận hạn hoặc đã hết hạn</small></article><article><span>Chờ phê duyệt</span><strong>{metrics.pending}</strong><small>Điều chỉnh chưa làm thay đổi số dư</small></article></section>
    <section className="warehouse-policy-note"><strong>Nguyên tắc an toàn dữ liệu</strong><p>{data.policyNotice}</p></section>
    <section className="warehouse-workbench-layout">
      <main className="warehouse-operation-card">
        <div className="warehouse-operation-tabs" role="tablist" aria-label="Chọn nghiệp vụ kho">
          <button className={operation === "RECEIPT" ? "active" : ""} type="button" onClick={() => changeOperation("RECEIPT")}>Nhập kho</button>
          <button className={operation === "ISSUE" ? "active" : ""} type="button" onClick={() => changeOperation("ISSUE")}>Xuất kho</button>
          <button className={operation === "ADJUSTMENT" ? "active" : ""} type="button" onClick={() => changeOperation("ADJUSTMENT")}>Hỏng / thất thoát / kiểm kê</button>
        </div>
        <form className="warehouse-operation-form" onSubmit={(event) => void submit(event)}>
          <header><div><h2>{operation === "RECEIPT" ? "Ghi nhận nhập kho" : operation === "ISSUE" ? "Ghi nhận xuất kho" : "Ghi nhận chênh lệch tồn"}</h2><p>Revision và mã chống gửi trùng được đính kèm tự động khi xác nhận.</p></div><span>Epic 09 · MVP</span></header>
          <div className="warehouse-form-grid">
            <label>SKU<select value={skuId} onChange={(event) => changeSku(event.target.value)} required>{data.skus.map((item) => <option value={item.skuId} key={item.skuId}>{item.skuCode} · {item.productName} · {item.skuLabel}</option>)}</select></label>
            {operation === "RECEIPT" ? <label>Cách ghi nhận Batch<select value={batchMode} onChange={(event) => setBatchMode(event.target.value as "EXISTING" | "NEW")}><option value="EXISTING">Nhận thêm vào Batch hiện có</option><option value="NEW">Tạo Batch từ lần nhập này</option></select></label> : null}
            {(operation !== "RECEIPT" || batchMode === "EXISTING") ? <label>Batch<select value={batchId} onChange={(event) => setBatchId(event.target.value)} required>{selectedSku?.batches.map((batch) => <option value={batch.batchId} key={batch.batchId}>{batch.batchCode} · khả dụng {QUANTITY.format(batch.balance.available)}</option>)}</select></label> : null}
            {operation === "RECEIPT" && batchMode === "NEW" ? <><label>Mã Batch<input value={batchCode} onChange={(event) => setBatchCode(event.target.value)} minLength={2} maxLength={80} required /></label><label>Ngày sản xuất<input type="date" value={manufacturingDate} onChange={(event) => setManufacturingDate(event.target.value)} required /></label><label>Hạn sử dụng<input type="date" value={expiresAt} onChange={(event) => setExpiresAt(event.target.value)} required /></label></> : null}
            {operation === "ISSUE" ? <label>Mục đích xuất<select value={purpose} onChange={(event) => setPurpose(event.target.value as typeof purpose)}><option value="ORDER">Xuất cho Order</option><option value="APPROVED_NON_ORDER">Xuất ngoài Order đã được phê duyệt</option></select></label> : null}
            {operation === "ADJUSTMENT" ? <label>Loại biến động<select value={adjustmentKind} onChange={(event) => setAdjustmentKind(event.target.value as typeof adjustmentKind)}><option value="DAMAGE">Hàng hỏng</option><option value="LOSS">Thất thoát</option><option value="COUNT">Điều chỉnh sau kiểm kê</option></select></label> : null}
            <label>{operation === "ADJUSTMENT" && adjustmentKind === "COUNT" ? "Số lượng kiểm đếm thực tế" : "Số lượng"}<input type="number" min={operation === "ADJUSTMENT" && adjustmentKind === "COUNT" ? 0 : 1} step={1} value={quantity} onChange={(event) => setQuantity(event.target.value)} required /></label>
            <label>{operation === "RECEIPT" ? "Nguồn tham chiếu" : operation === "ISSUE" && purpose === "ORDER" ? "Mã Order" : "Mã chứng từ / tham chiếu"}<input value={reference} onChange={(event) => setReference(event.target.value)} minLength={3} maxLength={100} placeholder={operation === "RECEIPT" ? "PO-2026-... hoặc SR-2026-..." : "ORD-2026-..."} required /></label>
          </div>
          {(operation === "ADJUSTMENT" || (operation === "ISSUE" && purpose === "APPROVED_NON_ORDER")) ? <label className="warehouse-reason">Lý do nghiệp vụ<textarea value={reason} onChange={(event) => setReason(event.target.value)} minLength={10} maxLength={500} placeholder="Mô tả nguyên nhân để Audit và phê duyệt..." required /></label> : null}
          {selectedBatch ? <div className="warehouse-selected-batch"><span>Batch đang chọn</span><strong>{selectedBatch.batchCode}</strong><small>Vật lý {QUANTITY.format(selectedBatch.balance.onHand)} · Đang giữ {QUANTITY.format(selectedBatch.balance.reserved)} · Khả dụng {QUANTITY.format(selectedBatch.balance.available)} · HSD {DATE.format(new Date(selectedBatch.expiresAt))}</small></div> : null}
          {operationError ? <div className="admin-inline-error">{operationError}</div> : null}
          {result ? <div className={`warehouse-result ${result.status.toLocaleLowerCase()}`}><strong>{result.statusLabel}</strong><p>{result.message}</p><small>{result.movementId ? `Movement ${result.movementId}` : `Yêu cầu ${result.approvalRequestId}`} · Revision {result.revision}</small></div> : null}
          <footer><span>Thực hiện bởi {session?.employee.displayName}. API quyết định áp dụng hay chờ duyệt.</span><button className="admin-primary-button" type="submit" disabled={!canSubmit || submitting || !selectedSku}>{submitting ? "Đang ghi nhận…" : !canSubmit ? "Không có quyền thao tác" : "Xác nhận nghiệp vụ"}</button></footer>
        </form>
      </main>
      <aside className="warehouse-context-card">
        <header><div><h2>Ngữ cảnh tồn kho</h2><p>Cập nhật {DATE_TIME.format(new Date(data.calculatedAt))}</p></div><b>{data.skus.length} SKU</b></header>
        <div className="warehouse-sku-context">{data.skus.map((item) => <button className={item.skuId === skuId ? "active" : ""} type="button" onClick={() => changeSku(item.skuId)} key={item.skuId}><span><strong>{item.skuCode}</strong><small>{item.productName} · {item.skuLabel}</small></span><span><strong>{QUANTITY.format(item.balance.available)}</strong><small>khả dụng</small></span></button>)}</div>
        <section><h3>Biến động gần đây</h3><div className="warehouse-movement-list">{data.recentMovements.map((item) => <article key={item.movementId}><span className={item.quantity >= 0 ? "positive" : "negative"}>{item.quantity > 0 ? "+" : ""}{QUANTITY.format(item.quantity)}</span><div><strong>{item.typeLabel} · {item.skuCode}</strong><small>{item.batchCode} · {item.reference}</small><small>{item.actorLabel} · {DATE_TIME.format(new Date(item.occurredAt))}</small></div></article>)}</div></section>
      </aside>
    </section>
  </div>;
}
