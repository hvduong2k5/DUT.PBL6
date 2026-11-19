import 'dart:convert';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:uuid/uuid.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/features/catalog/domain/catalog_models.dart';
import 'package:mobile_app/features/orders/domain/order_models.dart';
import 'package:mobile_app/features/orders/domain/orders_repository.dart';
import 'package:mobile_app/features/reviews/domain/reviews_repository.dart';

String _error(Object e) => e is ApiException
    ? e.failure.message
    : 'Chưa thể tải hoặc gửi đánh giá. Vui lòng thử lại.';

class ReviewsState {
  const ReviewsState({
    this.loading = false,
    this.loadingMore = false,
    this.data,
    this.error,
    this.pageError,
  });
  final bool loading, loadingMore;
  final ProductReviewsPage? data;
  final String? error, pageError;
}

class ReviewsCubit extends Cubit<ReviewsState> {
  ReviewsCubit(this._repository, this.productId) : super(const ReviewsState());
  final ReviewsRepository _repository;
  final String productId;
  int _revision = 0;
  Future<void> load() async {
    final revision = ++_revision;
    emit(const ReviewsState(loading: true));
    try {
      final data = await _repository.page(productId);
      if (!isClosed && revision == _revision) emit(ReviewsState(data: data));
    } catch (e) {
      if (!isClosed && revision == _revision) {
        emit(ReviewsState(error: _error(e)));
      }
    }
  }

  Future<void> more() async {
    final data = state.data;
    if (data == null || !data.hasMore || state.loadingMore) return;
    final revision = _revision;
    emit(ReviewsState(data: data, loadingMore: true));
    try {
      final next = await _repository.page(
        productId,
        page: data.page + 1,
        pageSize: data.pageSize,
      );
      if (isClosed || revision != _revision) return;
      final merged = {for (final r in data.reviews) r.id: r};
      final count = merged.length;
      for (final r in next.reviews) {
        merged[r.id] = r;
      }
      if (merged.length == count && merged.length < next.total) {
        throw const FormatException('No review pagination progress');
      }
      emit(
        ReviewsState(
          data: ProductReviewsPage(
            average: next.average,
            total: next.total,
            reviews: merged.values.toList(),
            page: next.page,
            pageSize: next.pageSize,
          ),
        ),
      );
    } catch (e) {
      if (!isClosed && revision == _revision) {
        emit(ReviewsState(data: data, pageError: _error(e)));
      }
    }
  }
}

class ReviewFormState {
  const ReviewFormState({
    this.loading = false,
    this.loadingMore = false,
    this.checkingOrder = false,
    this.pending = false,
    this.orders = const [],
    this.page = 0,
    this.total = 0,
    this.pageSize = 10,
    this.orderId,
    this.skus = const [],
    this.sku,
    this.created,
    this.error,
  });
  final bool loading, loadingMore, checkingOrder, pending;
  final List<CustomerOrder> orders;
  final int page, total, pageSize;
  final String? orderId, sku, error;
  final List<String> skus;
  final ProductReview? created;
  bool get hasMore => page * pageSize < total;
  ReviewFormState copy({
    bool? loading,
    bool? loadingMore,
    bool? checkingOrder,
    bool? pending,
    List<CustomerOrder>? orders,
    int? page,
    int? total,
    int? pageSize,
    String? orderId,
    List<String>? skus,
    String? sku,
    bool clearSelection = false,
    ProductReview? created,
    String? error,
  }) => ReviewFormState(
    loading: loading ?? this.loading,
    loadingMore: loadingMore ?? this.loadingMore,
    checkingOrder: checkingOrder ?? this.checkingOrder,
    pending: pending ?? this.pending,
    orders: orders ?? this.orders,
    page: page ?? this.page,
    total: total ?? this.total,
    pageSize: pageSize ?? this.pageSize,
    orderId: orderId ?? this.orderId,
    skus: clearSelection ? const [] : skus ?? this.skus,
    sku: clearSelection ? null : sku ?? this.sku,
    created: created,
    error: error,
  );
}

