import 'package:equatable/equatable.dart';

enum PrincipalType { customer, employee, unknown }

enum ProvisioningStatus { active, suspended, unknown }

/// Identity projection; unknown authority never defaults to an active Customer.
class IdentityPersona extends Equatable {
  IdentityPersona({
    required this.principalId,
    required this.email,
    required this.type,
    required this.status,
    required List<Object?> roles,
    required List<String> permissions,
    this.firstName,
    this.lastName,
  }) : roles = List.unmodifiable(roles),
       permissions = List.unmodifiable(permissions) {
    if (principalId.trim().isEmpty ||
        email.trim().isEmpty ||
        type == PrincipalType.customer && roles.isNotEmpty) {
      throw const FormatException('Invalid Identity persona');
    }
  }
  factory IdentityPersona.fromJson(Map<String, dynamic> json) {
    String requiredText(String key) {
      final value = json[key];
      if (value is! String || value.trim().isEmpty) {
        throw const FormatException('Incomplete Identity persona');
      }
      return value;
    }

    String? optionalText(String key) {
      final value = json[key];
      if (value != null && value is! String) {
        throw const FormatException('Invalid Identity name');
      }
      return value as String?;
    }

    final roles = json['roles'], permissions = json['permissions'];
    if (roles is! List ||
        permissions is! List ||
        permissions.any((p) => p is! String)) {
      throw const FormatException('Invalid Identity authority');
    }
    return IdentityPersona(
      principalId: requiredText('principalId'),
      email: requiredText('email'),
      type: switch (requiredText('principalType')) {
        'CUSTOMER' => PrincipalType.customer,
        'EMPLOYEE' => PrincipalType.employee,
        _ => PrincipalType.unknown,
      },
      status: switch (requiredText('provisioningStatus')) {
        'ACTIVE' => ProvisioningStatus.active,
        'SUSPENDED' => ProvisioningStatus.suspended,
        _ => ProvisioningStatus.unknown,
      },
      firstName: optionalText('firstName'),
      lastName: optionalText('lastName'),
      roles: roles,
      permissions: permissions.cast<String>(),
    );
  }
  final String principalId, email;
  final String? firstName, lastName;
  final PrincipalType type;
  final ProvisioningStatus status;
  final List<Object?> roles;
  final List<String> permissions;
  bool get isEmployee => type == PrincipalType.employee;
  bool get canPurchase =>
      type == PrincipalType.customer &&
      status == ProvisioningStatus.active &&
      roles.isEmpty;
  String get displayName => [
    firstName,
    lastName,
  ].whereType<String>().where((s) => s.trim().isNotEmpty).join(' ').trim();
  @override
  List<Object?> get props => [
    principalId,
    email,
    firstName,
    lastName,
    type,
    status,
    roles,
    permissions,
  ];
}

const employeePurchaseMessage =
    'Tài khoản nhân viên công vụ không được phép mua hàng cá nhân. Vui lòng đăng xuất và đăng nhập bằng tài khoản Khách hàng.';
const inactivePurchaseMessage =
    'Tài khoản chưa đủ điều kiện mua hàng. Vui lòng kiểm tra trạng thái tài khoản hoặc liên hệ hỗ trợ.';
