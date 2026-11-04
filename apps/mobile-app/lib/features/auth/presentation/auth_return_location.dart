import 'package:mobile_app/app/router/routes.dart';

/// Keep auth navigation within known Customer routes, never an external URL.
String safeAuthReturnLocation(String? value) {
  final uri = value == null ? null : Uri.tryParse(value);
  if (uri == null ||
      uri.hasScheme ||
      uri.hasAuthority ||
      !uri.path.startsWith('/') ||
      uri.path.startsWith('//') ||
      uri.path.contains('\\') ||
      uri.path.contains('..')) {
    return AppRoutes.account;
  }
  const routes = {
    '/',
    '/account',
    '/catalog',
    '/cart',
    '/checkout',
    '/checkout/quote',
    '/checkout/operation',
    '/orders',
    '/addresses',
    '/addresses/form',
    '/notifications',
    '/promotions',
    '/account/profile',
    '/account/loyalty',
    '/track-order',
    '/after-sales',
    '/support/request',
    '/trace',
    '/b2b',
  };
  final path = uri.path;
  if (!routes.contains(path) &&
      !RegExp(
        r'^/(product|orders|order-confirmation|payment)/[^/]+$',
      ).hasMatch(path) &&
      !RegExp(
        r'^/orders/[^/]+/(tracking|after-sales/request)$',
      ).hasMatch(path) &&
      !RegExp(
        r'^/(after-sales|support/tickets|trace|b2b/quotes)/[^/]+$',
      ).hasMatch(path)) {
    return AppRoutes.account;
  }
  return uri.toString();
}

String authLocation(String path, String returnLocation) =>
    Uri(path: path, queryParameters: {'returnUrl': returnLocation}).toString();
