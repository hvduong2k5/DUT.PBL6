"use client";

import { useEffect, useRef, useState, type FormEvent } from "react";
import { useEmployeeSession } from "@/components/auth/employee-session-provider";
import { availableActions, formSpec, records, rowText, UUID_PATTERN, type FormSpec, type RecordData, type ResourceKind } from "./forms";
import { BusinessForm } from "./business-form";

const labels: Record<string, string> = { id: "ID", code: "Mã", name: "Tên", number: "Số chứng từ", status: "Trạng thái", supplier_type: "Loại nhà cung cấp", contact_name: "Liên hệ", phone: "Điện thoại", email: "Email", address: "Địa chỉ", legal_identifier: "Mã pháp lý", description: "Mô tả", base_unit: "Đơn vị cơ sở", preferred_supplier_id: "Nhà cung cấp ưu tiên", supplier_id: "ID nhà cung cấp", supplier_name: "Nhà cung cấp", supplier_sku: "Mã hàng NCC", material_id: "ID nguyên liệu", material_code: "Mã nguyên liệu", material_name: "Nguyên liệu", unit: "Đơn vị", unit_price_minor: "Đơn giá (đơn vị tiền nhỏ nhất)", currency: "Tiền tệ", minimum_quantity: "Số lượng tối thiểu", lead_time_days: "Thời gian cung ứng (ngày)", effective_from: "Hiệu lực từ (UTC)", effective_to: "Mốc kết thúc loại trừ (UTC)", active: "Đang hoạt động", revision: "Revision", version: "Version", contract_number: "Số hợp đồng", scope: "Phạm vi hợp đồng", payment_terms: "Điều khoản thanh toán", total_amount_minor: "Tổng tiền (đơn vị tiền nhỏ nhất)", expected_delivery_date: "Ngày dự kiến nhận (UTC)", notes: "Ghi chú", created_by: "Người tạo", submitted_by: "Người trình", approved_by: "Người duyệt", sent_by: "Người ghi nhận gửi", supplier_response_note: "Phản hồi nhà cung cấp", rejection_reason: "Lý do từ chối", close_reason: "Lý do đóng", cancellation_reason: "Lý do hủy", created_at: "Tạo lúc (UTC)", updated_at: "Cập nhật lúc (UTC)", warehouse_code: "Kho", purchase_order_id: "ID PO", purchase_order_number: "Số PO", received_by: "Người nhận", inspected_by: "Người kiểm nhận", inspected_at: "Kiểm nhận lúc (UTC)", goods_receipt_line_id: "ID dòng phiếu", line_number: "Dòng", purchase_order_line_id: "ID dòng PO", quantity: "Số lượng đặt", received_quantity: "Đã nhận", accepted_quantity: "Đã chấp nhận", expected_open_quantity: "Dự kiến còn nhận", quantity_received: "Thực nhận", quantity_accepted: "Chấp nhận", quantity_rejected: "Từ chối", supplier_lot_code: "Lô nhà cung cấp", manufactured_on: "Ngày sản xuất (UTC)", expires_on: "Ngày hết hạn (UTC)", humidity_percent: "Độ ẩm (%)", inspection_notes: "Ghi chú kiểm nhận", quality_status: "Chất lượng", type: "Loại chênh lệch", expected_quantity: "Dự kiến", actual_quantity: "Thực tế", difference_quantity: "Chênh lệch", reason: "Lý do", decision: "Quyết định", resolution_note: "Ghi chú xử lý", resolved_by: "Người xử lý", resolved_at: "Xử lý lúc (UTC)" };

Object.assign(labels, { changed_by: "Người đổi revision", change_reason: "Lý do đổi revision", submitted_at: "Trình lúc (UTC)", approved_at: "Duyệt lúc (UTC)", rejected_by: "Người từ chối", rejected_at: "Từ chối lúc (UTC)", sent_at: "Ghi nhận gửi lúc (UTC)", supplier_responded_by: "Người ghi nhận phản hồi", supplier_responded_at: "Phản hồi lúc (UTC)", cancelled_by: "Người hủy", cancelled_at: "Hủy lúc (UTC)", closed_by: "Người đóng", closed_at: "Đóng lúc (UTC)" });

