import 'dart:convert';

const b2bPurposes = {
  'VIP_CUSTOMER_GIFT': 'Tri ân khách hàng VIP & đối tác',
  'EMPLOYEE_GIFT': 'Quà nhân viên',
  'EVENT_GIFT': 'Hội nghị & sự kiện',
  'PARTNER_GIFT': 'Quà ngoại giao',
};
const b2bBranding = {
  'engraveLogo': 'Khắc logo / ép nhũ',
  'customSleeve': 'Sleeve theo nhận diện',
  'greetingCard': 'Thiệp chúc mừng',
  'ribbon': 'Ruy băng thương hiệu',
};

class B2bOrganization {
  const B2bOrganization({
    required this.id,
    required this.name,
    required this.taxCode,
    this.representative = '',
    this.title = '',
    this.phone = '',
    this.email = '',
    this.address = '',
    this.verification = '',
  });
  final String id,
      name,
      taxCode,
      representative,
      title,
      phone,
      email,
      address,
      verification;
  factory B2bOrganization.fromJson(Map<String, dynamic> json) =>
      B2bOrganization(
        id: json['organizationId'] as String,
        name: json['legalName'] as String,
        taxCode: json['taxCode'] as String,
        representative: json['representativeName'] as String? ?? '',
        title: json['representativeTitle'] as String? ?? '',
        phone: json['phone'] as String? ?? '',
        email: json['email'] as String? ?? '',
        address: json['invoiceAddress'] as String? ?? '',
        verification: json['verificationStatus'] as String? ?? '',
      );
}

class B2bProduct {
  const B2bProduct({
    required this.sku,
    required this.name,
    required this.variant,
    required this.price,
    required this.minimum,
  });
  final String sku, name, variant;
  final int price, minimum;
  factory B2bProduct.fromJson(Map<String, dynamic> json) => B2bProduct(
    sku: json['skuId'] as String,
    name: json['name'] as String,
    variant: json['variant'] as String,
    price: json['unitPriceVnd'] as int,
    minimum: json['minimumQuantity'] as int,
  );
}

class B2bDiscountTier {
  const B2bDiscountTier({
    required this.minimum,
    this.maximum,
    required this.percent,
    required this.label,
  });
  final int minimum, percent;
  final int? maximum;
  final String label;
}

class B2bRecentRequest {
  const B2bRecentRequest({
    required this.id,
    required this.number,
    required this.status,
    required this.quantity,
    required this.purpose,
    this.quoteId,
  });
  final String id, number, status, purpose;
  final int quantity;
  final String? quoteId;
  factory B2bRecentRequest.fromJson(Map<String, dynamic> json) =>
      B2bRecentRequest(
        id: json['requestId'] as String,
        number: json['requestNumber'] as String,
        status: json['statusLabel'] as String,
        quantity: json['totalQuantity'] as int,
        purpose: json['purposeLabel'] as String,
        quoteId: json['quoteId'] as String?,
      );
}

class B2bContext {
  const B2bContext({
    required this.organization,
    required this.provinces,
    required this.products,
    required this.tiers,
    required this.requests,
  });
  final B2bOrganization organization;
  final List<({String code, String name})> provinces;
  final List<B2bProduct> products;
  final List<B2bDiscountTier> tiers;
  final List<B2bRecentRequest> requests;
  factory B2bContext.fromJson(Map<String, dynamic> json) => B2bContext(
    organization: B2bOrganization.fromJson(
      Map<String, dynamic>.from(json['organization'] as Map),
    ),
    provinces: (json['deliveryProvinces'] as List).map((e) {
      final p = e as Map;
      return (
        code: p['provinceCode'] as String,
        name: p['provinceName'] as String,
      );
    }).toList(),
    products: (json['catalog'] as List)
        .map((e) => B2bProduct.fromJson(Map<String, dynamic>.from(e as Map)))
        .toList(),
    tiers: (json['discountTiers'] as List).map((e) {
      final t = e as Map;
      return B2bDiscountTier(
        minimum: t['minimumQuantity'] as int,
        maximum: t['maximumQuantity'] as int?,
        percent: t['discountPercent'] as int,
        label: t['label'] as String,
      );
    }).toList(),
    requests: (json['recentRequests'] as List)
        .map(
          (e) => B2bRecentRequest.fromJson(Map<String, dynamic>.from(e as Map)),
        )
        .toList(),
  );
}

class B2bRequestDraft {
  const B2bRequestDraft({
    required this.organizationId,
    required this.purpose,
    required this.date,
    required this.provinceCode,
    required this.provinceName,
    required this.quantities,
    required this.branding,
    required this.invoice,
    required this.sample,
  });
  final String organizationId, purpose, date, provinceCode, provinceName;
  final Map<String, int> quantities;
  final Map<String, bool> branding;
  final bool invoice, sample;
}

