import 'dart:async';
import 'package:equatable/equatable.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:mobile_app/features/cart/domain/cart_models.dart';
import 'package:mobile_app/features/cart/domain/cart_repository.dart';
import 'package:mobile_app/features/catalog/domain/catalog_models.dart';
import 'package:mobile_app/core/network/api_exception.dart';

enum CartStatus { initial, loading, success, failure }

class CartState extends Equatable {
  const CartState({
    this.status = CartStatus.initial,
    this.cart = const Cart(),
    this.message,
    this.selectedItemIds,
    this.pending = false,
    this.cleanupFailed = false,
  });
  final CartStatus status;
  final Cart cart;
  final String? message;
  final Set<String>? selectedItemIds;
  final bool pending;
  final bool cleanupFailed;
  Set<String> get availableIds => cart.items
      .where((line) => line.product.inStock && line.variant.stock > 0)
      .map((line) => line.itemId)
      .toSet();
  Set<String> get selectedIds =>
      (selectedItemIds ?? availableIds).intersection(availableIds);
  Cart get selectedCart => Cart(
    id: cart.id,
    items: cart.items
        .where((line) => selectedIds.contains(line.itemId))
        .toList(),
  );
  CartState copyWith({
    CartStatus? status,
    Cart? cart,
    String? message,
    Set<String>? selectedItemIds,
    bool pending = false,
    bool cleanupFailed = false,
  }) => CartState(
    status: status ?? this.status,
    cart: cart ?? this.cart,
    message: message,
    selectedItemIds: selectedItemIds ?? this.selectedItemIds,
    pending: pending,
    cleanupFailed: cleanupFailed,
  );
  @override
  List<Object?> get props => [
    status,
    cart,
    message,
    selectedItemIds,
    pending,
    cleanupFailed,
  ];
}

class CartCubit extends Cubit<CartState> {
  CartCubit(this._repository, {this._principal})
    : super(const CartState());
  final CartRepository _repository;
  final String? Function()? _principal;
  int _generation = 0;
  bool _current(int generation, String? owner) =>
      !isClosed && generation == _generation && owner == _principal?.call();
  void identityChanged() {
    if (isClosed) return;
    ++_generation;
    emit(const CartState());
    unawaited(load());
  }

  void toggle(String itemId) {
    if (state.pending || !state.availableIds.contains(itemId)) return;
    final ids = {...state.selectedIds};
    if (!ids.add(itemId)) ids.remove(itemId);
    emit(state.copyWith(selectedItemIds: ids));
  }

  void toggleAll() {
    if (state.pending) return;
    emit(
      state.copyWith(
        selectedItemIds: state.selectedIds.length == state.availableIds.length
            ? <String>{}
            : state.availableIds,
      ),
    );
  }

  Future<void> load() async {
    if (isClosed || state.pending) return;
    final generation = ++_generation, owner = _principal?.call();
    emit(state.copyWith(status: CartStatus.loading, pending: true));
    try {
      final cart = await _repository.getCart();
      if (!_current(generation, owner)) return;
      emit(state.copyWith(status: CartStatus.success, cart: cart));
    } catch (_) {
      if (!_current(generation, owner)) return;
      emit(
        state.copyWith(
          status: CartStatus.failure,
          message: 'Không thể tải giỏ hàng.',
        ),
      );
    }
  }

  Future<bool> add(
    Product product,
    ProductVariant variant, {
    int quantity = 1,
  }) async {
    if (state.pending) return false;
    final existing =
        state.cart.items
            .where((line) => line.variant.sku == variant.sku)
            .firstOrNull
            ?.quantity ??
        0;
    if (quantity < 1 ||
        quantity + existing > maxCartItemQuantity ||
        !product.inStock ||
        variant.stock < 1) {
      emit(
        state.copyWith(
          message:
              'Chỉ có thể chọn từ 1 đến $maxCartItemQuantity sản phẩm cho mỗi quy cách khả dụng.',
        ),
      );
      return false;
    }
    return _mutate(
      () => _repository.add(product, variant, quantity),
      selectNew: true,
    );
  }

  Future<bool> update(String itemId, int quantity) async {
    if (quantity < 1 || quantity > maxCartItemQuantity) return false;
    return _mutate(() => _repository.update(itemId, quantity));
  }

  Future<bool> remove(String itemId) =>
      _mutate(() => _repository.remove(itemId));
  Future<bool> clear() => _mutate(_repository.clear);

  /// Remove only purchased lines. Creation success must never be retried for cleanup.
  Future<bool> consume(Set<String> purchasedIds) async {
    final generation = _generation + 1, owner = _principal?.call();
    final success = await _mutate(() async {
      var cart = state.cart;
      for (final id in purchasedIds) {
        cart = await _repository.remove(id);
      }
      return cart;
    });
    if (!success && _current(generation, owner)) {
      emit(
        state.copyWith(
          selectedItemIds: state.selectedIds.difference(purchasedIds),
          cleanupFailed: true,
          message:
              'Đơn đã được tạo nhưng giỏ chưa đồng bộ. Hãy kiểm tra đơn trước khi đặt lại.',
        ),
      );
    }
    return success;
  }

  Future<bool> _mutate(
    Future<Cart> Function() operation, {
    bool selectNew = false,
  }) async {
    if (isClosed || state.pending) return false;
    final generation = ++_generation, owner = _principal?.call();
    final selected = state.selectedIds;
    final previousIds = state.cart.items.map((line) => line.itemId).toSet();
    emit(state.copyWith(pending: true));
    try {
      final cart = await operation();
      if (!_current(generation, owner)) return false;
      emit(
        state.copyWith(
          status: CartStatus.success,
          cart: cart,
          selectedItemIds: {
            ...selected,
            if (selectNew)
              ...cart.items
                  .map((line) => line.itemId)
                  .where((id) => !previousIds.contains(id)),
          },
        ),
      );
      return true;
    } catch (error) {
      if (_current(generation, owner)) {
        emit(
          state.copyWith(
            status: CartStatus.success,
            message: error is ApiException
                ? error.failure.message
                : 'Chưa thể cập nhật giỏ hàng. Vui lòng thử lại.',
          ),
        );
      }
      return false;
    }
  }
}
