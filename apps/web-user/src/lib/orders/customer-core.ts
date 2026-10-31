interface Money {
  currency_code: string;
  units: number;
  nanos: number;
}

// Legacy provider DTO retained only for Returns eligibility until that feature
// is migrated. Order list/detail runtime no longer consumes this shape.
export interface CustomerCoreOrderDetail {
  order_id: string;
  status: string;
  shipping_address: {
    recipient_name: string;
    phone_number: string;
    street_address: string;
    ward: string;
    district: string;
    province: string;
  };
  items: Array<{ sku_code: string; quantity: number; price: Money }>;
  subtotal_amount: Money;
  discount_amount: Money;
  shipping_fee: Money;
  final_amount: Money;
  tracking_code: string | null;
  seal_code?: string | null;
  packing_video_url?: string | null;
  created_at: string;
}