class B2bRequestReceipt {
  const B2bRequestReceipt(this.id, this.number, this.message);
  final String id, number, message;
  factory B2bRequestReceipt.fromJson(Map<String, dynamic> json) =>
      B2bRequestReceipt(
        json['requestId'] as String,
        json['requestNumber'] as String,
        json['message'] as String,
      );
}

class B2bQuoteItem {
  const B2bQuoteItem({
    required this.sku,
    required this.name,
    required this.variant,
    required this.quantity,
    required this.price,
    required this.total,
  });
  final String sku, name, variant;
  final int quantity, price, total;
}

class B2bQuote {
  const B2bQuote({
    required this.id,
    required this.number,
    required this.version,
    required this.status,
    required this.label,
    required this.issued,
    required this.expires,
    required this.organization,
    required this.contact,
    required this.items,
    required this.customizations,
    required this.totals,
    required this.terms,
    required this.payment,
    required this.canAccept,
  });
  final String id, number, status, label;
  final int version;
  final DateTime issued, expires;
  final B2bOrganization organization;
  final Map<String, String> contact, payment;
  final List<B2bQuoteItem> items;
  final List<String> customizations;
  final Map<String, int> totals;
  final List<({String title, String description})> terms;
  final bool canAccept;
  bool get expired => !expires.isAfter(DateTime.now());
  String get agreementFingerprint => jsonEncode([
    id,
    number,
    version,
    issued.toIso8601String(),
    expires.toIso8601String(),
    organization.id,
    organization.name,
    organization.taxCode,
    items
        .map((i) => [i.sku, i.name, i.variant, i.quantity, i.price, i.total])
        .toList(),
    customizations,
    [
      'merchandiseVnd',
      'discountVnd',
      'customizationVnd',
      'vatVnd',
      'grandTotalVnd',
      'depositPercent',
      'depositVnd',
      'remainingVnd',
    ].map((k) => totals[k]).toList(),
    terms.map((t) => [t.title, t.description]).toList(),
    [
      'bankName',
      'accountName',
      'accountNumberMasked',
      'transferContent',
    ].map((k) => payment[k]).toList(),
  ]);
  factory B2bQuote.fromJson(Map<String, dynamic> json) => B2bQuote(
    id: json['quoteId'] as String,
    number: json['quoteNumber'] as String,
    version: json['version'] as int,
    status: json['status'] as String,
    label: json['statusLabel'] as String,
    issued: DateTime.parse(json['issuedAt'] as String),
    expires: DateTime.parse(json['expiresAt'] as String),
    organization: B2bOrganization.fromJson(
      Map<String, dynamic>.from(json['organization'] as Map),
    ),
    contact: Map<String, String>.from(json['contact'] as Map),
    payment: Map<String, String>.from(json['payment'] as Map),
    totals: _parseTotals(json['totals'] as Map),
    customizations: (json['customizations'] as List).cast<String>(),
    items: (json['items'] as List).map((e) {
      final i = e as Map;
      return B2bQuoteItem(
        sku: i['skuId'] as String,
        name: i['name'] as String,
        variant: i['variant'] as String,
        quantity: i['quantity'] as int,
        price: i['unitPriceVnd'] as int,
        total: i['lineTotalVnd'] as int,
      );
    }).toList(),
    terms: (json['terms'] as List).map((e) {
      final t = e as Map;
      return (
        title: t['title'] as String,
        description: t['description'] as String,
      );
    }).toList(),
    canAccept: json['canAccept'] as bool,
  );
  static Map<String, int> _parseTotals(Map value) {
    final totals = Map<String, int>.from(value);
    for (final key in const [
      'merchandiseVnd',
      'discountVnd',
      'customizationVnd',
      'vatVnd',
      'grandTotalVnd',
      'depositPercent',
      'depositVnd',
      'remainingVnd',
    ]) {
      if (!totals.containsKey(key)) {
        throw const FormatException('Incomplete quote totals');
      }
    }
    return totals;
  }
}

class B2bAcceptance {
  const B2bAcceptance({
    required this.orderId,
    required this.orderNumber,
    required this.message,
  });
  final String orderId, orderNumber, message;
  factory B2bAcceptance.fromJson(Map<String, dynamic> json) => B2bAcceptance(
    orderId: json['orderId'] as String,
    orderNumber: json['orderNumber'] as String,
    message: json['message'] as String,
  );
}

abstract interface class B2bRepository {
  Future<B2bContext> context();
  Future<B2bRequestReceipt> request(
    B2bRequestDraft draft, {
    required String idempotencyKey,
  });
  Future<B2bQuote> quote(String id);
  Future<B2bAcceptance> accept(String id, {required String idempotencyKey});
}
