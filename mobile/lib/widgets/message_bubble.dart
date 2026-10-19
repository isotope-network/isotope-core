// mobile/lib/widgets/message_bubble.dart
import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';
import 'package:flutter/material.dart';
import 'package:just_audio/just_audio.dart';
import 'package:path_provider/path_provider.dart';
import 'package:open_file/open_file.dart';
import 'package:video_player/video_player.dart';
import 'package:zero_cached_thumbnail/zero_cached_thumbnail.dart';
import '../models/message.dart';
import '../utils/time_format.dart';
import '../services/log_service.dart';

class MessageBubble extends StatelessWidget {
  final Message message;
  final VoidCallback? onLike;
  final VoidCallback? onDislike;
  /// Отображаемое имя отправителя (для входящих).
  /// Пусто — используется message.sender (PeerID).
  final String? senderName;
  /// Прогресс передачи файла (0-100). Для file-сообщений.
  final int Function(String mediaId)? progressFor;

  const MessageBubble({
    super.key,
    required this.message,
    this.onLike,
    this.onDislike,
    this.senderName,
    this.progressFor,
  });

  int _progressFor(String mediaId) {
    if (progressFor == null) return 0;
    return progressFor!(mediaId);
  }

  @override
  Widget build(BuildContext context) {
    final isOwn = message.isOwn;
    final isNetwork = !isOwn && message.sender != 'Вы';

    return Align(
      alignment: isOwn ? Alignment.centerRight : Alignment.centerLeft,
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
        child: Column(
          crossAxisAlignment: isOwn ? CrossAxisAlignment.end : CrossAxisAlignment.start,
          children: [
            Container(
              constraints: BoxConstraints(
                maxWidth: MediaQuery.of(context).size.width * 0.7,
              ),
              decoration: BoxDecoration(
                color: isOwn
                    ? const Color(0xFFDCF8C6)
                    : isNetwork
                        ? const Color(0xFFFFF3E0)
                        : Colors.white,
                borderRadius: BorderRadius.only(
                  topLeft: const Radius.circular(16),
                  topRight: const Radius.circular(16),
                  bottomLeft: Radius.circular(isOwn ? 16 : 4),
                  bottomRight: Radius.circular(isOwn ? 4 : 16),
                ),
                border: (!isOwn && !isNetwork)
                    ? Border.all(color: Colors.grey.shade200)
                    : null,
              ),
              child: Padding(
                padding: const EdgeInsets.all(10),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      isOwn
                          ? 'Вы'
                          : (senderName != null && senderName!.isNotEmpty
                              ? senderName!
                              : message.sender),
                      style: TextStyle(
                        fontSize: 12,
                        fontWeight: FontWeight.w600,
                        color: Colors.black54,
                      ),
                    ),
                    const SizedBox(height: 2),
                    Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        _weightBadge(message.weight),
                        const SizedBox(width: 6),
                        Text(
                          formatMessageTime(message.time),
                          style: const TextStyle(fontSize: 11, color: Colors.black45),
                        ),
                      ],
                    ),
                    const SizedBox(height: 6),
                    if (message.isVoice)
                      _VoicePlayer(message: message)
                    else if (message.isFile)
                      _buildFileView(context)
                    else if (message.isPhoto)
                      _PhotoView(message: message)
                    else
                      Text(
                        message.displayText,
                        style: const TextStyle(fontSize: 15),
                      ),
                    if (isOwn && message.messageStatus != null) ...[
                      const SizedBox(height: 4),
                      Align(
                        alignment: Alignment.centerRight,
                        child: Tooltip(
                          message: _statusTooltip(message.messageStatus),
                          child: Text(
                            message.messageStatusIcon,
                            style: TextStyle(
                              fontSize: 11,
                              color: Color(message.messageStatusColor),
                              fontWeight: message.messageStatus == 4
                                  ? FontWeight.w900
                                  : FontWeight.w600,
                            ),
                          ),
                        ),
                      ),
                    ] else if (isOwn && message.deliveryStatus != null) ...[
                      const SizedBox(height: 4),
                      Align(
                        alignment: Alignment.centerRight,
                        child: Text(
                          '${message.statusIcon} ${_statusText(message.deliveryStatus!)}',
                          style: TextStyle(
                            fontSize: 10,
                            color: Color(message.statusColor),
                          ),
                        ),
                      ),
                    ],
                  ],
                ),
              ),
            ),
            Padding(
              padding: const EdgeInsets.only(top: 6),
              child: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  _iconButton(
                    icon: Icons.thumb_up_alt_outlined,
                    color: message.score > 0 ? Colors.green : Colors.grey,
                    onTap: onLike,
                    size: 18,
                  ),
                  const SizedBox(width: 16),
                  _iconButton(
                    icon: Icons.thumb_down_alt_outlined,
                    color: message.score < 0 ? Colors.red : Colors.grey,
                    onTap: onDislike,
                    size: 18,
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildFileView(BuildContext context) {
    final p = _progressFor(message.mediaId);
    final kind = _fileKind(message.fileName);
    final hasPreview = message.localFilePath.isNotEmpty &&
        (kind == _FileKind.image || kind == _FileKind.video);
    final showProgress = p >= 0 && p < 100;

    // Превью + прогресс-бар под ним (пока идёт отправка/приём).
    if (hasPreview && showProgress) {
      return Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          kind == _FileKind.image
              ? _ImageFileView(message: message)
              : _VideoFileView(message: message),
          const SizedBox(height: 4),
          SizedBox(
            width: 200,
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  message.isOwn ? 'Отправка $p%' : 'Получение $p%',
                  style: const TextStyle(fontSize: 11, color: Colors.black54),
                ),
                const SizedBox(height: 2),
                ClipRRect(
                  borderRadius: BorderRadius.circular(3),
                  child: LinearProgressIndicator(
                    value: (p / 100).clamp(0.0, 1.0),
                    minHeight: 4,
                    backgroundColor: Colors.grey.shade300,
                    valueColor: const AlwaysStoppedAnimation<Color>(Color(0xFF4CAF50)),
                  ),
                ),
              ],
            ),
          ),
        ],
      );
    }

    // Превью без прогресса (готово).
    if (hasPreview) {
      return kind == _FileKind.image
          ? _ImageFileView(message: message)
          : _VideoFileView(message: message);
    }

    // Нет превью — обычный файл с прогрессом.
    return _FileRow(
      message: message,
      progress: p,
    );
  }

  Widget _weightBadge(double weight) {
    Color bg;
    Color fg;
    if (weight >= 0.7) {
      bg = const Color(0xFFC8E6C9);
      fg = const Color(0xFF2E7D32);
    } else if (weight <= 0.3) {
      bg = const Color(0xFFFFCDD2);
      fg = const Color(0xFFC62828);
    } else {
      bg = const Color(0xFFFFF9C4);
      fg = const Color(0xFFF57F17);
    }

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 1),
      decoration: BoxDecoration(
        color: bg,
        borderRadius: BorderRadius.circular(8),
      ),
      child: Text(
        '⚖${weight.toStringAsFixed(2)}',
        style: TextStyle(fontSize: 10, fontWeight: FontWeight.w600, color: fg),
      ),
    );
  }

  Widget _iconButton({
    required IconData icon,
    required Color color,
    VoidCallback? onTap,
    double size = 16,
  }) {
    return GestureDetector(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.all(4),
        child: Icon(icon, size: size, color: color),
      ),
    );
  }

  String _statusText(String status) {
    switch (status) {
      case 'sent':
        return 'Отправлено';
      case 'delivered':
        return 'Доставлено';
      case 'partial':
        return 'Частично';
      case 'filtered':
        return 'Тема вне интересов';
      default:
        return '';
    }
  }

  /// Тултип для иконки статуса (1/2/3/4).
  String _statusTooltip(int? status) {
    switch (status) {
      case 1:
        return 'Отправлено';
      case 2:
        return 'Доставлено. Ждём прочтения';
      case 3:
        return 'Доставлено. Прочтение неизвестно';
      case 4:
        return 'Прочитано';
      default:
        return '';
    }
  }
}

