import type { Metadata } from "next";
import { InventoryWorkspace } from "./inventory-workspace";

export const metadata: Metadata = { title: "Quản lý Tồn kho & Batch" };

export default function InventoryPage() { return <InventoryWorkspace />; }
