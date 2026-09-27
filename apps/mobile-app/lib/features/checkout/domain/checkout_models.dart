import 'package:equatable/equatable.dart';
import 'package:mobile_app/core/models/money.dart';
import 'package:mobile_app/features/cart/domain/cart_models.dart';

enum PaymentMethod { vietqr, cod }

extension PaymentMethodContract on PaymentMethod {
  String get apiValue => this == PaymentMethod.vietqr ? 'VIETQR' : 'COD';
  String get label =>
      this == PaymentMethod.vietqr ? 'VietQR' : 'Thanh toán khi nhận hàng';
}

class CheckoutDraft extends Equatable {
  const CheckoutDraft({
    required this.fullName,
    required this.phone,
    required this.streetAddress,
    required this.ward,
    required this.district,
    required this.province,
    required this.cart,
    this.paymentMethod = PaymentMethod.cod,
  });
  final String fullName;
  final String phone;
  final String streetAddress;
  final String ward;
  final String district;
  final String province;
  final Cart cart;
  final PaymentMethod paymentMethod;
  @override
  List<Object?> get props => [
    fullName,
    phone,
    streetAddress,
    ward,
    district,
    province,
    cart,
    paymentMethod,
  ];
}

class OrderConfirmation extends Equatable {
  const OrderConfirmation({
    required this.orderId,
    required this.status,
    required this.total,
  });
  final String orderId;
  final String status;
  final Money total;
  @override
  List<Object?> get props => [orderId, status, total];
}
