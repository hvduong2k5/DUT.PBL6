"use client";

import { useState, type FormEvent } from "react";

type Status = "READY" | "PARTIAL" | "UNAVAILABLE";
type Batch = { batchCode: string; expiresOn: string; sellableUnits: number; unitsAtRisk: number };
type Rank = {
  skuCode: string; status: Status; netUnits: number | null; unitsPerDay: string | null;
  salesCoveredDays: number; inventoryCoveredDays: number; stockoutDays: number | null;
  riskReason: string | null; snapshotAsOf: string | null;
};
type Workbench = {
  warehouseId: string; from: string; to: string; horizonDays: number; ranking: Rank[];
  skuLimitReached: boolean; batches: (Batch & { skuCode: string })[]; batchCount: number;
};
type Velocity = {
  soldUnits: number | null; returnedUnits: number | null;
  series: { date: string; soldUnits: number | null; returnedUnits: number | null; stockoutMinutes: number | null }[];
};
type Risk = { reasonCode: string | null; sourceAsOf: string | null; demandUnitsPerDay: string | null; batches: Batch[] };
type Detail = { sku: string; velocity: Velocity; risk: Risk };

const statusLabel: Record<Status, string> = { READY: "Đủ dữ liệu", PARTIAL: "Thiếu ngày", UNAVAILABLE: "Chưa có số bán" };
const riskLabel: Record<string, string> = {
  BATCH_SNAPSHOT_MISSING: "Chưa có snapshot lô",
  BATCH_SNAPSHOT_STALE: "Snapshot lô đã cũ",
  BASELINE_COVERAGE_GAP: "Thiếu dữ liệu trong 14 ngày cơ sở",
  INSUFFICIENT_IN_STOCK_DAYS: "Chưa đủ 7 ngày còn hàng",
};
const number = new Intl.NumberFormat("vi-VN", { maximumFractionDigits: 2 });
const timestamp = new Intl.DateTimeFormat("vi-VN", { dateStyle: "short", timeStyle: "short", timeZone: "Asia/Ho_Chi_Minh" });
const show = (value: number | string | null) => value === null ? "—" : number.format(Number(value));

function defaultDates() {
  const to = new Intl.DateTimeFormat("sv-SE", { timeZone: "Asia/Ho_Chi_Minh" }).format(new Date());
  const from = new Date(Date.parse(to) - 14 * 86_400_000).toISOString().slice(0, 10);
  return { from, to };
}

async function getReport<T>(name: string, query: Record<string, string>): Promise<T> {
  const response = await fetch(`/api/admin/analytics/${name}?${new URLSearchParams(query)}`, { cache: "no-store" });
  if (!response.ok) {
    const error = await response.json().catch(() => ({}));
    throw new Error(error.message ?? "Không thể tải báo cáo.");
  }
  return response.json();
}

