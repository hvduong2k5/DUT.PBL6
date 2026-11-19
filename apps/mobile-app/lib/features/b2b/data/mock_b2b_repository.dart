import 'package:uuid/uuid.dart';
import 'package:mobile_app/core/error/failures.dart';
import 'package:mobile_app/core/models/money.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/features/auth/domain/auth_models.dart';
import 'package:mobile_app/features/auth/data/mock_auth_repository.dart';
import 'package:mobile_app/features/orders/domain/order_models.dart';
import 'package:mobile_app/features/b2b/domain/b2b_repository.dart';

class MockB2bRepository implements B2bRepository {
  MockB2bRepository({required this.currentUser, required this.onAccepted});
  final AuthUser? Function() currentUser;
  final void Function(CustomerOrder) onAccepted;
  final Map<String, B2bRequestReceipt> _requests = {};
  final List<B2bRecentRequest> _recent = [];
  final Map<String, B2bAcceptance> _acceptances = {};
  late final Map<String, dynamic> _quote = {
    'quoteId': 'LOCAL-QUOTE-001',
    'quoteNumber': 'BG-LOCAL-001',
    'version': 1,
    'status': 'SENT',
    'statusLabel': 'Đã phát hành · dữ liệu local',
    'issuedAt': DateTime.now()
        .subtract(const Duration(days: 1))
        .toUtc()
        .toIso8601String(),
    'expiresAt': DateTime.now()
        .add(const Duration(days: 30))
        .toUtc()
        .toIso8601String(),
    'organization': {
      'organizationId': 'LOCAL-ORG',
      'legalName': 'Doanh nghiệp thử nghiệm local',
      'taxCode': 'LOCAL-TEST',
    },
    'contact': {
      'displayName': 'Chuyên viên local',
      'title': 'Hỗ trợ doanh nghiệp',
      'phone': '',
      'email': '',
    },
    'items': [
      {
        'skuId': 'LOCAL-GIFT',
        'name': 'Hộp quà Tri Kỷ',
        'variant': 'Bộ quà doanh nghiệp',
        'quantity': 20,
        'unitPriceVnd': 220000,
        'lineTotalVnd': 4400000,
      },
    ],
    'customizations': ['Thiệp chúc mừng riêng'],
    'totals': {
      'merchandiseVnd': 4400000,
      'discountVnd': 220000,
      'customizationVnd': 0,
      'vatVnd': 0,
      'grandTotalVnd': 4180000,
      'depositPercent': 30,
      'depositVnd': 1254000,
      'remainingVnd': 2926000,
    },
    'terms': [
      {
        'title': 'Giao nhận',
        'description':
            'Thời gian giao do chuyên viên xác nhận trong phiên thử nghiệm.',
      },
      {
        'title': 'Thanh toán',
        'description':
            'Đặt cọc 30%; chấp thuận chưa phải thanh toán. Không chuyển tiền thật trong chế độ local.',
      },
    ],
    'payment': {
      'bankName': 'Ngân hàng thử nghiệm',
      'accountName': 'DOANH NGHIEP LOCAL',
      'accountNumberMasked': '**** 0000',
      'transferContent': 'LOCAL TEST ONLY',
    },
    'canAccept': true,
  };
  void _verified() {
    if (currentUser()?.id != MockAuthRepository.b2bFixtureId) {
      throw const ApiException(
        Failure(
          kind: FailureKind.forbidden,
          message: 'Tổ chức chưa được xác minh hoặc bạn không có quyền B2B.',
        ),
      );
    }
  }

  @override
  Future<B2bContext> context() async {
    _verified();
    return B2bContext(
      organization: const B2bOrganization(
        id: 'LOCAL-ORG',
        name: 'Doanh nghiệp thử nghiệm local',
        taxCode: 'LOCAL-TEST',
        representative: 'Đại diện local',
        title: 'Người phụ trách',
        address: 'Địa chỉ hóa đơn mẫu',
        verification: 'VERIFIED',
      ),
      provinces: const [
        (code: 'HUE', name: 'Thành phố Huế'),
        (code: 'DA-NANG', name: 'Thành phố Đà Nẵng'),
      ],
      products: const [
        B2bProduct(
          sku: 'LOCAL-GIFT',
          name: 'Hộp quà Tri Kỷ',
          variant: 'Bộ quà doanh nghiệp',
          price: 220000,
          minimum: 20,
        ),
        B2bProduct(
          sku: 'LOCAL-TEA',
          name: 'Trà Sen',
          variant: 'Hộp 100g',
          price: 120000,
          minimum: 10,
        ),
      ],
      tiers: const [
        B2bDiscountTier(
          minimum: 20,
          maximum: 99,
          percent: 5,
          label: 'Từ 20 bộ',
        ),
        B2bDiscountTier(minimum: 100, percent: 10, label: 'Từ 100 bộ'),
      ],
      requests: [
        ..._recent,
        const B2bRecentRequest(
          id: 'LOCAL-REQUEST-001',
          number: 'YC-LOCAL-001',
          status: 'Đã có báo giá',
          quantity: 20,
          purpose: 'Quà đối tác',
          quoteId: 'LOCAL-QUOTE-001',
        ),
      ],
    );
  }

