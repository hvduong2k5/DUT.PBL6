import type { Metadata } from "next";
import { PaymentWorkspace } from "./payment-workspace";

export const metadata: Metadata = { title: "Vận hành thanh toán" };
export default function PaymentPage() { return <PaymentWorkspace />; }
