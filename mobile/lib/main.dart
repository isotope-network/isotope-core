// mobile/lib/main.dart
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'providers/chat_provider.dart';
import 'services/p2p_service.dart';
import 'services/log_service.dart';
import 'services/media_storage.dart';
import 'screens/connect_screen.dart';
import 'screens/onboarding_screen.dart';

const String FIRST_LAUNCH_KEY = 'first_launch_seen';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();
  await _initLogService();
  await MediaStorage.init();
  MediaStorage.cleanupOld();
  runApp(const IsoApp());
}

Future<void> _initLogService() async {
  try {
    const channel = MethodChannel('isotope/libp2p');
    final filesDir = await channel.invokeMethod<String>('getFilesDir');
    if (filesDir != null && filesDir.isNotEmpty) {
      await LogService.init(filesDir);
    } else {
      debugPrint('main: filesDir is empty, using default');
      await LogService.init('.');
    }
  } catch (e) {
    debugPrint('main: error initializing LogService: $e');
    await LogService.init('.');
  }
}

class IsoApp extends StatelessWidget {
  const IsoApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MultiProvider(
      providers: [
        ChangeNotifierProvider(create: (_) => ChatProvider()),
        Provider<P2PService>(create: (_) => P2PService()),
      ],
      child: MaterialApp(
        title: 'ИСО',
        debugShowCheckedModeBanner: false,
        theme: ThemeData(
          useMaterial3: true,
          colorSchemeSeed: Colors.green,
          appBarTheme: const AppBarTheme(
            backgroundColor: Colors.green,
            foregroundColor: Colors.white,
            elevation: 1,
          ),
        ),
        home: const _RootRouter(),
      ),
    );
  }
}

/// _RootRouter — решает, что показать: onboarding или ConnectScreen.
/// Читает first_launch_seen из SharedPreferences.
class _RootRouter extends StatefulWidget {
  const _RootRouter();

  @override
  State<_RootRouter> createState() => _RootRouterState();
}

class _RootRouterState extends State<_RootRouter> {
  bool? _firstLaunchSeen;

  @override
  void initState() {
    super.initState();
    _loadFlag();
  }

  Future<void> _loadFlag() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final seen = prefs.getBool(FIRST_LAUNCH_KEY) ?? false;
      if (mounted) {
        setState(() => _firstLaunchSeen = seen);
      }
    } catch (_) {
      // Ошибка чтения — считаем, что onboarding пройден.
      if (mounted) {
        setState(() => _firstLaunchSeen = true);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    if (_firstLaunchSeen == null) {
      // Пока читаем флаг — пустой экран (или splash).
      return const Scaffold(body: SizedBox.shrink());
    }
    if (_firstLaunchSeen == false) {
      return const OnboardingScreen();
    }
    return const ConnectScreen();
  }
}
// mobile/lib/main.dart