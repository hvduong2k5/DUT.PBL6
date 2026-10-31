export interface ProductReview {
  review_id: string;
  customer_name: string;
  rating: number;
  comment: string;
  media_urls: string[];
  is_verified_purchase: boolean;
  created_at: string;
  seller_reply?: string | null;
}

export interface ProductReviews {
  average_rating: number;
  total_reviews: number;
  reviews: ProductReview[];
}

export interface CreateReviewInput {
  product_id: string;
  rating: number;
  comment: string;
}

export interface TraceabilityRecord {
  qr_code: string;
  product_name: string;
  batch_code: string;
  ocop_star: number;
  ocop_certificate_no: string;
  production_date: string;
  expiry_date: string;
  workshop_location: string;
  artisan_name: string;
  food_safety_cert: string;
  raw_materials_origin: string;
  crafting_video_url?: string | null;
}

export interface Money {
  currency_code: string;
  units: number;
  nanos: number;
}

export interface Voucher {
  voucher_code: string;
  title: string;
  discount_type: "PERCENTAGE" | "FIXED_AMOUNT" | "FREESHIP";
  discount_value: number;
  min_order_amount: number;
  max_discount_amount: number;
  expires_at: string;
}

export interface VoucherList {
  vouchers: Voucher[];
}

export interface VoucherValidation {
  is_valid: boolean;
  discount_amount: Money;
  message: string;
}

export interface LoyaltyPoints {
  customer_id: string;
  current_points: number;
  tier: "BRONZE" | "SILVER" | "GOLD" | "DIAMOND";
  points_to_vnd_rate: number;
}

interface CustomerCoreErrorBody {
  error_code?: string;
  user_message?: string;
  code?: string;
  message?: string;
}

export class CustomerCoreApiError extends Error {
  readonly status: number;
  readonly code: string;

  constructor(status: number, body: CustomerCoreErrorBody) {
    super(body.user_message || body.message || "Không thể xử lý yêu cầu lúc này.");
    this.name = "CustomerCoreApiError";
    this.status = status;
    this.code = body.error_code || body.code || "UNKNOWN_ERROR";
  }
}
