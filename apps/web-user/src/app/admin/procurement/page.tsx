import { notFound } from "next/navigation";
import { ProcurementWorkspace } from "@/features/employee/procurement/workspace";
import "@/features/employee/procurement/workspace.css";

export const metadata = { title: "Thu mua & nhận nguyên liệu" };
export default function ProcurementPage() {
  if (process.env.ADMIN_PROCUREMENT_ENABLED !== "true") notFound();
  return <ProcurementWorkspace />;
}
