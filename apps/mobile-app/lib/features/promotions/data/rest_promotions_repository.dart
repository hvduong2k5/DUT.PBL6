import 'package:mobile_app/core/models/money.dart';
import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/features/promotions/domain/promotions_repository.dart';

class RestPromotionsRepository implements PromotionsRepository {
  const RestPromotionsRepository(this._client);
  final ApiClient _client;
  @override
  Future<List<Voucher>> vouchers() async {
    final json = await _client.getJson('/api/v1/promotions/vouchers');
    return (json['vouchers'] as List)
        .map((item) => Voucher.fromJson(Map<String, dynamic>.from(item as Map)))
        .toList();
  }

  @override
  Future<VoucherValidation> validate(String code, int subtotal) async {
    final json = await _client.postJson(
      '/api/v1/promotions/validate',
      data: {'voucher_code': code, 'subtotal_amount': subtotal},
    );
    return VoucherValidation(
      valid: json['is_valid'] as bool,
      discount: Money.fromJson(
        Map<String, dynamic>.from(json['discount_amount'] as Map),
      ),
      message: json['message'] as String,
    );
  }
}
