export const items = [
  {
    skuId: "SKU-002-200",
    slug: "me-xung-deo-ngu-thien",
    meta: "Hộp cao cấp: 200g",
    name: "Mè Xửng Dẻo Ngự Trà Thượng Hạng",
    image: "/images/goi-hue/photo-sp-me-xung-dia.jpg",
    group: 0,
    amount: 85000,
  },
  {
    skuId: "SKU-011-150",
    slug: "tra-sen-tinh-tam",
    meta: "Hộp thiếc cao cấp: 150g",
    name: "Trà Sen Bách Diệp Hồ Tịnh Tâm",
    image: "/images/goi-hue/photo-sp-tra-sen-lon.jpg",
    group: 1,
    amount: 380000,
  },
  {
    skuId: "SKU-009-500",
    slug: "bo-qua-hoang-trieu",
    meta: "Trọng lượng: 500g",
    name: "Hộp Quà Cố Đô Ngự Thiện",
    image: "/images/goi-hue/photo-sp-hop-qua-mo.jpg",
    group: 2,
    amount: 450000,
  },
  {
    skuId: "SKU-006-300",
    slug: "banh-dau-xanh-trai-cay",
    meta: "Hộp sơn mài: 12 cái",
    name: "Bánh Đậu Xanh Trái Cây Nghệ Nhân",
    image:
      "https://lh3.googleusercontent.com/aida-public/AB6AXuAGdlKPl8besUprgcIcaSNI-VXXEzt-aIkDTw6zE0If6NpcKmXbUXA58HO_oZbPDdSW8L_PwvVi7s3-n4fiI7yFLqHyWKiYM8zP30qlRMzUz2rDLvUDKnxRxHBMBN_TpgSNM9ciBOUI-kxDrnOfioqEgb9W16eSmlaDDnvGH_p6MK5CYsqFmT0ggFzCTnCpTmUHqbI7VNUkbm47hgj65LWqY9tqc9nNXP4jGIkreI45AUAse_CgYT8E",
    group: 3,
    amount: 120000,
  },
];

export type HomeItem = (typeof items)[number];
export const money = (value: number) => value.toLocaleString("vi-VN") + " ₫";
