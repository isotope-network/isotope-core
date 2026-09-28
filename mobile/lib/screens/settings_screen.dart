// mobile/lib/screens/settings_screen.dart
import 'package:flutter/material.dart';
import 'privacy_settings_screen.dart';
import 'messages_settings_screen.dart';

/// Экран «Настройки» — список разделов.
/// Личные — выше Системных (пользовательское важнее).
class SettingsScreen extends StatelessWidget {
  const SettingsScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Настройки'),
      ),
      body: ListView(
        children: [
          // ==== ЛИЧНЫЕ ====
          const _SectionHeader('Личные'),
          ListTile(
            leading: const Icon(Icons.lock_outline),
            title: const Text('Приватность'),
            subtitle: const Text('Кто видит ваши статусы'),
            trailing: const Icon(Icons.chevron_right),
            onTap: () {
              Navigator.push(
                context,
                MaterialPageRoute(builder: (_) => const PrivacySettingsScreen()),
              );
            },
          ),
          ListTile(
            leading: const Icon(Icons.message_outlined),
            title: const Text('Сообщения'),
            subtitle: const Text('Задержка, время удаления'),
            trailing: const Icon(Icons.chevron_right),
            onTap: () {
              Navigator.push(
                context,
                MaterialPageRoute(builder: (_) => const MessagesSettingsScreen()),
              );
            },
          ),

          const Divider(height: 1),

          // ==== СИСТЕМНЫЕ ====
          const _SectionHeader('Системные'),
          // TODO: Подключение (bootstrap) — перенести из connect_screen.
          // TODO: Ввести код контакта — перенести из connect_screen.
          // Пока — заглушки. Реальная логика — в connect_screen (bottom sheet).
          const ListTile(
            leading: Icon(Icons.link),
            title: Text('Подключение'),
            subtitle: Text('Адрес подключения'),
            enabled: false,
          ),
          const ListTile(
            leading: Icon(Icons.input),
            title: Text('Ввести код контакта'),
            subtitle: Text('Для продвинутых'),
            enabled: false,
          ),

          const SizedBox(height: 16),
        ],
      ),
    );
  }
}

/// Заголовок раздела.
class _SectionHeader extends StatelessWidget {
  final String title;
  const _SectionHeader(this.title);

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
      child: Text(
        title.toUpperCase(),
        style: TextStyle(
          fontSize: 12,
          fontWeight: FontWeight.bold,
          color: Colors.grey.shade600,
          letterSpacing: 0.5,
        ),
      ),
    );
  }
}
// mobile/lib/screens/settings_screen.dart