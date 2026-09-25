import type { Metadata } from "next";
import { Suspense } from "react";
import { SiteFooter } from "@/components/layout/site-footer";
import { SiteHeader } from "@/components/layout/site-header";
import { GuestOrderLookup } from "./guest-order-lookup";

export const metadata: Metadata = { title: "Tra cứu Order", description: "Xác minh quyền sở hữu để tra cứu Guest Order Ô Mạ Huế." };

export default function TrackOrderPage() {
  return <><SiteHeader /><Suspense fallback={null}><GuestOrderLookup /></Suspense><SiteFooter /></>;
}
