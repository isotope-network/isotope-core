// mobile/lib/screens/messages_settings_screen.dart
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../providers/chat_provider.dart';
import '../l10n/app_localizations.dart';
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

  /// Периоды TTL. Только значения для Go. Метки — через _periodLabel.
  static const List<String> _ttlPeriodValues = [
    '10s', '30s', '1m', '5m', '15m', '30m', '1h', '4h', '24h',
  ];

  /// Режимы удаления. Только значения для Go. Метки — через _modeLabel.
  static const List<String> _ttlModeValues = ['after_read', 'hard'];

  String _periodLabel(BuildContext context, String value) {
    final l = AppLocalizations.of(context);
    switch (value) {
      case '10s': return l.messagesTtl10s;
      case '30s': return l.messagesTtl30s;
      case '1m': return l.messagesTtl1m;
      case '5m': return l.messagesTtl5m;
      case '15m': return l.messagesTtl15m;
      case '30m': return l.messagesTtl30m;
      case '1h': return l.messagesTtl1h;
      case '4h': return l.messagesTtl4h;
      case '24h': return l.messagesTtl24h;
      default: return value;
    }
  }

  String _modeLabel(BuildContext context, String value) {
    final l = AppLocalizations.of(context);
    switch (value) {
      case 'after_read': return l.messagesModeAfterRead;
      case 'hard': return l.messagesModeHard;
      default: return value;
    }
  }

  String _ttlPeriod = 'never';
  String _ttlMode = '';
  // Последние выбранные период/режим (не «Не удаляются»).
  // Восстанавливаются при возврате с «Не удаляются» на «Удалять через».
  String _lastPeriod = '10s';
  String _lastMode = 'after_read';
  bool _loading = true;

  /// "Не удаляются" — любое из старых/новых значений.
  bool get _isForever => _ttlPeriod == 'never' || _ttlPeriod == 'forever';

  @override
  void initState() {
    super.initState();
    _loadTtl();
  }

  Future<void> _loadTtl() async {
    try {
      final ttl = await LibP2PService.getTtl();
      if (mounted) {
        var period = ttl['ttl_period'] ?? 'never';
        if (period == 'forever' || period.isEmpty) {
          period = 'never';
        }
        final mode = ttl['ttl_mode'] ?? '';
        setState(() {
          _ttlPeriod = period;
          _ttlMode = mode;
          if (period != 'never') {
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

  /// Переключение «Не удаляются» / «Удалять через».
  void _setForever(bool forever) {
    if (forever) {
      _saveTtl('never', '');
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
      case '30s':
        return 30;
      case '1m':
        return 60;
      case '5m':
        return 300;
      case '15m':
        return 900;
      case '30m':
        return 1800;
      case '1h':
        return 3600;
      case '4h':
        return 14400;
      case '24h':
        return 86400;
      default:
        return 0;
    }
  }

  /// Предупреждение о запрете скриншотов (TTL от 10 секунд до 1 минуты).
  bool get _showSecureWarning {
    final s = _periodSeconds(_ttlPeriod);
    return s >= 10 && s <= 60;
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(AppLocalizations.of(context).messagesTitle),
      ),
      body: Consumer<ChatProvider>(
        builder: (_, provider, __) {
          final l = AppLocalizations.of(context);
          return ListView(
            children: [
              // === ЗАДЕРЖКА ОТПРАВКИ ===
              Padding(
                padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
                child: Text(
                  l.messagesDelaySection,
                  style: const TextStyle(
                    fontSize: 12,
                    fontWeight: FontWeight.bold,
                    color: Colors.grey,
                    letterSpacing: 0.5,
                  ),
                ),
              ),
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: 16),
                child: Text(
                  l.messagesDelaySub,
                  style: const TextStyle(fontSize: 13, color: Colors.grey),
                ),
              ),
              ListTile(
                title: Text(l.messagesDelayLabel),
                trailing: DropdownButton<int>(
                  value: _delayOptions.contains(provider.sendDelay)
                      ? provider.sendDelay
                      : 0,
                  onChanged: (v) {
                    if (v != null) provider.setSendDelay(v);
                  },
                  items: _delayOptions
                      .map((sec) => DropdownMenuItem<int>(
                            value: sec,
                            child: Text(
                              sec == 0 ? l.messagesDelayNone : l.messagesDelaySeconds(sec),
                            ),
                          ))
                      .toList(),
                ),
              ),

              const Divider(),

              // === ВРЕМЯ УДАЛЕНИЯ ===
              Padding(
                padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
                child: Text(
                  l.messagesTtlSection,
                  style: const TextStyle(
                    fontSize: 12,
                    fontWeight: FontWeight.bold,
                    color: Colors.grey,
                    letterSpacing: 0.5,
                  ),
                ),
              ),
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: 16),
                child: Text(
                  l.messagesTtlSub,
                  style: const TextStyle(fontSize: 13, color: Colors.grey),
                ),
              ),

              if (_loading)
                const Padding(
                  padding: EdgeInsets.all(16),
                  child: Center(child: CircularProgressIndicator()),
                )
              else ...[
                RadioListTile<bool>(
                  title: Text(l.messagesTtlNever),
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
                      Text(l.messagesTtlAfter),
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
                          items: _ttlPeriodValues
                              .map((value) {
                                // 10s / 30s / 1m — периоды с запретом скриншотов.
                                final isSecure = value == '10s' ||
                                    value == '30s' ||
                                    value == '1m';
                                return DropdownMenuItem<String>(
                                  value: value,
                                  child: Text(
                                    _periodLabel(context, value),
                                    style: isSecure
                                        ? const TextStyle(color: Colors.orange)
                                        : null,
                                  ),
                                );
                              })
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
                  Padding(
                    padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
                    child: Text(
                      l.messagesModeSection,
                      style: const TextStyle(
                        fontSize: 12,
                        fontWeight: FontWeight.bold,
                        color: Colors.grey,
                        letterSpacing: 0.5,
                      ),
                    ),
                  ),
                  ..._ttlModeValues.map((value) {
                    final selected = _ttlMode == value;
                    return RadioListTile<String>(
                      title: Text(_modeLabel(context, value)),
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

                  Padding(
                    padding: const EdgeInsets.fromLTRB(16, 8, 16, 8),
                    child: Text(
                      l.messagesModeAfterReadSub,
                      style: const TextStyle(fontSize: 13, color: Colors.grey),
                    ),
                  ),
                  Padding(
                    padding: const EdgeInsets.fromLTRB(16, 8, 16, 8),
                    child: Text(
                      l.messagesModeHardSub,
                      style: const TextStyle(fontSize: 13, color: Colors.grey),
                    ),
                  ),

                  if (_showSecureWarning)
                    Padding(
                      padding: const EdgeInsets.fromLTRB(16, 8, 16, 16),
                      child: Text(
                        l.messagesSecureWarning,
                        style: const TextStyle(
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