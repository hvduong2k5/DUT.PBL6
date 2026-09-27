import 'package:equatable/equatable.dart';

class Address extends Equatable {
  const Address({
    required this.id,
    required this.recipientName,
    required this.phoneNumber,
    required this.streetAddress,
    required this.ward,
    required this.district,
    required this.province,
    this.isDefault = false,
  });

  final String id;
  final String recipientName;
  final String phoneNumber;
  final String streetAddress;
  final String ward;
  final String district;
  final String province;
  final bool isDefault;

  factory Address.fromApiJson(Map<String, dynamic> json) => Address(
    id: json['id']?.toString() ?? '',
    recipientName: json['recipient_name']?.toString() ?? '',
    phoneNumber: json['phone_number']?.toString() ?? '',
    streetAddress: json['street_address']?.toString() ?? '',
    ward: json['ward']?.toString() ?? '',
    district: json['district']?.toString() ?? '',
    province: json['province']?.toString() ?? '',
    isDefault: json['is_default'] as bool? ?? false,
  );

  String get formatted => '$streetAddress, $ward, $district, $province';

  Address copyWith({bool? isDefault}) => Address(
    id: id,
    recipientName: recipientName,
    phoneNumber: phoneNumber,
    streetAddress: streetAddress,
    ward: ward,
    district: district,
    province: province,
    isDefault: isDefault ?? this.isDefault,
  );

  Map<String, dynamic> toApiJson() => {
    'recipient_name': recipientName,
    'phone_number': phoneNumber,
    'street_address': streetAddress,
    'ward': ward,
    'district': district,
    'province': province,
    'is_default': isDefault,
  };

  @override
  List<Object?> get props => [
    id,
    recipientName,
    phoneNumber,
    streetAddress,
    ward,
    district,
    province,
    isDefault,
  ];
}
