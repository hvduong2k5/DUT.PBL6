import 'package:mobile_app/core/config/app_config.dart';

abstract final class ApiConfig {
  static String get baseUrl => AppConfig.apiBaseUrl;

  static const connectTimeout = Duration(seconds: 10);
  static const receiveTimeout = Duration(seconds: 15);
}
