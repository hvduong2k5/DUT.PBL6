import 'dart:convert';
import 'package:uuid/uuid.dart';
import 'package:mobile_app/core/error/failures.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/features/auth/domain/auth_models.dart';
import 'package:mobile_app/features/catalog/domain/catalog_repository.dart';
import 'package:mobile_app/features/orders/domain/order_models.dart';
import 'package:mobile_app/features/orders/domain/orders_repository.dart';
import 'package:mobile_app/features/reviews/domain/reviews_repository.dart';

class MockReviewsRepository implements ReviewsRepository {
  MockReviewsRepository({
    required this.catalog,
    required this.orders,
    required this.currentUser,
  });
  final CatalogRepository catalog;
  final OrdersRepository orders;
  final AuthUser? Function() currentUser;
  final Map<String, List<ProductReview>> _reviews = {};
  final Map<String, ({String fingerprint, ProductReview review})> _keys = {};
  final Set<String> _written = {};
  @override
  Future<ProductReviewsPage> page(
    String productId, {
    int page = 1,
    int pageSize = 10,
  }) async {
    final all = _reviews[productId] ?? [];
    return ProductReviewsPage(
      average: all.isEmpty
          ? 0
          : all.fold<int>(0, (sum, r) => sum + r.rating) / all.length,
      total: all.length,
      reviews: all.skip((page - 1) * pageSize).take(pageSize).toList(),
      page: page,
      pageSize: pageSize,
    );
  }

  @override
  Future<ProductReview> create(
    ReviewDraft draft, {
    required String idempotencyKey,
  }) async {
    final user = currentUser();
    if (user == null) {
      throw const ApiException(
        Failure(
          kind: FailureKind.unauthenticated,
          message: 'Đăng nhập để gửi đánh giá.',
        ),
      );
    }
    if (draft.rating < 1 ||
        draft.rating > 5 ||
        draft.comment.trim().length < 10 ||
        draft.comment.length > 1000) {
      throw const ApiException(
        Failure(
          kind: FailureKind.validation,
          message: 'Kiểm tra số sao và nội dung đánh giá.',
        ),
      );
    }
    final key = '${user.id}:$idempotencyKey',
        fingerprint = jsonEncode([
          draft.productId,
          draft.orderId,
          draft.sku,
          draft.rating,
          draft.comment,
        ]);
    final previous = _keys[key];
    if (previous != null) {
      if (previous.fingerprint != fingerprint) {
        throw const ApiException(
          Failure(
            kind: FailureKind.conflict,
            message: 'Yêu cầu đã thay đổi. Vui lòng gửi yêu cầu mới.',
          ),
        );
      }
      return previous.review;
    }
    final order = await orders.order(draft.orderId),
        product = await catalog.product(draft.productId);
    if (currentUser()?.id != user.id) {
      throw const ApiException(
        Failure(
          kind: FailureKind.unauthenticated,
          message: 'Phiên đăng nhập đã thay đổi. Vui lòng thử lại.',
        ),
      );
    }
    if (order.status != OrderStatus.delivered ||
        !order.items.any((i) => i.sku == draft.sku) ||
        !product.variants.any((v) => v.sku == draft.sku)) {
      throw const ApiException(
        Failure(
          kind: FailureKind.forbidden,
          message: 'Chưa xác minh được sản phẩm này trong đơn đã giao của bạn.',
        ),
      );
    }
    final pair = '${user.id}:${draft.orderId}:${draft.sku}';
    if (_written.contains(pair)) {
      throw const ApiException(
        Failure(
          kind: FailureKind.conflict,
          message: 'Sản phẩm trong đơn này đã được đánh giá.',
        ),
      );
    }
    final result = ProductReview(
      id: const Uuid().v4(),
      customer: user.fullName,
      rating: draft.rating,
      comment: draft.comment,
      media: const [],
      verified: true,
      createdAt: DateTime.now(),
      reply: '',
    );
    _reviews.putIfAbsent(draft.productId, () => []).insert(0, result);
    _keys[key] = (fingerprint: fingerprint, review: result);
    _written.add(pair);
    return result;
  }
}