export function RecordDetails({ row }: { row: RecordData }) {
  return <dl className="proc-details">{Object.entries(row).filter(([key, value]) => labels[key] && value !== null && value !== undefined && typeof value !== "object").map(([key, value]) => <div key={key}><dt>{labels[key]}</dt><dd>{typeof value === "boolean" ? value ? "Có" : "Không" : rowText(value) || "—"}</dd></div>)}</dl>;
}

async function read(path: string, signal?: AbortSignal): Promise<RecordData> {
  const response = await fetch(`/api/admin/procurement/${path}`, { credentials: "same-origin", cache: "no-store", signal });
  let payload: unknown;
  try { payload = await response.json(); } catch { throw new Error("Phản hồi không hợp lệ. Không hiển thị dữ liệu giả."); }
  const data = payload && typeof payload === "object" && !Array.isArray(payload) ? payload as RecordData : undefined;
  if (!response.ok) throw new Error(`${response.status === 401 ? "Vui lòng đăng nhập lại." : response.status === 403 ? "Thiếu quyền hoặc phạm vi phù hợp." : response.status === 404 ? "Không tìm thấy bản ghi." : "Không tải được dữ liệu."} ${data?.requestId ? `Mã yêu cầu: ${rowText(data.requestId)}` : ""}`);
  if (!data) throw new Error("Phản hồi không hợp lệ.");
  return data;
}

function RelatedCollection({ path, onSelect }: { path: string; onSelect: (row: RecordData) => void }) {
  const [offset, setOffset] = useState(0), [reload, setReload] = useState(0);
  const [data, setData] = useState<RecordData[]>(), [failure, setFailure] = useState("");
  const paged = /\/(?:contracts|revisions)$/u.test(path);
  useEffect(() => {
    const controller = new AbortController();
    void read(path + (paged ? `?limit=20&offset=${offset}` : ""), controller.signal).then((result) => {
      if (!Array.isArray(result.items)) throw new Error("Danh sách không hợp lệ.");
      if (!controller.signal.aborted) setData(records(result.items));
    }).catch((cause) => { if (!controller.signal.aborted) setFailure(cause instanceof Error ? cause.message : "Không tải được dữ liệu."); });
    return () => controller.abort();
  }, [path, offset, paged, reload]);
  function reset() { setData(undefined); setFailure(""); }
  return <section className="proc-card" aria-label="Danh sách liên quan">
    <button type="button" onClick={() => { reset(); setReload((value) => value + 1); }}>Tải lại danh sách</button>
    {failure ? <p role="alert">{failure}</p> : !data ? <p role="status">Đang tải…</p> : !data.length ? <p>Chưa có bản ghi.</p> : <div className="proc-record-grid">{data.map((row, index) => <article className="proc-card" key={rowText(row.id) || index}><RecordDetails row={row} /><button type="button" onClick={() => onSelect(row)}>Xem / thao tác bản ghi</button></article>)}</div>}
    {paged && <div className="proc-actions"><button type="button" disabled={offset === 0 || !data} onClick={() => { reset(); setOffset((value) => Math.max(0, value - 20)); }}>Trang trước</button><span>Trang {offset / 20 + 1}</span><button type="button" disabled={!data || data.length < 20} onClick={() => { reset(); setOffset((value) => value + 20); }}>Trang sau</button></div>}
  </section>;
}

