import 'dart:async';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';
import 'package:mobile_app/app/app.dart';
import 'package:mobile_app/app/app_dependencies.dart';
import 'package:mobile_app/core/config/data_source_mode.dart';
import 'package:mobile_app/core/models/page_result.dart';
import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/features/orders/data/mock_orders_repository.dart';
import 'package:mobile_app/features/orders/data/rest_orders_repository.dart';
import 'package:mobile_app/features/orders/domain/order_models.dart';
import 'package:mobile_app/features/orders/domain/orders_repository.dart';
import 'package:mobile_app/features/orders/presentation/orders_cubit.dart';

class _Orders extends Mock implements OrdersRepository {}

void main() {
  setUp(() => FlutterSecureStorage.setMockInitialValues({}));
  test(
    'history REST uses supported q/status/page/pageSize and exact raw status labels',
    () async {
      RequestOptions? request;
      final dio = Dio(BaseOptions(baseUrl: 'http://localhost'));
      dio.interceptors.add(
        InterceptorsWrapper(
          onRequest: (options, handler) {
            request = options;
            handler.resolve(
              Response(
                requestOptions: options,
                statusCode: 200,
                data: {
                  'orders': [
                    {
                      'order_id': 'O1',
                      'status': 'PAID',
                      'total_items': 1,
                      'final_amount': {
                        'currency_code': 'VND',
                        'units': 1000,
                        'nanos': 0,
                      },
                      'created_at': '2026-10-20T00:00:00Z',
                    },
                  ],
                  'total': 30,
                  'page': 2,
                  'page_size': 10,
                },
              ),
            );
          },
        ),
      );
      final result = await RestOrdersRepository(
        ApiClient(dio: dio),
      ).orderPage(query: 'O1', status: 'PAID', page: 2);
      expect(request!.queryParameters, {
        'q': 'O1',
        'status': 'PAID',
        'page': 2,
        'pageSize': 10,
      });
      expect(result.hasMore, isTrue);
      expect(result.items.single.statusLabel, 'Đã thanh toán');
    },
  );
  test(
    'page failure preserves cards; changed search discards stale page',
    () async {
      final fixture = (await MockOrdersRepository().orders()).first,
          repository = _Orders();
      when(
        () => repository.orderPage(
          query: '',
          status: null,
          page: 1,
          pageSize: 10,
        ),
      ).thenAnswer(
        (_) async =>
            PageResult(items: [fixture], total: 2, page: 1, pageSize: 1),
      );
      when(
        () =>
            repository.orderPage(query: '', status: null, page: 2, pageSize: 1),
      ).thenThrow(StateError('offline'));
      final cubit = OrdersCubit(repository);
      await cubit.load();
      await cubit.loadMore();
      expect(cubit.state.orders.single.id, fixture.id);
      expect(cubit.state.pageMessage, isNotNull);
      final pending = Completer<PageResult<CustomerOrder>>();
      when(
        () =>
            repository.orderPage(query: '', status: null, page: 2, pageSize: 1),
      ).thenAnswer((_) => pending.future);
      when(
        () => repository.orderPage(
          query: 'unknown',
          status: null,
          page: 1,
          pageSize: 10,
        ),
      ).thenAnswer(
        (_) async => PageResult(items: [], total: 0, page: 1, pageSize: 10),
      );
      final old = cubit.loadMore();
      await cubit.search('unknown');
      pending.complete(
        PageResult(items: [fixture], total: 2, page: 2, pageSize: 1),
      );
      await old;
      expect(cubit.state.orders, isEmpty);
      expect(cubit.state.query, 'unknown');
      expect(cubit.state.loadingMore, isFalse);
      await cubit.close();
    },
  );
  testWidgets(
    'history search empty recovery clears query and restores owned cards',
    (tester) async {
      final dependencies = RepositoryFactory.create(
        mode: DataSourceMode.mock,
        localMvpCheckout: false,
      );
      await tester.runAsync(() async {
        await dependencies.authRepository.login(
          phoneNumber: '0905123456',
          password: 'legacy',
        );
      });
      await tester.pumpWidget(
        OmaApp(initialLocation: '/orders', dependencies: dependencies),
      );
      await tester.pumpAndSettle();
      await tester.enterText(find.byType(TextField), 'UNKNOWN');
      await tester.testTextInput.receiveAction(TextInputAction.search);
      await tester.pumpAndSettle();
      expect(find.text('Chưa có đơn phù hợp'), findsOneWidget);
      await tester.tap(find.text('Xóa tìm kiếm & bộ lọc'));
      await tester.pumpAndSettle();
      expect(find.text('Chưa có đơn phù hợp'), findsNothing);
      expect(
        tester.widget<TextField>(find.byType(TextField)).controller!.text,
        '',
      );
      expect(find.text('#OM-8921'), findsOneWidget);
      expect(tester.takeException(), isNull);
    },
  );
}
