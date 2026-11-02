import type { Metadata } from "next";
import { PermissionCatalogWorkspace } from "./permission-catalog-workspace";

export const metadata: Metadata = { title: "Danh mục Permission" };

export default function PermissionCatalogPage() { return <PermissionCatalogWorkspace />; }
