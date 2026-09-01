import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'providers/chat_provider.dart';
import 'services/p2p_service.dart';
import 'screens/connect_screen.dart';

void main() {
  runApp(const IsoApp());
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