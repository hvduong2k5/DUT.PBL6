"use client";

import Link from "next/link";
import { useCallback, useEffect, useRef, useState } from "react";
import { useEmployeeSession } from "@/components/auth/employee-session-provider";
import { CertificateUpload } from "./certificate-upload";
import { QRLabel } from "./qr-label";
import { traceabilityService as api } from "./service";
import { TraceApiError, emptyTraceContent, type TraceBatchContext, type TraceBinding, type TraceContent, type TraceProfile, type TraceProfilePage } from "./types";
import "./workspace.css";

const STATE: Record<string, string> = { DRAFT: "Bản nháp", IN_REVIEW: "Chờ duyệt", APPROVED: "Đã duyệt", REJECTED: "Cần chỉnh sửa", PUBLISHED: "Đang công khai", ARCHIVED: "Đã lưu trữ", ACTIVE: "Đang hoạt động", SUSPENDED: "Tạm dừng công khai" };
export function TraceabilityWorkspace({ initial }: { initial: { id?: string; batchId?: string; skuCode?: string } }) {
  const { session } = useEmployeeSession(); const [page, setPage] = useState<TraceProfilePage>();
  const [profile, setProfile] = useState<TraceProfile>(); const [context, setContext] = useState<TraceBatchContext>();
  const [content, setContent] = useState<TraceContent>(emptyTraceContent); const [reason, setReason] = useState("");
  const [loading, setLoading] = useState(true); const [busy, setBusy] = useState(false); const [error, setError] = useState(""); const [notice, setNotice] = useState("");
  const [duplicateProfileId, setDuplicateProfileId] = useState<string>();
  const commands = useRef(new Map<string, string>());
  const authorization = JSON.stringify(session.permissionGrants ?? []);
  function can(permission: string, warehouse?: string) {
    return Boolean(session.permissions.some((item) => item === permission) && session.permissionGrants?.some((grant) => grant.permissionCode === permission && (grant.scopeType === "GLOBAL" && !grant.scopeId || grant.scopeType === "WAREHOUSE" && grant.scopeId === warehouse)));
  }
  const select = useCallback((value: TraceProfile) => {
    setProfile(value); setContext(undefined); setReason(""); setDuplicateProfileId(undefined);
    const latest = [...value.revisions].sort((a, b) => b.number - a.number)[0];
    setContent(latest ? structuredClone(latest.content) : emptyTraceContent());
  }, []);
  const load = useCallback(async (cursor = "") => {
    setLoading(true); setError("");
    try { setPage(await api.list(cursor)); } catch (cause) { setError(cause instanceof Error ? cause.message : "Chưa thể tải danh sách hồ sơ."); }
    finally { setLoading(false); }
  }, []);
  const open = useCallback(async (id: string) => {
    setLoading(true); setError("");
    try { select(await api.get(id)); } catch (cause) { setError(cause instanceof Error ? cause.message : "Chưa thể mở hồ sơ."); }
    finally { setLoading(false); }
  }, [select]);
  useEffect(() => {
    let active = true;
    void Promise.resolve().then(async () => {
      if (!active) return;
      setProfile(undefined); setContext(undefined); setNotice(""); setDuplicateProfileId(undefined); commands.current.clear();
      if (initial.id) { await open(initial.id); return; }
      if (initial.batchId && initial.skuCode) {
        setLoading(true); setError("");
        try {
          const source = await api.context(initial.batchId, initial.skuCode);
          const binding: TraceBinding = { product_id: source.product_id, sku_code: source.sku_code, batch_id: source.batch_id, source_namespace: source.source_namespace };
          const found = await api.lookup(binding);
          if (!active) return;
          if (found.items[0]) select(await api.get(found.items[0].profile_id));
          else { setContext(source); setContent(emptyTraceContent()); setNotice("Phiếu nhập đã được giữ nguyên. Lô này chưa có hồ sơ nguồn gốc; bạn có thể lập bản nháp nếu được cấp quyền."); }
        } catch (cause) { if (active) setError(cause instanceof Error ? cause.message : "Chưa thể lấy nguồn lô. Phiếu nhập kho vẫn giữ nguyên."); }
        finally { if (active) setLoading(false); }
      } else await load();
    });
    return () => { active = false; };
  }, [initial.id, initial.batchId, initial.skuCode, authorization, load, open, select]);
  const revision = profile ? [...profile.revisions].sort((a, b) => b.number - a.number)[0] : undefined;
  const dirty = Boolean(revision && JSON.stringify(content) !== JSON.stringify(revision.content));
  const warehouse = profile?.warehouse_id ?? context?.warehouse_id;
  const readable = !profile || can("traceability.read_internal", warehouse);
  const editable = profile ? profile.status === "ACTIVE" && revision?.status === "DRAFT" && can("traceability.edit", warehouse) : Boolean(context && can("traceability.create", warehouse));
  async function perform(kind: string, payload: unknown, operation: (key: string) => Promise<TraceProfile>) {
    setBusy(true); setError(""); setNotice("");
    const intent = `${session.employee.employeeId}:${profile?.id ?? context?.batch_id}:${kind}:${JSON.stringify(payload)}`;
    let key = commands.current.get(intent); if (!key) { key = crypto.randomUUID(); commands.current.set(intent, key); }
    try { const result = await operation(key); commands.current.delete(intent); select(result); setNotice("Đã ghi nhận thao tác hồ sơ. Phiếu nhập kho không thay đổi."); }
    catch (cause) { setError(cause instanceof Error ? cause.message : "Chưa thể hoàn tất thao tác."); }
    finally { setBusy(false); }
  }
  function save() {
    if (profile && revision) { void perform("edit", { content, version: profile.version }, (key) => api.edit(profile.id, revision.number, profile.version, content, key)); return; }
    if (!context) return;
    const binding: TraceBinding = { product_id: context.product_id, sku_code: context.sku_code, source_namespace: context.source_namespace, batch_id: context.batch_id };
    void perform("create", { binding, content }, async (key) => {
      try { return await api.create(binding, content, key); }
      catch (cause) { if (!(cause instanceof TraceApiError) || cause.status !== 409) throw cause; const found = await api.lookup(binding); if (found.items[0]) setDuplicateProfileId(found.items[0].profile_id); throw cause; }
    });
  }
  function action(value: string) {
    if (!profile || !revision) return;
    if (dirty) { setError("Lưu bản nháp trước khi thực hiện thao tác này."); return; }
    if (["approve", "reject", "suspend", "archive"].includes(value) && !reason.trim()) { setError("Nhập lý do trước khi ghi nhận quyết định."); return; }
    void perform(value, { version: profile.version, revision: revision.number, reason }, (key) => api.action(profile.id, revision.number, value, profile.version, reason, key));
  }
  function clone() {
    if (!profile || !revision) return;
    if (dirty) { setError("Lưu bản nháp trước khi tạo bản sửa đổi."); return; }
    if (revision.status === "DRAFT" && !reason.trim()) { setError("Nhập lý do thay thế bản nháp đang làm."); return; }
    void perform("revision", { version: profile.version, revision: revision.number, reason }, (key) => api.revision(profile.id, revision.number, profile.version, reason, key));
  }
  const contributor = revision?.contributors.includes(session.employee.employeeId);
  return <div className="admin-page trace-workspace">
    <header className="trace-header"><div><p>Quản lý nguồn gốc theo lô</p><h1>Hồ sơ nguồn gốc</h1><p>Lập hồ sơ độc lập với nhập kho; duyệt trước khi công bố và in tem.</p></div><Link href="/admin/work/warehouse">Về nhập kho</Link></header>
    {error ? <div className="admin-inline-error" role="alert"><p>{error}</p>{duplicateProfileId ? <p>Lô này đã có hồ sơ. Nội dung đang nhập chưa được lưu. <Link href={`/admin/traceability?id=${duplicateProfileId}`}>Mở hồ sơ đã có</Link></p> : null}<button disabled={busy} onClick={() => profile ? void open(profile.id) : void load()}>Tải lại dữ liệu</button></div> : null}
    {notice ? <p className="trace-notice" role="status">{notice}</p> : null}
    {loading ? <p aria-busy="true">Đang kiểm tra quyền và tải hồ sơ…</p> : null}
    {!loading && !readable ? <p role="alert">Bạn không còn quyền xem hồ sơ trong kho này.</p> : null}
    {!loading && readable && (profile || context) ? <>
      <section className="trace-card"><h2>{profile ? content.product_name || profile.sku_code : `Lập hồ sơ cho lô ${context?.batch_code}`}</h2>
        <p>SKU {profile?.sku_code ?? context?.sku_code} · {profile ? STATE[profile.status] : "Nguồn lô đã được xác minh theo kho được giao"}</p>
        {revision ? <p>Bản {revision.number} · {STATE[revision.status]} · Lần cập nhật {profile?.version}</p> : null}
        {context ? <p>NSX {context.production_date} · HSD {context.expiry_date}{context.restricted ? " · Lô đang bị hạn chế; chưa thể gửi duyệt/công bố." : ""}{context.depleted ? " · Lô đã hết tồn, thông tin nguồn gốc vẫn được giữ." : ""}</p> : null}
      </section>
      <section className="trace-card"><h2>Nội dung hồ sơ {revision ? `— bản ${revision.number}` : "— bản nháp mới"}</h2>
        <fieldset disabled={!editable || busy}>
          <div className="trace-form-grid">
            <label>Tên sản phẩm<input value={content.product_name} maxLength={250} onChange={(event) => setContent({ ...content, product_name: event.target.value })} /></label>
            <label>Cơ sở sản xuất<input value={content.producer} maxLength={250} onChange={(event) => setContent({ ...content, producer: event.target.value })} /></label>
            <label>Địa điểm sản xuất<input value={content.workshop_location} maxLength={1000} onChange={(event) => setContent({ ...content, workshop_location: event.target.value })} /></label>
            <label>Nghệ nhân (nếu có)<input value={content.artisan_name ?? ""} maxLength={250} onChange={(event) => setContent({ ...content, artisan_name: event.target.value || null })} /></label>
          </div>
          <label>Quy trình sản xuất<textarea rows={5} maxLength={8000} value={content.process} onChange={(event) => setContent({ ...content, process: event.target.value })} /></label>
          <h3>Nguyên liệu và nơi cung cấp</h3>
          {content.ingredients.map((item, index) => <div className="trace-form-grid" key={index}>
            <label>Nguyên liệu<input value={item.name} maxLength={250} onChange={(event) => setContent({ ...content, ingredients: content.ingredients.map((entry, position) => position === index ? { ...entry, name: event.target.value } : entry) })} /></label>
            <label>Nguồn cung<input value={item.origin} maxLength={1000} onChange={(event) => setContent({ ...content, ingredients: content.ingredients.map((entry, position) => position === index ? { ...entry, origin: event.target.value } : entry) })} /></label>
            <button type="button" onClick={() => setContent({ ...content, ingredients: content.ingredients.filter((_, position) => position !== index) })}>Bỏ nguyên liệu</button>
          </div>)}
          <button type="button" disabled={content.ingredients.length >= 20} onClick={() => setContent({ ...content, ingredients: [...content.ingredients, { name: "", origin: "" }] })}>Thêm nguyên liệu</button>
        </fieldset>
        <h3>Chứng nhận</h3>
        {content.certificates.length === 0 ? <p>Chưa có chứng nhận. Bạn có thể lưu nháp và bổ sung trước khi gửi duyệt.</p> : null}
        {content.certificates.map((certificate) => <div className="trace-certificate-row" key={certificate.id}><div><strong>{certificate.type} {certificate.stars ? `${certificate.stars} sao` : ""} · {certificate.number}</strong><p>{certificate.issuer} · Cấp {certificate.issued_on} · {certificate.valid_until ? `Hết hạn ${certificate.valid_until}` : "Theo lịch rà soát chứng nhận"}</p>
          {profile && revision ? certificate.evidence_ids.map((id) => content.evidence.find((item) => item.id === id)?.reference ? revision.content.evidence.some((item) => item.id === id) ? <a key={id} href={`/api/admin/trace-files/${profile.id}/${id}?revision=${revision.number}`} target="_blank" rel="noreferrer">Mở bản scan để đối chiếu</a> : <span key={id}>Lưu bản nháp để mở bản scan từ hồ sơ.</span> : <span key={id}>Bản scan cần quyền xem bằng chứng riêng.</span>) : null}</div>
          {editable ? <button type="button" disabled={busy} onClick={() => setContent({ ...content, certificates: content.certificates.filter((item) => item.id !== certificate.id) })}>Bỏ chứng nhận</button> : null}</div>)}
        {editable && !busy ? <CertificateUpload productId={profile?.product_id ?? context!.product_id} onAdd={(certificate, evidence) => setContent((current) => ({ ...current, certificates: [...current.certificates, certificate], evidence: [...current.evidence, evidence] }))} /> : null}
        {editable ? <button className="admin-primary-button" disabled={busy} onClick={save}>{busy ? "Đang ghi nhận…" : "Lưu bản nháp"}</button> : null}
      </section>
      {profile && revision ? <section className="trace-card"><h2>Duyệt và công bố</h2>
        <label>Lý do / ghi nhận quyết định<textarea rows={2} maxLength={1000} value={reason} disabled={busy} onChange={(event) => setReason(event.target.value)} /></label>
        {dirty ? <p role="status">Có nội dung chưa lưu. Lưu bản nháp trước khi gửi duyệt hoặc đổi trạng thái.</p> : null}
        {can("traceability.suspend", warehouse) || can("traceability.archive", warehouse) ? <p>Tạm dừng hoặc lưu trữ sẽ chặn công khai và in tem. Hiện chưa có thao tác khôi phục hồ sơ.</p> : null}
        <div className="trace-actions">
          {profile.status === "ACTIVE" && revision.status === "DRAFT" && can("traceability.submit", warehouse) ? <button disabled={busy || dirty} onClick={() => action("submit")}>Gửi duyệt bản đã lưu</button> : null}
          {profile.status === "ACTIVE" && revision.status === "IN_REVIEW" && can("traceability.approve", warehouse) && !contributor ? <><button disabled={busy || dirty} onClick={() => action("approve")}>Duyệt hồ sơ</button><button disabled={busy || dirty} onClick={() => action("reject")}>Yêu cầu chỉnh sửa</button></> : null}
          {revision.status === "IN_REVIEW" && contributor ? <p>Người đã soạn hoặc chỉnh sửa bản này không được tự duyệt.</p> : null}
          {profile.status === "ACTIVE" && revision.status === "APPROVED" && can("traceability.publish", warehouse) ? <button disabled={busy || dirty} onClick={() => action("publish")}>Công bố hồ sơ</button> : null}
          {profile.status === "ACTIVE" && can("traceability.edit", warehouse) ? <button disabled={busy || dirty} onClick={clone}>Tạo bản sửa đổi</button> : null}
          {profile.status === "ACTIVE" && can("traceability.suspend", warehouse) ? <button disabled={busy || dirty} onClick={() => action("suspend")}>Tạm dừng công khai</button> : null}
          {profile.status !== "ARCHIVED" && can("traceability.archive", warehouse) ? <button disabled={busy || dirty} onClick={() => action("archive")}>Lưu trữ hồ sơ</button> : null}
        </div><p>Gửi duyệt/công bố dùng nội dung đã lưu. Sau khi sửa, lưu bản nháp trước khi gửi.</p>
      </section> : null}
      {profile ? <QRLabel key={`${profile.id}:${profile.version}`} profile={profile} /> : null}
    </> : null}
    {!loading && !profile && !context ? <section className="trace-card"><h2>Hồ sơ trong phạm vi được giao</h2>
      {page?.items.length === 0 ? <p>Chưa có hồ sơ. Mở một lô từ kết quả nhập kho để lập hồ sơ nguồn gốc.</p> : null}
      <div className="trace-list">{page?.items.map((item) => <button key={item.id} onClick={() => void open(item.id)}><strong>{[...item.revisions].sort((a, b) => b.number - a.number)[0]?.content.product_name || item.sku_code}</strong><span>{item.sku_code} · {STATE[item.status]} · {item.public_revision ? "Đã có bản công khai" : "Chưa công bố"}</span></button>)}</div>
      {page?.next_cursor ? <button onClick={() => void load(page.next_cursor!)}>Trang tiếp theo</button> : null}
    </section> : null}
    {profile || context ? <button disabled={busy} onClick={() => { setProfile(undefined); setContext(undefined); void load(); }}>Về danh sách hồ sơ</button> : null}
  </div>;
}