/// Тип файла по расширению.
enum _FileKind { image, video, other }

/// Определяет тип файла по расширению. Нечувствительно к регистру.
_FileKind _fileKind(String fileName) {
  final lower = fileName.toLowerCase();
  if (lower.endsWith('.jpg') || lower.endsWith('.jpeg') ||
      lower.endsWith('.png') || lower.endsWith('.gif') ||
      lower.endsWith('.webp') || lower.endsWith('.bmp')) {
    return _FileKind.image;
  }
  if (lower.endsWith('.mp4') || lower.endsWith('.mov') ||
      lower.endsWith('.mkv') || lower.endsWith('.avi') ||
      lower.endsWith('.webm') || lower.endsWith('.3gp')) {
    return _FileKind.video;
  }
  return _FileKind.other;
}

/// Отображение фото: миниатюра + тап для полноэкранного просмотра.
class _PhotoView extends StatelessWidget {
  final Message message;

  const _PhotoView({required this.message});

  void _openFullscreen(BuildContext context) {
    final b64 = message.mediaBase64;
    if (b64.isEmpty) return;
    Navigator.of(context).push(
      MaterialPageRoute(
        builder: (_) => _PhotoFullscreen(photoBase64: b64),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final b64 = message.mediaBase64;
    if (b64.isEmpty) {
      return const SizedBox(
        width: 160,
        height: 120,
        child: Center(child: Text('Нет фото')),
      );
    }
    Uint8List? bytes;
    try {
      bytes = base64Decode(b64);
    } catch (_) {
      bytes = null;
    }
    if (bytes == null || bytes.isEmpty) {
      return const SizedBox(
        width: 160,
        height: 120,
        child: Center(child: Text('Ошибка фото')),
      );
    }

    return GestureDetector(
      onTap: () => _openFullscreen(context),
      child: ClipRRect(
        borderRadius: BorderRadius.circular(8),
        child: Image.memory(
          bytes,
          width: 200,
          height: 200,
          fit: BoxFit.cover,
          gaplessPlayback: true,
        ),
      ),
    );
  }
}

/// Полноэкранный просмотр фото. InteractiveViewer — zoom, pan.
class _PhotoFullscreen extends StatelessWidget {
  final String photoBase64;

  const _PhotoFullscreen({required this.photoBase64});

  @override
  Widget build(BuildContext context) {
    Uint8List? bytes;
    try {
      bytes = base64Decode(photoBase64);
    } catch (_) {
      bytes = null;
    }
    return Scaffold(
      backgroundColor: Colors.black,
      appBar: AppBar(
        backgroundColor: Colors.black,
        foregroundColor: Colors.white,
        elevation: 0,
      ),
      body: Center(
        child: bytes == null
            ? const Text(
                'Ошибка фото',
                style: TextStyle(color: Colors.white),
              )
            : InteractiveViewer(
                minScale: 1.0,
                maxScale: 5.0,
                child: Image.memory(
                  bytes,
                  fit: BoxFit.contain,
                  gaplessPlayback: true,
                ),
              ),
      ),
    );
  }
}

/// Отображение файла-изображения: миниатюра из localFilePath + fullscreen.
/// Пока файл не собран — прогресс через _FileRow.
class _ImageFileView extends StatelessWidget {
  final Message message;

  const _ImageFileView({required this.message});

  @override
  Widget build(BuildContext context) {
    final path = message.localFilePath;
    if (path.isEmpty) {
      return _FileRow(message: message, progress: 0);
    }
    final file = File(path);
    return GestureDetector(
      onTap: () {
        Navigator.of(context).push(
          MaterialPageRoute(
            builder: (_) => _ImageFileFullscreen(filePath: path),
          ),
        );
      },
      child: ClipRRect(
        borderRadius: BorderRadius.circular(8),
        child: Image.file(
          file,
          width: 200,
          height: 200,
          fit: BoxFit.cover,
          errorBuilder: (_, __, ___) => _FileRow(message: message, progress: 0),
        ),
      ),
    );
  }
}

/// Полноэкранный просмотр изображения-файла. InteractiveViewer — zoom, pan.
class _ImageFileFullscreen extends StatelessWidget {
  final String filePath;

  const _ImageFileFullscreen({required this.filePath});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: Colors.black,
      appBar: AppBar(
        backgroundColor: Colors.black,
        foregroundColor: Colors.white,
        elevation: 0,
      ),
      body: Center(
        child: InteractiveViewer(
          minScale: 1.0,
          maxScale: 5.0,
          child: Image.file(
            File(filePath),
            fit: BoxFit.contain,
            gaplessPlayback: true,
            errorBuilder: (_, __, ___) => const Text(
              'Ошибка изображения',
              style: TextStyle(color: Colors.white),
            ),
          ),
        ),
      ),
    );
  }
}

