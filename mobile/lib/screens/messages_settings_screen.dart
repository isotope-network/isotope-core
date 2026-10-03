// mobile/lib/screens/messages_settings_screen.dart
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../providers/chat_provider.dart';
import '../services/libp2p_service.dart';
import '../services/log_service.dart';

/// Экран «Сообщения».
/// Задержка отправки + TTL (время и режим удаления сообщений).
class MessagesSettingsScreen extends StatefulWidget {
  const MessagesSettingsScreen({super.key});

  @override
  State<MessagesSettingsScreen> createState() => _MessagesSettingsScreenState();
}

class _MessagesSettingsScreenState extends State<MessagesSettingsScreen> {
  static const List<int> _delayOptions = [0, 3, 5, 10];

  /// Периоды TTL. Ключ — значение для Go, метка — для UI.
  static const List<Map<String, String>> _ttlPeriods = [
    {'value': '10s', 'label': '10 секунд'},
    {'value': '1m', 'label': '1 минута'},
    {'value': '10m', 'label': '10 минут'},
    {'value': '1h', 'label': '1 час'},
    {'value': '24h', 'label': '24 часа'},
    {'value': '7d', 'label': '7 дней'},
    {'value': '30d', 'label': '30 дней'},
  ];

  /// Режимы удаления.
  static const List<Map<String, String>> _ttlModes = [
    {'value': 'after_read', 'label': 'После прочтения'},
    {'value': 'hard', 'label': 'Жёсткий'},
  ];

  String _ttlPeriod = 'forever';
  String _ttlMode = '';
  // Последние выбранные период/режим (не «Вечно»).
  // Восстанавливаются при возврате с «Вечно» на «Удалять через».
  String _lastPeriod = '10s';
  String _lastMode = 'after_read';
  bool _loading = true;

  bool get _isForever => _ttlPeriod == 'forever';

  @override
  void initState() {
    super.initState();
    _loadTtl();
  }

