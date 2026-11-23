"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import { HeritageBanner } from "@/components/heritage/heritage-banner";
import { Motif } from "@/components/heritage/motif";
import { TraceSearchForm } from "@/components/heritage/trace-search-form";
import type { TraceabilityRecord } from "@/lib/customer-core/types";
import { CustomerCoreApiError } from "@/lib/customer-core/types";
import { customerCoreService } from "@/services/customer-core-service";

const DATE = new Intl.DateTimeFormat("vi-VN", { day: "2-digit", month: "2-digit", year: "numeric" });

function formatDate(value: string) {
  const date = new Date(`${value}T00:00:00`);
  return Number.isNaN(date.valueOf()) ? value : DATE.format(date);
}

export function TraceExperience({ qrCode }: { qrCode: string }) {
  const searchParams = useSearchParams();
  const scenario = searchParams.get("mockScenario") ?? undefined;
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

  const suffix = scenario ? `?mockScenario=${encodeURIComponent(scenario)}` : "";

  return (
    <main className="gh-page" id="main-content">
      <section className="gh-home-band" aria-labelledby="trace-title">
        <Motif name="09-cong-thanh" className="gh-band__gate" />
        <Motif name="02-canh-hoa-phai" className="gh-band__blossom-right" />
        <div className="gh-wrap">
          <HeritageBanner level={1} titleId="trace-title" kicker="Hồ sơ nguồn gốc OCOP" title="Kiểm chứng hành trình của từng thức quà"
            lead="Nhập mã QR in trên bao bì để xem lô sản xuất, nghệ nhân, nguyên liệu và chứng nhận an toàn thực phẩm."
            image={{ src: "/images/heritage/photos/sp-goi-le-tra.jpg", alt: "Túi mè xửng và trà cung đình Gói Huế có nhãn mã QR truy xuất" }}>
            <TraceSearchForm initialCode={qrCode} suffix={suffix} />
          </HeritageBanner>
        </div>
      </section>

      <section className="gh-page-section" aria-live="polite">
        <Motif name="03-canh-tra" className="gh-page-section__branch" />
        <div className="gh-wrap">
          {loading ? <div className="gh-skeleton-grid" aria-busy="true" aria-label="Đang tải hồ sơ truy xuất">
            {[0, 1, 2].map((index) => <div key={index} className="gh-frame-card gh-skeleton-card" />)}
          </div> : null}

          {!loading && error ? <div className="gh-frame-functional gh-frame-functional--fold gh-state" role="alert">
            <span className="gh-kicker">{error.status === 404 ? "Mã chưa có trong hệ thống" : "Kết nối gián đoạn"}</span>
            <h2>{error.status === 404 ? "Không tìm thấy mã truy xuất" : "Chưa thể mở hồ sơ nguồn gốc"}</h2>
            <p>{error.message}</p>
            <div className="gh-state__actions">
              {error.status !== 404 ? <button type="button" className="gh-btn gh-btn--brown" onClick={retryTraceability}>Thử lại</button> : null}
              <Link className="gh-btn gh-btn--paper" href="/products">Khám phá sản phẩm</Link>
            </div>
          </div> : null}

          {!loading && !error && record ? <>
            <div className="gh-frame-plaque gh-trace-cert">
              <div>
                <span className="gh-badge gh-badge--gold">Hồ sơ hợp lệ</span>
                <h2>{record.product_name}</h2>
                <p>Mã truy xuất <strong>{record.qr_code}</strong> đã được hệ thống Gói Huế xác nhận.</p>
              </div>
              <div className="gh-trace-cert__seal gh-surface-imperial">
                <span>OCOP</span>
                <strong>{record.ocop_star} sao</strong>
                <small>{record.ocop_certificate_no}</small>
              </div>
            </div>

            <div className="gh-trace-grid">
              <article className="gh-frame-card gh-trace-card">
                <span className="gh-kicker">Lô sản xuất</span>
                <h3>{record.batch_code}</h3>
                <dl>
                  <div><dt>Ngày sản xuất</dt><dd>{formatDate(record.production_date)}</dd></div>
                  <div><dt>Hạn sử dụng</dt><dd>{formatDate(record.expiry_date)}</dd></div>
                  <div><dt>Nghệ nhân phụ trách</dt><dd>{record.artisan_name}</dd></div>
                  <div><dt>Xưởng sản xuất</dt><dd>{record.workshop_location}</dd></div>
                </dl>
              </article>
              <article className="gh-frame-card gh-trace-card">
                <Motif name="07-hat-me" className="gh-trace-card__motif" />
                <span className="gh-kicker">Nguồn nguyên liệu</span>
                <h3>Từ thổ nhưỡng xứ Huế</h3>
                <p>{record.raw_materials_origin}</p>
              </article>
              <article className="gh-frame-card gh-trace-card">
                <Motif name="12-hoa-sen-cung-dinh" className="gh-trace-card__motif" />
                <span className="gh-kicker">An toàn thực phẩm</span>
                <h3>Chứng nhận kiểm định</h3>
                <p>{record.food_safety_cert}</p>
              </article>
            </div>

            <div className="gh-frame-functional gh-frame-functional--fold gh-trace-actions">
              <div>
                <span className="gh-kicker">Minh bạch từ xưởng</span>
                <h2>Xem quy trình tạo nên thức quà</h2>
                <p>Hồ sơ được liên kết với đúng mã lô để khách hàng có thể kiểm chứng trước khi sử dụng hoặc mua lại.</p>
              </div>
              <div className="gh-state__actions">
                {record.crafting_video_url ? <a className="gh-btn gh-btn--brown gh-btn--fold" href={record.crafting_video_url} target="_blank" rel="noreferrer">Xem video chế tác ↗</a> : null}
                <Link className="gh-btn gh-btn--paper" href={`/products?q=${encodeURIComponent(record.product_name)}`}>Tìm sản phẩm</Link>
              </div>
            </div>
          </> : null}
        </div>
      </section>
    </main>
  );
}
