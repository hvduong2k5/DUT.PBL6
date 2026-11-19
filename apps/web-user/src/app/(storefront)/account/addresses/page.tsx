import type { Metadata } from "next";
import { AddressBookFlow } from "./address-book-flow";

export const metadata: Metadata = { title: "Sổ địa chỉ" };

export default function AddressesPage() {
  return <AddressBookFlow />;
}
