import 'package:mobile_app/features/notifications/domain/app_notification.dart';

/// Local boundary: the supplied OpenAPI/GraphQL contracts define no notification endpoint.
class LocalNotificationsRepository implements NotificationsRepository {
  @override
  Future<List<AppNotification>> notifications() async => const [
    AppNotification(
      id: '1',
      title: 'Đơn hàng đang được giao',
      message: 'Đơn #OM-8921 đã chuyển sang trạng thái đang giao.',
      timeLabel: '5 phút',
      unread: true,
    ),
    AppNotification(
      id: '2',
      title: 'Đơn hàng đã xác nhận',
      message: 'Đơn #OM-8921 đã được xác nhận và đang chuẩn bị.',
      timeLabel: '2 giờ',
      unread: true,
    ),
    AppNotification(
      id: '3',
      title: 'Gợi ý từ Gói Huế',
      message: 'Khám phá thêm các sản phẩm đặc sản Cố Đô trong bộ sưu tập.',
      timeLabel: 'Hôm qua',
    ),
  ];
}

/// Real-data mode: no notification endpoint exists in the MVP customer
/// contract yet, so the list stays empty instead of showing sample data.
class EmptyNotificationsRepository implements NotificationsRepository {
  const EmptyNotificationsRepository();
  @override
  Future<List<AppNotification>> notifications() async => const [];
}
