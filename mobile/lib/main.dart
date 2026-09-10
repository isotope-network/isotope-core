import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';
import 'providers/chat_provider.dart';
import 'services/p2p_service.dart';
import 'services/log_service.dart';
import 'screens/connect_screen.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();
  await _initLogService();
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
        home: const ConnectScreen(),
      ),
    );
  }
}