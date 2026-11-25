import { PublicApiError } from "@/lib/api-contract";

/**
 * User-facing Vietnamese messages for storefront API error codes.
 *
 * The API deliberately returns generic English messages for guest traffic
 * (for example "The guest request could not be completed."), so the UI must
 * translate the machine-readable `code` into something a shopper can act on.
 * Never render the raw server message for a coded error.
 */
const CODE_MESSAGES: Record<string, string> = {
  // Address and form-format problems: the shopper can fix these.
  INVALID_ADDRESS: "Thông tin giao hàng chưa hợp lệ. Kiểm tra họ tên, số điện thoại (10 số, đầu số 03/05/07/08/09) và địa chỉ cụ thể.",
  INVALID_ADMINISTRATIVE_UNIT: "Phường/Xã chưa khớp với Tỉnh/Thành phố đã chọn. Vui lòng chọn lại Tỉnh/Thành phố và Phường/Xã.",
  ADDRESS_REQUIRED: "Vui lòng nhập đầy đủ địa chỉ giao hàng.",
  ADDRESS_NOT_FOUND: "Địa chỉ đã lưu không còn tồn tại. Hãy nhập địa chỉ mới hoặc chọn địa chỉ khác.",
  INVALID_ADDRESS_ID: "Địa chỉ đã lưu không còn hợp lệ. Hãy nhập địa chỉ mới hoặc chọn địa chỉ khác.",
  ADDRESS_CHANGED: "Địa chỉ giao hàng vừa thay đổi. Kiểm tra lại thông tin trước khi đặt hàng.",
  AMBIGUOUS_ADDRESS: "Địa chỉ chưa rõ ràng. Kiểm tra lại Tỉnh/Thành phố, Phường/Xã và địa chỉ cụ thể.",
  INVALID_QUOTE_INPUT: "Thông tin báo giá chưa hợp lệ. Kiểm tra lại giỏ hàng và địa chỉ giao hàng.",
  INVALID_QUOTE: "Quote không hợp lệ. Hệ thống sẽ tạo quote mới.",
  INVALID_PAYMENT_METHOD: "Phương thức thanh toán không hợp lệ. Vui lòng chọn lại phương thức thanh toán.",
  INVALID_QUANTITY: "Số lượng sản phẩm không hợp lệ.",
  INVALID_ITEMS: "Danh sách sản phẩm trong giỏ không hợp lệ.",
  INVALID_SELECTION: "Lựa chọn sản phẩm trong giỏ không hợp lệ.",
  INVALID_CART_INPUT: "Thông tin giỏ hàng không hợp lệ.",
  TOO_MANY_ITEMS: "Giỏ hàng vượt quá số lượng sản phẩm tối đa cho phép.",

  // Quote lifecycle: usually resolved by letting the page re-quote.
  QUOTE_EXPIRED: "Quote đã hết hạn. Hệ thống đang tạo quote mới, đợi một chút rồi thử lại.",
  QUOTE_NOT_FOUND: "Quote không còn hiệu lực. Hệ thống sẽ tạo quote mới.",
  QUOTE_ALREADY_USED: "Quote này đã được dùng. Hệ thống sẽ tạo quote mới.",
  QUOTE_UNAVAILABLE: "Chưa thể báo giá cho giỏ hàng hiện tại. Thử lại sau ít phút.",
  QUOTE_INPUT_MISMATCH: "Giỏ hàng hoặc địa chỉ đã thay đổi so với lúc báo giá. Hệ thống sẽ tạo quote mới.",

  // Cart and catalogue drift.
  CART_CHANGED: "Giỏ hàng vừa thay đổi. Trang đang tải lại thông tin mới nhất.",
  CART_REPLACED: "Giỏ hàng vừa được thay thế bởi một phiên khác. Trang đang tải lại thông tin mới nhất.",
  VERSION_CONFLICT: "Dữ liệu vừa thay đổi ở nơi khác. Tải lại trang rồi thử lại.",
  PRICE_CHANGED: "Giá sản phẩm vừa thay đổi. Kiểm tra lại tổng tiền trước khi đặt hàng.",
  INSUFFICIENT_STOCK: "Một sản phẩm trong giỏ không đủ tồn kho. Kiểm tra lại giỏ hàng.",
  SKU_NOT_AVAILABLE: "Một sản phẩm trong giỏ hiện không còn bán. Kiểm tra lại giỏ hàng.",
  CATALOG_SNAPSHOT_INVALID: "Dữ liệu sản phẩm đang được cập nhật. Thử lại sau ít phút.",
  CATALOG_SNAPSHOT_UNKNOWN: "Dữ liệu sản phẩm đang được cập nhật. Thử lại sau ít phút.",
  CATALOG_INCONSISTENT: "Dữ liệu sản phẩm đang được cập nhật. Thử lại sau ít phút.",
  INVALID_PRICE: "Giá sản phẩm đang được cập nhật. Thử lại sau ít phút.",

  // Payment-method availability.
  COD_NOT_AVAILABLE: "Thanh toán khi nhận hàng (COD) hiện không khả dụng cho đơn này. Chọn VietQR hoặc thử lại sau.",

  // Guest session and feature gates.
  GUEST_CHECKOUT_DISABLED: "Guest Checkout hiện đang tắt. Đăng nhập để tiếp tục đặt hàng.",
  GUEST_SESSION_UNAVAILABLE: "Phiên khách đã hết hạn. Tải lại trang để bắt đầu phiên mới.",
  GUEST_CART_MERGE_DISABLED: "Tính năng gộp giỏ hàng hiện đang tắt.",
  GUEST_PENDING_ORDER_LIMIT: "Bạn đang có quá nhiều đơn khách đang xử lý. Đợi các đơn hoàn tất rồi thử lại.",
  GUEST_PAYMENT_ACCESS_UNAVAILABLE: "Chưa thể mở quyền thanh toán cho đơn khách. Thử lại sau ít phút.",

  // Submission safety.
  CHECKOUT_ALREADY_SUBMITTED: "Đơn hàng này đã được gửi. Kiểm tra trạng thái đơn thay vì gửi lại.",
  IDEMPOTENCY_CONFLICT: "Yêu cầu bị trùng với một yêu cầu khác. Tải lại trang rồi thử lại.",

  // Identity and account state.
  UNAUTHENTICATED: "Phiên đăng nhập đã hết hạn. Đăng nhập lại để tiếp tục.",
  AUTH_REQUIRED: "Phiên đăng nhập đã hết hạn. Đăng nhập lại để tiếp tục.",
  CUSTOMER_REQUIRED: "Tính năng này cần tài khoản khách hàng. Đăng nhập để tiếp tục.",
  CUSTOMER_VERIFICATION_REQUIRED: "Tài khoản cần hoàn thiện hồ sơ (lưu số điện thoại) trước khi đặt hàng.",
  ACCOUNT_ACTIVATION_REQUIRED: "Tài khoản cần hoàn thiện hồ sơ (lưu số điện thoại) trước khi mua hàng.",
  CUSTOMER_FORBIDDEN: "Tài khoản không có quyền thực hiện thao tác này.",
  PERMISSION_DENIED: "Tài khoản không có quyền thực hiện thao tác này.",
  PRINCIPAL_FORBIDDEN: "Tài khoản không có quyền thực hiện thao tác này.",

  // Dependency and system problems: retry later.
  DEPENDENCY_UNKNOWN: "Hệ thống đang bận hoặc một dịch vụ tạm thời không phản hồi. Thử lại sau ít phút.",
  DEPENDENCY_INVALID_RESPONSE: "Hệ thống đang bận hoặc một dịch vụ tạm thời không phản hồi. Thử lại sau ít phút.",
  PROFILE_UNAVAILABLE: "Hệ thống đang bận hoặc một dịch vụ tạm thời không phản hồi. Thử lại sau ít phút.",
  IDENTITY_UNAVAILABLE: "Hệ thống đang bận hoặc một dịch vụ tạm thời không phản hồi. Thử lại sau ít phút.",
  SHIPPING_PROVIDER_UNAVAILABLE: "Chưa thể tính phí vận chuyển lúc này. Thử lại sau ít phút.",
  FULFILLMENT_PROVIDER_UNAVAILABLE: "Hệ thống đang bận hoặc một dịch vụ tạm thời không phản hồi. Thử lại sau ít phút.",
  CART_UNAVAILABLE: "Giỏ hàng tạm thời không truy cập được. Thử lại sau ít phút.",
  TARGET_CHECKOUT_NOT_READY: "Hệ thống đang bận hoặc một dịch vụ tạm thời không phản hồi. Thử lại sau ít phút."
};

