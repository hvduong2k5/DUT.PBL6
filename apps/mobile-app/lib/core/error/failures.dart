import 'package:equatable/equatable.dart';

enum FailureKind {
  validation,
  unauthenticated,
  forbidden,
  notFound,
  conflict,
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
  });
  final FailureKind kind;
  final String message;
  final String? code;
  final String? domain;
  final Map<String, String> fieldErrors;

  @override
  List<Object?> get props => [kind, message, code, domain, fieldErrors];
}
