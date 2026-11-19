class CustomerCapabilities {
  const CustomerCapabilities({
    required this.actor,
    required this.values,
    required this.version,
  });
  final String actor;
  final Set<String> values;
  final int version;
  bool allows(String capability) => values.contains(capability);
  factory CustomerCapabilities.fromJson(Map<String, dynamic> json) {
    final actor = json['actor'] as String;
    if (!const {
      'GUEST',
      'REGISTERED',
      'B2B',
      'MARKETPLACE',
      'OFFLINE',
    }.contains(actor)) {
      throw const FormatException('Unknown customer actor');
    }
    return CustomerCapabilities(
      actor: actor,
      values: (json['capabilities'] as List).cast<String>().toSet(),
      version: json['version'] as int,
    );
  }
}

abstract interface class CustomerCapabilitiesRepository {
  Future<CustomerCapabilities> load();
}
