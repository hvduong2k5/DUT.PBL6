import type { Metadata } from "next";
import { WarehouseWorkbench } from "./warehouse-workbench";

export const metadata: Metadata = { title: "Warehouse Operations Workbench" };

export default function WarehouseWorkbenchPage() { return <WarehouseWorkbench />; }
