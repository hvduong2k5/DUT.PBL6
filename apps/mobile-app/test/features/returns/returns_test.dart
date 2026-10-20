import 'package:dio/dio.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';
import 'package:mobile_app/core/models/money.dart';
import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/features/orders/data/mock_orders_repository.dart';
import 'package:mobile_app/features/orders/domain/orders_repository.dart';
import 'package:mobile_app/features/orders/domain/order_models.dart';
import 'package:mobile_app/features/returns/data/rest_returns_repository.dart';
import 'package:mobile_app/features/returns/data/mock_returns_repository.dart';
import 'package:mobile_app/features/returns/domain/returns_repository.dart';
import 'package:mobile_app/features/returns/presentation/returns_cubit.dart';

class _Orders extends Mock implements OrdersRepository {}

class _Returns extends Mock implements ReturnsRepository {}

ReturnDraft draft(String line, {int quantity = 1}) => ReturnDraft(
  orderId: 'OM-8874',
  lineId: line,
  quantity: quantity,
  reason: 'QUALITY_ISSUE',
  details: 'Nội dung thử nghiệm local.',
  evidenceUrls: const ['https://example.com/evidence.jpg'],
  bank: 'LOCALBANK',
  account: '12345678',
  holder: 'KHACH LOCAL',
);
void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  setUpAll(() => registerFallbackValue(draft('FALLBACK')));
  setUp(() => FlutterSecureStorage.setMockInitialValues({}));
  test(
    'stable return payload exact; production form does not invent line IDs or call draft eligibility',
    () async {
      final orders = _Orders();
      when(() => orders.order('OM-8874')).thenAnswer(
        (_) async => CustomerOrder(
          id: 'OM-8874',
          status: OrderStatus.delivered,
          totalItems: 1,
          total: Money.vnd(1),
          createdAt: DateTime(2026),
        ),
      );
      final requests = <RequestOptions>[];
      final dio = Dio(BaseOptions(baseUrl: 'http://localhost'));
      dio.interceptors.add(
        InterceptorsWrapper(
          onRequest: (options, handler) {
            requests.add(options);
            handler.resolve(
              Response(
                requestOptions: options,
                statusCode: 201,
                data: {
                  'return_id': 'R/1',
                  'order_id': 'OM-8874',
                  'status': 'APPROVED',
                  'reason': 'Đang xử lý',
                  'refund_amount': {
                    'currency_code': 'VND',
                    'units': 10000,
                    'nanos': 0,
                  },
                  'created_at': '2026-10-20T00:00:00Z',
                },
              ),
            );
          },
        ),
      );
      final repository = RestReturnsRepository(ApiClient(dio: dio), orders);
      final ctx = await repository.context('OM-8874');
      expect(ctx.blockingMessage, isNotNull);
      expect(ctx.lines, isEmpty);
      expect(requests, isEmpty);
      final result = await repository.create(
        draft('SERVER-LINE'),
        idempotencyKey: 'retry',
      );
      expect(requests.single.path, '/api/v1/returns');
      expect(requests.single.data, {
        'order_id': 'OM-8874',
        'lines': [
          {'line_id': 'SERVER-LINE', 'quantity': 1},
        ],
        'reason_code': 'QUALITY_ISSUE',
        'details': 'Nội dung thử nghiệm local.',
        'evidence_media_urls': ['https://example.com/evidence.jpg'],
        'refund_bank_code': 'LOCALBANK',
        'refund_account_number': '12345678',
        'refund_account_holder': 'KHACH LOCAL',
        'idempotency_key': 'retry',
      });
      expect(requests.single.headers['Idempotency-Key'], 'retry');
      expect(result.statusLabel, 'Đã duyệt yêu cầu');
      expect(result.status, isNot('REFUNDED'));
      await repository.detail('R/1');
      expect(requests.last.path, '/api/v1/returns/R%2F1');
    },
  );
  test(
    'local return owns server-fixture lines, caps quantities and keeps case ownership',
    () async {
      String? owner = 'C1';
      final repository = MockReturnsRepository(
        MockOrdersRepository(),
        () => owner,
      );
      final ctx = await repository.context('OM-8874');
      final line = ctx.lines.single;
      expect(line.id, startsWith('LOCAL-LINE-'));
      await expectLater(
        repository.create(draft('FORGED'), idempotencyKey: 'bad'),
        throwsA(isA<ApiException>()),
      );
      final result = await repository.create(
        draft(line.id),
        idempotencyKey: 'same',
      );
      expect(
        (await repository.create(draft(line.id), idempotencyKey: 'same')).id,
        result.id,
      );
      expect((await repository.context('OM-8874')).lines.single.maximum, 1);
      expect(result.refund, Money.vnd(0));
      owner = 'C2';
      await expectLater(
        repository.detail(result.id),
        throwsA(isA<ApiException>()),
      );
      owner = null;
      await expectLater(
        repository.detail(result.id),
        throwsA(isA<ApiException>()),
      );
    },
  );
  test(
    'blocked production form never submits even with client-supplied line ID',
    () async {
      final repository = _Returns();
      when(() => repository.context('OM-8874')).thenAnswer(
        (_) async => const ReturnFormContext(
          orderId: 'OM-8874',
          blockingMessage: 'Unavailable',
        ),
      );
      final cubit = ReturnsCubit(repository);
      await cubit.load('OM-8874');
      await cubit.submit(draft('FORGED'));
      verifyNever(
        () => repository.create(
          any(),
          idempotencyKey: any(named: 'idempotencyKey'),
        ),
      );
      expect(cubit.state.created, isNull);
      await cubit.close();
    },
  );
  test('return retry keeps key; editing financial request renews it', () async {
    final repository = _Returns(), keys = <String>[];
    when(() => repository.context('OM-8874')).thenAnswer(
      (_) async => ReturnFormContext(
        orderId: 'OM-8874',
        lines: [
          ReturnLine(
            id: 'L1',
            name: 'Mẫu',
            variant: 'Hộp',
            maximum: 2,
            price: Money.vnd(1),
          ),
        ],
      ),
    );
    when(
      () => repository.create(
        any(),
        idempotencyKey: any(named: 'idempotencyKey'),
      ),
    ).thenAnswer((invocation) async {
      keys.add(invocation.namedArguments[#idempotencyKey] as String);
      throw StateError('offline');
    });
    final cubit = ReturnsCubit(repository);
    await cubit.load('OM-8874');
    await cubit.submit(draft('L1'));
    await cubit.submit(draft('L1'));
    await cubit.submit(draft('L1', quantity: 2));
    expect(keys[0], keys[1]);
    expect(keys[2], isNot(keys[1]));
    expect(cubit.state.pending, isFalse);
    await cubit.close();
  });
}