export function ResourceOperations({ kind, row: initial, onChanged }: { kind: ResourceKind; row: RecordData; onChanged?: () => void }) {
  const { hasPermission, session } = useEmployeeSession();
  const [current, setCurrent] = useState(initial), [form, setForm] = useState<FormSpec>();
  const [related, setRelated] = useState<string>(), [selected, setSelected] = useState<{ kind: "contract" | "offer" | "revision"; row: RecordData }>();
  const [message, setMessage] = useState(""), [reading, setReading] = useState(false), [fresh, setFresh] = useState(true);
  const busy = useRef(false), alive = useRef(true);
  const request = useRef<AbortController | null>(null);
  useEffect(() => { alive.current = true; return () => { alive.current = false; request.current?.abort(); }; }, []);
  const actor = session?.employee.employeeId ?? "";
  const actions = fresh ? availableActions(kind, current, actor, hasPermission) : [];
  async function refresh() {
    if (busy.current) return false;
    busy.current = true; setReading(true); setFresh(false); setRelated(undefined); setSelected(undefined); setForm(undefined);
    request.current?.abort(); const controller = new AbortController(); request.current = controller;
    try {
      const result = await read(`${kind}/${rowText(current.id)}`, controller.signal);
      if (!alive.current || controller.signal.aborted) return false;
      setCurrent(result); setFresh(true); setMessage("");
      return true;
    } catch (cause) { if (alive.current && !controller.signal.aborted) setMessage(cause instanceof Error ? cause.message : "Không tải lại được dữ liệu."); return false; }
    finally { busy.current = false; if (alive.current) setReading(false); }
  }
  async function success(result: RecordData) {
    setMessage(`Đã ghi nhận${result.id ? `: ${rowText(result.id)}` : ""}. Đang đọc lại trạng thái hiện tại…`);
    onChanged?.();
    const refreshed = await refresh();
    if (alive.current) setMessage((old) => `Đã ghi nhận yêu cầu${result.id ? ` (${rowText(result.id)})` : ""}. ${refreshed ? "Đã tải trạng thái hiện tại." : `${old} Dùng nút tải lại; không gửi lại yêu cầu mới.`}`);
  }
  function open(action: string, data = current) { setForm(formSpec(action, data)); setMessage(""); }
  async function selectRelated(data: RecordData) {
    if (related?.endsWith("/contracts")) {
      request.current?.abort(); const controller = new AbortController(); request.current = controller;
      try { const value = await read(`contracts/${rowText(data.id)}`, controller.signal); if (alive.current && !controller.signal.aborted) setSelected({ kind: "contract", row: value }); }
      catch (cause) { if (alive.current && !controller.signal.aborted) setMessage(cause instanceof Error ? cause.message : "Không tải được hợp đồng."); }
    } else setSelected({ kind: related?.endsWith("/offers") ? "offer" : "revision", row: data });
  }
  return <section className="proc-card" aria-label="Chi tiết và thao tác nghiệp vụ">
    <h2>Chi tiết & thao tác</h2><RecordDetails row={current} />
    {records(current.lines).length > 0 && <section aria-label="Dòng chứng từ"><h3>Các dòng chứng từ</h3><div className="proc-record-grid">{records(current.lines).map((line, index) => <article className="proc-card" key={rowText(line.id) || index}><RecordDetails row={line} /></article>)}</div></section>}
    <div className="proc-actions"><button type="button" disabled={reading} onClick={() => void refresh()}>{reading ? "Đang tải…" : "Tải lại bản hiện tại"}</button>
      {actions.map((action) => <button type="button" key={action} disabled={reading} onClick={() => open(action)}>{formSpec(action, current).title}</button>)}
      {fresh && kind === "contracts" && hasPermission("PROCUREMENT_MANAGE") && <button type="button" onClick={() => open("contract-create", { ...current, id: current.supplier_id })}>Tạo phiên bản hợp đồng mới</button>}
      {fresh && ["suppliers", "materials"].includes(kind) && hasPermission("PROCUREMENT_VIEW") && <button type="button" onClick={() => { setSelected(undefined); setRelated(`${kind}/${rowText(current.id)}/offers`); }}>Bảng giá</button>}
      {fresh && kind === "suppliers" && hasPermission("PROCUREMENT_MANAGE") && <button type="button" onClick={() => { setSelected(undefined); setRelated(`suppliers/${rowText(current.id)}/contracts`); }}>Hợp đồng</button>}
      {fresh && kind === "purchase-orders" && hasPermission("PROCUREMENT_VIEW") && <button type="button" onClick={() => { setSelected(undefined); setRelated(`purchase-orders/${rowText(current.id)}/revisions`); }}>Lịch sử revision</button>}
    </div>
    {!fresh && <p role="alert">Dữ liệu chưa được xác nhận mới nhất. Các thao tác đã khóa cho tới khi tải lại thành công.</p>}
    {kind === "purchase-orders" && current.status === "PENDING_APPROVAL" && current.created_by === actor && <p>Người tạo PO không được tự duyệt hoặc tự từ chối PO.</p>}
    {kind === "goods-receipts" && <section><h3>Chênh lệch nhận hàng</h3>{!records(current.variances).length ? <p>Không có chênh lệch.</p> : records(current.variances).map((variance) => <article className="proc-card" key={rowText(variance.id)}><RecordDetails row={variance} />{fresh && hasPermission("PROCUREMENT_VARIANCE_RESOLVE") && variance.status === "OPEN" && <button type="button" onClick={() => open("variance-resolve", { ...variance, receipt_id: current.id, version: current.version })}>Xử lý chênh lệch</button>}</article>)}<p>Kiểm nhận không tự giải phóng QUARANTINE ở Inventory hoặc tạo nghiệp vụ hoàn tiền.</p></section>}
    {message && <p role="status">{message}</p>}
    {related && <RelatedCollection key={related} path={related} onSelect={(value) => void selectRelated(value)} />}
    {selected?.kind === "contract" && <ResourceOperations key={`contract-${rowText(selected.row.id)}`} kind="contracts" row={selected.row} onChanged={() => { setRelated(undefined); setSelected(undefined); setMessage("Đã ghi nhận thao tác hợp đồng. Mở lại danh sách hợp đồng để xem trạng thái mới nhất."); }} />}
    {selected?.kind === "offer" && <section className="proc-card"><h3>Phiên bản bảng giá</h3><RecordDetails row={selected.row} />{fresh && hasPermission("PROCUREMENT_MANAGE") && <button type="button" onClick={() => open("offer-status", selected.row)}>Bật / tắt bảng giá</button>}</section>}
    {selected?.kind === "revision" && <section className="proc-card"><h3>Bản phát hành lưu trữ — chỉ đọc</h3><RecordDetails row={selected.row} /><RecordDetails row={(selected.row.snapshot as RecordData) ?? {}} />{records((selected.row.snapshot as RecordData)?.lines).map((line, index) => <RecordDetails key={index} row={line} />)}</section>}
    {form && <BusinessForm key={`${form.action}-${form.path}-${rowText(form.source.version ?? form.source.updated_at)}`} spec={form} onSuccess={success} onCancel={() => { setForm(undefined); void refresh(); }} />}
  </section>;
}

