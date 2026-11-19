// Decorative Hue heritage watermark. Always aria-hidden and non-interactive (see .gh-motif in heritage.css).
export type MotifName =
  | "01-hien-nha" | "01-hien-nha-phai" | "02-canh-hoa" | "02-canh-hoa-phai" | "03-canh-tra" | "04-song-nuoc" | "05-may"
  | "06-mai-ngoi" | "06-mai-ngoi-trai" | "07-hat-me" | "09-cong-thanh" | "09-cong-thanh-phai" | "10-tuong-thanh"
  | "10-tuong-thanh-phai" | "11-cau-truong-tien" | "11-cau-truong-tien-phai" | "12-hoa-sen-cung-dinh" | "12-hoa-sen-cung-dinh-phai";

export function Motif({ name, className = "" }: { name: MotifName; className?: string }) {
  // eslint-disable-next-line @next/next/no-img-element
  return <img className={`gh-motif ${className}`} src={`/images/heritage/motifs/${name}.webp`} alt="" aria-hidden="true" loading="lazy" decoding="async" />;
}
