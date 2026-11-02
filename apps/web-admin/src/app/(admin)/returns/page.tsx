import type { Metadata } from "next";
import { ReturnWorkspace } from "./return-workspace";

export const metadata: Metadata = { title: "Quản lý đổi trả và hoàn tiền" };
export default function ReturnPage() { return <ReturnWorkspace />; }
