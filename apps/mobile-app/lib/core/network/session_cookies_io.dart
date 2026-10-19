import 'package:cookie_jar/cookie_jar.dart';
import 'package:dio/dio.dart';
import 'package:dio_cookie_manager/dio_cookie_manager.dart';

class SessionCookies {
  SessionCookies(Dio dio) {
    dio.options.followRedirects = false;
    dio.interceptors.add(CookieManager(_jar));
  }
  final CookieJar _jar = CookieJar();
  Future<void> clear() => _jar.deleteAll();
}
