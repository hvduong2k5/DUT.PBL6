import 'package:mobile_app/core/error/failures.dart';

class ApiException implements Exception {
  const ApiException(this.failure);
  final Failure failure;
  @override
  String toString() => failure.message;
}
