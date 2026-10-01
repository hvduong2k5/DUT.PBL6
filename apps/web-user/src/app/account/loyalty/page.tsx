import type { Metadata } from "next";
import { Suspense } from "react";
import { LoyaltyFlow } from "./loyalty-flow";

export const metadata: Metadata = { title: "Ví điểm Tri Kỷ" };

export default function LoyaltyPage() {
  return <Suspense fallback={null}><LoyaltyFlow /></Suspense>;
}
