"use client";

import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";

const SAMPLE_CODE = "QR-OMA-20261015-LOT08";

/** QR lookup form shared by the home traceability banner and the /trace page. Navigates to /trace/[code]. */
export function TraceSearchForm({ initialCode = "", suffix = "", inputId = "trace-code" }: { initialCode?: string; suffix?: string; inputId?: string }) {
  const router = useRouter();
  const [code, setCode] = useState(initialCode);

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const value = code.trim();
    if (!value) return;
    router.push(`/trace/${encodeURIComponent(value)}${suffix}`);
  }

  return <form className="gh-trace-form" onSubmit={submit}>
    <label className="gh-trace-form__label" htmlFor={inputId}>Mã QR trên nhãn hộp</label>
    <div className="gh-trace-form__row">
      <input id={inputId} className="gh-field" value={code} required maxLength={160} onChange={(event) => setCode(event.target.value)} placeholder={`Ví dụ: ${SAMPLE_CODE}`} />
      <button type="submit" className="gh-btn gh-btn--fold">Kiểm tra <span className="gh-btn__arrow" aria-hidden="true">→</span></button>
    </div>
    <button type="button" className="gh-trace-form__sample" onClick={() => setCode(SAMPLE_CODE)}>Dùng mã mẫu</button>
  </form>;
}
