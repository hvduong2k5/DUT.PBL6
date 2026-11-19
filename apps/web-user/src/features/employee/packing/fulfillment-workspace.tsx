"use client";

import Link from "next/link";
import { useCallback, useEffect, useRef, useState } from "react";
import { useEmployeeSession } from "@/components/auth/employee-session-provider";
import { canUseFulfillment, FulfillmentError, type FulfillmentPage, type FulfillmentHistoryPage, type FulfillmentTask, type PackingMedia, type Parcel, type PendingPackingCommand } from "./fulfillment-types";
import { fulfillmentService } from "./fulfillment-service";
import styles from "./fulfillment.module.css";

const labels: Record<string, string> = { READY: "Chờ nhận việc", ASSIGNED: "Đã nhận việc", IN_PROGRESS: "Đang đóng gói", BLOCKED: "Bị chặn", COMPLETED: "Đã đóng gói", CREATED: "Đã tải video", VERIFYING: "Đang kiểm tra video", VERIFIED: "Video hợp lệ", REJECTED: "Video bị từ chối", PASS: "Đạt", FAIL: "Không đạt", PENDING: "Chưa xác nhận" };
const date = (raw: string | null) => raw ? new Intl.DateTimeFormat("vi-VN", { dateStyle: "short", timeStyle: "short" }).format(new Date(raw)) : "Chưa có";
const message = (error: unknown) => error instanceof Error ? error.message : "Không thể xử lý yêu cầu đóng gói.";

