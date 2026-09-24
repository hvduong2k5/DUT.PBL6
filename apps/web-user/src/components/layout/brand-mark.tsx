import Link from "next/link";

export function BrandMark() {
  return (
    <Link className="brand" href="/" aria-label="Ô Mạ - Trang chủ">
      <span className="brand-seal" aria-hidden="true">Ô</span>
      <span>
        <strong>Ô MẠ</strong>
        <small>Tinh hoa bánh mứt Cố Đô</small>
      </span>
    </Link>
  );
}
