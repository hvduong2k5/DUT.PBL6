import Link from "next/link";
import Image from "next/image";

export function BrandMark() {
  return (
    <Link href="/" className="brand-logo" aria-label="Gói Huế - Trang chủ">
      <Image
        className="brand-logo__mark"
        width={44}
        height={44}
        src="/images/goi-hue/logo-goi-hue.svg"
        alt=""
      />
      <span className="brand-logo__text">Gói Huế</span>
    </Link>
  );
}
