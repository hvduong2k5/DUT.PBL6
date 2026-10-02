import type { Metadata } from "next";
import { PackingWorkspace } from "./packing-workspace";

export const metadata: Metadata = { title: "Quản lý đóng gói" };
export default function PackingPage() { return <PackingWorkspace />; }
