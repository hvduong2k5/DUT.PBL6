import type { Metadata } from "next";
import { SecurityFlow } from "./security-flow";

export const metadata: Metadata = { title: "Đổi mật khẩu" };

export default function SecurityPage() {
  return <SecurityFlow />;
}
