import type { Metadata } from "next";
import { PackingWorkbench } from "./packing-workbench";

export const metadata: Metadata = { title: "Workbench đóng gói" };

export default function PackingWorkbenchPage() { return <PackingWorkbench />; }