/// Отображение видео-файла: первый кадр (thumbnail) + иконка play + fullscreen.
/// Пока файл не собран — прогресс через _FileRow.
class _VideoFileView extends StatefulWidget {
  final Message message;

  const _VideoFileView({required this.message});

  @override
  State<_VideoFileView> createState() => _VideoFileViewState();
}

class _VideoFileViewState extends State<_VideoFileView> {
  Uint8List? _thumbnail;
  bool _loading = false;

  @override
  void initState() {
    super.initState();
    _generateThumbnail();
  }

  Future<void> _generateThumbnail() async {
    final path = widget.message.localFilePath;
    if (path.isEmpty) return;
    if (_loading) return;
    _loading = true;
    try {
      final bytes = await ZeroCachedThumbnail.bytes(
        ThumbnailSource.path(path),
        maxWidth: 400,
        quality: 75,
      );
      if (mounted) {
        setState(() => _thumbnail = bytes);
      }
    } catch (_) {
      // Тихо — покажем иконку без превью.
    } finally {
      _loading = false;
    }
  }

  @override
  Widget build(BuildContext context) {
    final path = widget.message.localFilePath;
    if (path.isEmpty) {
      return _FileRow(message: widget.message, progress: 0);
    }
    final thumb = _thumbnail;
    return GestureDetector(
      onTap: () {
        Navigator.of(context).push(
          MaterialPageRoute(
            builder: (_) => _VideoFileFullscreen(filePath: path),
          ),
        );
      },
      child: ClipRRect(
        borderRadius: BorderRadius.circular(8),
        child: SizedBox(
          width: 200,
          height: 200,
          child: Stack(
            fit: StackFit.expand,
            children: [
              if (thumb != null)
                Image.memory(thumb, fit: BoxFit.cover, gaplessPlayback: true)
              else
                Container(
                  color: Colors.black26,
                  child: const Center(
                    child: Icon(Icons.videocam, color: Colors.white, size: 48),
                  ),
                ),
              const Center(
                child: Icon(
                  Icons.play_circle_fill,
                  color: Colors.white,
                  size: 56,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// Полноэкранное воспроизведение видео-файла.
class _VideoFileFullscreen extends StatefulWidget {
  final String filePath;

  const _VideoFileFullscreen({required this.filePath});

  @override
  State<_VideoFileFullscreen> createState() => _VideoFileFullscreenState();
}

class _VideoFileFullscreenState extends State<_VideoFileFullscreen> {
  VideoPlayerController? _controller;
  bool _initialized = false;
  String? _error;

  @override
  void initState() {
    super.initState();
    _init();
  }

  Future<void> _init() async {
    try {
      final c = VideoPlayerController.file(File(widget.filePath));
      await c.initialize();
      if (!mounted) {
        c.dispose();
        return;
      }
      c.setLooping(false);
      await c.play();
      setState(() {
        _controller = c;
        _initialized = true;
      });
    } catch (e) {
      if (mounted) {
        setState(() => _error = e.toString());
      }
    }
  }

  @override
  void dispose() {
    _controller?.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: Colors.black,
      appBar: AppBar(
        backgroundColor: Colors.black,
        foregroundColor: Colors.white,
        elevation: 0,
      ),
      body: Center(
        child: _error != null
            ? Text(
                'Ошибка видео: $_error',
                style: const TextStyle(color: Colors.white),
              )
            : !_initialized || _controller == null
                ? const CircularProgressIndicator(color: Colors.white)
                : AspectRatio(
                    aspectRatio: _controller!.value.aspectRatio,
                    child: Stack(
                      alignment: Alignment.bottomCenter,
                      children: [
                        VideoPlayer(_controller!),
                        VideoProgressIndicator(
                          _controller!,
                          allowScrubbing: true,
                          colors: const VideoProgressColors(
                            playedColor: Color(0xFF4CAF50),
                          ),
                        ),
                        Positioned(
                          bottom: 8,
                          child: IconButton(
                            icon: Icon(
                              _controller!.value.isPlaying
                                  ? Icons.pause_circle_filled
                                  : Icons.play_circle_fill,
                              color: Colors.white,
                              size: 56,
                            ),
                            onPressed: () {
                              setState(() {
                                if (_controller!.value.isPlaying) {
                                  _controller!.pause();
                                } else {
                                  _controller!.play();
                                }
                              });
                            },
                          ),
                        ),
                      ],
                    ),
                  ),
      ),
    );
  }
}

/// Отображение файла: иконка + имя + размер + прогресс/кнопка «Открыть».
/// Пока файл не собран:
///   - получатель: «Получение N%» (по числу чанков на диске).
///   - отправитель: «Отправка N%» (по доставленным чанкам).
/// После сборки/доставки: «Открыть» / «Отправлено».
class _FileRow extends StatelessWidget {
  final Message message;
  final int progress;

  const _FileRow({required this.message, this.progress = 0});

  String _fmtSize(int bytes) {
    if (bytes < 1024) return '$bytes Б';
    if (bytes < 1024 * 1024) return '${(bytes / 1024).toStringAsFixed(1)} КБ';
    return '${(bytes / (1024 * 1024)).toStringAsFixed(1)} МБ';
  }

  Future<void> _open() async {
    final path = message.localFilePath;
    if (path.isEmpty) return;
    final result = await OpenFile.open(path);
    LogService.log('FILE: open $path → ${result.type}');
  }

  @override
  Widget build(BuildContext context) {
    final name = message.fileName.isNotEmpty ? message.fileName : 'Файл';
    final ready = message.localFilePath.isNotEmpty;

    // Состояние файла (по приоритету):
    // 1. Готов (localFilePath есть) → «Открыть».
    // 2. Сборка идёт (progress == 100, но не готов) → «Сборка…».
    // 3. Прогресс 0-99 → прогресс-бар + «N%».
    // 4. Прогресс неизвестен (progress < 0) → «Подготовка…».
    Widget statusWidget;
    final openWidget = ready
        ? GestureDetector(
            onTap: _open,
            child: const Text(
              'Открыть',
              style: TextStyle(
                fontSize: 12,
                fontWeight: FontWeight.w600,
                color: Color(0xFF4CAF50),
              ),
            ),
          )
        : null;
    final progressBar = (progress >= 0 && progress < 100)
        ? Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                '${message.isOwn ? "Отправка" : "Получение"} $progress%',
                style: const TextStyle(fontSize: 11, color: Colors.black54),
              ),
              const SizedBox(height: 3),
              ClipRRect(
                borderRadius: BorderRadius.circular(3),
                child: LinearProgressIndicator(
                  value: (progress / 100).clamp(0.0, 1.0),
                  minHeight: 4,
                  backgroundColor: Colors.grey.shade300,
                  valueColor: const AlwaysStoppedAnimation<Color>(Color(0xFF4CAF50)),
                ),
              ),
            ],
          )
        : null;

    if (openWidget != null && progressBar != null) {
      // Файл уже на диске, но передача ещё идёт: «Открыть» + бар.
      statusWidget = Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          openWidget,
          const SizedBox(height: 3),
          progressBar,
        ],
      );
    } else if (openWidget != null) {
      statusWidget = openWidget;
    } else if (progressBar != null) {
      statusWidget = progressBar;
    } else if (progress >= 100) {
      // Все чанки есть.
      // Отправитель: «Отправлено» (у него файла на диске не будет).
      // Получатель: «Сборка…» (tryAssemble идёт, скоро будет готов).
      statusWidget = Text(
        message.isOwn ? 'Отправлено' : 'Сборка…',
        style: const TextStyle(fontSize: 12, color: Colors.black54),
      );
    } else {
      // progress < 0 (неизвестно) или 0 (ещё не началось) → «Подготовка…».
      statusWidget = const Text(
        'Подготовка…',
        style: TextStyle(fontSize: 12, color: Colors.black54),
      );
    }

    return SizedBox(
      width: 220,
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            width: 36,
            height: 36,
            decoration: const BoxDecoration(
              color: Color(0xFF4CAF50),
              shape: BoxShape.circle,
            ),
            child: const Icon(Icons.insert_drive_file, color: Colors.white, size: 20),
          ),
          const SizedBox(width: 8),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  name,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w600),
                ),
                const SizedBox(height: 2),
                Text(
                  _fmtSize(message.fileSize),
                  style: const TextStyle(fontSize: 11, color: Colors.black54),
                ),
                const SizedBox(height: 4),
                statusWidget,
              ],
            ),
          ),
        ],
      ),
    );
  }
}

