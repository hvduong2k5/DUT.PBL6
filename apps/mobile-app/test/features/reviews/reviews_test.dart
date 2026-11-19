import 'dart:async';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';
import 'package:mobile_app/app/app_dependencies.dart';
import 'package:mobile_app/core/config/data_source_mode.dart';
import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/features/catalog/data/mock_catalog_repository.dart';
import 'package:mobile_app/features/orders/data/mock_orders_repository.dart';
import 'package:mobile_app/features/orders/domain/orders_repository.dart';
import 'package:mobile_app/features/orders/domain/order_models.dart';
import 'package:mobile_app/features/reviews/data/rest_reviews_repository.dart';
import 'package:mobile_app/features/reviews/domain/reviews_repository.dart';
import 'package:mobile_app/features/reviews/presentation/reviews_cubits.dart';

class _Orders extends Mock implements OrdersRepository {}

class _Reviews extends Mock implements ReviewsRepository {}

const draft = ReviewDraft(
  productId: 'PROD-MX-GION-500',
  orderId: 'OM-8860',
  sku: 'MX-GION-500G',
  rating: 5,
  comment: 'Mè giòn thơm, đóng gói cẩn thận.',
);
final reviewJson = <String, dynamic>{
  'review_id': 'R1',
  'customer_name': 'Khách',
  'rating': 5,
  'comment': draft.comment,
  'media_urls': <String>[],
  'is_verified_purchase': true,
  'created_at': '2026-10-20T00:00:00Z',
  'seller_reply': '',
};
void main() {
  setUpAll(() => registerFallbackValue(draft));
  test(
    'REST uses exact ZIP review body and server aggregate, no product_id',
    () async {
      final requests = <RequestOptions>[];
      final dio = Dio(BaseOptions(baseUrl: 'http://localhost'));
      dio.interceptors.add(
        InterceptorsWrapper(
          onRequest: (o, h) {
            requests.add(o);
            h.resolve(
              Response(
                requestOptions: o,
                statusCode: o.method == 'POST' ? 201 : 200,
                data: o.method == 'POST'
                    ? reviewJson
                    : {
                        'average_rating': 4.2,
                        'total_reviews': 21,
                        'reviews': [reviewJson],
                      },
              ),
            );
          },
        ),
      );
      final repo = RestReviewsRepository(ApiClient(dio: dio));
      final page = await repo.page(draft.productId, page: 2);
      expect(page.average, 4.2);
      expect(page.total, 21);
      expect(page.hasMore, isTrue);
      expect(requests.first.queryParameters, {'page': 2, 'pageSize': 10});
      final created = await repo.create(draft, idempotencyKey: 'retry-key');
      expect(created.verified, isTrue);
      expect(requests.last.data, {
        'order_id': 'OM-8860',
        'sku_code': 'MX-GION-500G',
        'rating': 5,
        'comment': draft.comment,
        'media_urls': [],
      });
      expect(requests.last.headers['Idempotency-Key'], 'retry-key');
    },
  );
  test(
    'local adapter requires purchase, replays same key and rejects duplicate review',
    () async {
      final deps = RepositoryFactory.create(mode: DataSourceMode.mock),
          repo = deps.reviewsRepository;
      await expectLater(
        repo.create(draft, idempotencyKey: 'k'),
        throwsA(isA<ApiException>()),
      );
      await deps.authRepository.login(
        phoneNumber: '0905123456',
        password: 'local-fixture',
      );
      await expectLater(
        repo.create(
          const ReviewDraft(
            productId: 'PROD-MX-GION-500',
            orderId: 'OM-8921',
            sku: 'MX-GION-500G',
            rating: 5,
            comment: 'Sản phẩm này cần kiểm tra.',
          ),
          idempotencyKey: 'bad',
        ),
        throwsA(isA<ApiException>()),
      );
      final created = await repo.create(draft, idempotencyKey: 'k');
      expect(created.verified, isTrue);
      expect((await repo.create(draft, idempotencyKey: 'k')).id, created.id);
      await expectLater(
        repo.create(draft, idempotencyKey: 'other'),
        throwsA(isA<ApiException>()),
      );
      final page = await repo.page(draft.productId);
      expect(page.total, 1);
      expect(page.average, 5);
    },
  );
  test(
    'form rejects stale order, mismatched SKU and revoked capability; failed identical submit reuses key',
    () async {
      final orders = _Orders(), reviews = _Reviews();
      final fixture = MockOrdersRepository(),
          product = await MockCatalogRepository().product(draft.productId);
      final pending = Completer<CustomerOrder>();
      when(() => orders.order('OLD')).thenAnswer((_) => pending.future);
      when(
        () => orders.order('OM-8860'),
      ).thenAnswer((_) => fixture.order('OM-8860'));
      when(
        () => orders.order('OM-8874'),
      ).thenAnswer((_) => fixture.order('OM-8874'));
      var allowed = true;
      final cubit = ReviewFormCubit(
        reviews,
        orders,
        product,
        mayCreate: () => allowed,
      );
      final old = cubit.selectOrder('OLD');
      await cubit.selectOrder('OM-8860');
      pending.complete(await fixture.order('OM-8874'));
      await old;
      expect(cubit.state.sku, 'MX-GION-500G');
      final keys = <String>[];
      when(
        () =>
            reviews.create(any(), idempotencyKey: any(named: 'idempotencyKey')),
      ).thenAnswer((i) async {
        keys.add(i.namedArguments[#idempotencyKey] as String);
        throw StateError('offline');
      });
      await cubit.submit(5, draft.comment);
      await cubit.submit(5, draft.comment);
      expect(keys.length, 2);
      expect(keys[0], keys[1]);
      await cubit.submit(4, draft.comment);
      expect(keys.last, isNot(keys.first));
      allowed = false;
      await cubit.submit(5, draft.comment);
      expect(keys.length, 3);
      allowed = true;
      await cubit.selectOrder('OM-8874');
      expect(cubit.state.skus, isEmpty);
      await cubit.submit(5, draft.comment);
      expect(keys.length, 3);
      await cubit.close();
    },
  );
}
