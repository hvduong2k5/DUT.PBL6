import type { Metadata } from "next";
import { EmployeePermissionGate } from "@/components/auth/employee-permission-gate";
import { ProductCatalogWorkspace } from "@/features/employee/catalog/product-catalog-workspace";

export const metadata: Metadata = { title: "Quản lý Sản phẩm & SKU" };

export default function ProductCatalogPage() { return <EmployeePermissionGate permission="PRODUCT_VIEW"><ProductCatalogWorkspace /></EmployeePermissionGate>; }