/// Плеер голосового сообщения.
/// Stateful — потому что держит AudioPlayer, состояние воспроизведения.
/// Первый тап — декодирует base64, пишет в temp-файл, играет.
class _VoicePlayer extends StatefulWidget {
  final Message message;

  const _VoicePlayer({required this.message});

  @override
  State<_VoicePlayer> createState() => _VoicePlayerState();
}

class _VoicePlayerState extends State<_VoicePlayer> {
  AudioPlayer? _player;
  bool _preparing = false;
  bool _playing = false;
  Duration _position = Duration.zero;
  Duration _duration = Duration.zero;
  String? _filePath;
  StreamSubscription? _posSub;
  StreamSubscription? _stateSub;
  StreamSubscription? _durSub;
  String? _error;

  @override
  void initState() {
    super.initState();
    final d = widget.message.duration;
    if (d > 0) _duration = Duration(seconds: d);
  }

  @override
  void dispose() {
    _posSub?.cancel();
    _stateSub?.cancel();
    _durSub?.cancel();
    _player?.dispose();
    super.dispose();
  }

  Future<void> _ensureFile() async {
    if (_filePath != null) return;
    final b64 = widget.message.mediaBase64;
    if (b64.isEmpty) {
      throw Exception('Пустое голосовое');
    }
    final bytes = base64Decode(b64);
    final dir = await getTemporaryDirectory();
    final path = '${dir.path}/voice_${widget.message.id}.m4a';
    final f = File(path);
    if (!await f.exists()) {
      await f.writeAsBytes(bytes);
    }
    _filePath = path;
  }

