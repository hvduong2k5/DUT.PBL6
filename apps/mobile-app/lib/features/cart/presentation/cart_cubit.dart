import 'package:equatable/equatable.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:mobile_app/features/cart/domain/cart_models.dart';
import 'package:mobile_app/features/cart/domain/cart_repository.dart';
import 'package:mobile_app/features/catalog/domain/catalog_models.dart';

enum CartStatus { initial, loading, success, failure }

class CartState extends Equatable {
  const CartState({
    this.status = CartStatus.initial,
    this.cart = const Cart(),
    this.message,
  });
  final CartStatus status;
  final Cart cart;
  final String? message;
  CartState copyWith({CartStatus? status, Cart? cart, String? message}) =>
      CartState(
        status: status ?? this.status,
        cart: cart ?? this.cart,
        message: message,
      );
  @override
  List<Object?> get props => [status, cart, message];
}

class CartCubit extends Cubit<CartState> {
  CartCubit(this._repository) : super(const CartState());
  final CartRepository _repository;
  Future<void> load() async {
    emit(state.copyWith(status: CartStatus.loading));
    try {
      emit(
        state.copyWith(
          status: CartStatus.success,
          cart: await _repository.getCart(),
        ),
      );
    } catch (_) {
      emit(
        state.copyWith(
          status: CartStatus.failure,
          message: 'Không thể tải giỏ hàng.',
        ),
      );
    }
  }

  Future<void> add(
    Product product,
    ProductVariant variant, {
    int quantity = 1,
  }) async {
    final cart = await _repository.add(product, variant, quantity);
    emit(
      CartState(
        status: CartStatus.success,
        cart: cart,
        message: 'Đã thêm vào giỏ hàng',
      ),
    );
  }

  Future<void> update(String itemId, int quantity) async => emit(
    CartState(
      status: CartStatus.success,
      cart: await _repository.update(itemId, quantity),
    ),
  );
  Future<void> remove(String itemId) async => emit(
    CartState(
      status: CartStatus.success,
      cart: await _repository.remove(itemId),
    ),
  );
  Future<void> clear() async => emit(
    CartState(status: CartStatus.success, cart: await _repository.clear()),
  );
}