export function FulfillmentWorkspace({ management = false }: { management?: boolean }) {
  const { employee } = useEmployeeSession();
  const authorized = canUseFulfillment(employee);
  const scope = `packing:${employee.principalId}`;
  const [page, setPage] = useState<FulfillmentPage | null>(null);
  const [state, setState] = useState("");
  const [warehouse, setWarehouse] = useState("");
  const [task, setTask] = useState<FulfillmentTask | null>(null);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [pending, setPending] = useState<PendingPackingCommand | null>(null);
  const [media, setMedia] = useState<PackingMedia | null>(null);
  const [video, setVideo] = useState<File | null>(null);
  const [parcel, setParcel] = useState<Parcel>({ weight_grams: 0, length_cm: 0, width_cm: 0, height_cm: 0, fragile: false });
  const [seal, setSeal] = useState("");
  const [confirming, setConfirming] = useState(false);
  const [history, setHistory] = useState<FulfillmentHistoryPage | null>(null);
  const selection = useRef(0);
  const running = useRef(false);

  const loadQueue = useCallback(async (cursor?: string) => {
    if (!authorized) { setLoading(false); return; }
    setLoading(true); setError("");
    try {
      const query = new URLSearchParams({ limit: "25" });
      if (state) query.set("workflow_state", state);
      if (warehouse) query.set("warehouse_id", warehouse);
      if (cursor) query.set("cursor", cursor);
      const next = await fulfillmentService.queue(query);
      setPage((current) => cursor && current ? { ...next, items: [...current.items, ...next.items.filter((item) => !current.items.some((old) => old.task_id === item.task_id))] } : next);
    } catch (cause) { setError(message(cause)); }
    finally { setLoading(false); }
  }, [authorized, state, warehouse]);

  useEffect(() => { void Promise.resolve().then(() => loadQueue()); }, [loadQueue]);
  useEffect(() => {
    void Promise.resolve().then(() => { try {
      const saved = sessionStorage.getItem(scope + ":command");
      if (saved) setPending(JSON.parse(saved));
    } catch { setError("Không đọc được yêu cầu đang chờ. Hãy kiểm tra trạng thái task trước khi thao tác."); } });
  }, [scope]);

  const openTask = useCallback(async (id: string) => {
    const generation = ++selection.current;
    setError(""); setMedia(null); setVideo(null); setTask(null); setConfirming(false); setHistory(null);
    try {
      const next = await fulfillmentService.task(id);
      if (generation !== selection.current) return;
      setTask(next); setParcel(next.package.parcel ?? { weight_grams: 0, length_cm: 0, width_cm: 0, height_cm: 0, fragile: false }); setSeal(next.package.seal_code ?? "");
      const fileId = next.package.primary_evidence_id ?? sessionStorage.getItem(`${scope}:media:${id}`);
      if (fileId) { const saved = await fulfillmentService.media(fileId); if (generation === selection.current) setMedia(saved); }
    } catch (cause) { if (generation === selection.current) setError(message(cause)); }
  }, [scope]);

  async function loadHistory(cursor?: string) {
    if (!task) return;
    const generation = selection.current;
    try {
      const next = await fulfillmentService.history(task.task_id, cursor);
      if (generation === selection.current) setHistory((current) => cursor && current ? { ...next, items: [...current.items, ...next.items] } : next);
    } catch (cause) { setError(message(cause)); }
  }

  // Poll reads only. A scanner outage leaves the video pending and never seals.
  useEffect(() => {
    if (!media || !["CREATED", "VERIFYING"].includes(media.status)) return;
    const controller = new AbortController(); let timer: ReturnType<typeof setTimeout>; let attempts = 0;
    const poll = async () => {
      try {
        const next = await fulfillmentService.media(media.file_id, controller.signal);
        if (controller.signal.aborted) return;
        setMedia(next);
        if (["VERIFIED", "REJECTED"].includes(next.status)) return;
        if (++attempts >= 120) { setNotice("Video chưa được xác minh. Bạn có thể kiểm tra lại; chưa thể niêm phong kiện."); return; }
        timer = setTimeout(poll, 2000);
      } catch (cause) { if (!controller.signal.aborted) setError(message(cause)); }
    };
    timer = setTimeout(poll, 1500);
    return () => { controller.abort(); clearTimeout(timer); };
  }, [media?.file_id, media?.status]); // eslint-disable-line react-hooks/exhaustive-deps

  function remember(command: PendingPackingCommand | null) {
    if (command) sessionStorage.setItem(scope + ":command", JSON.stringify(command));
    else sessionStorage.removeItem(scope + ":command");
    setPending(command);
  }
  async function execute(command: PendingPackingCommand) {
    if (running.current) return;
    running.current = true; setBusy(true); setError(""); setNotice("");
    try {
      remember(command);
      await fulfillmentService.command(command);
      remember(null);
      await openTask(command.taskId); await loadQueue();
      setNotice(command.action === "seal" ? "Đã đóng gói. Order sẽ xử lý tạo vận đơn; xem kết quả tại Đơn hàng/Giao vận." : "Đã lưu kết quả vào hệ thống.");
    } catch (cause) {
      if (cause instanceof FulfillmentError && cause.status >= 400 && cause.status < 500 && ![401, 403, 429].includes(cause.status)) {
        remember(null);
        if ([409, 412].includes(cause.status)) { await openTask(command.taskId); await loadQueue(); }
      }
      setError(message(cause));
    } finally { running.current = false; setBusy(false); setConfirming(false); }
  }
  function mutate(action: string, body: Record<string, unknown> = {}, method: "POST" | "PUT" = "POST") {
    if (!task || pending || running.current) return;
    void execute({ taskId: task.task_id, action, body, method, version: task.version, key: crypto.randomUUID() });
  }

  async function upload() {
    if (!task || !video || running.current) return;
    const current = task; const chosen = video; const generation = selection.current;
    if (!["video/mp4", "video/webm"].includes(chosen.type) || chosen.size <= 0 || chosen.size > 209715200) { setError("Chọn video MP4/WebM không quá 200 MB."); return; }
    running.current = true; setBusy(true); setError("");
    let intent = "";
    try {
      const hash = Array.from(new Uint8Array(await crypto.subtle.digest("SHA-256", await chosen.arrayBuffer())), (n) => n.toString(16).padStart(2, "0")).join("");
      intent = `${scope}:upload:${current.task_id}:${hash}:${chosen.size}`;
      const key = sessionStorage.getItem(intent) ?? crypto.randomUUID();
      sessionStorage.setItem(intent, key);
      const uploaded = await fulfillmentService.upload(current, chosen, hash, key);
      sessionStorage.setItem(`${scope}:media:${current.task_id}`, uploaded.file_id);
      // The stable completion key permits recovery after a lost response.
      const result = await fulfillmentService.completeMedia(uploaded.file_id, uploaded.file_id + "-complete");
      if (generation === selection.current) { setMedia(result); setNotice("Video đã tải lên, đang chờ kiểm tra."); }
    } catch (cause) {
      if (cause instanceof FulfillmentError && cause.status === 410 && intent) sessionStorage.removeItem(intent);
      setError(cause instanceof FulfillmentError && cause.status === 410 ? "Phiên upload đã hết hạn. Bấm Tải video để tạo phiên mới cho tệp đã chọn." : message(cause));
    }
    finally { running.current = false; setBusy(false); }
  }
  async function checkMedia() {
    if (!media || running.current) return;
    running.current = true; setBusy(true); setError("");
    try {
      if (media.status === "CREATED") await fulfillmentService.completeMedia(media.file_id, `${media.file_id}-complete`);
      setMedia(await fulfillmentService.media(media.file_id));
    } catch (cause) { setError(message(cause)); }
    finally { running.current = false; setBusy(false); }
  }

  if (!authorized) return <main className="admin-state" role="alert"><h1>Chưa có quyền đóng gói</h1><p>Luồng Fulfillment hiện yêu cầu Employee ACTIVE với ADMIN tại kho hoặc toàn hệ thống.</p></main>;
  const mine = task?.assignee_id === employee.principalId;
  const working = Boolean(mine && task?.workflow_state === "IN_PROGRESS" && !busy && !pending && !management);
  const missing: string[] = [];
  if (task) {
    if (!mine || task.workflow_state !== "IN_PROGRESS") missing.push("Nhận việc và bắt đầu đóng gói.");
    if (task.active_blocks.length) missing.push("Task có điều kiện chặn cần xử lý.");
    if (!task.allocations.length || task.allocations.some((a) => !a.currently_eligible || a.counted_quantity !== a.allocated_quantity || a.batch_valid_until <= task.queue_as_of)) missing.push("Xác nhận đủ số lượng từng lô hợp lệ.");
    if (task.checklist.some((c) => c.required && c.result !== "PASS")) missing.push("Hoàn thành checklist bắt buộc.");
    if (task.evidence_mode === "VIDEO_REQUIRED" && (media?.status !== "VERIFIED" || media.task_id !== task.task_id || media.package_id !== task.package.package_id)) missing.push("Video của kiện phải được xác minh hợp lệ.");
    if (!/^[A-Z0-9][A-Z0-9-]{5,79}$/u.test(seal)) missing.push("Nhập mã niêm phong 6–80 ký tự in hoa, số hoặc dấu gạch ngang.");
    if (![parcel.weight_grams, parcel.length_cm, parcel.width_cm, parcel.height_cm].every((v) => Number.isSafeInteger(v) && v > 0)) missing.push("Nhập cân nặng và kích thước thực tế bằng số nguyên dương.");
  }

  return <div className={`admin-page ${styles.page}`}>
    <header className="admin-page-header"><div><h1>{management ? "Quản lý đóng gói" : "Workbench đóng gói"}</h1><p>Nhận việc, xác nhận hàng và niêm phong theo dữ liệu Order/Inventory.</p></div><button className="admin-secondary-button" disabled={loading || busy} onClick={() => void loadQueue()}>Làm mới danh sách</button></header>
    {error && <div className="admin-inline-error" role="alert">{error}</div>}
    {notice && <p className={styles.notice} role="status">{notice}</p>}
    {pending && <section className={styles.pending} role="alert"><strong>Có yêu cầu chưa rõ kết quả</strong><p>Task {pending.taskId} · {pending.action}. Thử lại giữ nguyên nội dung, revision và mã chống trùng.</p><button className="admin-primary-button" disabled={busy} onClick={() => void execute(pending)}>Kiểm tra lại bằng cùng yêu cầu</button></section>}
    <div className={styles.filters}><label>Trạng thái<select value={state} onChange={(e) => setState(e.target.value)}><option value="">Đang hoạt động</option>{["READY", "ASSIGNED", "IN_PROGRESS", "BLOCKED", "COMPLETED"].map((s) => <option key={s} value={s}>{labels[s]}</option>)}</select></label><label>Kho<input value={warehouse} onChange={(e) => setWarehouse(e.target.value)} placeholder="Tất cả kho được phép" /></label>{page && <small>Dữ liệu lúc {date(page.queue_as_of)}</small>}</div>
    <div className={styles.layout}>
      <aside className={styles.queue}><h2>Hàng đợi đóng gói</h2>{loading && !page && <p role="status">Đang tải task…</p>}{!loading && !page?.items.length && <p>Không có task phù hợp.</p>}{page?.items.map((row) => <button disabled={busy} aria-pressed={task?.task_id === row.task_id} key={row.task_id} onClick={() => void openTask(row.task_id)}><strong>Đơn {row.order_id}</strong><span>{labels[row.workflow_state] ?? row.workflow_state} · {row.warehouse_id}</span><small>{row.line_count} dòng · {row.total_quantity} sản phẩm · {row.assignee_id === employee.principalId ? "Của bạn" : row.assignee_id ? "Đã có người nhận" : "Chưa nhận"}</small><small>Hạn: {date(row.packing_due_at)}</small></button>)}{page?.next_cursor && <button disabled={loading || busy} onClick={() => void loadQueue(page.next_cursor!)}>Tải thêm</button>}</aside>
      <main className={styles.detail}>{!task ? <p>Chọn task để xem và xử lý.</p> : <>
        <header><h2>Đơn {task.order_id}</h2><p>{labels[task.workflow_state] ?? task.workflow_state} · Kho {task.warehouse_id} · Phiên bản {task.version}</p><small>Task {task.task_id}</small></header>
        {management ? <Link className="admin-primary-button" href="/admin/work/packing">Mở Workbench để thao tác</Link> : <div className={styles.actions}>
          {task.workflow_state === "READY" && <button disabled={busy || Boolean(pending)} className="admin-primary-button" onClick={() => mutate("assignments")}>Nhận việc</button>}
          {mine && task.workflow_state === "ASSIGNED" && <button disabled={busy || Boolean(pending)} className="admin-primary-button" onClick={() => mutate("start")}>Bắt đầu đóng gói</button>}
          <button disabled={busy} className="admin-secondary-button" onClick={() => void openTask(task.task_id)}>Tải lại task</button>
        </div>}
        {task.active_blocks.map((block) => <p role="alert" className={styles.pending} key={block.block_id}>Bị chặn: {block.reason_code} ({block.source})</p>)}
        <section><h3>1. Xác nhận hàng theo lô</h3>{task.items.map((line) => <article key={line.order_line_id}><strong>{line.product_name} · {line.sku_code} × {line.quantity}</strong><p>{line.packaging_text}</p>{task.allocations.filter((a) => a.order_line_id === line.order_line_id).map((a) => <form className={styles.pick} key={a.allocation_id} onSubmit={(e) => { e.preventDefault(); const values = new FormData(e.currentTarget); mutate(`picks/${a.allocation_id}`, { counted_quantity: Number(values.get("quantity")), allocation_version: a.allocation_version }, "PUT"); }}>
          <div><strong>Lô {a.batch_code}</strong><small>HSD {date(a.batch_valid_until)} · {a.currently_eligible ? "Hợp lệ ở lần kiểm tra gần nhất" : "Không hợp lệ"}</small></div><label>Đã đếm / {a.allocated_quantity}<input key={`${a.allocation_id}:${task.version}`} aria-label={`Số lượng lô ${a.batch_code}`} name="quantity" type="number" min="0" max={a.allocated_quantity} defaultValue={a.counted_quantity} disabled={!working} required /></label><button disabled={!working || !a.currently_eligible} className="admin-secondary-button">Lưu số lượng</button>
        </form>)}</article>)}</section>
        <section><h3>2. Checklist</h3>{task.checklist.map((c) => <form key={`${c.item_code}:${task.version}`} onSubmit={(e) => { e.preventDefault(); const values = new FormData(e.currentTarget); mutate(`checklist/${c.item_code}`, { result: values.get("result"), note: String(values.get("note") ?? "") }, "PUT"); }}><label><strong>{c.definition}{c.required ? " *" : ""}</strong><span> — {labels[c.result] ?? "Chưa xác nhận"}</span></label><div className={styles.actions}><select name="result" defaultValue={c.result === "FAIL" ? "FAIL" : "PASS"} disabled={!working} aria-label={`Kết quả ${c.definition}`}><option value="PASS">Đạt</option><option value="FAIL">Không đạt</option></select><input name="note" placeholder="Ghi chú" defaultValue={c.note ?? ""} disabled={!working} /><button className="admin-secondary-button" disabled={!working}>Lưu checklist</button></div>{c.confirmed_at && <small>Đã lưu lúc {date(c.confirmed_at)}</small>}</form>)}</section>
        <section><h3>3. Video đóng gói</h3><p>Chọn MP4/WebM tối đa 200 MB. Chỉ video được hệ thống xác minh mới dùng để niêm phong.</p><div className={styles.actions}><input aria-label="Video đóng gói" type="file" accept="video/mp4,video/webm" disabled={!working} onChange={(e) => setVideo(e.target.files?.[0] ?? null)} /><button className="admin-secondary-button" disabled={!working || !video} onClick={() => void upload()}>{busy ? "Đang xử lý…" : "Tải video"}</button></div>{media && <div><p role="status">{labels[media.status] ?? media.status}</p><button disabled={busy} className="admin-secondary-button" onClick={() => void checkMedia()}>Kiểm tra video</button>{media.status === "REJECTED" && <p role="alert">Video không đạt. Hãy chọn video hợp lệ khác.</p>}</div>}</section>
        <section><h3>4. Kiện hàng và niêm phong</h3><p>Kiện {task.package.package_id}</p><div className={styles.dimensions}>{([["weight_grams", "Cân nặng (g)"], ["length_cm", "Dài (cm)"], ["width_cm", "Rộng (cm)"], ["height_cm", "Cao (cm)"]] as const).map(([key, label]) => <label key={key}>{label}<input type="number" min="1" step="1" value={parcel[key] || ""} disabled={!working} onChange={(e) => setParcel((old) => ({ ...old, [key]: Number(e.target.value) }))} /></label>)}</div><label><input type="checkbox" checked={parcel.fragile} disabled={!working} onChange={(e) => setParcel((old) => ({ ...old, fragile: e.target.checked }))} /> Hàng dễ vỡ</label><label className={styles.seal}>Mã niêm phong<input value={seal} disabled={!working} onChange={(e) => setSeal(e.target.value.toUpperCase())} placeholder="VD: SEAL-000001" /></label>
          {task.workflow_state === "COMPLETED" ? <p role="status">Kiện đã được niêm phong. Xem tiến độ vận đơn ở Đơn hàng/Giao vận.</p> : <><ul>{missing.map((item) => <li key={item}>{item}</li>)}</ul><button className="admin-primary-button" disabled={!working || missing.length > 0} onClick={() => setConfirming(true)}>Hoàn tất đóng gói</button></>}
        </section>
        <section><h3>Lịch sử thao tác</h3><button className="admin-secondary-button" disabled={busy} onClick={() => void loadHistory()}>Xem lịch sử</button>{history?.items.map((item) => <p key={item.history_id}>{date(item.occurred_at)} · {item.action} · Phiên bản {item.task_version}<small> · {item.actor_id}{item.reason ? ` · ${item.reason}` : ""}</small></p>)}{history?.next_cursor && <button className="admin-secondary-button" onClick={() => void loadHistory(history.next_cursor!)}>Lịch sử tiếp theo</button>}</section>
      </>}</main>
    </div>
    {confirming && task && <div className="admin-modal-backdrop"><section role="dialog" aria-modal="true" aria-labelledby="seal-title" className={`admin-modal ${styles.dialog}`}><h2 id="seal-title">Xác nhận niêm phong kiện</h2><p>Mã {seal}. Số lượng, checklist và video sẽ được kiểm tra lại tại backend. Kiện đã niêm phong chưa hỗ trợ mở lại.</p><div className={styles.actions}><button className="admin-secondary-button" disabled={busy} onClick={() => setConfirming(false)}>Quay lại</button><button className="admin-primary-button" disabled={!working || missing.length > 0} onClick={() => mutate("seal", { package_id: task.package.package_id, seal_code: seal, parcel, ...(media?.status === "VERIFIED" ? { primary_evidence_id: media.file_id } : {}) })}>Xác nhận hoàn tất</button></div></section></div>}
  </div>;
}