function statusMessage(status: number, fallback: string): string {
  if (status === 401) return CODE_MESSAGES.UNAUTHENTICATED;
  if (status === 403) return "Tài khoản không có quyền thực hiện thao tác này.";
  if (status === 404) return "Không tìm thấy dữ liệu yêu cầu. Tải lại trang rồi thử lại.";
  if (status === 409) return "Dữ liệu vừa thay đổi ở nơi khác. Tải lại trang rồi thử lại.";
  if (status === 410) return "Yêu cầu đã hết hạn. Tạo lại yêu cầu mới.";
  if (status === 422) return "Thông tin chưa hợp lệ. Kiểm tra lại các trường đã nhập.";
  if (status === 429) return "Quá nhiều yêu cầu trong thời gian ngắn. Đợi một chút rồi thử lại.";
  if (status >= 500) return "Hệ thống đang bận hoặc mất kết nối. Thử lại sau ít phút.";
  return fallback;
}

/** Message for a machine-readable error code, falling back to the HTTP status. */
export function userErrorCodeMessage(code: string | undefined, status = 0, fallback = "Chưa thể xử lý yêu cầu lúc này."): string {
  if (code && CODE_MESSAGES[code]) return CODE_MESSAGES[code];
  return statusMessage(status, fallback);
}

/** Message for a caught error: coded API errors are translated, anything else uses the caller's fallback. */
export function userErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof PublicApiError) return userErrorCodeMessage(error.code, error.status, fallback);
  return fallback;
}