  @override
  Future<B2bRequestReceipt> request(
    B2bRequestDraft draft, {
    required String idempotencyKey,
  }) async {
    _verified();
    if (_requests.containsKey(idempotencyKey)) {
      return _requests[idempotencyKey]!;
    }
    final ctx = await context();
    if (draft.organizationId != ctx.organization.id ||
        !draft.quantities.entries.any((e) => e.value > 0) ||
        draft.quantities.entries.any(
          (e) =>
              e.value > 0 &&
              !ctx.products.any((p) => p.sku == e.key && e.value >= p.minimum),
        )) {
      throw const ApiException(
        Failure(
          kind: FailureKind.validation,
          message: 'Kiểm tra tổ chức, sản phẩm và số lượng tối thiểu.',
        ),
      );
    }
    final id = const Uuid().v4(), number = 'YC-LOCAL-${_requests.length + 2}';
    final receipt = B2bRequestReceipt(
      id,
      number,
      'Yêu cầu local đã được ghi nhận. Không gửi đến chuyên viên thật.',
    );
    _requests[idempotencyKey] = receipt;
    _recent.insert(
      0,
      B2bRecentRequest(
        id: id,
        number: number,
        status: 'Đã tiếp nhận',
        quantity: draft.quantities.values.fold(0, (a, b) => a + b),
        purpose: b2bPurposes[draft.purpose] ?? draft.purpose,
      ),
    );
    return receipt;
  }

  @override
  Future<B2bQuote> quote(String id) async {
    _verified();
    if (id != _quote['quoteId']) {
      throw const ApiException(
        Failure(
          kind: FailureKind.notFound,
          message: 'Không tìm thấy báo giá đã phát hành thuộc tổ chức.',
        ),
      );
    }
    return B2bQuote.fromJson(_quote);
  }

  @override
  Future<B2bAcceptance> accept(
    String id, {
    required String idempotencyKey,
  }) async {
    final value = await quote(id);
    if (_acceptances.containsKey(idempotencyKey)) {
      return _acceptances[idempotencyKey]!;
    }
    if (!value.canAccept || value.expired) {
      throw const ApiException(
        Failure(
          kind: FailureKind.conflict,
          message: 'Báo giá đã hết hạn hoặc không còn có thể chấp thuận.',
        ),
      );
    }
    const orderId = 'LOCAL-B2B-001';
    onAccepted(
      CustomerOrder(
        id: orderId,
        status: OrderStatus.pending,
        rawStatus: 'PENDING_PAYMENT',
        totalItems: value.items.fold(0, (a, i) => a + i.quantity),
        total: Money.vnd(value.totals['grandTotalVnd']!),
        createdAt: DateTime.now(),
        subtotal: Money.vnd(value.totals['merchandiseVnd']!),
        discount: Money.vnd(value.totals['discountVnd']!),
        items: value.items
            .map(
              (i) => OrderItem(
                name: i.name,
                variantName: i.variant,
                sku: i.sku,
                quantity: i.quantity,
                price: Money.vnd(i.price),
              ),
            )
            .toList(),
      ),
    );
    const result = B2bAcceptance(
      orderId: orderId,
      orderNumber: orderId,
      message:
          'Đã chấp thuận báo giá local, đơn đang chờ thanh toán. Chưa có khoản tiền nào được chuyển.',
    );
    _acceptances[idempotencyKey] = result;
    _quote['canAccept'] = false;
    _quote['status'] = 'ACCEPTED';
    _quote['statusLabel'] = 'Đã chấp thuận';
    return result;
  }
}
