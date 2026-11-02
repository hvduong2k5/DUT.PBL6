"use client";

import Link from "next/link";
import { useAuth } from "@/context/AuthContext";

export function LoginForm({ returnUrl }: { returnUrl: string }) {
  const { login } = useAuth();
  const query = returnUrl === "/" ? "" : `?returnUrl=${encodeURIComponent(returnUrl)}`;

  return (
    <div className="auth-form oidc-login-panel">
      <div className="oidc-login-mark" aria-hidden="true">Ô</div>
      <div>
        <h2>Đăng nhập an toàn</h2>
        <p>Bạn sẽ được chuyển đến cổng định danh O Mạ để đăng nhập bằng Email/Mật khẩu hoặc Google.</p>
      </div>
      <button className="primary-button full-width" type="button" onClick={() => login(returnUrl)}>
        Tiếp tục đăng nhập <span aria-hidden="true">→</span>
      </button>
      <p className="switch-flow">Chưa có tài khoản Tri Kỷ? <Link href={`/register${query}`}>Đăng ký ngay</Link></p>
      {returnUrl.startsWith("/thanh-toan") ? <Link className="guest-link" href={returnUrl}>Tiếp tục thanh toán với tư cách khách</Link> : null}
      <div className="security-note">Phiên đăng nhập được bảo vệ bằng PKCE và cookie HttpOnly. Mật khẩu không đi qua ứng dụng Web O Mạ.</div>
    </div>
  );
}
