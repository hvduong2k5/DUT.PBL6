"use client";

import { useEffect, useRef, useState } from "react";
import { createPortal, flushSync } from "react-dom";
import QRCode from "qrcode";
import { customerCoreService } from "@/services/customer-core-service";
import type { TraceabilityRecord } from "@/lib/customer-core/types";
import type { TraceProfile } from "./types";

function publicURL(profile: TraceProfile): string {
  const url = new URL(profile.qr_url ?? "");
  const origins = new Set([window.location.origin, process.env.NEXT_PUBLIC_TRACEABILITY_PUBLIC_ORIGIN].filter(Boolean));
  if (!origins.has(url.origin) || url.username || url.password || url.search || url.hash || url.pathname !== `/trace/${encodeURIComponent(profile.qr_code)}`) throw new Error("Đường dẫn tem chưa đúng địa chỉ tra cứu được cấu hình.");
  return url.href;
}
export function QRLabel({ profile }: { profile: TraceProfile }) {
  const canvas = useRef<HTMLCanvasElement>(null); const [record, setRecord] = useState<TraceabilityRecord>();
  const printCanvas = useRef<HTMLCanvasElement>(null);
  const [printRecord, setPrintRecord] = useState<TraceabilityRecord>();
  useEffect(() => {
    const clear = () => setPrintRecord(undefined);
    window.addEventListener("afterprint", clear);
    return () => window.removeEventListener("afterprint", clear);
  }, []);
  const [error, setError] = useState(""); const [busy, setBusy] = useState(false);
  async function eligible() {
    const url = publicURL(profile);
    if (profile.status !== "ACTIVE" || !profile.public_revision) throw new Error("Hồ sơ chưa được công bố hoặc đã dừng công khai.");
    const value = await customerCoreService.getTraceability(profile.qr_code);
    if (value.qr_code !== profile.qr_code || !value.batch_code || !value.product_name) throw new Error("Thông tin công khai chưa khớp tem.");
    return { url, value };
  }
  async function prepare() {
    setBusy(true); setError(""); setRecord(undefined);
    try { const { url, value } = await eligible(); setRecord(value); if (canvas.current) await QRCode.toCanvas(canvas.current, url, { width: 256, margin: 4, errorCorrectionLevel: "M", color: { dark: "#000000", light: "#ffffff" } }); }
    catch (cause) { setError(cause instanceof Error ? cause.message : "Không thể chuẩn bị tem."); }
    finally { setBusy(false); }
  }
  async function exportLabel(format: "png" | "svg" | "print") {
    setBusy(true); setError("");
    try {
      const { url, value } = await eligible();
      flushSync(() => setRecord(value));
      if (format === "print") {
        flushSync(() => setPrintRecord(value));
        if (!printCanvas.current) throw new Error("Chưa thể chuẩn bị bố cục tem.");
        await QRCode.toCanvas(printCanvas.current, url, { width: 256, margin: 4, errorCorrectionLevel: "M" });
        window.print(); return;
      }
      const data = format === "svg" ? URL.createObjectURL(new Blob([await QRCode.toString(url, { type: "svg", width: 512, margin: 4, errorCorrectionLevel: "M" })], { type: "image/svg+xml" })) : await QRCode.toDataURL(url, { width: 512, margin: 4, errorCorrectionLevel: "M" });
      const link = document.createElement("a"); link.href = data; link.download = `ma-nguon-goc-${value.batch_code.replace(/[^A-Za-z0-9_-]/gu, "_")}.${format}`; link.click();
      if (format === "svg") setTimeout(() => URL.revokeObjectURL(data), 1000);
    } catch (cause) { setPrintRecord(undefined); setRecord(undefined); setError(cause instanceof Error ? cause.message : "Tem chưa đủ điều kiện tải/in."); }
    finally { setBusy(false); }
  }
  return <section className="trace-card">
    <h2>Tem nguồn gốc theo lô</h2><p>In theo thông tin đang công khai. Mã giữ nguyên khi in lại; bản nháp mới không thay thông tin trên tem.</p>
    <button type="button" disabled={busy} onClick={() => void prepare()}>Kiểm tra và chuẩn bị tem</button>
    {error ? <p role="alert" className="admin-inline-error">{error}</p> : null}
    <div className="trace-print-label" hidden={!record}>
      <strong>{record?.product_name}</strong><span>Lô {record?.batch_code}</span>
      <canvas ref={canvas} aria-label="Mã QR dẫn đến hồ sơ nguồn gốc công khai" />
      <small>NSX {record?.production_date} · HSD {record?.expiry_date}</small><small>Quét mã để xem nguồn gốc</small>
    </div>
    {record ? <div className="trace-actions"><button disabled={busy} onClick={() => void exportLabel("png")}>Tải mã PNG</button><button disabled={busy} onClick={() => void exportLabel("svg")}>Tải mã SVG</button><button disabled={busy} onClick={() => void exportLabel("print")}>In tem</button></div> : null}
    {printRecord && typeof document !== "undefined" ? createPortal(<div className="trace-print-sheet"><div className="trace-print-label">
      <strong>{printRecord.product_name}</strong><span>Lô {printRecord.batch_code}</span>
      <canvas ref={printCanvas} aria-label={`Mã truy xuất lô ${printRecord.batch_code}`} />
      <small>NSX {printRecord.production_date} · HSD {printRecord.expiry_date}</small>
      <small>Quét QR để xem nguồn gốc</small>
    </div></div>, document.body) : null}
  </section>;
}