export function CreateOperations({ onChanged }: { onChanged: () => void }) {
  const { hasPermission } = useEmployeeSession();
  const [form, setForm] = useState<FormSpec>(), [message, setMessage] = useState("");
  const [targetAction, setTargetAction] = useState<"offer-create" | "contract-create">(), [supplierID, setSupplierID] = useState("");
  const create = [...(hasPermission("PROCUREMENT_MANAGE") ? ["supplier-create", "material-create", "po-create"] : []), ...(hasPermission("PROCUREMENT_RECEIVE") ? ["receipt-create"] : [])];
  return <section className="proc-card" aria-label="Tạo mới nghiệp vụ"><h2>Tạo mới</h2><div className="proc-actions">{create.map((action) => <button type="button" key={action} onClick={() => { setMessage(""); setTargetAction(undefined); setForm(formSpec(action)); }}>{formSpec(action).title}</button>)}
    {hasPermission("PROCUREMENT_MANAGE") && <><button type="button" onClick={() => { setForm(undefined); setMessage(""); setTargetAction("offer-create"); }}>Tạo bảng giá theo ID nhà cung cấp</button><button type="button" onClick={() => { setForm(undefined); setMessage(""); setTargetAction("contract-create"); }}>Tạo hợp đồng theo ID nhà cung cấp</button></>}
    </div>
    {!create.length && <p>Chưa có quyền tạo dữ liệu.</p>}
    {message && <p role="status">{message}</p>}
    {targetAction && <form className="proc-actions" onSubmit={(event) => { event.preventDefault(); if (!UUID_PATTERN.test(supplierID.trim())) { setMessage("ID nhà cung cấp phải là UUID hợp lệ."); return; } setForm(formSpec(targetAction, { id: supplierID.trim() })); setTargetAction(undefined); setMessage(""); }}><label>ID nhà cung cấp <input required value={supplierID} onChange={(event) => setSupplierID(event.target.value)} /></label><button type="submit">Mở form</button><button type="button" onClick={() => setTargetAction(undefined)}>Hủy</button></form>}
    {form && <BusinessForm key={form.action} spec={form} onCancel={() => setForm(undefined)} onSuccess={async (result) => { setForm(undefined); setMessage(`Đã ghi nhận${result.id ? `, ID: ${rowText(result.id)}` : ""}. Tra cứu bản hiện tại trước khi thao tác bước tiếp theo.`); onChanged(); }} />}
  </section>;
}

