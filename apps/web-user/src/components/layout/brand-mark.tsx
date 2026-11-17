import Image from "next/image";
import Link from "next/link";

export function BrandMark() {
  return <Link className="gh-brand" href="/" aria-label="Gói Huế · Trang chủ">
    <Image src="/images/goi-hue/logo-goi-hue.svg" width={36} height={36} alt="" />
    <span>Gói Huế</span>
  </Link>;
}