class ReviewFormCubit extends Cubit<ReviewFormState> {
  ReviewFormCubit(
    this._reviews,
    this._orders,
    this.product, {
    required this.mayCreate,
  }) : super(const ReviewFormState());
  final ReviewsRepository _reviews;
  final OrdersRepository _orders;
  final Product product;
  final bool Function() mayCreate;
  String? _key, _fingerprint;
  int _orderRevision = 0;
  Future<void> loadOrders({bool more = false}) async {
    if (!mayCreate() ||
        state.loading ||
        state.loadingMore ||
        state.pending ||
        more && !state.hasMore) {
      return;
    }
    final next = more ? state.page + 1 : 1;
    emit(state.copy(loading: !more, loadingMore: more));
    try {
      final result = await _orders.orderPage(
        status: 'DELIVERED',
        page: next,
        pageSize: more ? state.pageSize : 10,
      );
      if (isClosed) return;
      final merged = {
        if (more)
          for (final order in state.orders) order.id: order,
      };
      final count = merged.length;
      for (final order in result.items) {
        merged[order.id] = order;
      }
      if (result.page != next ||
          more && merged.length == count && merged.length < result.total) {
        throw const FormatException('No order pagination progress');
      }
      emit(
        state.copy(
          loading: false,
          loadingMore: false,
          orders: merged.values.toList(),
          page: result.page,
          total: result.total,
          pageSize: result.pageSize,
        ),
      );
    } catch (e) {
      if (!isClosed) {
        emit(state.copy(loading: false, loadingMore: false, error: _error(e)));
      }
    }
  }

  Future<void> selectOrder(String id) async {
    if (state.pending || !mayCreate() || id.trim().isEmpty) return;
    final revision = ++_orderRevision;
    emit(
      state.copy(orderId: id.trim(), checkingOrder: true, clearSelection: true),
    );
    try {
      final order = await _orders.order(id.trim());
      if (isClosed || revision != _orderRevision) return;
      if (order.id != id.trim()) {
        throw const FormatException('Order identifier mismatch');
      }
      final known = product.variants.map((v) => v.sku).toSet();
      final skus = order.status == OrderStatus.delivered
          ? order.items
                .map((i) => i.sku)
                .whereType<String>()
                .where(known.contains)
                .toSet()
                .toList()
          : <String>[];
      emit(
        state.copy(
          checkingOrder: false,
          skus: skus,
          sku: skus.length == 1 ? skus.single : null,
          error: skus.isEmpty
              ? 'Đơn này chưa có quy cách phù hợp đã giao. Hãy chọn đơn khác hoặc liên hệ hỗ trợ.'
              : null,
        ),
      );
    } catch (e) {
      if (!isClosed && revision == _orderRevision) {
        emit(state.copy(checkingOrder: false, error: _error(e)));
      }
    }
  }

  void selectSku(String value) {
    if (!state.pending && state.skus.contains(value)) {
      emit(state.copy(sku: value));
    }
  }

  Future<void> submit(int rating, String input) async {
    if (state.pending || state.checkingOrder || state.created != null) return;
    final comment = input.trim();
    if (!mayCreate() ||
        state.orderId == null ||
        state.sku == null ||
        !state.skus.contains(state.sku) ||
        rating < 1 ||
        rating > 5 ||
        comment.length < 10 ||
        comment.length > 1000) {
      emit(
        state.copy(
          error:
              'Chọn đơn/quy cách đã mua, mức 1–5 sao và nội dung 10–1000 ký tự.',
        ),
      );
      return;
    }
    final fingerprint = jsonEncode([state.orderId, state.sku, rating, comment]);
    if (_fingerprint != fingerprint) {
      _fingerprint = fingerprint;
      _key = const Uuid().v4();
    }
    emit(state.copy(pending: true));
    try {
      final review = await _reviews.create(
        ReviewDraft(
          productId: product.id,
          orderId: state.orderId!,
          sku: state.sku!,
          rating: rating,
          comment: comment,
        ),
        idempotencyKey: _key!,
      );
      if (!isClosed) emit(state.copy(pending: false, created: review));
    } catch (e) {
      if (!isClosed) emit(state.copy(pending: false, error: _error(e)));
    }
  }
}
