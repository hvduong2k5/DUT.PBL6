"use client";

import { useRef, useState } from "react";
import type { Certificate, Evidence } from "./types";

interface Session { uploadSessionId: string; fileId: string; uploadUrl: string; uploadPermit: string; requiredHeaders: Record<string, string> }
interface Attempt { fingerprint: string; createKey: string; completeKey: string; session?: Session; uploaded?: boolean }
async function media<T>(path: string, method = "GET", body?: object, key?: string, permit?: string): Promise<T> {
  const response = await fetch(`/api/admin/trace-media/${path}`, { method, credentials: "include", cache: "no-store", headers: { Accept: "application/json", ...(body ? { "Content-Type": "application/json", "Idempotency-Key": key! } : {}), ...(permit ? { "X-Upload-Permit": permit } : {}) }, body: body ? JSON.stringify(body) : undefined });
  const result = await response.json();
  if (!response.ok) throw new Error(result.message ?? "Không thể tải bản scan lúc này.");
  return result as T;
}
export function CertificateUpload({ productId, onAdd }: { productId: string; onAdd: (certificate: Certificate, evidence: Evidence) => void }) {
  const [file, setFile] = useState<File>(); const [number, setNumber] = useState(""); const [issuer, setIssuer] = useState("");
  const [issued, setIssued] = useState(""); const [until, setUntil] = useState(""); const [stars, setStars] = useState(3);
  const [publicConsent, setPublicConsent] = useState(false); const [verified, setVerified] = useState<string>();
  const [busy, setBusy] = useState(false); const [notice, setNotice] = useState(""); const attempt = useRef<Attempt | undefined>(undefined);
  async function add() {
    if (!file || !number.trim() || !issuer.trim() || !issued || !until || !publicConsent || !verified) { setNotice("Điền đủ thông tin và xác nhận đã đối chiếu bản scan được phép công khai."); return; }
    if (!new Set(["application/pdf", "image/jpeg", "image/png", "image/webp"]).has(file.type) || file.size < 1 || file.size > 20971520) { setNotice("Chọn PDF, JPEG, PNG hoặc WebP không quá 20 MB."); return; }
    if (until < issued) { setNotice("Ngày hết hạn phải từ ngày cấp trở đi."); return; }
    setBusy(true); setNotice("Đang chuẩn bị bản scan…");
    try {
      const hash = [...new Uint8Array(await crypto.subtle.digest("SHA-256", await file.arrayBuffer()))].map((value) => value.toString(16).padStart(2, "0")).join("");
      const fingerprint = `${file.name}:${file.size}:${hash}`;
      if (attempt.current?.fingerprint !== fingerprint) attempt.current = { fingerprint, createKey: crypto.randomUUID(), completeKey: crypto.randomUUID() };
      const current = attempt.current;
      if (!current) throw new Error("Chưa thể tạo phiên tải lên.");
      if (!current.session) current.session = await media<Session>("upload-sessions", "POST", { purpose: "OCOP_CERTIFICATE", fileName: file.name, mediaType: file.type, sizeBytes: file.size, sha256: hash }, current.createKey);
      const session = current.session;
      if (!current.uploaded) {
        setNotice("Đang tải bản scan lên kho tài liệu…");
        const uploaded = await fetch(session.uploadUrl, { method: "PUT", headers: session.requiredHeaders, body: file, credentials: "omit" });
        if (!uploaded.ok) throw new Error("Kho tài liệu chưa nhận được bản scan. Bạn có thể thử lại.");
        current.uploaded = true;
      }
      setNotice("Đang kiểm tra tập tin…");
      await media(`upload-sessions/${session.uploadSessionId}/complete`, "POST", {}, current.completeKey, session.uploadPermit);
      let ready = false;
      for (let index = 0; index < 20; index++) {
        const info = await media<{ fileId: string; status: string; sha256Checksum: string }>(`files/${session.fileId}`, "GET", undefined, undefined, session.uploadPermit);
        if (info.status === "READY") { if (info.sha256Checksum !== hash) throw new Error("Thông tin kiểm tra tập tin không khớp. Hãy thử tải lại bản scan."); ready = true; break; }
        if (info.status === "PROCESSING_FAILED") throw new Error("Bản scan chưa xử lý được; hãy chọn một tập tin khác.");
        await new Promise((resolve) => setTimeout(resolve, 1000));
      }
      if (!ready) throw new Error("Bản scan vẫn đang xử lý. Bấm lại để kiểm tra, hồ sơ đã lưu vẫn được giữ nguyên.");
      const evidenceId = crypto.randomUUID();
      onAdd({ id: crypto.randomUUID(), type: "OCOP", number: number.trim(), issuer: issuer.trim(), scope: "PRODUCT", scope_id: productId, issued_on: issued, valid_until: until, no_expiry_stated: false, review_due_at: null, stars, verification_method: "MANUAL", verified_at: verified, revoked: false, evidence_ids: [evidenceId] }, { id: evidenceId, reference: `document:${session.fileId}`, kind: "CERTIFICATE", description: `Bản scan chứng nhận OCOP ${number.trim()}` });
      setNotice("Đã thêm chứng nhận vào bản nháp đang sửa. Bấm Lưu bản nháp để ghi nhận.");
      setFile(undefined); setNumber(""); setIssuer(""); setVerified(undefined); setPublicConsent(false); attempt.current = undefined;
    } catch (error) { setNotice(error instanceof Error ? error.message : "Chưa thể thêm chứng nhận."); }
    finally { setBusy(false); }
  }
  return <fieldset className="trace-certificate" disabled={busy}>
    <legend>Thêm chứng nhận OCOP</legend>
    <div className="trace-form-grid">
      <label>Số chứng nhận<input value={number} onChange={(event) => { setNumber(event.target.value); setVerified(undefined); }} maxLength={250} /></label>
      <label>Đơn vị cấp<input value={issuer} onChange={(event) => { setIssuer(event.target.value); setVerified(undefined); }} maxLength={250} /></label>
      <label>Ngày cấp<input type="date" value={issued} onChange={(event) => { setIssued(event.target.value); setVerified(undefined); }} /></label>
      <label>Ngày hết hạn<input type="date" value={until} onChange={(event) => { setUntil(event.target.value); setVerified(undefined); }} /></label>
      <label>Hạng OCOP<select value={stars} onChange={(event) => { setStars(Number(event.target.value)); setVerified(undefined); }}>{[1, 2, 3, 4, 5].map((value) => <option key={value} value={value}>{value} sao</option>)}</select></label>
      <label>Bản scan<input type="file" accept="application/pdf,image/jpeg,image/png,image/webp" onChange={(event) => { setFile(event.target.files?.[0]); setVerified(undefined); }} /></label>
    </div>
    <label className="trace-check"><input type="checkbox" checked={publicConsent} onChange={(event) => setPublicConsent(event.target.checked)} />Bản scan này được phép công khai, không chứa tài liệu nội bộ hoặc thông tin cá nhân cần giữ kín.</label>
    <label className="trace-check"><input type="checkbox" checked={Boolean(verified)} onChange={(event) => setVerified(event.target.checked ? new Date().toISOString() : undefined)} />Tôi đã đối chiếu số, đơn vị cấp, hạng và thời hạn với bản scan.</label>
    <button type="button" className="admin-primary-button" disabled={busy} onClick={() => void add()}>{busy ? "Đang tải và kiểm tra…" : "Tải và thêm chứng nhận"}</button>
    {notice ? <p role="status">{notice}</p> : null}
  </fieldset>;
}
