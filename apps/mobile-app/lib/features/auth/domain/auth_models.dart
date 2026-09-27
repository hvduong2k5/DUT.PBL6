import 'package:equatable/equatable.dart';

class AuthUser extends Equatable {
  const AuthUser({
    required this.id,
    required this.phoneNumber,
    required this.fullName,
    this.email,
  });
  final String id;
  final String phoneNumber;
  final String fullName;
  final String? email;

  factory AuthUser.fromJson(Map<String, dynamic> json) => AuthUser(
    id: json['customer_id'] as String? ?? '',
    phoneNumber: json['phone_number'] as String? ?? '',
    fullName: json['full_name'] as String? ?? '',
    email: json['email'] as String?,
  );
  @override
  List<Object?> get props => [id, phoneNumber, fullName, email];
}

class AuthSession extends Equatable {
  const AuthSession({
    required this.user,
    required this.accessToken,
    required this.refreshToken,
  });
  final AuthUser user;
  final String accessToken;
  final String refreshToken;
  @override
  List<Object?> get props => [user, accessToken, refreshToken];
}
