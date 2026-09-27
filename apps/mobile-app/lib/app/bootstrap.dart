import 'package:flutter/widgets.dart';
import 'package:mobile_app/app/app.dart';
import 'package:mobile_app/core/storage/onboarding_store.dart';

Future<void> bootstrap() async {
  WidgetsFlutterBinding.ensureInitialized();
  final onboardingStore = OnboardingStore();
  var onboardingCompleted = false;
  try {
    onboardingCompleted = await onboardingStore.isCompleted();
  } catch (_) {
    // A storage failure must not block entry to the app.
  }
  runApp(
    OmaApp(
      onboardingStore: onboardingStore,
      onboardingCompleted: onboardingCompleted,
    ),
  );
}
