// mobile/lib/screens/settings_screen.dart
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../providers/locale_provider.dart';
import '../l10n/app_localizations.dart';
import 'privacy_settings_screen.dart';
import 'messages_settings_screen.dart';
import 'profile_screen.dart';

/// Экран «Настройки» — список разделов.
/// Личные — выше Системных (пользовательское важнее).
class SettingsScreen extends StatelessWidget {
  /// Колбэк «Подключение» — открывает диалог bootstrap.
  /// Передаётся из ConnectScreen (там — контроллеры и логика).
  final VoidCallback? onConnection;

  /// Колбэк «Ввести код контакта» — открывает диалог ручного ввода.
  final VoidCallback? onManualCode;

  const SettingsScreen({
    super.key,
    this.onConnection,
    this.onManualCode,
  });

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(AppLocalizations.of(context).settingsTitle),
      ),
      body: ListView(
        children: [
          // ==== ЛИЧНЫЕ ====
          _SectionHeader(AppLocalizations.of(context).settingsPersonal),
          ListTile(
            leading: const Icon(Icons.person_outline),
            title: Text(AppLocalizations.of(context).settingsProfile),
            subtitle: Text(AppLocalizations.of(context).settingsProfileSub),
            trailing: const Icon(Icons.chevron_right),
            onTap: () {
              Navigator.push(
                context,
                MaterialPageRoute(builder: (_) => const ProfileScreen()),
              );
            },
          ),
          ListTile(
            leading: const Icon(Icons.lock_outline),
            title: Text(AppLocalizations.of(context).privacy),
            subtitle: Text(AppLocalizations.of(context).privacySubtitle),
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
            title: Text(AppLocalizations.of(context).settingsMessages),
            subtitle: Text(AppLocalizations.of(context).settingsMessagesSub),
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
          _SectionHeader(AppLocalizations.of(context).settingsSystem),
          ListTile(
            leading: const Icon(Icons.language),
            title: Text(AppLocalizations.of(context).language),
            subtitle: Text(_currentLanguageName(context)),
            trailing: const Icon(Icons.chevron_right),
            onTap: () => _showLanguageDialog(context),
          ),
          ListTile(
            leading: const Icon(Icons.link),
            title: Text(AppLocalizations.of(context).settingsConnection),
            subtitle: Text(AppLocalizations.of(context).settingsConnectionSub),
            enabled: onConnection != null,
            onTap: onConnection,
          ),
          ListTile(
            leading: const Icon(Icons.input),
            title: Text(AppLocalizations.of(context).settingsManualCode),
            subtitle: Text(AppLocalizations.of(context).settingsManualCodeSub),
            enabled: onManualCode != null,
            onTap: onManualCode,
          ),

          const SizedBox(height: 16),
        ],
      ),
    );
  }

  String _currentLanguageName(BuildContext context) {
    final provider = context.watch<LocaleProvider>();
    switch (provider.code) {
      case 'en':
        return 'English';
      case 'ru':
        return 'Русский';
      default:
        return 'Системный';
    }
  }

  void _showLanguageDialog(BuildContext context) {
    final provider = context.read<LocaleProvider>();
    showDialog<void>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Язык'),
        content: RadioGroup<String>(
          groupValue: provider.code,
          onChanged: (v) {
            if (v == null) return;
            provider.setLocale(v);
            Navigator.of(ctx).pop();
          },
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              RadioListTile<String>(
                title: Text(AppLocalizations.of(context).languageSystem),
                value: '',
              ),
              RadioListTile<String>(
                title: Text(AppLocalizations.of(context).languageEnglish),
                value: 'en',
              ),
              RadioListTile<String>(
                title: Text(AppLocalizations.of(context).languageRussian),
                value: 'ru',
              ),
            ],
          ),
        ),
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