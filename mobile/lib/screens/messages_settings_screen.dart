// mobile/lib/screens/messages_settings_screen.dart
import 'package:flutter/material.dart';

/// Экран «Сообщения».
/// Пока — заглушки: задержка отправки + TTL.
/// Реальная логика — перенос из chat_screen (следующий шаг).
class MessagesSettingsScreen extends StatelessWidget {
  const MessagesSettingsScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Сообщения'),
      ),
      body: ListView(
        children: const [
          ListTile(
            leading: Icon(Icons.hourglass_empty),
            title: Text('Задержка отправки'),
            subtitle: Text('Будет перенесено из чата'),
            enabled: false,
          ),
          ListTile(
            leading: Icon(Icons.timer_outlined),
            title: Text('Время удаления (TTL)'),
            subtitle: Text('Будет перенесено из чата'),
            enabled: false,
          ),
        ],
      ),
    );
  }
}
// mobile/lib/screens/messages_settings_screen.dart