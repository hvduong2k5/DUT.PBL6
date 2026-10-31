import type { Metadata } from "next";
import { ProductCatalogWorkspace } from "./product-catalog-workspace";

export const metadata: Metadata = { title: "Quản lý Sản phẩm & SKU" };

export default function ProductCatalogPage() { return <ProductCatalogWorkspace />; }
