import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/features/notifications/data/local_notifications_repository.dart';
import 'package:mobile_app/features/notifications/presentation/notifications_cubit.dart';

void main() {
  test(
    'local notification boundary supplies the designed default state',
    () async {
      final cubit = NotificationsCubit(LocalNotificationsRepository());
      await cubit.load();
      expect(cubit.state.loading, isFalse);
      expect(cubit.state.items, isNotEmpty);
      expect(cubit.state.items.any((item) => item.unread), isTrue);
      await cubit.close();
    },
  );
}
