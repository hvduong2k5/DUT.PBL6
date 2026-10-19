class RecoveryVerification {
  const RecoveryVerification({
    required this.resetProof,
    required this.expiresAt,
  });
  final String resetProof;
  final DateTime expiresAt;
}