export function SupplyWorkspace({ initialWarehouse }: { initialWarehouse: string }) {
  const [filters, setFilters] = useState(() => ({ ...defaultDates(), warehouseId: initialWarehouse, horizonDays: "14" }));
  const [data, setData] = useState<Workbench>();
  const [detail, setDetail] = useState<Detail>();
  const [busy, setBusy] = useState("");
  const [error, setError] = useState("");
  const [slowFirst, setSlowFirst] = useState(false);

  async function load(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy("workbench");
    setError("");
    setData(undefined);
    setDetail(undefined);
    try {
      setData(await getReport<Workbench>("workbench", { ...filters, warehouseId: filters.warehouseId.trim() }));
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Không thể tải báo cáo.");
    } finally {
      setBusy("");
    }
  }

  async function openSku(sku: string) {
    if (!data) return;
    setBusy(sku);
    setError("");
    setDetail(undefined);
    const scope = { skuCode: sku, warehouseId: data.warehouseId };
    try {
      const [velocity, risk] = await Promise.all([
        getReport<Velocity>("velocity", { ...scope, from: data.from, to: data.to }),
        getReport<Risk>("expiry-risk", { ...scope, horizonDays: String(data.horizonDays) }),
      ]);
      setDetail({ sku, velocity, risk });
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Không thể tải chi tiết SKU.");
    } finally {
      setBusy("");
    }
  }

  const ranking = slowFirst && data
    ? [...data.ranking.filter((row) => row.status === "READY").reverse(), ...data.ranking.filter((row) => row.status !== "READY")]
    : data?.ranking;

  return (
    <main className="admin-page supply-workspace" aria-busy={!!busy}>
      <header className="admin-page-header"><div>
        <nav>Nhân viên <span>›</span> Kho vận</nav>
        <h1>Phân tích cung ứng</h1>
        <p>Theo dõi tốc độ bán và ưu tiên kiểm tra các lô có nguy cơ cận hạn theo kho.</p>
      </div></header>
      <form className="supply-filters" onSubmit={load}>
        <label>Mã kho<input required value={filters.warehouseId} maxLength={64} onChange={(e) => setFilters({ ...filters, warehouseId: e.target.value })} /></label>
        <label>Từ ngày<input required type="date" value={filters.from} onChange={(e) => setFilters({ ...filters, from: e.target.value })} /></label>
        <label>Đến trước ngày<input required type="date" value={filters.to} onChange={(e) => setFilters({ ...filters, to: e.target.value })} /></label>
        <label>Số ngày cảnh báo<input required type="number" min={1} max={90} value={filters.horizonDays} onChange={(e) => setFilters({ ...filters, horizonDays: e.target.value })} /></label>
        <button className="admin-primary-button" disabled={!!busy}>Xem báo cáo</button>
      </form>
      {busy && <p role="status">Đang tải {busy === "workbench" ? "báo cáo" : busy}…</p>}
      {error && <p role="alert">{error}</p>}
      {data && <>
        <section className="supply-report">
          <header><div><h2>Xếp hạng tốc độ bán</h2><p>Kho {data.warehouseId} · {data.from} đến trước {data.to}</p></div>
          <button className="admin-primary-button" type="button" aria-pressed={slowFirst} onClick={() => setSlowFirst(!slowFirst)}>{slowFirst ? "Chậm nhất trước" : "Nhanh nhất trước"}</button></header>
          {data.skuLimitReached && <p>Hiển thị tối đa 100 mã SKU của pilot.</p>}
          {!ranking?.length ? <p>Chưa nhận được dữ liệu cho kho này.</p> : <div className="supply-scroll"><table className="admin-table">
            <thead><tr><th>SKU</th><th>Đơn vị ròng</th><th>Đơn vị/ngày</th><th>Ngày có số bán / tồn</th><th>Ngày hết hàng</th><th>Tình trạng dữ liệu</th></tr></thead>
            <tbody>{ranking.map((row) => <tr key={row.skuCode}>
              <td><button className="supply-sku" type="button" disabled={!!busy} onClick={() => void openSku(row.skuCode)}>{row.skuCode}</button></td>
              <td>{show(row.netUnits)}</td><td>{show(row.unitsPerDay)}</td>
              <td>{row.salesCoveredDays} / {row.inventoryCoveredDays}</td><td>{show(row.stockoutDays)}</td>
              <td>{statusLabel[row.status]}{row.riskReason && <small>{riskLabel[row.riskReason] ?? row.riskReason}</small>}</td>
            </tr>)}</tbody>
          </table></div>}
        </section>
        <section className="supply-report">
          <header><div><h2>Lô cần ưu tiên kiểm tra</h2><p>{data.batchCount} lô có lượng dự kiến còn lại khi hết hạn trong {data.horizonDays} ngày.</p></div></header>
          {data.batchCount > data.batches.length && <p>Hiển thị {data.batches.length} lô đầu tiên theo ngày hết hạn.</p>}
          {data.batches.length ? <div className="supply-scroll"><table className="admin-table">
            <thead><tr><th>SKU</th><th>Lô</th><th>Hết hạn</th><th>Tồn bán được</th><th>Đơn vị có nguy cơ</th></tr></thead>
            <tbody>{data.batches.map((batch) => <tr key={`${batch.skuCode}/${batch.batchCode}`}>
              <td><button className="supply-sku" type="button" disabled={!!busy} onClick={() => void openSku(batch.skuCode)}>{batch.skuCode}</button></td>
              <td>{batch.batchCode}</td><td>{batch.expiresOn}</td><td>{show(batch.sellableUnits)}</td><td>{show(batch.unitsAtRisk)}</td>
            </tr>)}</tbody>
          </table></div> : <p>Chưa ghi nhận lô có nguy cơ trong phần dữ liệu tính được.</p>}
        </section>
      </>}
      {detail && <section className="supply-report" aria-label={`Chi tiết ${detail.sku}`}>
        <header><div><h2>Chi tiết {detail.sku}</h2><p>Bán {show(detail.velocity.soldUnits)} · Trả {show(detail.velocity.returnedUnits)} · Nhu cầu FEFO {show(detail.risk.demandUnitsPerDay)} đơn vị/ngày</p></div></header>
        {detail.risk.sourceAsOf && <p>Snapshot tồn lô {timestamp.format(new Date(detail.risk.sourceAsOf))}</p>}
        {detail.risk.reasonCode && <p>{riskLabel[detail.risk.reasonCode] ?? detail.risk.reasonCode}</p>}
        {detail.risk.batches.length > 0 && <div className="supply-scroll"><table className="admin-table">
          <thead><tr><th>Lô</th><th>Hết hạn</th><th>Tồn bán được</th><th>Đơn vị có nguy cơ</th></tr></thead>
          <tbody>{detail.risk.batches.map((batch) => <tr key={batch.batchCode}><td>{batch.batchCode}</td><td>{batch.expiresOn}</td><td>{show(batch.sellableUnits)}</td><td>{show(batch.unitsAtRisk)}</td></tr>)}</tbody>
        </table></div>}
        <details><summary>Số bán, hàng trả và phút hết hàng theo ngày</summary><div className="supply-scroll"><table className="admin-table">
          <thead><tr><th>Ngày</th><th>Số bán</th><th>Hàng trả</th><th>Phút hết hàng</th></tr></thead>
          <tbody>{detail.velocity.series.map((day) => <tr key={day.date}><td>{day.date}</td><td>{show(day.soldUnits)}</td><td>{show(day.returnedUnits)}</td><td>{show(day.stockoutMinutes)}</td></tr>)}</tbody>
        </table></div></details>
      </section>}
    </main>
  );
}
