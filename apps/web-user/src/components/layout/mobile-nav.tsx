"use client";

import Link from "next/link";
import { useEffect, useState, type ReactNode } from "react";
import { COLLECTIONS } from "./collections";

type Sheet = "categories" | "search" | null;

interface MobileNavProps {
  itemCount: number | null;
  accountHref: string;
  accountLabel: string;
  showCart: boolean;
}

function Icon({ children }: { children: ReactNode }) {
  return <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" aria-hidden="true">{children}</svg>;
}

/** Bottom navigation + bottom sheets for phones and tablets (< 1100px). Pure navigation: no API calls. */
export function MobileNav({ itemCount, accountHref, accountLabel, showCart }: MobileNavProps) {
  const [sheet, setSheet] = useState<Sheet>(null);

  useEffect(() => {
    if (!sheet) return;
    const onKey = (event: KeyboardEvent) => { if (event.key === "Escape") setSheet(null); };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [sheet]);

  const title = sheet === "search" ? "Tìm sản phẩm" : "Bộ sưu tập";
  return <>
    <nav className="gh-bottom-nav" aria-label="Điều hướng điện thoại">
      <Link href="/" onClick={() => setSheet(null)}><Icon><path d="M3 10.5 12 3l9 7.5V20a1 1 0 0 1-1 1h-5v-6h-6v6H4a1 1 0 0 1-1-1z" /></Icon><span>Trang chủ</span></Link>
      <button type="button" onClick={() => setSheet("categories")} aria-haspopup="dialog"><Icon><rect x="3" y="3" width="7" height="7" /><rect x="14" y="3" width="7" height="7" /><rect x="3" y="14" width="7" height="7" /><rect x="14" y="14" width="7" height="7" /></Icon><span>Danh mục</span></button>
      <button type="button" onClick={() => setSheet("search")} aria-haspopup="dialog"><Icon><circle cx="11" cy="11" r="7" /><path d="m16 16 4.5 4.5" strokeLinecap="round" /></Icon><span>Tìm kiếm</span></button>
      {showCart ? <Link href="/cart" aria-label={itemCount === null ? "Mở giỏ hàng" : `Mở giỏ hàng, ${itemCount} sản phẩm`}>
        <Icon><path d="M4 8h16l-1.2 12H5.2z" /><path d="M8 8V6a4 4 0 0 1 8 0v2" /></Icon><span>Giỏ hàng</span>
        {itemCount ? <b>{itemCount > 99 ? "99+" : itemCount}</b> : null}
      </Link> : null}
      <Link href={accountHref}><Icon><circle cx="12" cy="8" r="4" /><path d="M4 21a8 8 0 0 1 16 0" /></Icon><span>{accountLabel}</span></Link>
    </nav>

    {sheet ? <div className="gh-sheet-layer">
      <button type="button" className="gh-sheet-backdrop" aria-label="Đóng" onClick={() => setSheet(null)} />
      <div className="gh-sheet" role="dialog" aria-modal="true" aria-label={title}>
        <div className="gh-sheet__handle" aria-hidden="true" />
        <div className="gh-sheet__head">
          <h2>{title}</h2>
          <button type="button" className="gh-btn-icon" aria-label="Đóng cửa sổ" onClick={() => setSheet(null)}>×</button>
        </div>
        {sheet === "categories" ? <ul className="gh-sheet__list">
          {COLLECTIONS.map((item) => <li key={item.href}><Link href={item.href} onClick={() => setSheet(null)}><strong>{item.name}</strong><small>{item.note}</small></Link></li>)}
          <li><Link href="/products" onClick={() => setSheet(null)}><strong>Xem tất cả sản phẩm</strong><small>Toàn bộ bộ sưu tập Gói Huế</small></Link></li>
        </ul> : <form action="/products" method="get" role="search" className="gh-sheet__search">
          <label className="gh-field-label" htmlFor="mobile-search">Nhập tên sản phẩm</label>
          <input id="mobile-search" className="gh-field" type="search" name="q" minLength={2} placeholder="Tìm mè xửng, trà sen…" autoFocus />
          <button type="submit" className="gh-btn gh-btn--brown">Tìm kiếm</button>
        </form>}
      </div>
    </div> : null}
  </>;
}
