import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/core/models/money.dart';

void main() {
  group('Money', () {
    test('parses integer units without floating point loss', () {
      final money = Money.fromJson({
        'currency_code': 'VND',
        'units': '9007199254740993',
        'nanos': 0,
      });
      expect(money.units, BigInt.parse('9007199254740993'));
    });
    test('adds values with nanos carry', () {
      final result =
          Money(currencyCode: 'USD', units: BigInt.one, nanos: 700000000) +
          Money(currencyCode: 'USD', units: BigInt.two, nanos: 600000000);
      expect(
        result,
        Money(currencyCode: 'USD', units: BigInt.from(4), nanos: 300000000),
      );
    });
    test('formats VND', () => expect(Money.vnd(255000).format(), '255.000đ'));
  });
}
