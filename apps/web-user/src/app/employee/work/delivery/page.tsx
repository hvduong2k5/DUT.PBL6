import type { Metadata } from "next";
import { DeliveryWorkbench } from "./delivery-workbench";

export const metadata: Metadata = { title: "Công việc giao hàng" };

export default function DeliveryWorkbenchPage() { return <DeliveryWorkbench />; }
