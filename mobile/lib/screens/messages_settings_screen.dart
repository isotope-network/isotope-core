// mobile/lib/screens/messages_settings_screen.dart
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../providers/chat_provider.dart';

/// Экран «Сообщения».
/// Пока — задержка отправки. TTL — заглушка (перенесём из chat_screen).
class MessagesSettingsScreen extends StatelessWidget {
  const MessagesSettingsScreen({super.key});

  static const List<int> _delayOptions = [0, 3, 5, 10];

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Сообщения'),
      ),
      body: Consumer<ChatProvider>(
        builder: (_, provider, __) {
          return ListView(
            children: [
              const Padding(
                padding: EdgeInsets.fromLTRB(16, 16, 16, 8),
                child: Text(
                  'ЗАДЕРЖКА ОТПРАВКИ',
                  style: TextStyle(
                    fontSize: 12,
                    fontWeight: FontWeight.bold,
                    color: Colors.grey,
                    letterSpacing: 0.5,
                  ),
                ),
              ),
              const Padding(
                padding: EdgeInsets.symmetric(horizontal: 16),
                child: Text(
                  'Даёт возможность передумать. Сообщение отправляется через выбранное время.',
                  style: TextStyle(fontSize: 13, color: Colors.grey),
                ),
              ),
              ..._delayOptions.map((sec) {
                final label = sec == 0 ? 'Без задержки' : '$sec секунд';
                final isSelected = provider.sendDelay == sec;
                return RadioListTile<int>(
                  title: Text(label),
                  value: sec,
                  // ignore: deprecated_member_use
                  groupValue: provider.sendDelay,
                  // ignore: deprecated_member_use
                  onChanged: (v) {
                    if (v != null) provider.setSendDelay(v);
                  },
                  selected: isSelected,
                );
              }),
              const Divider(),
              const Padding(
                padding: EdgeInsets.fromLTRB(16, 16, 16, 8),
                child: Text(
                  'ВРЕМЯ УДАЛЕНИЯ (TTL)',
                  style: TextStyle(
                    fontSize: 12,
                    fontWeight: FontWeight.bold,
                    color: Colors.grey,
                    letterSpacing: 0.5,
                  ),
                ),
              ),
              const ListTile(
                leading: Icon(Icons.timer_outlined),
                title: Text('Время удаления'),
                subtitle: Text('Скоро — перенос из чата'),
                enabled: false,
              ),
            ],
          );
        },
      ),
    );
  }
}
// mobile/lib/screens/messages_settings_screen.dart