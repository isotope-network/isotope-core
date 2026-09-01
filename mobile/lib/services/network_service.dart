import 'dart:async';
import 'dart:io';
import 'package:connectivity_plus/connectivity_plus.dart';

/// Слушатель изменений сети
class NetworkService {
  String? _currentIp;
  StreamSubscription? _connectivitySub;

  final _ipChangedController = StreamController<String?>.broadcast();
  Stream<String?> get onIpChanged => _ipChangedController.stream;

  /// Запускает мониторинг сети
  void startMonitoring() {
    _connectivitySub = Connectivity().onConnectivityChanged.listen((result) async {
      final ip = await _getLocalIp();
      if (ip != _currentIp) {
        _currentIp = ip;
        _ipChangedController.add(ip);
      }
    });
  }

  /// Останавливает мониторинг
  void stopMonitoring() {
    _connectivitySub?.cancel();
  }

  /// Получает текущий IP
  Future<String?> _getLocalIp() async {
    try {
      final interfaces = await NetworkInterface.list();
      for (final interface in interfaces) {
        for (final addr in interface.addresses) {
          if (addr.type == InternetAddressType.IPv4 && !addr.isLoopback) {
            return addr.address;
          }
        }
      }
    } catch (_) {}
    return null;
  }

  /// Текущий IP
  String? get currentIp => _currentIp;

  void dispose() {
    stopMonitoring();
    _ipChangedController.close();
  }
}