import type { Metadata } from "next";
import { AuthPage } from "@/components/layout/auth-page";
import { getSafeReturnUrl } from "@/lib/auth/return-url";
import { RecoveryFlow } from "./recovery-flow";

export const metadata: Metadata = { title: "Khôi phục quyền truy cập" };

export default async function RecoveryPage({ searchParams }: { searchParams: Promise<{ returnUrl?: string | string[]; recoveryToken?: string | string[] }> }) {
  const params = await searchParams;
  const token = Array.isArray(params.recoveryToken) ? params.recoveryToken[0] : params.recoveryToken;
  return (
    <AuthPage eyebrow="◈ Cổng bảo mật Khách hàng Tri Kỷ" title="Khôi Phục Quyền Truy Cập" description="Nhập email đã đăng ký. Nếu phù hợp, Ô Mạ sẽ gửi một liên kết đặt lại mật khẩu an toàn." asideTitle="Gìn giữ an tâm, trọn vẹn phong vị Cố Đô" asideBody="Mọi yêu cầu đều được phản hồi trung tính để bảo vệ quyền riêng tư và nhật ký thưởng thức của Quý khách." benefits={[{ icon: "♢", title: "Cam kết Bảo hộ Tài khoản", body: "Liên kết một lần, hết hạn sau 15 phút." }, { icon: "↻", title: "Đồng bộ Mọi Thiết bị", body: "Mật khẩu mới thu hồi toàn bộ phiên cũ." }, { icon: "◇", title: "Không tiết lộ tài khoản", body: "Phản hồi giống nhau cho mọi địa chỉ email." }]}>
      <RecoveryFlow recoveryToken={token} returnUrl={getSafeReturnUrl(params.returnUrl)} />
    </AuthPage>
  );
}
