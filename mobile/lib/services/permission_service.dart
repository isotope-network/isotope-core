// mobile/lib/services/permission_service.dart
import 'dart:io';
import 'package:flutter/services.dart';

/// Сервис запросов разрешений Android.
/// Оборачивает MethodChannel isotope/libp2p — case requestPermissions.
/// По мере надобности: notifications при старте, camera при QR,
/// nearby при «Найти рядом».
class PermissionService {
  static const MethodChannel _channel = MethodChannel('isotope/libp2p');

  /// Запрашивает список разрешений.
  /// Возвращает true — все выданы. false — хоть одно отказано.
  static Future<bool> request(List<String> permissions) async {
    if (!Platform.isAndroid) return true;
    if (permissions.isEmpty) return true;
    try {
      final result = await _channel.invokeMethod<bool>(
        'requestPermissions',
        {'permissions': permissions},
      );
      return result ?? false;
    } on PlatformException {
      return false;
    }
  }

  /// Открывает настройки приложения (для fallback после отказа).
  static Future<void> openAppSettings() async {
    if (!Platform.isAndroid) return;
    try {
      await _channel.invokeMethod<String>('openAppSettings');
    } on PlatformException {
      // ignore
    }
  }

  /// Разрешения для уведомлений (Android 13+).
  static List<String> get notifications => [
        'android.permission.POST_NOTIFICATIONS',
      ];

  /// Разрешения для камеры (QR-сканер).
  static List<String> get camera => [
        'android.permission.CAMERA',
      ];

  /// Разрешения для микрофона (голосовые сообщения).
  static List<String> get microphone => [
        'android.permission.RECORD_AUDIO',
      ];

  /// Разрешения для «Найти рядом» (BLE + Wi-Fi + локация).
  static List<String> get nearby => [
        'android.permission.BLUETOOTH_SCAN',
        'android.permission.BLUETOOTH_ADVERTISE',
        'android.permission.BLUETOOTH_CONNECT',
        'android.permission.ACCESS_FINE_LOCATION',
        'android.permission.NEARBY_WIFI_DEVICES',
      ];

  /// Универсальный помощник: если не выданы — открыть настройки.
  static Future<void> requestWithSettingsFallback(
    List<String> permissions,
    void Function()? onDenied,
  ) async {
    final ok = await request(permissions);
    if (!ok && onDenied != null) {
      onDenied();
    }
  }

  /// Не используется сейчас — задел на будущее.
  static Future<Map<String, dynamic>> getStatus() async {
    return {};
  }
}
// mobile/lib/services/permission_service.dart