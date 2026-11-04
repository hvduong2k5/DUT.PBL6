import 'package:equatable/equatable.dart';
import 'package:mobile_app/core/models/money.dart';

class Voucher extends Equatable {
  const Voucher({
    required this.code,
    required this.title,
    required this.type,
    required this.value,
    required this.minimum,
    required this.maximum,
    required this.expiresAt,
  });
  factory Voucher.fromJson(Map<String, dynamic> json) => Voucher(
    code: json['voucher_code'] as String,
    title: json['title'] as String,
    type: json['discount_type'] as String,
    value: json['discount_value'] as int,
    minimum: json['min_order_amount'] as int,
    maximum: json['max_discount_amount'] as int,
    expiresAt: DateTime.parse(json['expires_at'] as String),
  );
  final String code;
  final String title;
  final String type;
  final int value;
  final int minimum;
  final int maximum;
  final DateTime expiresAt;
  String get label => switch (type) {
    'PERCENTAGE' => '$value%',
    'FREESHIP' => 'Miễn phí giao hàng',
    _ => Money.vnd(value).format(),
  };
  @override
  List<Object?> get props => [
    code,
    title,
    type,
    value,
    minimum,
    maximum,
    expiresAt,
  ];
}

class VoucherValidation extends Equatable {
  const VoucherValidation({
    required this.valid,
    required this.discount,
    required this.message,
  });
  final bool valid;
  final Money discount;
  final String message;
  @override
  List<Object?> get props => [valid, discount, message];
}

abstract interface class PromotionsRepository {
  Future<List<Voucher>> vouchers();
  Future<VoucherValidation> validate(String code, int subtotal);
}
