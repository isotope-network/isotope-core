import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'package:flutter/material.dart';
import 'package:just_audio/just_audio.dart';
import 'package:path_provider/path_provider.dart';
import '../models/message.dart';
import '../utils/time_format.dart';

class MessageBubble extends StatelessWidget {
  final Message message;
  final VoidCallback? onLike;
  final VoidCallback? onDislike;
  /// Отображаемое имя отправителя (для входящих).
  /// Пусто — используется message.sender (PeerID).
  final String? senderName;

  const MessageBubble({
    super.key,
    required this.message,
    this.onLike,
    this.onDislike,
    this.senderName,
  });

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