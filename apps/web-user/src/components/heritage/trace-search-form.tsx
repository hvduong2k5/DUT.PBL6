"use client";

import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";
import { traceCode } from "@/lib/customer-core/trace-code";

/** QR lookup form shared by the home traceability banner and the /trace page. Navigates to /trace/[code]. */
export function TraceSearchForm({ initialCode = "", suffix = "", inputId = "trace-code" }: { initialCode?: string; suffix?: string; inputId?: string }) {
  const router = useRouter();
  const [code, setCode] = useState(initialCode);
  const [error, setError] = useState("");

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const value = traceCode(code, [window.location.origin, process.env.NEXT_PUBLIC_TRACEABILITY_PUBLIC_ORIGIN ?? ""]);
    if (!value) { setError("Nhập mã trên tem hoặc đường dẫn truy xuất của cửa hàng."); return; }
    setError("");
    router.push(`/trace/${encodeURIComponent(value)}${suffix}`);
  }

  return <form className="gh-trace-form" onSubmit={submit}>
    <label className="gh-trace-form__label" htmlFor={inputId}>Mã QR trên nhãn hộp</label>
    <div className="gh-trace-form__row">
      <input id={inputId} className="gh-field" value={code} required maxLength={512} onChange={(event) => setCode(event.target.value)} placeholder="Nhập mã hoặc dán đường dẫn từ tem QR" />
      <button type="submit" className="gh-btn gh-btn--fold">Kiểm tra <span className="gh-btn__arrow" aria-hidden="true">→</span></button>
    </div>
    {error ? <p role="alert">{error}</p> : <p>Dùng camera điện thoại quét tem QR để mở trực tiếp hồ sơ.</p>}
  </form>;
}
