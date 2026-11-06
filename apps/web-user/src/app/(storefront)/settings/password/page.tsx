import type { Metadata } from "next";
import { SiteFooter } from "@/components/layout/site-footer";
import { SiteHeader } from "@/components/layout/site-header";
import { ChangePasswordFlow } from "./change-password-flow";

export const metadata: Metadata = { title: "Đổi mật khẩu" };

export default function ChangePasswordPage() {
  return <><SiteHeader /><ChangePasswordFlow /><SiteFooter /></>;
}
