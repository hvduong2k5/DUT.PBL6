import 'package:mobile_app/core/config/app_config.dart';
import 'package:mobile_app/core/config/data_source_mode.dart';

/// Customer features whose public API is not yet in the MVP customer contract
/// (`docs/03_api_specs/mvp-customer.openapi.yaml`) or routed by the gateway
/// (`services/api-gateway/config/kong.yml`).
///
/// With real data (`DATA_SOURCE=backend`/`mockoon`) they are off unless turned
/// on with `--dart-define=FEATURE_<NAME>=true` once the backend ships them.
/// Mock mode keeps them on for local UI work and tests.
enum AppFeature {
  promotions('Ưu đãi'),
  loyalty('Ví điểm & đặc quyền'),
  returns('Đổi trả & hậu mãi'),
  reviews('Đánh giá sản phẩm'),
  support('Hỗ trợ khách hàng'),
  trace('Truy xuất nguồn gốc'),
  b2b('Quà biếu doanh nghiệp'),
  guestOrderLookup('Tra cứu đơn khách');

  const AppFeature(this.label);
  final String label;
}

class AppFeatures {
  const AppFeatures.all() : _enabled = null;
  const AppFeatures.only(Set<AppFeature> enabled) : _enabled = enabled;

  /// `null` means every feature is on.
  final Set<AppFeature>? _enabled;

  bool enabled(AppFeature feature) =>
      _enabled == null || _enabled.contains(feature);

  static const _defines = <AppFeature, bool>{
    AppFeature.promotions: bool.fromEnvironment('FEATURE_PROMOTIONS'),
    AppFeature.loyalty: bool.fromEnvironment('FEATURE_LOYALTY'),
    AppFeature.returns: bool.fromEnvironment('FEATURE_RETURNS'),
    AppFeature.reviews: bool.fromEnvironment('FEATURE_REVIEWS'),
    AppFeature.support: bool.fromEnvironment('FEATURE_SUPPORT'),
    AppFeature.trace: bool.fromEnvironment('FEATURE_TRACE'),
    AppFeature.b2b: bool.fromEnvironment('FEATURE_B2B'),
    AppFeature.guestOrderLookup: bool.fromEnvironment(
      'FEATURE_GUEST_ORDER_LOOKUP',
    ),
  };

  static AppFeatures resolve(
    DataSourceMode mode, {
    Map<AppFeature, bool> defines = _defines,
  }) => mode.isRemote
      ? AppFeatures.only({
          for (final entry in defines.entries)
            if (entry.value) entry.key,
        })
      : const AppFeatures.all();

  static AppFeatures current = resolve(AppConfig.dataSourceMode);

  /// Which gated feature owns a route path, if any.
  static AppFeature? forPath(String path) {
    bool under(String prefix) => path == prefix || path.startsWith('$prefix/');
    if (under('/promotions')) return AppFeature.promotions;
    if (under('/account/loyalty')) return AppFeature.loyalty;
    if (under('/after-sales') ||
        RegExp(r'^/orders/[^/]+/after-sales(/|$)').hasMatch(path)) {
      return AppFeature.returns;
    }
    if (under('/support')) return AppFeature.support;
    if (under('/trace')) return AppFeature.trace;
    if (under('/b2b')) return AppFeature.b2b;
    if (under('/track-order')) return AppFeature.guestOrderLookup;
    return null;
  }
}