  Future<void> _toggle() async {
    if (_preparing) return;
    setState(() {
      _preparing = true;
      _error = null;
    });
    try {
      if (_player == null) {
        await _ensureFile();
        final p = AudioPlayer();
        await p.setFilePath(_filePath!);
        _player = p;
        _durSub = p.durationStream.listen((d) {
          if (d != null && mounted) {
            setState(() => _duration = d);
          }
        });
        _posSub = p.positionStream.listen((pos) {
          if (mounted) setState(() => _position = pos);
        });
        _stateSub = p.playerStateStream.listen((s) {
          if (!mounted) return;
          final playing = s.playing && s.processingState != ProcessingState.completed;
          if (_playing != playing) {
            setState(() => _playing = playing);
          }
          if (s.processingState == ProcessingState.completed) {
            p.seek(Duration.zero);
            p.pause();
          }
        });
      }

      if (_playing) {
        await _player!.pause();
      } else {
        // play() возвращает Future, который завершается только при
        // остановке/завершении. await заблокировал бы _toggle → пауза
        // не срабатывала бы. Fire-and-forget.
        unawaited(_player!.play());
      }
    } catch (e) {
      if (mounted) setState(() => _error = e.toString());
    } finally {
      if (mounted) setState(() => _preparing = false);
    }
  }

