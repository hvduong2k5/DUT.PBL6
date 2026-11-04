import 'package:equatable/equatable.dart';

enum FailureKind {
  validation,
  unauthenticated,
  forbidden,
  notFound,
  conflict,
  expired,
  rateLimited,
  server,
  network,
  timeout,
  unknown,
}

class Failure extends Equatable {
  const Failure({
    required this.kind,
    required this.message,
    this.code,
    this.domain,
    this.fieldErrors = const {},
    this.requestId,
    this.retryAfterSeconds,
    this.operationId,
  });
  final FailureKind kind;
  final String message;
  final String? code;
  final String? domain;
  final Map<String, String> fieldErrors;
  final String? requestId;
  final int? retryAfterSeconds;
  final String? operationId;

  @override
  List<Object?> get props => [
    kind,
    message,
    code,
    domain,
    fieldErrors,
    requestId,
    retryAfterSeconds,
    operationId,
  ];
}
