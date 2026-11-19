import 'package:mobile_app/core/models/money.dart';

// Public MVP target at dev7f22226. Deliberately separate from ZIP legacy DTOs.
int _integer(Object? value, {int minimum = 0}) {
  if (value is! int || value < minimum) {
    throw const FormatException('Invalid MVP integer');
  }
  return value;
}

String _text(Object? value) {
  if (value is! String || value.trim().isEmpty) {
    throw const FormatException('Missing MVP value');
  }
  return value;
}

Map<String, dynamic> _object(Object? value) {
  if (value is! Map) throw const FormatException('Expected MVP object');
  return Map<String, dynamic>.from(value);
}

Money _vnd(Object? value) => Money.vnd(_integer(value));

class MvpCartItem {
  MvpCartItem.fromJson(Map<String, dynamic> json)
    : sku = _text(json['sku_code']),
      quantity = _integer(json['quantity'], minimum: 1),
      id = json['item_id'] as String?,
      name = json['product_name'] as String?,
      unitPrice = json['unit_price'] == null ? null : _vnd(json['unit_price']);

  final String sku;
  final int quantity;
  final String? id, name;
  // Missing price is unavailable, never a free item or invented stock flag.
  final Money? unitPrice;
}

List<MvpCartItem> _items(Object? value) {
  if (value is! List) throw const FormatException('Missing MVP items');
  return List.unmodifiable(value.map((v) => MvpCartItem.fromJson(_object(v))));
}

class MvpCart {
  MvpCart.fromJson(Map<String, dynamic> json)
    : revision = _integer(json['revision']),
      items = _items(json['items']);
  final int revision;
  final List<MvpCartItem> items;
}

class MvpSavedAddress {
  MvpSavedAddress.fromJson(Map<String, dynamic> json)
    : id = json['id'] as String? ?? '',
      recipient = json['recipient_name'] as String? ?? '',
      phone = json['phone_number'] as String? ?? '',
      street = json['street_address'] as String? ?? '',
      provinceCode = json['province_code'] as String? ?? '',
      wardCode = json['ward_code'] as String? ?? '',
      provinceName = json['province_name'] as String?,
      wardName = json['ward_name'] as String?,
      version = json['version'] == null
          ? null
          : _integer(json['version'], minimum: 1),
      isDefault = json['is_default'] == true,
      isDeleted = json['is_deleted'] == true;

  final String id, recipient, phone, street, provinceCode, wardCode;
  final String? provinceName, wardName;
  final int? version;
  final bool isDefault, isDeleted;
  bool get usable =>
      !isDeleted &&
      [
        id,
        recipient,
        phone,
        street,
        provinceCode,
        wardCode,
      ].every((v) => v.trim().isNotEmpty);
}

class MvpQuoteAddress {
  MvpQuoteAddress.fromJson(Map<String, dynamic> json)
    : recipient = _text(json['recipient_name']),
      phone = _text(json['phone']),
      street = _text(json['street']),
      wardCode = _text(json['ward_code']),
      provinceCode = _text(json['province_code']);
  final String recipient, phone, street, wardCode, provinceCode;
}

DateTime _date(Object? value) {
  final text = _text(value);
  // RFC3339 values carry an offset; never interpret server expiry as local time.
  if (!RegExp(r'(Z|[+-]\d{2}:\d{2})$').hasMatch(text)) {
    throw const FormatException('MVP timestamp requires timezone');
  }
  return DateTime.parse(text);
}

class MvpQuote {
  MvpQuote.fromJson(Map<String, dynamic> json)
    : id = _text(json['quote_id']),
      cartRevision = _integer(json['cart_revision']),
      paymentMethod = _text(json['payment_method']),
      items = _items(json['items']),
      address = MvpQuoteAddress.fromJson(_object(json['address'])),
      sourceAddressId = json['source_address_id'] as String?,
      sourceAddressVersion = json['source_address_version'] == null
          ? null
          : _integer(json['source_address_version'], minimum: 1),
      catalogSnapshotId = _text(json['catalog_snapshot_id']),
      shippingQuoteReference = _text(json['shipping_quote_reference']),
      subtotal = _vnd(json['subtotal']),
      shippingFee = _vnd(json['shipping_fee']),
      total = _vnd(json['final_amount']),
      expiresAt = _date(json['expires_at']),
      catalogExpiresAt = _date(json['catalog_expires_at']),
      shippingExpiresAt = _date(json['shipping_expires_at']);

  final String id, paymentMethod, catalogSnapshotId, shippingQuoteReference;
  final String? sourceAddressId;
  final int cartRevision;
  final int? sourceAddressVersion;
  final List<MvpCartItem> items;
  final MvpQuoteAddress address;
  final Money subtotal, shippingFee, total;
  final DateTime expiresAt, catalogExpiresAt, shippingExpiresAt;

  DateTime get validUntil => [catalogExpiresAt, shippingExpiresAt].fold(
    expiresAt,
    (earliest, value) => value.isBefore(earliest) ? value : earliest,
  );

  bool isUsableAt(DateTime now) =>
      items.isNotEmpty &&
      [
        expiresAt,
        catalogExpiresAt,
        shippingExpiresAt,
      ].every((expiry) => expiry.isAfter(now));
}
