import type { ReactNode } from "react";
import { ProductImage } from "@/components/product/product-image";

interface HeritageBannerProps {
  kicker: string;
  title: ReactNode;
  titleId: string;
  /** h1 for page heroes, h2 for banners inside a page. */
  level?: 1 | 2;
  lead?: ReactNode;
  image: { src: string; alt: string };
  priority?: boolean;
  children?: ReactNode;
  className?: string;
}

/**
 * Dark lacquer stage with the kit's double stepped frame (khung-doi), copy on the left and a photo
 * fading in from the right. Shared by the home hero, traceability, promotions and other page heroes.
 */
export function HeritageBanner({ kicker, title, titleId, level = 2, lead, image, priority = false, children, className = "" }: HeritageBannerProps) {
  const Heading = level === 1 ? "h1" : "h2";
  return <div className={`gh-banner gh-surface-imperial ${className}`}>
    <div className="gh-banner__copy">
      <span className="gh-banner__kicker">{kicker}</span>
      <Heading id={titleId} className="gh-banner__title">{title}</Heading>
      <div className="gh-banner__rule" aria-hidden="true" />
      {lead ? <p className="gh-banner__lead">{lead}</p> : null}
      {children}
    </div>
    <div className="gh-banner__visual">
      <ProductImage fill priority={priority} sizes="(max-width: 960px) 100vw, 55vw" src={image.src} alt={image.alt} />
    </div>
  </div>;
}
