export const MVP_FEATURE_GATES = Object.freeze({
  cod: process.env.NEXT_PUBLIC_MVP_COD_ENABLED === "true",
  voucher: process.env.NEXT_PUBLIC_MVP_VOUCHER_ENABLED === "true",
  guestOrder: process.env.NEXT_PUBLIC_MVP_GUEST_ORDER_ENABLED === "true"
});
