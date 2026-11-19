import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/features/checkout/domain/mvp_purchase_models.dart';
import 'package:mobile_app/features/checkout/domain/mvp_purchase_repository.dart';

enum MvpQuoteStatus {
  initial,
  loading,
  emptyCart,
  noAddress,
  editing,
  quoting,
  ready,
  failure,
}

class MvpQuoteState {
  const MvpQuoteState({
    this.status = MvpQuoteStatus.initial,
    this.cart,
    this.addresses = const [],
    this.addressId,
    this.quote,
    this.message,
  });
  final MvpQuoteStatus status;
  final MvpCart? cart;
  final List<MvpSavedAddress> addresses;
  final String? addressId, message;
  final MvpQuote? quote;
  bool usableAt(DateTime now) =>
      status == MvpQuoteStatus.ready && (quote?.isUsableAt(now) ?? false);
}

class MvpQuoteCubit extends Cubit<MvpQuoteState> {
  MvpQuoteCubit(this._repository, {DateTime Function()? now})
    : _now = now ?? DateTime.now,
      super(const MvpQuoteState());
  final MvpPurchaseRepository _repository;
  final DateTime Function() _now;
  int _revision = 0;

  Future<void> load() async {
    final revision = ++_revision;
    final selected = state.addressId;
    emit(const MvpQuoteState(status: MvpQuoteStatus.loading));
    try {
      final cart = await _repository.cart();
      if (isClosed || revision != _revision) return;
      if (cart.items.isEmpty) {
        emit(MvpQuoteState(status: MvpQuoteStatus.emptyCart, cart: cart));
        return;
      }
      final addresses = List<MvpSavedAddress>.unmodifiable(
        (await _repository.savedAddresses()).where((address) => address.usable),
      );
      if (isClosed || revision != _revision) return;
      final address =
          addresses.where((a) => a.id == selected).firstOrNull ??
          addresses.where((a) => a.isDefault).firstOrNull ??
          addresses.firstOrNull;
      emit(
        MvpQuoteState(
          status: address == null
              ? MvpQuoteStatus.noAddress
              : MvpQuoteStatus.editing,
          cart: cart,
          addresses: addresses,
          addressId: address?.id,
        ),
      );
    } catch (error) {
      if (!isClosed && revision == _revision) {
        emit(
          MvpQuoteState(
            status: MvpQuoteStatus.failure,
            message: _message(error),
          ),
        );
      }
    }
  }

  void selectAddress(String id) {
    if (!state.addresses.any((a) => a.id == id)) return;
    ++_revision;
    emit(
      MvpQuoteState(
        status: MvpQuoteStatus.editing,
        cart: state.cart,
        addresses: state.addresses,
        addressId: id,
      ),
    );
  }

  Future<void> refreshQuote() async {
    // Re-read cart and saved addresses before asking for a replacement quote.
    final revision = _revision + 1;
    await load();
    if (!isClosed &&
        revision == _revision &&
        state.status == MvpQuoteStatus.editing) {
      await requestQuote();
    }
  }

  Future<void> requestQuote() async {
    if (state.status == MvpQuoteStatus.quoting ||
        state.cart == null ||
        state.cart!.items.isEmpty ||
        state.addressId == null) {
      return;
    }
    final revision = ++_revision,
        cart = state.cart!,
        addressId = state.addressId!;
    final addresses = state.addresses;
    emit(
      MvpQuoteState(
        status: MvpQuoteStatus.quoting,
        cart: cart,
        addresses: addresses,
        addressId: addressId,
      ),
    );
    try {
      final quote = await _repository.quote(
        cartRevision: cart.revision,
        addressId: addressId,
      );
      if (isClosed || revision != _revision) return;
      if (quote.cartRevision != cart.revision ||
          quote.paymentMethod != 'VIETQR' ||
          quote.sourceAddressId != null && quote.sourceAddressId != addressId ||
          !quote.isUsableAt(_now())) {
        throw const FormatException('Expired or mismatched quote');
      }
      emit(
        MvpQuoteState(
          status: MvpQuoteStatus.ready,
          cart: cart,
          addresses: addresses,
          addressId: addressId,
          quote: quote,
        ),
      );
    } catch (error) {
      if (!isClosed && revision == _revision) {
        // Discard stale authority. Retry starts with load(), not a new order.
        emit(
          MvpQuoteState(
            status: MvpQuoteStatus.failure,
            message: _message(error),
          ),
        );
      }
    }
  }

  String _message(Object error) => error is ApiException
      ? error.failure.message
      : 'Chưa thể lấy báo giá hợp lệ. Vui lòng tải lại giỏ và địa chỉ.';
}
