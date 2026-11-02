"use client";

import Link from "next/link";
import { FormEvent, useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import type { TraceabilityRecord } from "@/lib/customer-core/types";
import { CustomerCoreApiError } from "@/lib/customer-core/types";
import { customerCoreService } from "@/services/customer-core-service";

const DATE = new Intl.DateTimeFormat("vi-VN", { day: "2-digit", month: "2-digit", year: "numeric" });

function formatDate(value: string) {
  const date = new Date(`${value}T00:00:00`);
  return Number.isNaN(date.valueOf()) ? value : DATE.format(date);
}

export function TraceExperience({ qrCode }: { qrCode: string }) {
  const router = useRouter();
  const searchParams = useSearchParams();
  const scenario = searchParams.get("mockScenario") ?? undefined;
  const [query, setQuery] = useState(qrCode);
  const [record, setRecord] = useState<TraceabilityRecord>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<{ status?: number; message: string }>();
  const [retryKey, setRetryKey] = useState(0);

  useEffect(() => {
    let active = true;
    customerCoreService.getTraceability(qrCode, scenario)
      .then((payload) => { if (active) setRecord(payload); })
      .catch((cause: unknown) => {
        if (!active) return;
        setError(cause instanceof CustomerCoreApiError
          ? { status: cause.status, message: cause.message }
          : { message: "Chưa thể truy xuất thông tin sản phẩm lúc này." });
      })
      .finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, [qrCode, retryKey, scenario]);

  function retryTraceability() {
    setLoading(true);
    setError(undefined);
    setRecord(undefined);
    setRetryKey((value) => value + 1);
  }

  function lookup(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const value = query.trim();
    if (!value) return;
    const suffix = scenario ? `?mockScenario=${encodeURIComponent(scenario)}` : "";
    router.push(`/trace/${encodeURIComponent(value)}${suffix}`);
  }

  return (
    <main className="trace-page">
      <section className="trace-search-hero">
        <p className="eyebrow">Hồ sơ nguồn gốc OCOP</p>
        <h1>Kiểm chứng hành trình của từng thức quà</h1>
        <p>Nhập mã QR in trên bao bì để xem lô sản xuất, nghệ nhân, nguyên liệu và chứng nhận an toàn thực phẩm.</p>
        <form onSubmit={lookup}>
          <label className="sr-only" htmlFor="trace-code">Mã QR sản phẩm</label>
          <input id="trace-code" value={query} required maxLength={160} onChange={(event) => setQuery(event.target.value)} placeholder="Ví dụ: QR-OMA-20261015-LOT08" />
          <button type="submit">Truy xuất ngay</button>
        </form>
      </section>

      {loading ? <section className="trace-loading" aria-busy="true" aria-label="Đang tải hồ sơ truy xuất"><span /><span /><span /><span /></section> : null}

      {!loading && error ? <section className="trace-state" role="alert">
        <span aria-hidden="true">{error.status === 404 ? "⌕" : "!"}</span>
        <h2>{error.status === 404 ? "Không tìm thấy mã truy xuất" : "Chưa thể mở hồ sơ nguồn gốc"}</h2>
        <p>{error.message}</p>
        <div>{error.status !== 404 ? <button type="button" onClick={retryTraceability}>Thử lại</button> : null}<Link href="/products">Khám phá sản phẩm</Link></div>
      </section> : null}

      {!loading && !error && record ? <>
        <section className="trace-authenticity">
          <div className="trace-seal" aria-hidden="true">✺<small>OCOP</small></div>
          <div><span className="trace-valid">✓ Hồ sơ hợp lệ</span><h2>{record.product_name}</h2><p>Mã truy xuất <strong>{record.qr_code}</strong> đã được hệ thống Ô Mạ xác nhận.</p></div>
          <div className="trace-stars"><strong>{record.ocop_star} sao</strong><span>{"★".repeat(Math.max(0, Math.min(5, record.ocop_star)))}</span><small>{record.ocop_certificate_no}</small></div>
        </section>

        <section className="trace-record-grid">
          <article className="trace-batch-card">
            <p className="eyebrow">Lô sản xuất</p><h2>{record.batch_code}</h2>
            <dl><div><dt>Ngày sản xuất</dt><dd>{formatDate(record.production_date)}</dd></div><div><dt>Hạn sử dụng</dt><dd>{formatDate(record.expiry_date)}</dd></div><div><dt>Nghệ nhân phụ trách</dt><dd>{record.artisan_name}</dd></div><div><dt>Xưởng sản xuất</dt><dd>{record.workshop_location}</dd></div></dl>
          </article>
          <article className="trace-origin-card"><span aria-hidden="true">♧</span><p className="eyebrow">Nguồn nguyên liệu</p><h2>Từ thổ nhưỡng xứ Huế</h2><p>{record.raw_materials_origin}</p></article>
          <article className="trace-safety-card"><span aria-hidden="true">✓</span><p className="eyebrow">An toàn thực phẩm</p><h2>Chứng nhận kiểm định</h2><p>{record.food_safety_cert}</p></article>
        </section>

        <section className="trace-actions">
          <div><p className="eyebrow">Minh bạch từ xưởng</p><h2>Xem quy trình tạo nên thức quà</h2><p>Hồ sơ được liên kết với đúng mã lô để khách hàng có thể kiểm chứng trước khi sử dụng hoặc mua lại.</p></div>
          <div>{record.crafting_video_url ? <a className="primary-button" href={record.crafting_video_url} target="_blank" rel="noreferrer">Xem video chế tác ↗</a> : null}<Link className="secondary-button" href={`/products?q=${encodeURIComponent(record.product_name)}`}>Tìm sản phẩm</Link></div>
        </section>
      </> : null}
    </main>
  );
}
