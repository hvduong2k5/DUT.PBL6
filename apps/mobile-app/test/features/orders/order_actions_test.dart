import 'dart:async';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';
import 'package:mobile_app/app/app.dart';
import 'package:mobile_app/app/app_dependencies.dart';
import 'package:mobile_app/core/config/data_source_mode.dart';
import 'package:mobile_app/core/models/money.dart';
import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/features/cart/domain/cart_models.dart';
import 'package:mobile_app/features/checkout/domain/checkout_models.dart';
import 'package:mobile_app/features/orders/data/mock_orders_repository.dart';
import 'package:mobile_app/features/orders/data/rest_orders_repository.dart';
import 'package:mobile_app/features/orders/domain/order_models.dart';
import 'package:mobile_app/features/orders/domain/orders_repository.dart';
import 'package:mobile_app/features/orders/presentation/order_cancellation_cubit.dart';

class _Orders extends Mock implements OrdersRepository {}

Map<String, Object> money(int units) => {
  'currency_code': 'VND',
  'units': units,
  'nanos': 0,
};
void main() {
  setUp(() => FlutterSecureStorage.setMockInitialValues({}));
  test(
    'stable order preserves SKU/raw price/totals; cancellation and tracking match ZIP',
    () async {
      final requests = <RequestOptions>[];
      final dio = Dio(BaseOptions(baseUrl: 'http://localhost'));
      dio.interceptors.add(
        InterceptorsWrapper(
          onRequest: (options, handler) {
            requests.add(options);
            handler.resolve(
              Response(
                requestOptions: options,
                statusCode: 200,
                data: options.path.endsWith('/cancel')
                    ? {'is_cancelled': true, 'status': 'CANCELLED_BY_USER'}
                    : options.path.endsWith('/tracking')
                    ? {
                        'order_id': 'O/1',
                        'tracking_code': '',
                        'carrier_name': '',
                        'current_status': 'PENDING',
                        'seal_code': '',
                        'packing_video_url': '',
                        'shipper_name': '',
                        'shipper_phone': '',
                        'checkpoints': [],
                      }
                    : {
                        'order_id': 'O/1',
                        'status': 'NEW_STATE',
                        'created_at': '2026-10-20T00:00:00Z',
                        'items': [
                          {
                            'sku_code': 'SKU-1',
                            'quantity': 3,
                            'price': money(10000),
                          },
                        ],
                        'subtotal_amount': money(30000),
                        'discount_amount': money(1000),
                        'shipping_fee': money(5000),
                        'final_amount': money(34000),
                      },
              ),
            );
          },
        ),
      );
      final repository = RestOrdersRepository(ApiClient(dio: dio));
      final order = await repository.order('O/1');
      expect(order.status, OrderStatus.unknown);
      expect(order.rawStatus, 'NEW_STATE');
      expect(order.items.single.sku, 'SKU-1');
      expect(order.items.single.unitPriceKnown, isFalse);
      expect(order.items.single.price, Money.vnd(10000));
      expect(order.total, Money.vnd(34000));
      expect((await repository.tracking('O/1')).checkpoints, isEmpty);
      expect(requests.last.path, '/api/v1/orders/O%2F1/tracking');
      expect(
        await repository.cancel(
          'O/1',
          'CUSTOMER_CHANGED_MIND',
          idempotencyKey: 'retry',
        ),
        'CANCELLED_BY_USER',
      );
      expect(requests.last.data, {'reason': 'CUSTOMER_CHANGED_MIND'});
      expect(requests.last.headers['Idempotency-Key'], 'retry');
    },
  );
  test(
    'cancel guards duplicates and retains retry key until reason changes',
    () async {
      final repository = _Orders(), keys = <String>[];
      final pending = Completer<String>();
      var count = 0;
      when(
        () => repository.cancel(
          'O1',
          any(),
          idempotencyKey: any(named: 'idempotencyKey'),
        ),
      ).thenAnswer((invocation) {
        keys.add(invocation.namedArguments[#idempotencyKey] as String);
        return ++count == 1
            ? pending.future
            : Future<String>.error(StateError('offline'));
      });
      final cubit = OrderCancellationCubit(repository, 'O1');
      final first = cubit.cancel('Thay đổi nhu cầu');
      await cubit.cancel('Thay đổi nhu cầu');
      expect(count, 1);
      pending.completeError(StateError('offline'));
      await first;
      await cubit.cancel('Thay đổi nhu cầu');
      await cubit.cancel('Lý do khác');
      expect(keys[0], keys[1]);
      expect(keys[2], isNot(keys[1]));
      expect(cubit.state.pending, isFalse);
      expect(cubit.state.status, isNull);
      await cubit.close();
    },
  );
  test(
    'local cancellation is authoritative, idempotent and rejects paid/shipping orders',
    () async {
      final repository = MockOrdersRepository();
      repository.record(
        OrderConfirmation(
          orderId: 'LOCAL-PENDING',
          status: 'PENDING_PAYMENT',
          total: Money.vnd(0),
        ),
        const CheckoutDraft(
          fullName: 'Khách local',
          phone: '0905123456',
          streetAddress: 'Đường mẫu',
          ward: 'Phường mẫu',
          province: 'Huế',
          cart: Cart(id: 'c', items: []),
        ),
      );
      expect(
        await repository.cancel(
          'LOCAL-PENDING',
          'CUSTOMER_CHANGED_MIND',
          idempotencyKey: 'same',
        ),
        'CANCELLED_BY_USER',
      );
      expect(
        await repository.cancel(
          'LOCAL-PENDING',
          'CUSTOMER_CHANGED_MIND',
          idempotencyKey: 'same',
        ),
        'CANCELLED_BY_USER',
      );
      expect(
        (await repository.order('LOCAL-PENDING')).status,
        OrderStatus.cancelled,
      );
      await expectLater(
        repository.cancel(
          'OM-8921',
          'CUSTOMER_CHANGED_MIND',
          idempotencyKey: 'new',
        ),
        throwsA(isA<ApiException>()),
      );
    },
  );
  testWidgets(
    'registered customer confirms cancellation and detail reflects server status',
    (tester) async {
      final dependencies = RepositoryFactory.create(mode: DataSourceMode.mock);
      await tester.runAsync(() async {
        await dependencies.authRepository.login(
          phoneNumber: '0905123456',
          password: 'legacy',
        );
      });
      (dependencies.ordersRepository as MockOrdersRepository).record(
        OrderConfirmation(
          orderId: 'LOCAL-PENDING',
          status: 'PENDING_PAYMENT',
          total: Money.vnd(0),
        ),
        const CheckoutDraft(
          fullName: 'Khách local',
          phone: '0905123456',
          streetAddress: 'Đường mẫu',
          ward: 'Phường mẫu',
          province: 'Huế',
          cart: Cart(id: 'c', items: []),
        ),
      );
      await tester.pumpWidget(
        OmaApp(
          initialLocation: '/orders/LOCAL-PENDING',
          dependencies: dependencies,
        ),
      );
      await tester.pumpAndSettle();
      await tester.tap(find.text('Yêu cầu hủy đơn'));
      await tester.pumpAndSettle();
      await tester.enterText(find.byType(TextField), 'CUSTOMER_CHANGED_MIND');
      await tester.tap(find.text('Xác nhận hủy'));
      await tester.pumpAndSettle();
      expect(find.text('Yêu cầu hủy đơn'), findsNothing);
      expect(find.text('Đã hủy'), findsWidgets);
      expect(tester.takeException(), isNull);
    },
  );
  testWidgets(
    'guest tracking requires actual grant and then displays public checkpoints',
    (tester) async {
      final dependencies = RepositoryFactory.create(mode: DataSourceMode.mock);
      await tester.runAsync(() async {
        final challenge = await dependencies.guestOrdersRepository.challenge(
          'OM-8921',
          '0905123456',
          idempotencyKey: 'challenge',
        );
        await dependencies.guestOrdersRepository.verify(
          challenge.id,
          '789214',
          idempotencyKey: 'verify',
        );
      });
      await tester.pumpWidget(
        OmaApp(
          initialLocation: '/orders/OM-8921/tracking',
          dependencies: dependencies,
        ),
      );
      await tester.pumpAndSettle();
      expect(find.text('Mã vận đơn: LOCAL-OM-8921'), findsOneWidget);
      await tester.scrollUntilVisible(
        find.text('Các mốc công khai'),
        250,
        scrollable: find.byType(Scrollable).first,
      );
      expect(find.text('Các mốc công khai'), findsOneWidget);
      expect(tester.takeException(), isNull);
    },
  );
}
