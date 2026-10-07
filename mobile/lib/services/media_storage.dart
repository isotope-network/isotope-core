// mobile/lib/services/media_storage.dart
import 'dart:convert';
import 'dart:io';
import 'package:path_provider/path_provider.dart';
import 'log_service.dart';

/// Хранилище медиа-файлов (чанковая сборка).
/// Структура на диске:
///   <app_dir>/isotope_media/chunks/<MediaID>/chunk_<index>.bin
///   <app_dir>/isotope_media/chunks/<MediaID>/meta.json
///   <app_dir>/isotope_media/files/<MediaID>.bin
/// После сборки папка chunks/<MediaID>/ удаляется.
class MediaStorage {
  static Directory? _root;
  static Directory? _chunksDir;
  static Directory? _filesDir;

  /// Инициализация. Вызывается один раз при старте приложения.
  static Future<void> init() async {
    final appDir = await getApplicationDocumentsDirectory();
    _root = Directory('${appDir.path}/isotope_media');
    _chunksDir = Directory('${_root!.path}/chunks');
    _filesDir = Directory('${_root!.path}/files');
    if (!await _chunksDir!.exists()) {
      await _chunksDir!.create(recursive: true);
    }
    if (!await _filesDir!.exists()) {
      await _filesDir!.create(recursive: true);
    }
    LogService.log('MEDIA: init done');
  }

  static Directory get _chunksRoot {
    if (_chunksDir == null) {
      throw StateError('MediaStorage not initialized');
    }
    return _chunksDir!;
  }

  static Directory get _filesRoot {
    if (_filesDir == null) {
      throw StateError('MediaStorage not initialized');
    }
    return _filesDir!;
  }

  /// Сохраняет один чанк. Возвращает true — принят.
  /// meta.json пишется при первом чанке (chunkIndex == 0).
  static Future<bool> saveChunk({
    required String mediaId,
    required int chunkIndex,
    required int chunkTotal,
    required String fileName,
    required int fileSize,
    required String base64Data,
  }) async {
    try {
      final dir = Directory('${_chunksRoot.path}/$mediaId');
      if (!await dir.exists()) {
        await dir.create(recursive: true);
      }

      // meta.json — только на chunk 0 (или если его нет).
      final metaFile = File('${dir.path}/meta.json');
      if (!await metaFile.exists()) {
        final meta = {
          'chunkTotal': chunkTotal,
          'fileSize': fileSize,
          'fileName': fileName,
        };
        await metaFile.writeAsString(jsonEncode(meta));
      }

      final bytes = base64Decode(base64Data);
      final chunkFile = File('${dir.path}/chunk_$chunkIndex.bin');
      await chunkFile.writeAsBytes(bytes);
      LogService.log('MEDIA: saved chunk $chunkIndex/$chunkTotal of $mediaId (${bytes.length} B)');
      return true;
    } catch (e) {
      LogService.log('MEDIA: saveChunk failed for $mediaId[$chunkIndex]: $e');
      return false;
    }
  }

  /// Проверяет — все ли чанки есть. Если да — склеивает в файл.
  /// Возвращает путь к собранному файлу или null, если ещё не все.
  static Future<String?> tryAssemble(String mediaId) async {
    try {
      final dir = Directory('${_chunksRoot.path}/$mediaId');
      if (!await dir.exists()) return null;

      final metaFile = File('${dir.path}/meta.json');
      if (!await metaFile.exists()) return null;

      final meta = jsonDecode(await metaFile.readAsString()) as Map<String, dynamic>;
      final chunkTotal = meta['chunkTotal'] as int? ?? 0;
      if (chunkTotal <= 0) return null;

      // Проверяем все чанки.
      final chunks = <File>[];
      for (int i = 0; i < chunkTotal; i++) {
        final f = File('${dir.path}/chunk_$i.bin');
        if (!await f.exists()) return null;
        chunks.add(f);
      }

      // Все на месте — склеиваем.
      final outPath = '${_filesRoot.path}/$mediaId.bin';
      final out = File(outPath);
      final sink = out.openWrite();
      for (final c in chunks) {
        sink.add(await c.readAsBytes());
      }
      await sink.close();

      // Удаляем папку с чанками. Если её уже удалил другой поток —
      // это не ошибка.
      try {
        if (await dir.exists()) {
          await dir.delete(recursive: true);
        }
      } catch (_) {}

      LogService.log('MEDIA: assembled $mediaId → $outPath');
      return outPath;
    } catch (e) {
      LogService.log('MEDIA: tryAssemble failed for $mediaId: $e');
      return null;
    }
  }

  /// Возвращает путь к собранному файлу, если он есть.
  static Future<String?> getFilePath(String mediaId) async {
    final path = '${_filesRoot.path}/$mediaId.bin';
    final f = File(path);
    if (await f.exists()) return path;
    return null;
  }

  /// Очистка: удаляет чанки старше N часов. По умолчанию — 24.
  static Future<void> cleanupOld({int hours = 24}) async {
    try {
      final cutoff = DateTime.now().subtract(Duration(hours: hours));
      final dirs = await _chunksRoot.list().toList();
      for (final e in dirs) {
        if (e is Directory) {
          final stat = await e.stat();
          if (stat.modified.isBefore(cutoff)) {
            await e.delete(recursive: true);
            LogService.log('MEDIA: cleanup removed ${e.path}');
          }
        }
      }
    } catch (e) {
      LogService.log('MEDIA: cleanupOld failed: $e');
    }
  }
}
// mobile/lib/services/media_storage.dart