/** Allows direct contract lookup (MANAGE) and receipt lookup (RECEIVE), without
 * inventing a supplier/PO VIEW grant for these independent permissions. */
export function DirectLookup({ onChanged }: { onChanged: () => void }) {
  const { hasPermission } = useEmployeeSession();
  const options: Array<[ResourceKind, string]> = [...(hasPermission("PROCUREMENT_VIEW") ? [["suppliers", "Nhà cung cấp"], ["materials", "Nguyên liệu"], ["purchase-orders", "PO"]] as Array<[ResourceKind, string]> : []), ...(hasPermission("PROCUREMENT_MANAGE") ? [["contracts", "Hợp đồng"]] as Array<[ResourceKind, string]> : []), ...(hasPermission("PROCUREMENT_RECEIVE") || hasPermission("PROCUREMENT_VIEW") || hasPermission("PROCUREMENT_VARIANCE_RESOLVE") ? [["goods-receipts", "Phiếu nhận hàng"]] as Array<[ResourceKind, string]> : [])];
  const [kind, setKind] = useState<ResourceKind>(options[0]?.[0] ?? "goods-receipts"), [id, setID] = useState("");
  const [result, setResult] = useState<{ kind: ResourceKind; row: RecordData }>(), [message, setMessage] = useState("");
  const request = useRef<AbortController | null>(null);
  useEffect(() => () => request.current?.abort(), []);
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); request.current?.abort(); setResult(undefined);
    if (!options.some(([value]) => value === kind) || !UUID_PATTERN.test(id.trim())) { setMessage("Chọn loại bản ghi và UUID hợp lệ."); return; }
    const controller = new AbortController(); request.current = controller; setMessage("Đang tra cứu…");
    try { const row = await read(`${kind}/${id.trim()}`, controller.signal); if (!controller.signal.aborted) { setResult({ kind, row }); setMessage(""); } }
    catch (cause) { if (!controller.signal.aborted) setMessage(cause instanceof Error ? cause.message : "Không tải được dữ liệu."); }
  }
  if (!options.length) return null;
  return <section className="proc-card" aria-label="Tra cứu theo ID"><h2>Tra cứu trực tiếp theo ID</h2><form onSubmit={(event) => void submit(event)} className="proc-actions">
    <label>Loại bản ghi <select value={kind} onChange={(event) => { request.current?.abort(); setResult(undefined); setMessage(""); setKind(event.target.value as ResourceKind); }}>{options.map(([value, label]) => <option key={value} value={value}>{label}</option>)}</select></label>
    <label>ID bản ghi <input required value={id} onChange={(event) => setID(event.target.value)} /></label><button>Tra cứu</button>
  </form><p>Quyền xử lý chênh lệch không tự cấp quyền đọc phiếu; cần VIEW hoặc RECEIVE phù hợp để xem chi tiết.</p>{message && <p role="status">{message}</p>}
    {result && <ResourceOperations key={`${result.kind}-${rowText(result.row.id)}-${rowText(result.row.updated_at)}`} kind={result.kind} row={result.row} onChanged={onChanged} />}
  </section>;
}
