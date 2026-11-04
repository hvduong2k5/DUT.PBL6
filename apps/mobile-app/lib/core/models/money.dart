import 'package:equatable/equatable.dart';

class Money extends Equatable implements Comparable<Money> {
  const Money({
    required this.currencyCode,
    required this.units,
    this.nanos = 0,
  });
  factory Money.fromJson(Map<String, dynamic> json) => Money(
    currencyCode: json['currency_code'] as String? ?? 'VND',
    units: BigInt.parse(json['units'].toString()),
    nanos: (json['nanos'] as num?)?.toInt() ?? 0,
  );
  factory Money.vnd(int units) =>
      Money(currencyCode: 'VND', units: BigInt.from(units));
  final String currencyCode;
  final BigInt units;
  final int nanos;

  Money operator +(Money other) {
    _ensureCurrency(other);
    final totalNanos = nanos + other.nanos;
    return Money(
      currencyCode: currencyCode,
      units: units + other.units + BigInt.from(totalNanos ~/ 1000000000),
      nanos: totalNanos % 1000000000,
    );
  }

  Money times(int quantity) => Money(
    currencyCode: currencyCode,
    units: units * BigInt.from(quantity),
    nanos: nanos * quantity,
  );
  String format({String locale = 'vi_VN'}) {
    if (currencyCode == 'VND' && nanos == 0) {
      return '${_groupThousands(units)}đ';
    }
    final fraction = nanos
        .toString()
        .padLeft(9, '0')
        .replaceFirst(RegExp(r'0+$'), '');
    return fraction.isEmpty
        ? '$units $currencyCode'
        : '$units.$fraction $currencyCode';
  }

  @override
  int compareTo(Money other) {
    _ensureCurrency(other);
    final result = units.compareTo(other.units);
    return result == 0 ? nanos.compareTo(other.nanos) : result;
  }

  void _ensureCurrency(Money other) {
    if (currencyCode != other.currencyCode) {
      throw ArgumentError('Cannot operate on different currencies.');
    }
  }

  static String _groupThousands(BigInt value) {
    final negative = value.isNegative;
    final digits = value.abs().toString();
    final firstGroup = digits.length % 3 == 0 ? 3 : digits.length % 3;
    final parts = <String>[digits.substring(0, firstGroup)];
    for (var index = firstGroup; index < digits.length; index += 3) {
      parts.add(digits.substring(index, index + 3));
    }
    return '${negative ? '-' : ''}${parts.join('.')}';
  }

  @override
  List<Object?> get props => [currencyCode, units, nanos];
}
