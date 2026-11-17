import Link from "next/link";
import { BrandMark } from "./brand-mark";
import { collections, serviceLinks } from "./storefront-links";

export function SiteFooter() {
  return <footer className="gh-footer"><div className="gh-wrap">
    <div><BrandMark /><p>Mè xửng, trà và quà biếu từ Huế.</p></div>
    <nav aria-label="Bộ sưu tập"><strong>Bộ sưu tập</strong>{collections.map(link => <Link key={link.href} href={link.href}>{link.name}</Link>)}</nav>
    <nav aria-label="Thông tin và hỗ trợ"><strong>Gói Huế</strong>{serviceLinks.map(link => <Link key={link.href} href={link.href}>{link.name}</Link>)}</nav>
  </div></footer>;
}