import type { Metadata } from "next";
import { Suspense } from "react";
import { B2BQuoteWorkspace } from "./quote-workspace";

export const metadata: Metadata = { title: "Quản lý báo giá B2B" };

export default function B2BQuoteWorkspacePage() {
  return <Suspense fallback={null}><B2BQuoteWorkspace /></Suspense>;
}
