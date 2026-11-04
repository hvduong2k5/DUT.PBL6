import type { Metadata } from "next";
import { AuthPage } from "@/components/layout/auth-page";
import { getSafeReturnUrl } from "@/lib/auth/return-url";
import { RegistrationFlow } from "./registration-flow";
import { OidcRegistrationFlow } from "./oidc-registration-flow";
import { usesOidc, usesIdentityBff } from "@/lib/auth/oidc-config";

export const metadata: Metadata = { title: "Đăng ký tài khoản Tri Kỷ" };

export default async function RegistrationPage({ searchParams }: { searchParams: Promise<{ returnUrl?: string | string[]; verificationToken?: string | string[] }> }) {
  const params = await searchParams;
  const token = Array.isArray(params.verificationToken) ? params.verificationToken[0] : params.verificationToken;
  if (usesOidc() || usesIdentityBff()) return <AuthPage eyebrow="Khách hàng mới" title="Đăng ký tài khoản" description="Dùng email để tạo tài khoản đăng nhập. Hồ sơ khách hàng được tạo sau đó và cần hoàn tất xác minh trước khi mua hàng." asideTitle="Tài khoản Ô Mạ" asideBody="Quản lý hồ sơ và địa chỉ giao hàng của bạn." benefits={[]}><OidcRegistrationFlow returnUrl={getSafeReturnUrl(params.returnUrl)} /></AuthPage>;
  return (
    <AuthPage eyebrow="Ghi danh hội viên mới" title="Khởi Tạo Thẻ Tri Kỷ" description="Đăng ký bằng số điện thoại để tiếp tục mua sắm và quản lý đơn hàng." asideTitle="Trở thành Khách Tri Kỷ Ô Mạ" asideBody="Kết duyên cùng phong vị Cố Đô. Thưởng ngoạn trọn vẹn thức quà ngự thiện với các đặc quyền được chọn lọc." benefits={[{ icon: "◉", title: "Kích hoạt tức thì", body: "Tài khoản sẵn sàng ngay sau khi đăng ký thành công." }, { icon: "%", title: "Đặc quyền mùa lễ", body: "Nhận tin về bộ sưu tập và dịp lễ truyền thống." }, { icon: "✎", title: "Thư pháp Cố Đô", body: "Cá nhân hóa thiệp quà trên hành trình sau." }]}>
      <RegistrationFlow verificationToken={token} returnUrl={getSafeReturnUrl(params.returnUrl)} />
    </AuthPage>
  );
}
