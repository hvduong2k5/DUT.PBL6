"use client";

import { useEffect, useRef, useState, type FormEvent } from "react";
import { useEmployeeSession } from "@/components/auth/employee-session-provider";
import { CreateOperations, DirectLookup, ResourceOperations } from "./operations";

type Row = Record<string, unknown>;
type State = { kind: "loading" } | { kind: "success"; items: Row[]; total?: number } | { kind: "error"; status: number; requestId?: string };
const resources = { suppliers: "Nhà cung cấp", materials: "Nguyên liệu", "purchase-orders": "Đơn mua hàng" };
type Resource = keyof typeof resources;
const text = (value: unknown) => typeof value === "string" || typeof value === "number" ? String(value) : "—";

/** Signed grants on the server, not this UI projection, decide access. */
export function ProcurementWorkspace() {
  const { hasPermission } = useEmployeeSession();
  const [resource, setResource] = useState<Resource>("suppliers");
  const [offset, setOffset] = useState(0);
  const [reload, setReload] = useState(0);
  const [state, setState] = useState<State>({ kind: "loading" });
  const [detail, setDetail] = useState<Row>();
  const [detailStatus, setDetailStatus] = useState("");
  const detailRequest = useRef<AbortController | null>(null);
  const view = hasPermission("PROCUREMENT_VIEW");
  const receive = hasPermission("PROCUREMENT_RECEIVE");
  const [receiptId, setReceiptId] = useState("");
  const [receipt, setReceipt] = useState<Row>();
  const [receiptStatus, setReceiptStatus] = useState("");
  const receiptRequest = useRef<AbortController | null>(null);

  useEffect(() => () => { detailRequest.current?.abort(); receiptRequest.current?.abort(); }, []);
  useEffect(() => {
    if (!view) return;
    const controller = new AbortController();
    detailRequest.current?.abort();
    void fetch(`/api/admin/procurement/${resource}?limit=20&offset=${offset}`, { cache: "no-store", credentials: "same-origin", signal: controller.signal })
      .then(async (response) => {
        const data: unknown = await response.json();
        if (controller.signal.aborted) return;
        const body = data as Row;
        if (!response.ok) { setState({ kind: "error", status: response.status, requestId: typeof body?.requestId === "string" ? body.requestId : undefined }); return; }
        if (!body || !Array.isArray(body.items) || body.items.some((row) => !row || typeof row !== "object" || Array.isArray(row))) throw new Error("invalid response");
        setState({ kind: "success", items: body.items as Row[], total: typeof body.total === "number" ? body.total : undefined });
      }).catch(() => { if (!controller.signal.aborted) setState({ kind: "error", status: 503 }); });
    return () => controller.abort();
  }, [view, resource, offset, reload]);

  function resetList() {
    detailRequest.current?.abort();
    setState({ kind: "loading" }); setDetail(undefined); setDetailStatus("");
  }

  function afterChange() { setState({ kind: "loading" }); setReload((value) => value + 1); }

  async function showDetail(id: string) {
    detailRequest.current?.abort();
    const controller = new AbortController(); detailRequest.current = controller;
    setDetail(undefined); setDetailStatus("Đang tải chi tiết…");
    try {
      const response = await fetch(`/api/admin/procurement/${resource}/${encodeURIComponent(id)}`, { credentials: "same-origin", cache: "no-store", signal: controller.signal });
      const data: unknown = await response.json();
      if (controller.signal.aborted) return;
      if (!response.ok) { setDetailStatus(`Không thể xem chi tiết (${response.status}).`); return; }
      if (!data || typeof data !== "object" || Array.isArray(data)) throw new Error();
      setDetail(data as Row); setDetailStatus("");
    } catch { if (!controller.signal.aborted) setDetailStatus("Không thể tải chi tiết. Vui lòng thử lại."); }
  }

  async function lookupReceipt(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); receiptRequest.current?.abort();
    const controller = new AbortController(); receiptRequest.current = controller;
    setReceipt(undefined); setReceiptStatus("Đang tra cứu phiếu…");
    try {
      const response = await fetch(`/api/admin/procurement/goods-receipts/${encodeURIComponent(receiptId.trim())}`, { credentials: "same-origin", cache: "no-store", signal: controller.signal });
      const data: unknown = await response.json();
      if (controller.signal.aborted) return;
      if (!response.ok) { setReceiptStatus(response.status === 403 ? "Phiếu nằm ngoài quyền hoặc phạm vi kho của bạn." : `Không thể xem phiếu (${response.status}).`); return; }
      if (!data || typeof data !== "object" || Array.isArray(data)) throw new Error();
      setReceipt(data as Row); setReceiptStatus("");
    } catch { if (!controller.signal.aborted) setReceiptStatus("Không thể tra cứu phiếu. Vui lòng thử lại."); }
  }

  return <main className="admin-workspace procurement-workspace">
    <h1>Thu mua & nhận nguyên liệu</h1>
    <p>Dữ liệu từ Procurement. Quyền và phạm vi kho được kiểm tra lại ở dịch vụ; không gồm dự báo nguồn hàng.</p>
    <CreateOperations onChanged={afterChange} />
    {view ? <section aria-label="Tra cứu Procurement">
      <label>Danh mục <select value={resource} onChange={(event) => { resetList(); setResource(event.target.value as Resource); setOffset(0); }}>
        {Object.entries(resources).map(([key, label]) => <option key={key} value={key}>{label}</option>)}
      </select></label> <button onClick={() => { resetList(); setReload((value) => value + 1); }}>Tải lại</button>
      <div aria-live="polite">
        {state.kind === "loading" && <p role="status">Đang tải dữ liệu…</p>}
        {state.kind === "error" && <p role="alert">{state.status === 401 ? "Phiên đăng nhập không hợp lệ. Vui lòng đăng nhập lại." : state.status === 403 ? "Bạn không có quyền hoặc phạm vi phù hợp." : "Không thể tải dữ liệu. Hãy thử lại."} {state.requestId ? `Mã yêu cầu: ${state.requestId}` : ""}</p>}
        {state.kind === "success" && (state.items.length === 0 ? <p>Không có dữ liệu ở trang này.</p> : <>
          <table><caption>{resources[resource]}{state.total === undefined ? "" : ` — ${state.total} bản ghi`}</caption>
            <thead><tr><th>Mã</th><th>Tên / nhà cung cấp</th><th>Trạng thái</th><th>Chi tiết</th></tr></thead>
            <tbody>{state.items.map((row) => <tr key={text(row.id)}><td>{text(row.code ?? row.number ?? row.id)}</td><td>{text(row.name ?? row.supplier_name ?? row.supplier_id)}</td><td>{text(row.status)}</td><td><button type="button" onClick={() => void showDetail(text(row.id))}>Xem chi tiết</button></td></tr>)}</tbody>
          </table>
        </>)}
      </div>
      <button disabled={offset === 0 || state.kind === "loading"} onClick={() => { resetList(); setOffset((value) => Math.max(0, value - 20)); }}>Trang trước</button>
      <span> Trang {offset / 20 + 1} </span>
      <button disabled={state.kind !== "success" || state.items.length < 20} onClick={() => { resetList(); setOffset((value) => value + 20); }}>Trang sau</button>
      {detailStatus && <p role="status">{detailStatus}</p>}
      {detail && <ResourceOperations key={`${resource}-${String(detail.id)}-${String(detail.updated_at)}`} kind={resource} row={detail} onChanged={afterChange} />}
    </section> : <p>Chưa có quyền tra cứu danh mục/PO. Các quyền thao tác không tự cấp quyền đọc danh mục.</p>}
    {(view || receive) && <section aria-label="Tra cứu phiếu nhận hàng"><h2>Phiếu nhận hàng</h2><p>Tra cứu bằng ID phiếu; chỉ trả dữ liệu trong phạm vi được cấp.</p>
      <form onSubmit={(event) => void lookupReceipt(event)}><label>ID phiếu <input required value={receiptId} onChange={(event) => setReceiptId(event.target.value)} pattern="[0-9a-f]{8}(-[0-9a-f]{4}){3}-[0-9a-f]{12}" /></label><button>Tra cứu</button></form>
      {receiptStatus && <p role="status">{receiptStatus}</p>}
      {receipt && <ResourceOperations key={`receipt-${String(receipt.id)}-${String(receipt.updated_at)}`} kind="goods-receipts" row={receipt} onChanged={afterChange} />}
    </section>}
    <DirectLookup onChanged={afterChange} />
    <p>Hợp đồng hiện quản lý dữ liệu; chưa hỗ trợ tải tệp hoặc ký điện tử. Không có chức năng dự báo nguồn hàng.</p>
  </main>;
}
