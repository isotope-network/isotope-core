// mobile/lib/screens/messages_settings_screen.dart
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../providers/chat_provider.dart';
import '../services/libp2p_service.dart';
import '../services/log_service.dart';

/// Экран «Сообщения».
/// Задержка отправки + TTL (время удаления сообщений).
class MessagesSettingsScreen extends StatefulWidget {
  const MessagesSettingsScreen({super.key});

  @override
  State<MessagesSettingsScreen> createState() => _MessagesSettingsScreenState();
}

class _MessagesSettingsScreenState extends State<MessagesSettingsScreen> {
  static const List<int> _delayOptions = [0, 3, 5, 10];

  // TTL-опции: секунды.
  static const List<Map<String, dynamic>> _ttlOptions = [
    {'label': '10 секунд', 'value': '10'},
    {'label': '1 минута', 'value': '60'},
    {'label': '10 минут', 'value': '600'},
    {'label': '1 час', 'value': '3600'},
    {'label': '24 часа', 'value': '86400'},
    {'label': '7 дней', 'value': '604800'},
    {'label': '30 дней', 'value': '2592000'},
    {'label': 'Вечно', 'value': '0'},
  ];

  String _myTtl = '0';
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    _loadTtl();
  }

  Future<void> _loadTtl() async {
    try {
      final ttl = await LibP2PService.getMyTtl();
      if (mounted) {
        final normalized = ttl.isEmpty ? '0' : ttl;
        setState(() {
          _myTtl = normalized;
          _loading = false;
        });
        // Синхронизируем с ChatProvider — на случай, если настройка
        // менялась в предыдущей сессии, а ChatProvider._currentTtl
        // загрузился со старым значением.
        final parsed = int.tryParse(normalized) ?? 0;
        context.read<ChatProvider>().setTtl(parsed);
      }
    } catch (e) {
      LogService.log('MessagesSettings: load TTL error: $e');
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _setTtl(String ttl) async {
    setState(() => _myTtl = ttl);
    try {
      final result = await LibP2PService.setMyTtl(ttl);
      if (result.containsKey('error')) {
        LogService.log('MessagesSettings: set TTL error: ${result['error']}');
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text('Ошибка: ${result['error']}')),
          );
        }
      } else {
        LogService.log('MessagesSettings: my_ttl=$ttl');
        // Обновляем TTL в ChatProvider — иначе он останется со старым
        // значением (загруженным один раз при старте libp2p) и _sendNow
        // не будет ставить expiresAt.
        if (mounted) {
          final parsed = int.tryParse(ttl) ?? 0;
          context.read<ChatProvider>().setTtl(parsed);
        }
      }
    } catch (e) {
      LogService.log('MessagesSettings: set TTL exception: $e');
    }
  }

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
                  'ВРЕМЯ УДАЛЕНИЯ',
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
                  'Через какое время сообщения удаляются у вас и у собеседника.',
                  style: TextStyle(fontSize: 13, color: Colors.grey),
                ),
              ),
              if (_loading)
                const Padding(
                  padding: EdgeInsets.all(16),
                  child: Center(child: CircularProgressIndicator()),
                )
              else
                ..._ttlOptions.map((option) {
                  final value = option['value'] as String;
                  final label = option['label'] as String;
                  final isSelected = _myTtl == value;
                  return RadioListTile<String>(
                    title: Text(label),
                    value: value,
                    // ignore: deprecated_member_use
                    groupValue: _myTtl,
                    // ignore: deprecated_member_use
                    onChanged: (v) {
                      if (v != null) _setTtl(v);
                    },
                    selected: isSelected,
                  );
                }),
            ],
          );
        },
      ),
    );
  }
}
// mobile/lib/screens/messages_settings_screen.dart