import type { Metadata } from "next";
import { RoleCatalogWorkspace } from "./role-catalog-workspace";

export const metadata: Metadata = { title: "Quản lý Role" };

export default function RoleCatalogPage() { return <RoleCatalogWorkspace />; }
