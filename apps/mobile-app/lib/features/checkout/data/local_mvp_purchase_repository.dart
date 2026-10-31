import 'dart:convert';
import 'package:mobile_app/core/error/failures.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/features/cart/domain/cart_models.dart';
import 'package:mobile_app/features/cart/domain/cart_repository.dart';
import 'package:mobile_app/features/checkout/domain/mvp_purchase_models.dart';
import 'package:mobile_app/features/checkout/domain/mvp_purchase_repository.dart';

// Explicit local-only fixtures, never a fallback for a failing remote quote.
class LocalMvpPurchaseRepository implements MvpPurchaseRepository {
  LocalMvpPurchaseRepository(
    this._cart,
    this._authenticated, {
    DateTime Function()? now,
  }) : _now = now ?? DateTime.now;
  final CartRepository _cart;
  final bool Function() _authenticated;
  final DateTime Function() _now;
  int _revision = 0;
  String? _fingerprint;
  static const addressId = '11111111-1111-4111-8111-111111111111';

  void _authorize() {
    if (!_authenticated()) {
      throw const ApiException(
        Failure(
          kind: FailureKind.unauthenticated,
          message: 'Vui lòng đăng nhập để xem báo giá.',
        ),
      );
    }
  }

  Map<String, dynamic> _item(CartLine line) => {
    'item_id': line.itemId,
    'sku_code': line.variant.sku,
    'product_name': '${line.product.name} · ${line.variant.name}',
    'quantity': line.quantity,
    'unit_price': int.parse(line.variant.price.units.toString()),
  };

  @override
  Future<MvpCart> cart() async {
    _authorize();
    final value = await _cart.getCart();
    _authorize();
    final items = value.items.map(_item).toList();
    final fingerprint = jsonEncode(items);
    if (fingerprint != _fingerprint) {
      _fingerprint = fingerprint;
      _revision++;
    }
    return MvpCart.fromJson({'revision': _revision, 'items': items});
  }

  @override
  Future<List<MvpSavedAddress>> savedAddresses() async {
    _authorize();
    return [
      MvpSavedAddress.fromJson({
        'id': addressId,
        'recipient_name': 'Người nhận thử nghiệm',
        'phone_number': '0900000000',
        'street_address': 'Đường mẫu — dữ liệu local',
        'province_code': '01',
        'ward_code': '00001',
        'province_name': 'Tỉnh mẫu (local)',
        'ward_name': 'Phường mẫu (local)',
        'version': 1,
        'is_default': true,
      }),
    ];
  }

  @override
  Future<MvpQuote> quote({
    required int cartRevision,
    required String addressId,
  }) async {
    final current = await cart();
    final lines = (await _cart.getCart()).items;
    _authorize();
    if (addressId != LocalMvpPurchaseRepository.addressId ||
        cartRevision != current.revision ||
        lines.isEmpty ||
        lines.any((l) => !l.product.inStock || l.variant.stock < l.quantity)) {
      throw const ApiException(
        Failure(
          kind: FailureKind.conflict,
          message:
              'Giỏ hoặc địa chỉ đã thay đổi. Vui lòng tải lại để xem báo giá mới.',
        ),
      );
    }
    final subtotal = lines.fold<int>(
      0,
      (sum, l) => sum + int.parse(l.subtotal.units.toString()),
    );
    final expiry = _now()
        .toUtc()
        .add(const Duration(minutes: 2))
        .toIso8601String();
    return MvpQuote.fromJson({
      'quote_id': 'LOCAL-QUOTE-$cartRevision-${_now().microsecondsSinceEpoch}',
      'cart_revision': cartRevision,
      'payment_method': 'VIETQR',
      'source_address_id': addressId,
      'source_address_version': 1,
      'items': lines.map(_item).toList(),
      'address': {
        'recipient_name': 'Người nhận thử nghiệm',
        'phone': '0900000000',
        'street': 'Đường mẫu — dữ liệu local',
        'province_code': '01',
        'ward_code': '00001',
      },
      'catalog_snapshot_id': 'LOCAL-CATALOG',
      'shipping_quote_reference': 'LOCAL-SHIPPING',
      'subtotal': subtotal,
      'shipping_fee': 15000,
      'final_amount': subtotal + 15000,
      'expires_at': expiry,
      'catalog_expires_at': expiry,
      'shipping_expires_at': expiry,
    });
  }
}
