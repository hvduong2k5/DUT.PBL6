import { AuthApiError } from "./types";

const SAFE_ERROR_COPY: Record<string, string> = {
  AUTHENTICATION_FAILED: "Email hoặc mật khẩu chưa chính xác. Vui lòng thử lại.",
  ACCOUNT_UNAVAILABLE: "Tài khoản hiện không thể đăng nhập. Vui lòng liên hệ hỗ trợ.",
  REGISTRATION_NOT_AVAILABLE:
    "Không thể tạo tài khoản bằng thông tin này. Bạn có thể đăng nhập hoặc khôi phục quyền truy cập.",
  VERIFICATION_INVALID: "Liên kết xác minh không hợp lệ.",
  VERIFICATION_EXPIRED: "Liên kết xác minh đã hết hạn. Vui lòng yêu cầu gửi lại email.",
  VERIFICATION_ALREADY_USED: "Liên kết xác minh đã được sử dụng.",
  VERIFICATION_REQUEST_INVALID: "Yêu cầu xác minh không còn hiệu lực.",
  RECOVERY_REQUEST_INVALID: "Liên kết khôi phục không hợp lệ.",
  RECOVERY_REQUEST_EXPIRED: "Liên kết khôi phục đã hết hạn. Vui lòng bắt đầu lại.",
  RECOVERY_REQUEST_ALREADY_USED: "Liên kết khôi phục đã được sử dụng.",
  RESET_PROOF_INVALID: "Phiên đặt lại mật khẩu không hợp lệ.",
  RESET_PROOF_EXPIRED: "Phiên đặt lại mật khẩu đã hết hạn. Vui lòng bắt đầu lại.",
  RESET_PROOF_ALREADY_USED: "Phiên đặt lại mật khẩu đã được sử dụng.",
  RATE_LIMITED: "Bạn đã gửi quá nhiều yêu cầu. Vui lòng chờ rồi thử lại.",
  NOTIFICATION_DELIVERY_FAILED: "Chưa thể gửi email lúc này. Vui lòng thử lại.",
  AUTH_UPSTREAM_UNAVAILABLE: "Không thể kết nối dịch vụ xác thực. Vui lòng thử lại sau.",
  INTERNAL_SERVER_ERROR: "Hệ thống đang bận. Vui lòng thử lại sau."
};

export function getFieldErrors(error: unknown): Record<string, string> {
  if (!(error instanceof AuthApiError)) return {};
  return Object.fromEntries(error.errors.map((item) => [item.field, item.message]));
}

export function getSafeErrorMessage(error: unknown): string {
  if (error instanceof AuthApiError) {
    return SAFE_ERROR_COPY[error.code] ?? "Không thể xử lý yêu cầu lúc này. Vui lòng thử lại.";
  }
  return "Không thể kết nối hệ thống. Vui lòng kiểm tra mạng và thử lại.";
}
