class CheckoutProvince {
  const CheckoutProvince({
    required this.code,
    required this.name,
    required this.wards,
  });
  factory CheckoutProvince.fromJson(Map<String, dynamic> json) =>
      CheckoutProvince(
        code: json['provinceCode'] as String,
        name: json['provinceName'] as String,
        wards: (json['wards'] as List).map((raw) {
          final ward = Map<String, dynamic>.from(raw as Map);
          return CheckoutWard(
            code: ward['wardCode'] as String,
            name: ward['wardName'] as String,
          );
        }).toList(),
      );
  final String code;
  final String name;
  final List<CheckoutWard> wards;
}

class CheckoutWard {
  const CheckoutWard({required this.code, required this.name});
  final String code;
  final String name;
}
