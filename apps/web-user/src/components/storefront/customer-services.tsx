"use client";

import { type FormEvent, useState } from "react";
import { useRouter } from "next/navigation";

export function CustomerServices() {
  const router = useRouter();
  const [batchCode, setBatchCode] = useState("");
  const [newsletterSent, setNewsletterSent] = useState(false);
  function trace(event: FormEvent) {
    event.preventDefault();
    if (batchCode.trim())
      router.push("/trace/" + encodeURIComponent(batchCode.trim()));
  }
  return (
    <section className="wrap service-forms section-block">
      <div id="ocop-traceability">
        <h2>Tra cứu nguồn gốc</h2>
        <form onSubmit={trace}>
          <label htmlFor="gh-batch">Mã trên nhãn sản phẩm</label>
          <input
            className="field"
            id="gh-batch"
            value={batchCode}
            onChange={(event) => setBatchCode(event.target.value)}
            required
          />
          <div>
            <button className="btn btn--brown" disabled={!batchCode.trim()}>
              Tra cứu
            </button>
            <button
              type="button"
              className="chip-btn"
              onClick={() => setBatchCode("QR-OMA-20261015-LOT08")}
            >
              Dùng mã mẫu
            </button>
          </div>
        </form>
      </div>
      <div>
        <h2>Nhận tin từ Gói Huế</h2>
        <form
          onSubmit={(event) => {
            event.preventDefault();
            event.currentTarget.reset();
            setNewsletterSent(true);
          }}
        >
          <label htmlFor="gh-email">Địa chỉ email</label>
          <input
            className="field"
            id="gh-email"
            type="email"
            autoComplete="email"
            required
          />
          <button className="btn btn--brown">Đăng ký</button>
        </form>
        {newsletterSent && <p role="status">Đã ghi nhận trong bản xem thử.</p>}
      </div>
    </section>
  );
}
