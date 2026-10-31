"use client";

import Image, { type ImageProps } from "next/image";
import { useMemo, useState } from "react";

export const PRODUCT_IMAGE_FALLBACK = "/images/product-placeholder.svg";

export function resolveProductImageSource(source?: string | null): string {
  const value = source?.trim();
  if (!value) return PRODUCT_IMAGE_FALLBACK;
  if (value.startsWith("/")) return value;

  try {
    const url = new URL(value);
    if (url.hostname.toLocaleLowerCase() === "cdn.omama.vn") return PRODUCT_IMAGE_FALLBACK;
    return value;
  } catch {
    return PRODUCT_IMAGE_FALLBACK;
  }
}

type ProductImageProps = Omit<ImageProps, "src"> & {
  src?: string | null;
};

export function ProductImage({ src, alt, onError, ...props }: ProductImageProps) {
  const resolvedSource = useMemo(() => resolveProductImageSource(src), [src]);
  const [failedSource, setFailedSource] = useState<string>();
  const displaySource = failedSource === resolvedSource ? PRODUCT_IMAGE_FALLBACK : resolvedSource;

  return (
    <Image
      {...props}
      src={displaySource}
      alt={alt}
      unoptimized={displaySource === PRODUCT_IMAGE_FALLBACK || props.unoptimized}
      onError={(event) => {
        onError?.(event);
        if (displaySource !== PRODUCT_IMAGE_FALLBACK) setFailedSource(resolvedSource);
      }}
    />
  );
}
