import 'package:mobile_app/core/models/money.dart';
import 'package:mobile_app/features/promotions/domain/promotions_repository.dart';

class MockPromotionsRepository implements PromotionsRepository {
  // Same local voucher as the supplied Web/contract fixture; never a live offer.
  final _voucher = Voucher(
    code: 'OMAMA_FREESHIP_30K',
    title: 'Miễn phí vận chuyển 30k cho đơn từ 200k',
    type: 'FREESHIP',
    value: 30000,
    minimum: 200000,
    maximum: 30000,
    expiresAt: DateTime.utc(2026, 11, 30, 23, 59, 59),
  );
  @override
  Future<List<Voucher>> vouchers() async {
    await Future<void>.delayed(const Duration(milliseconds: 200));
    return [_voucher];
  }

  @override
  Future<VoucherValidation> validate(String code, int subtotal) async {
    await Future<void>.delayed(const Duration(milliseconds: 250));
    final valid =
        code == _voucher.code &&
        subtotal >= _voucher.minimum &&
        _voucher.expiresAt.isAfter(DateTime.now());
    return VoucherValidation(
      valid: valid,
      discount: Money.vnd(valid ? _voucher.maximum : 0),
      message: valid
          ? 'Mã đáp ứng điều kiện giá trị hàng. Mức miễn phí giao hàng tối đa 30.000đ.'
          : 'Mã không hợp lệ, đã hết hạn hoặc chưa đạt giá trị đơn tối thiểu.',
    );
  }
}
