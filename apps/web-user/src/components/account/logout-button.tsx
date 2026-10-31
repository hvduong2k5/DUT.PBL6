"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useCustomerSession } from "@/components/auth/customer-session-provider";

export function LogoutButton() {
  const router = useRouter();
  const { logout: endSession } = useCustomerSession();
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");

  async function logout() {
    setPending(true); setError("");
    try {
      await endSession();
      router.push("/login");
      router.refresh();
    } catch {
      setError("Chưa thể đăng xuất. Vui lòng thử lại.");
      setPending(false);
    }
  }

  return <div className="account-logout"><button type="button" className="secondary-button" onClick={logout} disabled={pending}>{pending ? "Đang đăng xuất…" : "Đăng xuất"}</button>{error ? <small role="alert">{error}</small> : null}</div>;
}