  Future<void> _loadTtl() async {
    try {
      final ttl = await LibP2PService.getTtl();
      if (mounted) {
        final period = ttl['ttl_period'] ?? 'forever';
        final mode = ttl['ttl_mode'] ?? '';
        setState(() {
          _ttlPeriod = period;
          _ttlMode = mode;
          if (period != 'forever') {
            _lastPeriod = period;
            _lastMode = mode.isEmpty ? 'after_read' : mode;
          }
          _loading = false;
        });
        context.read<ChatProvider>().setTtl(period, mode);
        LogService.log('MessagesSettings: loaded period=$period mode=$mode');
      }
    } catch (e) {
      LogService.log('MessagesSettings: load TTL error: $e');
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _saveTtl(String period, String mode) async {
    setState(() {
      _ttlPeriod = period;
      _ttlMode = mode;
    });
    try {
      final result = await LibP2PService.setTtl(period, mode);
      if (result.containsKey('error')) {
        LogService.log('MessagesSettings: set TTL error: ${result['error']}');
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text('Ошибка: ${result['error']}')),
          );
        }
      } else {
        LogService.log('MessagesSettings: ttl set period=$period mode=$mode');
        if (mounted) {
          context.read<ChatProvider>().setTtl(period, mode);
        }
      }
    } catch (e) {
      LogService.log('MessagesSettings: set TTL exception: $e');
    }
  }

  /// Переключение «Вечно» / «Удалять через».
  void _setForever(bool forever) {
    if (forever) {
      _saveTtl('forever', '');
    } else {
      _saveTtl(_lastPeriod, _lastMode);
    }
  }

  /// Выбор периода TTL.
  void _setPeriod(String period) {
    final mode = _ttlMode.isEmpty ? 'after_read' : _ttlMode;
    _lastPeriod = period;
    _lastMode = mode;
    _saveTtl(period, mode);
  }

  /// Выбор режима удаления.
  void _setMode(String mode) {
    final period = _isForever ? _lastPeriod : _ttlPeriod;
    _lastPeriod = period;
    _lastMode = mode;
    _saveTtl(period, mode);
  }

  /// Период в секундах — для проверки диапазона FLAG_SECURE.
  int _periodSeconds(String period) {
    switch (period) {
      case '10s':
        return 10;
      case '1m':
        return 60;
      case '10m':
        return 600;
      case '1h':
        return 3600;
      case '24h':
        return 86400;
      case '7d':
        return 604800;
      case '30d':
        return 2592000;
      default:
        return 0;
    }
  }

  /// Предупреждение о запрете скриншотов (TTL от 10 сек до 1 часа).
  bool get _showSecureWarning {
    final s = _periodSeconds(_ttlPeriod);
    return s >= 10 && s <= 3600;
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
              // === ЗАДЕРЖКА ОТПРАВКИ ===
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

              // === ВРЕМЯ УДАЛЕНИЯ ===
              const Padding(
                padding: EdgeInsets.fromLTRB(16, 16, 16, 8),
                child: Text(
                  'ВРЕМЯ УДАЛЕНИЯ СООБЩЕНИЙ',
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
              else ...[
                RadioListTile<bool>(
                  title: const Text('Вечно'),
                  value: true,
                  // ignore: deprecated_member_use
                  groupValue: _isForever,
                  // ignore: deprecated_member_use
                  onChanged: (v) {
                    if (v == true) _setForever(true);
                  },
                ),
                RadioListTile<bool>(
                  title: Row(
                    children: [
                      const Text('Удалять через:'),
                      const SizedBox(width: 12),
                      Expanded(
                        child: DropdownButton<String>(
                          isExpanded: true,
                          value: _isForever ? '10s' : _ttlPeriod,
                          onChanged: _isForever
                              ? null
                              : (v) {
                                  if (v != null) _setPeriod(v);
                                },
                          items: _ttlPeriods
                              .map((p) => DropdownMenuItem<String>(
                                    value: p['value'],
                                    child: Text(p['label'] ?? ''),
                                  ))
                              .toList(),
                        ),
                      ),
                    ],
                  ),
                  value: false,
                  // ignore: deprecated_member_use
                  groupValue: _isForever,
                  // ignore: deprecated_member_use
                  onChanged: (v) {
                    if (v == false) _setForever(false);
                  },
                ),

                if (!_isForever) ...[
                  const Padding(
                    padding: EdgeInsets.fromLTRB(16, 16, 16, 8),
                    child: Text(
                      'РЕЖИМ УДАЛЕНИЯ',
                      style: TextStyle(
                        fontSize: 12,
                        fontWeight: FontWeight.bold,
                        color: Colors.grey,
                        letterSpacing: 0.5,
                      ),
                    ),
                  ),
                  ..._ttlModes.map((m) {
                    final value = m['value']!;
                    final label = m['label']!;
                    final selected = _ttlMode == value;
                    return RadioListTile<String>(
                      title: Text(label),
                      value: value,
                      // ignore: deprecated_member_use
                      groupValue: _ttlMode.isEmpty ? 'after_read' : _ttlMode,
                      // ignore: deprecated_member_use
                      onChanged: (v) {
                        if (v != null) _setMode(v);
                      },
                      selected: selected,
                    );
                  }),

                  const Padding(
                    padding: EdgeInsets.fromLTRB(16, 8, 16, 8),
                    child: Text(
                      'После прочтения — сообщение удаляется через выбранное время после того, как собеседник его прочитал.\n\n'
                      'Если прочтение неизвестно (собеседник не делится статусом) — автоматически используется жёсткий режим.\n\n'
                      'Если сообщение не прочитано в течение 48 часов — оно удаляется принудительно.',
                      style: TextStyle(fontSize: 13, color: Colors.grey),
                    ),
                  ),
                  const Padding(
                    padding: EdgeInsets.fromLTRB(16, 8, 16, 8),
                    child: Text(
                      'Жёсткий — сообщение удаляется через выбранное время после отправки, независимо от прочтения.',
                      style: TextStyle(fontSize: 13, color: Colors.grey),
                    ),
                  ),

                  if (_showSecureWarning)
                    const Padding(
                      padding: EdgeInsets.fromLTRB(16, 8, 16, 16),
                      child: Text(
                        '⚠ При выборе от 10 секунд до 1 часа скриншоты чата с такими сообщениями временно запрещены — до их удаления.',
                        style: TextStyle(
                          fontSize: 13,
                          color: Colors.orange,
                          fontWeight: FontWeight.w500,
                        ),
                      ),
                    ),
                ],
              ],
            ],
          );
        },
      ),
    );
  }
}
// mobile/lib/screens/messages_settings_screen.dart