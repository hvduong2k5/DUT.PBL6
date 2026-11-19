import Link from "next/link";

export function BrandMark() {
  return (
    <Link className="gh-brand" href="/" aria-label="Gói Huế - Trang chủ">
      {/* Static SVG emblem; next/image adds nothing for a tiny vector. */}
      {/* eslint-disable-next-line @next/next/no-img-element */}
      <img className="gh-brand__mark" src="/images/heritage/logo-goi-hue.svg" alt="" width={38} height={38} />
      <span className="gh-brand__text">Gói Huế</span>
    </Link>
  );
}
