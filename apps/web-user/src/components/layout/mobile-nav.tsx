"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useState } from "react";
import { COLLECTIONS } from "./collections";
import { CartIcon, GiftIcon, GridIcon, HomeIcon, SearchIcon } from "./icons";

type Sheet = "categories" | "search" | null;

interface MobileNavProps {
  itemCount: number | null;
  showCart: boolean;
}

/** Bottom navigation + bottom sheets for phones and tablets (< 1100px). Pure navigation: no API calls. */
export function MobileNav({ itemCount, showCart }: MobileNavProps) {
  const pathname = usePathname();
  const [sheet, setSheet] = useState<Sheet>(null);

  useEffect(() => {
    if (!sheet) return;
    const onKey = (event: KeyboardEvent) => { if (event.key === "Escape") setSheet(null); };
    document.addEventListener("keydown", onKey);
    document.body.style.overflow = "hidden";
    return () => { document.removeEventListener("keydown", onKey); document.body.style.overflow = ""; };
  }, [sheet]);

  const title = sheet === "search" ? "Tìm sản phẩm" : "Bộ sưu tập";
  return <>
    <nav className="gh-bottom-nav gh-surface-imperial" aria-label="Điều hướng điện thoại">
      <Link href="/" aria-current={pathname === "/" ? "page" : undefined}><HomeIcon /><span>Trang chủ</span></Link>
      <button type="button" onClick={() => setSheet("categories")} aria-haspopup="dialog" aria-expanded={sheet === "categories"}><GridIcon /><span>Danh mục</span></button>
      <Link href="/promotions" aria-current={pathname?.startsWith("/promotions") ? "page" : undefined}><GiftIcon /><span>Ưu đãi</span></Link>
      <button type="button" onClick={() => setSheet("search")} aria-haspopup="dialog" aria-expanded={sheet === "search"}><SearchIcon /><span>Tìm kiếm</span></button>
      {showCart ? <Link href="/cart" aria-current={pathname?.startsWith("/cart") ? "page" : undefined}
        aria-label={itemCount === null ? "Mở giỏ hàng" : `Mở giỏ hàng, ${itemCount} sản phẩm`}>
        <CartIcon /><span>Giỏ hàng</span>
        {itemCount ? <b>{itemCount > 99 ? "99+" : itemCount}</b> : null}
      </Link> : null}
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