  String _fmt(Duration d) {
    final m = d.inMinutes.toString();
    final s = (d.inSeconds % 60).toString().padLeft(2, '0');
    return '$m:$s';
  }

  @override
  Widget build(BuildContext context) {
    if (_error != null) {
      return Text(
        'Ошибка голосового: $_error',
        style: const TextStyle(fontSize: 12, color: Colors.red),
      );
    }

    final total = _duration.inMilliseconds > 0
        ? _duration.inMilliseconds
        : (widget.message.duration * 1000);
    final pos = _position.inMilliseconds.clamp(0, total <= 0 ? 1 : total);
    final progress = total > 0 ? pos / total : 0.0;

    return SizedBox(
      width: 220,
      child: Row(
        children: [
          GestureDetector(
            onTap: _toggle,
            child: Container(
              width: 36,
              height: 36,
              decoration: const BoxDecoration(
                color: Color(0xFF4CAF50),
                shape: BoxShape.circle,
              ),
              child: _preparing
                  ? const Padding(
                      padding: EdgeInsets.all(10),
                      child: CircularProgressIndicator(
                        strokeWidth: 2,
                        valueColor: AlwaysStoppedAnimation<Color>(Colors.white),
                      ),
                    )
                  : Icon(
                      _playing ? Icons.pause : Icons.play_arrow,
                      color: Colors.white,
                      size: 22,
                    ),
            ),
          ),
          const SizedBox(width: 8),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                SliderTheme(
                  data: SliderTheme.of(context).copyWith(
                    trackHeight: 3,
                    thumbShape: const RoundSliderThumbShape(enabledThumbRadius: 6),
                    overlayShape: const RoundSliderOverlayShape(overlayRadius: 12),
                    activeTrackColor: const Color(0xFF4CAF50),
                    inactiveTrackColor: Colors.grey.shade300,
                    thumbColor: const Color(0xFF4CAF50),
                    overlayColor: const Color(0x334CAF50),
                  ),
                  child: Slider(
                    value: progress.clamp(0.0, 1.0),
                    onChanged: (v) async {
                      if (_player == null) return;
                      final seekTo = Duration(
                        milliseconds: (v * total).round(),
                      );
                      await _player!.seek(seekTo);
                    },
                  ),
                ),
                Text(
                  _playing || _position.inSeconds > 0
                      ? '${_fmt(_position)} / ${_fmt(_duration)}'
                      : _fmt(_duration),
                  style: const TextStyle(fontSize: 11, color: Colors.black54),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
// mobile/lib/widgets/message_bubble.dart