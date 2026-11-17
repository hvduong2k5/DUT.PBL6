"use client";

import { useEffect, useState } from "react";

export function StorefrontDataNotice() {
  const [sample, setSample] = useState(false);
  useEffect(() => {
    const update = () => setSample(true);
    window.addEventListener("storefront-sample-data", update);
    if (document.documentElement.dataset.storefrontSample === "true")
      queueMicrotask(update);
    return () => window.removeEventListener("storefront-sample-data", update);
  }, []);
  if (!sample) return null;
  return (
    <p role="status" className="storefront-data-notice">
      Đang xem dữ liệu mẫu từ bản sao Gói Huế · phục vụ duyệt giao diện.
    </p>
  );
}
