import 'package:dio/dio.dart';

/// The browser owns cookie transport; native cookie libraries require dart:io.
class SessionCookies {
  SessionCookies(Dio dio);
  Future<void> clear() async {}
}
