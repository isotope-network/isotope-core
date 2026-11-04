// mobile/lib/screens/privacy_settings_screen.dart
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../providers/chat_provider.dart';
import '../services/libp2p_service.dart';
import '../services/log_service.dart';
import '../l10n/app_localizations.dart';

/// Экран «Приватность».
/// Пока один тумблер: «Показывать статус прочтения».
class PrivacySettingsScreen extends StatefulWidget {
  const PrivacySettingsScreen({super.key});

  @override
  State<PrivacySettingsScreen> createState() => _PrivacySettingsScreenState();
}

class _PrivacySettingsScreenState extends State<PrivacySettingsScreen> {
  bool _myReadEnabled = true;
  bool _showNotificationContent = true;
  bool _notifCount = false;
  bool _notifNames = false;
  bool _notifProgress = false;
  bool _sphinxEnabled = false;
  bool _mixingEnabled = false;
  bool _maskingEnabled = false;
  int _fileRequestThreshold = -1;
  String _seedPhrase = '';
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final readEnabled = await LibP2PService.getMyReadEnabled();
      final showContent = await LibP2PService.getShowNotificationContent();
      final notifSettings = await LibP2PService.getNotificationSettings();
      final sphinxEnabled = await LibP2PService.getSphinxEnabled();
      final mixingEnabled = await LibP2PService.getMixingEnabled();
      final maskingEnabled = await LibP2PService.getMaskingEnabled();
      final fileRequestThreshold = await LibP2PService.getFileRequestThreshold();
      final seedPhrase = await LibP2PService.getSeedPhrase();
      if (mounted) {
        setState(() {
          _myReadEnabled = readEnabled;
          _showNotificationContent = showContent;
          _notifCount = notifSettings['count'] ?? false;
          _notifNames = notifSettings['names'] ?? false;
          _notifProgress = notifSettings['progress'] ?? false;
          _sphinxEnabled = sphinxEnabled;
          _mixingEnabled = mixingEnabled;
          _maskingEnabled = maskingEnabled;
          _fileRequestThreshold = fileRequestThreshold;
          _seedPhrase = seedPhrase;
          _loading = false;
        });
      }
    } catch (e) {
      LogService.log('PrivacySettings: load failed: $e');
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _setEnabled(bool value) async {
    setState(() => _myReadEnabled = value);
    try {
      final result = await LibP2PService.setMyReadEnabled(value);
      if (result.containsKey('error')) {
        LogService.log('PrivacySettings: set failed: ${result['error']}');
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text('Ошибка: ${result['error']}')),
          );
          // Откат.
          setState(() => _myReadEnabled = !value);
        }
      } else {
        LogService.log('PrivacySettings: my_read_enabled=$value');
      }
    } catch (e) {
      LogService.log('PrivacySettings: set exception: $e');
      if (mounted) setState(() => _myReadEnabled = !value);
    }
  }

  Future<void> _setNotifSetting(String key, bool value) async {
    // Оптимистичное обновление.
    final prev = {
      'count': _notifCount,
      'names': _notifNames,
      'progress': _notifProgress,
    };
    setState(() {
      switch (key) {
        case 'count':
          _notifCount = value;
          break;
        case 'names':
          _notifNames = value;
          break;
        case 'progress':
          _notifProgress = value;
          break;
      }
    });
    try {
      final settings = {
        'count': _notifCount,
        'names': _notifNames,
        'progress': _notifProgress,
      };
      final result = await LibP2PService.setNotificationSettings(settings);
      if (result.containsKey('error')) {
        LogService.log('PrivacySettings: setNotifSetting failed: ${result['error']}');
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text('Ошибка: ${result['error']}')),
          );
          // Откат.
          setState(() {
            _notifCount = prev['count']!;
            _notifNames = prev['names']!;
            _notifProgress = prev['progress']!;
          });
        }
      } else {
        LogService.log('PrivacySettings: $key=$value');
        // Перечитать настройки в ChatProvider — шторка обновится сразу.
        if (mounted) {
          try {
            await context.read<ChatProvider>().reloadNotificationSettings();
          } catch (_) {}
        }
      }
    } catch (e) {
      LogService.log('PrivacySettings: setNotifSetting exception: $e');
      if (mounted) {
        setState(() {
          _notifCount = prev['count']!;
          _notifNames = prev['names']!;
          _notifProgress = prev['progress']!;
        });
      }
    }
  }

  Future<void> _setSphinxEnabled(bool value) async {
    setState(() => _sphinxEnabled = value);
    try {
      final result = await LibP2PService.setSphinxEnabled(value);
      if (result.containsKey('error')) {
        LogService.log('PrivacySettings: setSphinxEnabled failed: ${result['error']}');
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text('Ошибка: ${result['error']}')),
          );
          setState(() => _sphinxEnabled = !value);
        }
      } else {
        LogService.log('PrivacySettings: sphinx_enabled=$value');
      }
    } catch (e) {
      LogService.log('PrivacySettings: setSphinxEnabled exception: $e');
      if (mounted) setState(() => _sphinxEnabled = !value);
    }
  }

  Future<void> _setMixingEnabled(bool value) async {
    setState(() => _mixingEnabled = value);
    try {
      final result = await LibP2PService.setMixingEnabled(value);
      if (result.containsKey('error')) {
        LogService.log('PrivacySettings: setMixingEnabled failed: ${result['error']}');
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text('Ошибка: ${result['error']}')),
          );
          setState(() => _mixingEnabled = !value);
        }
      } else {
        LogService.log('PrivacySettings: mixing_enabled=$value');
      }
    } catch (e) {
      LogService.log('PrivacySettings: setMixingEnabled exception: $e');
      if (mounted) setState(() => _mixingEnabled = !value);
    }
  }

  Future<void> _setMaskingEnabled(bool value) async {
    setState(() => _maskingEnabled = value);
    try {
      final result = await LibP2PService.setMaskingEnabled(value);
      if (result.containsKey('error')) {
        LogService.log('PrivacySettings: setMaskingEnabled failed: ${result['error']}');
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text('Ошибка: ${result['error']}')),
          );
          setState(() => _maskingEnabled = !value);
        }
      } else {
        LogService.log('PrivacySettings: masking_enabled=$value');
      }
    } catch (e) {
      LogService.log('PrivacySettings: setMaskingEnabled exception: $e');
      if (mounted) setState(() => _maskingEnabled = !value);
    }
  }

  Future<void> _setFileRequestThreshold(int value) async {
    final prev = _fileRequestThreshold;
    setState(() => _fileRequestThreshold = value);
    try {
      final result = await LibP2PService.setFileRequestThreshold(value);
      if (result.containsKey('error')) {
        LogService.log('PrivacySettings: setFileRequestThreshold failed: ${result['error']}');
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text('Ошибка: ${result['error']}')),
          );
          setState(() => _fileRequestThreshold = prev);
        }
      } else {
        LogService.log('PrivacySettings: file_request_threshold=$value');
      }
    } catch (e) {
      LogService.log('PrivacySettings: setFileRequestThreshold exception: $e');
      if (mounted) setState(() => _fileRequestThreshold = prev);
    }
  }

  Future<void> _setShowContent(bool value) async {
    setState(() => _showNotificationContent = value);
    try {
      final result = await LibP2PService.setShowNotificationContent(value);
      if (result.containsKey('error')) {
        LogService.log('PrivacySettings: setShowContent failed: ${result['error']}');
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text('Ошибка: ${result['error']}')),
          );
          setState(() => _showNotificationContent = !value);
        }
      } else {
        LogService.log('PrivacySettings: show_notification_content=$value');
      }
    } catch (e) {
      LogService.log('PrivacySettings: setShowContent exception: $e');
      if (mounted) setState(() => _showNotificationContent = !value);
    }
  }

  Future<void> _showSeedDialog() async {
    final hasSeed = _seedPhrase.isNotEmpty;
    final l10n = AppLocalizations.of(context);
    await showDialog<void>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(l10n.seedDialogTitle),
        content: Text(
          hasSeed ? l10n.seedDialogHasSeed : l10n.seedDialogNoSeed,
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(),
            child: Text(l10n.cancel),
          ),
          if (!hasSeed)
            TextButton(
              onPressed: () {
                Navigator.of(ctx).pop();
                _createSeed();
              },
              child: Text(l10n.seedCreate),
            ),
          if (hasSeed)
            TextButton(
              onPressed: () {
                Navigator.of(ctx).pop();
                _showSeed();
              },
              child: Text(l10n.seedShow),
            ),
          TextButton(
            onPressed: () {
              Navigator.of(ctx).pop();
              _restoreSeed();
            },
            child: Text(l10n.seedRestore),
          ),
        ],
      ),
    );
  }

  Future<void> _createSeed() async {
    final l10n = AppLocalizations.of(context);
    final phrase = await LibP2PService.generateSeedPhrase(12);
    if (phrase.isEmpty) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(l10n.seedCreateFailed)),
        );
      }
      return;
    }
    if (!mounted) return;
    await showDialog<void>(
      context: context,
      barrierDismissible: false,
      builder: (ctx) => AlertDialog(
        title: Text(l10n.seedWriteDownTitle),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              l10n.seedWriteDownWarning,
              style: const TextStyle(color: Colors.red, fontWeight: FontWeight.bold),
            ),
            const SizedBox(height: 16),
            SelectableText(
              phrase,
              style: const TextStyle(fontSize: 16, fontFamily: 'monospace'),
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(),
            child: Text(l10n.cancel),
          ),
          FilledButton(
            onPressed: () async {
              Navigator.of(ctx).pop();
              final result = await LibP2PService.setSeedPhraseBeforeStart(phrase);
              if (!mounted) return;
              if (result.containsKey('error')) {
                ScaffoldMessenger.of(context).showSnackBar(
                  SnackBar(content: Text('${l10n.error}: ${result['error']}')),
                );
                return;
              }
              final savedToKeystore = await LibP2PService.saveSeedToKeystore(phrase);
              if (!mounted) return;
              setState(() => _seedPhrase = phrase);
              ScaffoldMessenger.of(context).showSnackBar(
                SnackBar(
                  content: Text(savedToKeystore ? l10n.seedSaved : l10n.seedSaveFailed),
                ),
              );
            },
            child: Text(l10n.seedIWroteIt),
          ),
        ],
      ),
    );
  }

  Future<void> _showSeed() async {
    final l10n = AppLocalizations.of(context);
    await showDialog<void>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(l10n.seedYourPhraseTitle),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              l10n.seedYourPhraseWarning,
              style: const TextStyle(color: Colors.red),
            ),
            const SizedBox(height: 16),
            SelectableText(
              _seedPhrase,
              style: const TextStyle(fontSize: 16, fontFamily: 'monospace'),
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(),
            child: Text(l10n.close),
          ),
        ],
      ),
    );
  }

  Future<void> _restoreSeed() async {
    final l10n = AppLocalizations.of(context);
    var localPhrase = '';
    final phrase = await showDialog<String>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(l10n.seedRestoreTitle),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              l10n.seedRestoreWarning,
              style: const TextStyle(color: Colors.red),
            ),
            const SizedBox(height: 16),
            TextField(
              maxLines: 4,
              onChanged: (v) => localPhrase = v,
              decoration: const InputDecoration(
                hintText: 'word1 word2 word3 ...',
                border: OutlineInputBorder(),
              ),
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(),
            child: Text(l10n.cancel),
          ),
          FilledButton(
            onPressed: () => Navigator.of(ctx).pop(localPhrase.trim()),
            child: Text(l10n.seedRestore),
          ),
        ],
      ),
    );
    if (phrase == null || phrase.isEmpty) return;

    final result = await LibP2PService.setSeedPhraseBeforeStart(phrase);
    if (!mounted) return;
    if (result.containsKey('error')) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('${l10n.error}: ${result['error']}')),
      );
      return;
    }
    final savedToKeystore = await LibP2PService.saveSeedToKeystore(phrase);
    if (!mounted) return;
    setState(() => _seedPhrase = phrase);
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(savedToKeystore ? l10n.seedSaved : l10n.seedSaveFailed),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(AppLocalizations.of(context).privacy),
      ),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : ListView(
              children: [
                SwitchListTile(
                  title: Text(AppLocalizations.of(context).privacyShowReadReceipts),
                  subtitle: Text(AppLocalizations.of(context).privacyShowReadReceiptsSub),
                  value: _myReadEnabled,
                  onChanged: _setEnabled,
                ),
                SwitchListTile(
                  title: Text(AppLocalizations.of(context).privacyShowNotificationContent),
                  subtitle: Text(AppLocalizations.of(context).privacyShowNotificationContentSub),
                  value: _showNotificationContent,
                  onChanged: _setShowContent,
                ),
                const Divider(),
                Padding(
                  padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
                  child: Text(
                    AppLocalizations.of(context).privacyNetworkNotification,
                    style: const TextStyle(fontWeight: FontWeight.bold),
                  ),
                ),
                Padding(
                  padding: const EdgeInsets.symmetric(horizontal: 16),
                  child: Text(
                    AppLocalizations.of(context).privacyNetworkNotificationSub,
                    style: const TextStyle(fontSize: 12, color: Colors.grey),
                  ),
                ),
                SwitchListTile(
                  title: Text(AppLocalizations.of(context).privacyShowCount),
                  subtitle: Text(AppLocalizations.of(context).privacyShowCountSub),
                  value: _notifCount,
                  onChanged: (v) => _setNotifSetting('count', v),
                ),
                SwitchListTile(
                  title: Text(AppLocalizations.of(context).privacyShowNames),
                  subtitle: Text(AppLocalizations.of(context).privacyShowNamesSub),
                  value: _notifNames,
                  onChanged: (v) => _setNotifSetting('names', v),
                ),
                SwitchListTile(
                  title: Text(AppLocalizations.of(context).privacyShowFileProgress),
                  subtitle: Text(AppLocalizations.of(context).privacyShowFileProgressSub),
                  value: _notifProgress,
                  onChanged: (v) => _setNotifSetting('progress', v),
                ),
                const Divider(),
                Padding(
                  padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
                  child: Text(
                    AppLocalizations.of(context).privacyAnonymity,
                    style: const TextStyle(fontWeight: FontWeight.bold),
                  ),
                ),
                SwitchListTile(
                  title: Text(AppLocalizations.of(context).privacySphinx),
                  subtitle: Text(AppLocalizations.of(context).privacySphinxSub),
                  value: _sphinxEnabled,
                  onChanged: _setSphinxEnabled,
                ),
                SwitchListTile(
                  title: Text(AppLocalizations.of(context).privacyMixing),
                  subtitle: Text(
                    _sphinxEnabled
                        ? AppLocalizations.of(context).privacyMixingOn
                        : AppLocalizations.of(context).privacyMixingOff,
                  ),
                  value: _mixingEnabled,
                  onChanged: _sphinxEnabled ? _setMixingEnabled : null,
                ),
                SwitchListTile(
                  title: Text(AppLocalizations.of(context).privacyMasking),
                  subtitle: Text(AppLocalizations.of(context).privacyMaskingSub),
                  value: _maskingEnabled,
                  onChanged: _setMaskingEnabled,
                ),
                const Divider(),
                Padding(
                  padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
                  child: Text(
                    AppLocalizations.of(context).privacyFileRequestSection,
                    style: const TextStyle(fontWeight: FontWeight.bold),
                  ),
                ),
                Padding(
                  padding: const EdgeInsets.symmetric(horizontal: 16),
                  child: Text(
                    AppLocalizations.of(context).privacyFileRequestSectionSub,
                    style: const TextStyle(fontSize: 12, color: Colors.grey),
                  ),
                ),
                ListTile(
                  leading: _fileRequestThreshold == -1
                      ? const Icon(Icons.check, color: Colors.green)
                      : const Icon(Icons.check, color: Colors.transparent),
                  title: Text(AppLocalizations.of(context).privacyFileRequestAlways),
                  onTap: () => _setFileRequestThreshold(-1),
                ),
                ListTile(
                  leading: _fileRequestThreshold > 0
                      ? const Icon(Icons.check, color: Colors.green)
                      : const Icon(Icons.check, color: Colors.transparent),
                  title: Text(AppLocalizations.of(context).privacyFileRequestFromMB),
                  subtitle: _fileRequestThreshold > 0
                      ? Padding(
                          padding: const EdgeInsets.only(top: 8),
                          child: TextFormField(
                            initialValue: _fileRequestThreshold.toString(),
                            keyboardType: TextInputType.number,
                            decoration: const InputDecoration(
                              labelText: 'МБ',
                              isDense: true,
                              border: OutlineInputBorder(),
                            ),
                            onFieldSubmitted: (v) {
                              final n = int.tryParse(v.trim()) ?? 10;
                              if (n > 0) _setFileRequestThreshold(n);
                            },
                          ),
                        )
                      : null,
                  onTap: () => _setFileRequestThreshold(10),
                ),
                ListTile(
                  leading: _fileRequestThreshold == 0
                      ? const Icon(Icons.check, color: Colors.green)
                      : const Icon(Icons.check, color: Colors.transparent),
                  title: Text(AppLocalizations.of(context).privacyFileRequestNever),
                  onTap: () => _setFileRequestThreshold(0),
                ),
                const Divider(),
                Padding(
                  padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
                  child: Text(
                    AppLocalizations.of(context).privacySeedSection,
                    style: const TextStyle(fontWeight: FontWeight.bold),
                  ),
                ),
                Padding(
                  padding: const EdgeInsets.symmetric(horizontal: 16),
                  child: Text(
                    AppLocalizations.of(context).privacySeedSectionSub,
                    style: const TextStyle(fontSize: 12, color: Colors.grey),
                  ),
                ),
                ListTile(
                  leading: Icon(
                    _seedPhrase.isEmpty ? Icons.key_off : Icons.key,
                    color: _seedPhrase.isEmpty ? Colors.grey : Colors.green,
                  ),
                  title: Text(
                    _seedPhrase.isEmpty
                        ? AppLocalizations.of(context).privacySeedNotSet
                        : AppLocalizations.of(context).privacySeedSet,
                  ),
                  subtitle: Text(
                    _seedPhrase.isEmpty
                        ? AppLocalizations.of(context).privacySeedCreateOrRestore
                        : AppLocalizations.of(context).privacySeedShowRestoreOther,
                  ),
                  trailing: const Icon(Icons.chevron_right),
                  onTap: _showSeedDialog,
                ),
              ],
            ),
    );
  }
}
// mobile/lib/screens/privacy_settings_screen.dart