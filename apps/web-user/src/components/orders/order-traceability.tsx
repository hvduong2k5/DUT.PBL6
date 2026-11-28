"use client";

import { useEffect, useState } from "react";
import type { OrderLine } from "@/lib/orders/types";
import { matchesFrozenOrder, type OrderTraceability } from "@/lib/orders/traceability";
import { orderService } from "@/services/order-service";

const MESSAGE = { PACKING_PENDING: "Thông tin lô sẽ có sau khi đơn được đóng gói và niêm phong.", NOT_APPLICABLE: "Đơn đã hủy trước khi có bản đóng gói niêm phong.", UNAVAILABLE: "Chưa thể đọc thông tin nguồn gốc. Chi tiết đơn hàng vẫn được giữ nguyên." };
export function OrderTraceabilityPanel({ orderId, lines }: { orderId: string; lines: OrderLine[] }) {
  const [state, setState] = useState<{ result?: OrderTraceability; error?: string }>({});
  const [attempt, setAttempt] = useState(0);
  useEffect(() => {
    let active = true;
    void orderService.traceability(orderId).then((result) => {
      if (!matchesFrozenOrder(result, lines)) throw new Error("Thông tin lô chưa khớp các dòng hàng đã chốt.");
      if (active) setState({ result });
    }).catch((cause) => { if (active) setState({ error: cause instanceof Error ? cause.message : MESSAGE.UNAVAILABLE }); });
    return () => { active = false; };
  }, [orderId, lines, attempt]);
  const result = state.result;
  return <section className="order-detail-card" id="nguon-goc"><header><span className="gh-emblem" aria-hidden="true" /><div><h2>Nguồn gốc các lô đã đóng gói</h2><p>Đối chiếu theo lô trong kiện hàng đã niêm phong.</p></div></header>
    {!result && !state.error ? <p role="status" aria-busy="true">Đang tải nguồn gốc…</p> : null}
    {state.error ? <p role="alert">{state.error}</p> : null}
    {result && result.state !== "READY" ? <p role="status">{MESSAGE[result.state]}</p> : null}
    {result?.state === "READY" ? <div className="order-lines">{lines.map((line) => <article key={line.lineId}><div><strong>{line.productName}</strong>{result.items.filter((row) => row.sku_code === line.skuCode).map((row) => <div key={`${row.order_line_id}:${row.batch_id}`}><p>Lô {row.batch_code} · {row.packed_quantity} sản phẩm</p>
      {row.qr_url ? <><a className="primary-link" href={row.qr_url}>Xem nguồn gốc lô</a>{row.status === "NO_LONGER_PUBLIC" ? <p>Hồ sơ hiện đã dừng công khai.</p> : null}</> : <p>{row.status === "NOT_AVAILABLE" ? "Lô này chưa có hồ sơ được công bố." : "Chưa thể tải hồ sơ cho lô này."}</p>}
    </div>)}</div></article>)}</div> : null}
    {state.error || result?.state === "UNAVAILABLE" || result?.items.some((row) => row.status === "UNAVAILABLE") ? <button type="button" className="secondary-button" onClick={() => { setState({}); setAttempt((value) => value + 1); }}>Tải lại nguồn gốc</button> : null}
  </section>;
}
