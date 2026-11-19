import 'package:equatable/equatable.dart';
import 'package:mobile_app/features/auth/domain/auth_models.dart';

class CustomerProfile extends Equatable {
  const CustomerProfile({
    required this.user,
    required this.tier,
    required this.points,
    this.avatarUrl = '',
  });
  factory CustomerProfile.fromJson(Map<String, dynamic> json) =>
      CustomerProfile(
        user: AuthUser.fromJson(json),
        tier: json['membership_tier'] as String,
        points: json['loyalty_points'] as int,
        avatarUrl: json['avatar_url'] as String,
      );
  final AuthUser user;
  final String tier;
  final int points;
  final String avatarUrl;
  @override
  List<Object?> get props => [user, tier, points, avatarUrl];
}

class LoyaltyOverview extends Equatable {
  const LoyaltyOverview({
    required this.customerId,
    required this.points,
    required this.tier,
    required this.vndRate,
  });
  factory LoyaltyOverview.fromJson(Map<String, dynamic> json) =>
      LoyaltyOverview(
        customerId: json['customer_id'] as String,
        points: json['current_points'] as int,
        tier: json['tier'] as String,
        vndRate: json['points_to_vnd_rate'] as int,
      );
  final String customerId;
  final int points;
  final String tier;
  final int vndRate;
  String get tierLabel => switch (tier) {
    'BRONZE' => 'Đồng',
    'SILVER' => 'Bạc',
    'GOLD' => 'Vàng',
    'DIAMOND' => 'Kim Cương',
    _ => tier,
  };
  @override
  List<Object?> get props => [customerId, points, tier, vndRate];
}

abstract interface class CustomerAccountRepository {
  Future<CustomerProfile> profile();
  Future<CustomerProfile> updateProfile(String fullName, String email);
  Future<LoyaltyOverview> loyalty();
}
