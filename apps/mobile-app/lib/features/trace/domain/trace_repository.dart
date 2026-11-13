class TraceRecord {
  const TraceRecord({
    required this.code,
    required this.product,
    required this.batch,
    required this.stars,
    required this.certificate,
    required this.productionDate,
    required this.expiryDate,
    required this.workshop,
    required this.artisan,
    required this.safety,
    required this.origin,
    required this.video,
  });
  final String code,
      product,
      batch,
      certificate,
      productionDate,
      expiryDate,
      workshop,
      artisan,
      safety,
      origin,
      video;
  final int stars;
  factory TraceRecord.fromJson(Map<String, dynamic> json) => TraceRecord(
    code: json['qr_code'] as String,
    product: json['product_name'] as String,
    batch: json['batch_code'] as String,
    stars: json['ocop_star'] as int,
    certificate: json['ocop_certificate_no'] as String,
    productionDate: json['production_date'] as String,
    expiryDate: json['expiry_date'] as String,
    workshop: json['workshop_location'] as String,
    artisan: json['artisan_name'] as String,
    safety: json['food_safety_cert'] as String,
    origin: json['raw_materials_origin'] as String,
    video: json['crafting_video_url'] as String,
  );
}

abstract interface class TraceRepository {
  Future<TraceRecord> lookup(String code);
}
