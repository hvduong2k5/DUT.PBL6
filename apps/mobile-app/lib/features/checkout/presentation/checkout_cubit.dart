import 'package:equatable/equatable.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:mobile_app/features/account/domain/address.dart';
import 'package:mobile_app/features/account/domain/address_repository.dart';
import 'package:mobile_app/features/cart/domain/cart_models.dart';
import 'package:mobile_app/features/checkout/domain/checkout_models.dart';
import 'package:mobile_app/features/checkout/domain/checkout_repository.dart';
import 'package:uuid/uuid.dart';

enum CheckoutStatus { editing, submitting, success, failure }

class CheckoutState extends Equatable {
  const CheckoutState({
    this.status = CheckoutStatus.editing,
    this.fullName = '',
    this.phone = '',
    this.streetAddress = '',
    this.ward = '',
    this.district = '',
    this.province = '',
    this.savedAddress,
    this.useAccountAddress = false,
    this.loadingSavedAddress = false,
    this.checkedSavedAddresses = false,
    this.paymentMethod = PaymentMethod.cod,
    this.fieldErrors = const {},
    this.message,
    this.confirmation,
  });
  final CheckoutStatus status;
  final String fullName;
  final String phone;
  final String streetAddress;
  final String ward;
  final String district;
  final String province;
  final Address? savedAddress;
  final bool useAccountAddress;
  final bool loadingSavedAddress;
  final bool checkedSavedAddresses;
  final PaymentMethod paymentMethod;
  final Map<String, String> fieldErrors;
  final String? message;
  final OrderConfirmation? confirmation;
  CheckoutState copyWith({
    CheckoutStatus? status,
    String? fullName,
    String? phone,
    String? streetAddress,
    String? ward,
    String? district,
    String? province,
    Address? savedAddress,
    bool? useAccountAddress,
    bool? loadingSavedAddress,
    bool? checkedSavedAddresses,
    PaymentMethod? paymentMethod,
    Map<String, String>? fieldErrors,
    String? message,
    OrderConfirmation? confirmation,
  }) => CheckoutState(
    status: status ?? this.status,
    fullName: fullName ?? this.fullName,
    phone: phone ?? this.phone,
    streetAddress: streetAddress ?? this.streetAddress,
    ward: ward ?? this.ward,
    district: district ?? this.district,
    province: province ?? this.province,
    savedAddress: savedAddress ?? this.savedAddress,
    useAccountAddress: useAccountAddress ?? this.useAccountAddress,
    loadingSavedAddress: loadingSavedAddress ?? this.loadingSavedAddress,
    checkedSavedAddresses: checkedSavedAddresses ?? this.checkedSavedAddresses,
    paymentMethod: paymentMethod ?? this.paymentMethod,
    fieldErrors: fieldErrors ?? this.fieldErrors,
    message: message,
    confirmation: confirmation ?? this.confirmation,
  );
  @override
  List<Object?> get props => [
    status,
    fullName,
    phone,
    streetAddress,
    ward,
    district,
    province,
    savedAddress,
    useAccountAddress,
    loadingSavedAddress,
    checkedSavedAddresses,
    paymentMethod,
    fieldErrors,
    message,
    confirmation,
  ];
}

class CheckoutCubit extends Cubit<CheckoutState> {
  CheckoutCubit(this._repository, this._cart, {String? idempotencyKey})
    : _idempotencyKey = idempotencyKey ?? const Uuid().v4(),
      super(const CheckoutState());
  final CheckoutRepository _repository;
  final Cart _cart;
  final String _idempotencyKey;
  void setName(String value) => emit(
    state.copyWith(
      fullName: value,
      fieldErrors: {...state.fieldErrors}..remove('fullName'),
    ),
  );
  void setPhone(String value) => emit(
    state.copyWith(
      phone: value,
      fieldErrors: {...state.fieldErrors}..remove('phone'),
    ),
  );
  void setStreetAddress(String value) => emit(
    state.copyWith(
      streetAddress: value,
      fieldErrors: {...state.fieldErrors}..remove('streetAddress'),
    ),
  );
  void setWard(String value) => emit(
    state.copyWith(
      ward: value,
      fieldErrors: {...state.fieldErrors}..remove('ward'),
    ),
  );
  void setDistrict(String value) => emit(
    state.copyWith(
      district: value,
      fieldErrors: {...state.fieldErrors}..remove('district'),
    ),
  );
  void setProvince(String value) => emit(
    state.copyWith(
      province: value,
      fieldErrors: {...state.fieldErrors}..remove('province'),
    ),
  );

