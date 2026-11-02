import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import '../l10n/app_localizations.dart';
import '../services/log_service.dart';

class LogScreen extends StatelessWidget {
  const LogScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final logs = LogService.logs;

    return Scaffold(
      appBar: AppBar(
        title: Text(AppLocalizations.of(context).logTitle),
        actions: [
          IconButton(
            icon: const Icon(Icons.save),
            onPressed: () async {
              LogService.log('LogScreen: Save button pressed');
              const channel = MethodChannel('isotope/libp2p');
              final result = await channel.invokeMethod('saveLog', {
                'text': logs.join('\n'),
              });
              LogService.log('LogScreen: saveLog result=$result');
              if (context.mounted && result != null) {
                ScaffoldMessenger.of(context).showSnackBar(
                  SnackBar(content: Text('$result')),
                );
              }
            },
            tooltip: AppLocalizations.of(context).logSave,
          ),
          IconButton(
            icon: const Icon(Icons.delete_outline),
            onPressed: () async {
              LogService.log('LogScreen: Clear button pressed');
              await LogService.clear();
              if (context.mounted) {
                ScaffoldMessenger.of(context).showSnackBar(
                  SnackBar(content: Text(AppLocalizations.of(context).logCleared)),
                );
              }
            },
            tooltip: AppLocalizations.of(context).logClear,
          ),
          IconButton(
            icon: const Icon(Icons.refresh),
            onPressed: () {
              LogService.log('LogScreen: Refresh button pressed');
              if (context.mounted) {
                (context as Element).markNeedsBuild();
              }
            },
            tooltip: AppLocalizations.of(context).logRefresh,
          ),
        ],
      ),
      body: logs.isEmpty
          ? Center(child: Text(AppLocalizations.of(context).logEmpty))
          : SingleChildScrollView(
              child: SelectableText(
                logs.reversed.join('\n'),
                style: const TextStyle(
                  fontFamily: 'monospace',
                  fontSize: 11,
                ),
              ),
            ),
    );
  }
}