  Future<void> loadSavedAddress(AddressRepository repository) async {
    emit(state.copyWith(loadingSavedAddress: true));
    try {
      final addresses = await repository.addresses();
      final saved =
          addresses.where((item) => item.isDefault).firstOrNull ??
          addresses.firstOrNull;
      emit(
        state.copyWith(
          loadingSavedAddress: false,
          checkedSavedAddresses: true,
          savedAddress: saved,
        ),
      );
    } catch (_) {
      emit(
        state.copyWith(loadingSavedAddress: false, checkedSavedAddresses: true),
      );
    }
  }

  void useSavedAddress(bool enabled) {
    final address = state.savedAddress;
    if (!enabled || address == null) {
      emit(state.copyWith(useAccountAddress: false));
      return;
    }
    emit(
      state.copyWith(
        useAccountAddress: true,
        fullName: address.recipientName,
        phone: address.phoneNumber,
        streetAddress: address.streetAddress,
        ward: address.ward,
        district: address.district,
        province: address.province,
        fieldErrors: const {},
      ),
    );
  }

  void setPayment(PaymentMethod value) =>
      emit(state.copyWith(paymentMethod: value));
  Map<String, String> validate() {
    final errors = <String, String>{};
    if (state.fullName.trim().length < 2) {
      errors['fullName'] = 'Vui lòng nhập họ và tên.';
    }
    if (!RegExp(r'^0\d{9}$').hasMatch(state.phone.trim())) {
      errors['phone'] = 'Số điện thoại gồm 10 chữ số và bắt đầu bằng 0.';
    }
    if (state.streetAddress.trim().length < 3) {
      errors['streetAddress'] = 'Vui lòng nhập số nhà và tên đường.';
    }
    if (state.ward.trim().isEmpty) {
      errors['ward'] = 'Vui lòng nhập phường/xã.';
    }
    if (state.district.trim().isEmpty) {
      errors['district'] = 'Vui lòng nhập quận/huyện.';
    }
    if (state.province.trim().isEmpty) {
      errors['province'] = 'Vui lòng nhập tỉnh/thành phố.';
    }
    return errors;
  }

  Future<void> submit() async {
    if (state.status == CheckoutStatus.submitting ||
        state.status == CheckoutStatus.success) {
      return;
    }
    final errors = validate();
    if (errors.isNotEmpty) {
      emit(
        state.copyWith(
          status: CheckoutStatus.editing,
          fieldErrors: errors,
          message: 'Vui lòng kiểm tra thông tin bắt buộc.',
        ),
      );
      return;
    }
    emit(
      state.copyWith(status: CheckoutStatus.submitting, fieldErrors: const {}),
    );
    try {
      final confirmation = await _repository.submit(
        CheckoutDraft(
          fullName: state.fullName.trim(),
          phone: state.phone.trim(),
          streetAddress: state.streetAddress.trim(),
          ward: state.ward.trim(),
          district: state.district.trim(),
          province: state.province.trim(),
          cart: _cart,
          paymentMethod: state.paymentMethod,
        ),
        idempotencyKey: _idempotencyKey,
      );
      emit(
        state.copyWith(
          status: CheckoutStatus.success,
          confirmation: confirmation,
        ),
      );
    } catch (_) {
      emit(
        state.copyWith(
          status: CheckoutStatus.failure,
          message: 'Không thể đặt hàng. Vui lòng thử lại.',
        ),
      );
    }
  }
}
