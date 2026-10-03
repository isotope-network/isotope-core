import 'dart:async';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../models/message.dart';
import '../providers/chat_provider.dart';
import '../utils/time_format.dart';
import '../services/api_service.dart';
import '../services/ws_service.dart';
import '../services/p2p_service.dart';
import '../services/libp2p_service.dart';
import '../services/log_service.dart';
import '../widgets/message_bubble.dart';

class ChatScreen extends StatefulWidget {
  final String nodeAddress;
  final P2PService p2pService;
  /// Имя контакта для AppBar. Пусто — фолбэк на PeerID.
  final String contactName;

  const ChatScreen({
    super.key,
    required this.nodeAddress,
    required this.p2pService,
    this.contactName = '',
  });

  @override
  State<ChatScreen> createState() => _ChatScreenState();
}

class _ChatScreenState extends State<ChatScreen> {
  final _controller = TextEditingController();
  final _scrollController = ScrollController();
  StreamSubscription? _messageSub;
  StreamSubscription? _recallSub;
  Timer? _ttlTimer;
  Timer? _draftThrottleTimer;
  VoidCallback? _providerListener;
  bool _isNearBottom = true;
  // Последнее выставленное значение FLAG_SECURE (null — ещё не выставлялось).
  bool? _lastSecureFlag;
  // Сохраняем provider — context.read в dispose невалиден.
  ChatProvider? _provider;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _setup();
    });
  }

  void _setup() {
    if (!mounted) return;

    final api = ApiService(baseUrl: 'http://${widget.nodeAddress}');
    final ws = WsService(wsUrl: 'ws://${widget.nodeAddress}/ws');

    final provider = context.read<ChatProvider>();
    _provider = provider;
    provider.configure(
      api: api,
      ws: ws,
      p2p: widget.p2pService,
      nodeIp: widget.nodeAddress,
    );
    provider.setChatOpen(true, provider.currentNodeIp);

    // Восстанавливаем черновик для этого чата (если есть).
    final peerID = provider.currentNodeIp;
    if (provider.hasDraftFor(peerID)) {
      _controller.text = provider.draftFor(peerID);
      _controller.selection = TextSelection.fromPosition(
        TextPosition(offset: _controller.text.length),
      );
    }

    _scrollController.addListener(_onScroll);
    _scrollToBottom();

    // Подписка на изменения текста — сохраняем черновик (throttle 500 мс).
    _controller.addListener(_onControllerChanged);

    _providerListener = () {
      if (mounted) {
        if (_isNearBottom) {
          _scrollToBottom();
        }
        _updateSecureFlag();
      }
    };
    provider.addListener(_providerListener!);

    // Первичное вычисление FLAG_SECURE при открытии чата.
    _updateSecureFlag();

    provider.loadMessages();

    final history = widget.p2pService.getHistory(widget.nodeAddress);
    for (final msg in history) {
      provider.addExternalMessage(msg);
    }

    widget.p2pService.resetUnread(widget.nodeAddress);

    _messageSub = widget.p2pService.onMessage.listen((data) {
      if (mounted) {
        provider.addExternalMessage(data);
        widget.p2pService.resetUnread(widget.nodeAddress);
        if (_isNearBottom) {
          _scrollToBottom();
        }
      }
    });

    _recallSub = widget.p2pService.onRecall.listen((messageId) {
      if (mounted) {
        provider.deleteMessage(messageId);
        setState(() {});
      }
    });

    _ttlTimer = Timer.periodic(const Duration(seconds: 5), (_) {
      if (mounted) {
        widget.p2pService.purgeExpiredMessages();
        provider.purgeExpired();
        setState(() {});
        _updateSecureFlag();
      }
    });

    setState(() {});
    _scrollToBottom();
  }

  /// Вызывается при каждом изменении текста в поле ввода.
  /// Throttle 500 мс: сохраняем черновик не чаще раза в 500 мс.
  void _onControllerChanged() {
    if (!mounted) return;
    _draftThrottleTimer?.cancel();
    _draftThrottleTimer = Timer(const Duration(milliseconds: 500), () {
      _flushDraft();
    });
  }

  /// Немедленно сохраняет текущий черновик.
  /// Вызывается из throttle-таймера и при dispose.
  void _flushDraft() {
    final provider = _provider;
    if (provider == null) return;
    final peerID = provider.currentNodeIp;
    if (peerID.isEmpty) return;
    provider.saveDraft(peerID, _controller.text);
  }

  void _onScroll() {
    if (_scrollController.hasClients) {
      final pos = _scrollController.position.pixels;
      final max = _scrollController.position.maxScrollExtent;
      _isNearBottom = max - pos < 50;
    }
  }

  /// Пересчитывает FLAG_SECURE: включается, если в чате есть
  /// активное TTL-сообщение с периодом от 10 секунд до 1 часа.
  /// Не вызывает setSecureFlag, если значение не изменилось.
  void _updateSecureFlag() {
    final provider = _provider;
    if (provider == null) return;
    final peerID = provider.currentNodeIp;
    final messages = provider.messagesFor(peerID);
    bool hasActiveTtl = false;
    for (final m in messages) {
      if (m.isExpired) continue;
      final s = m.ttlPeriodSeconds;
      if (s >= 10 && s <= 3600) {
        hasActiveTtl = true;
        break;
      }
    }
    LogService.log('SECURE: _updateSecureFlag hasActiveTtl=$hasActiveTtl last=$_lastSecureFlag');
    if (_lastSecureFlag == hasActiveTtl) return;
    _lastSecureFlag = hasActiveTtl;
    LibP2PService.setSecureFlag(hasActiveTtl);
  }

  @override
  void dispose() {
    // Немедленно сохраняем черновик (не ждём throttle).
    _draftThrottleTimer?.cancel();
    _flushDraft();

    final provider = _provider;
    if (provider != null && _providerListener != null) {
      provider.removeListener(_providerListener!);
    }
    _scrollController.removeListener(_onScroll);
    _controller.removeListener(_onControllerChanged);
    _controller.dispose();
    _scrollController.dispose();
    _messageSub?.cancel();
    _recallSub?.cancel();
    _ttlTimer?.cancel();

    // Снимаем FLAG_SECURE при закрытии чата.
    LogService.log('SECURE: dispose → setSecureFlag(false)');
    LibP2PService.setSecureFlag(false);

    _provider = null;
    super.dispose();
  }

  void _scrollToBottom() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (_scrollController.hasClients) {
        _scrollController.jumpTo(_scrollController.position.maxScrollExtent);
      }
    });
  }

  void _sendMessage() {
    final text = _controller.text.trim();
    if (text.isEmpty) return;

    final provider = context.read<ChatProvider>();

    // Если есть pending — блокируем (защита).
    if (provider.hasPending) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Дождитесь отправки предыдущего сообщения')),
      );
      return;
    }

    _controller.clear();
    _isNearBottom = true;
    _scrollToBottom();

    // Отправляем через provider (с учётом задержки).
    provider.sendMessage(text);
  }

  /// Строка для pending-сообщения: бабл с кругом и «Отмена».
  /// Без полупрозрачности — текст должен быть читаем.
  Widget _buildPendingRow(Message msg, ChatProvider provider) {
    final secs = provider.pendingSecondsFor(msg.id) ?? 0;
    final delay = provider.sendDelay;
    final progress = delay > 0 ? 1.0 - (secs / delay) : 1.0;

    return Dismissible(
      key: ValueKey(msg.id),
      direction: DismissDirection.endToStart,
      onDismissed: (_) {
        provider.deletePending(msg.id);
      },
      background: Container(
        color: Colors.red,
        alignment: Alignment.centerRight,
        padding: const EdgeInsets.only(right: 16),
        child: const Icon(Icons.delete, color: Colors.white),
      ),
      child: Align(
        alignment: Alignment.centerRight,
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
          child: Container(
            constraints: BoxConstraints(
              maxWidth: MediaQuery.of(context).size.width * 0.7,
            ),
            decoration: BoxDecoration(
              color: const Color(0xFFDCF8C6),
              borderRadius: const BorderRadius.only(
                topLeft: Radius.circular(16),
                topRight: Radius.circular(16),
                bottomLeft: Radius.circular(16),
                bottomRight: Radius.circular(4),
              ),
            ),
            child: Padding(
              padding: const EdgeInsets.all(10),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  const Text(
                    'Вы',
                    style: TextStyle(
                      fontSize: 12,
                      fontWeight: FontWeight.w600,
                      color: Colors.black54,
                    ),
                  ),
                  const SizedBox(height: 6),
                  Text(
                    msg.displayText,
                    style: const TextStyle(fontSize: 15),
                  ),
                  const SizedBox(height: 6),
                  Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      SizedBox(
                        width: 16,
                        height: 16,
                        child: CircularProgressIndicator(
                          value: progress.clamp(0.0, 1.0),
                          strokeWidth: 2,
                          backgroundColor: Colors.grey.shade300,
                          valueColor: const AlwaysStoppedAnimation<Color>(Colors.green),
                        ),
                      ),
                      const SizedBox(width: 6),
                      Text(
                        '$secs сек',
                        style: const TextStyle(fontSize: 11, color: Colors.black54),
                      ),
                      const SizedBox(width: 12),
                      TextButton(
                        style: TextButton.styleFrom(
                          padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 0),
                          minimumSize: const Size(0, 28),
                          tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                        ),
                        onPressed: () {
                          final text = provider.cancelPending(msg.id);
                          if (text != null) {
                            _controller.text = text;
                            _controller.selection = TextSelection.fromPosition(
                              TextPosition(offset: _controller.text.length),
                            );
                          }
                        },
                        child: const Text('Отмена', style: TextStyle(fontSize: 12)),
                      ),
                    ],
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return PopScope(
      canPop: true,
      onPopInvokedWithResult: (didPop, result) async {
        if (didPop) return;
        final provider = context.read<ChatProvider>();
        if (provider.hasPending) {
          await provider.pendingToDraft();
          if (!context.mounted) return;
          if (mounted) {
            ScaffoldMessenger.of(context).showSnackBar(
              const SnackBar(content: Text('Сообщение не отправлено. Сохранено как черновик.')),
            );
          }
        }
        if (mounted) Navigator.pop(context);
      },
      child: Scaffold(
      appBar: AppBar(
        title: Text(
          widget.contactName.isNotEmpty
              ? widget.contactName
              : 'Контакт ${widget.nodeAddress.length > 12 ? widget.nodeAddress.substring(0, 12) : widget.nodeAddress}',
          overflow: TextOverflow.ellipsis,
        ),
      ),
      body: Column(
        children: [
          if (context.watch<ChatProvider>().error != null)
            Container(
              width: double.infinity,
              color: Colors.red.shade50,
              padding: const EdgeInsets.all(8),
              child: Text(
                context.read<ChatProvider>().error!,
                style: const TextStyle(color: Colors.red),
                textAlign: TextAlign.center,
              ),
            ),
          Expanded(
            child: Consumer<ChatProvider>(
              builder: (_, provider, __) {
                final messages = provider.messagesFor(provider.currentNodeIp);
                final pending = provider.pendingMessages;
                final unreadCount = provider.unreadSnapshotFor(provider.currentNodeIp);
                final totalCount = messages.length + pending.length;

                if (totalCount == 0) {
                  return const Center(child: Text('Нет сообщений'));
                }

                return ListView.builder(
                  controller: _scrollController,
                  itemCount: totalCount,
                  itemBuilder: (_, index) {
                    // Сначала — обычные, потом — pending.
                    final isPending = index >= messages.length;
                    if (isPending) {
                      final pmsg = pending[index - messages.length];
                      return _buildPendingRow(pmsg, provider);
                    }
                    final msg = messages[index];
                    final isOwn = msg.isOwn;
                    final unreadStart = messages.length - unreadCount;
                    final showDivider = !isOwn && unreadCount > 0 && index == unreadStart;

                    // Разделитель дат: показываем, если это первое сообщение
                    // или день отличается от предыдущего.
                    bool showDateDivider = false;
                    if (index == 0) {
                      showDateDivider = true;
                    } else {
                      final prev = parseIsoLocal(messages[index - 1].time);
                      final cur = parseIsoLocal(msg.time);
                      if (prev == null || cur == null) {
                        showDateDivider = true;
                      } else {
                        final pDay = DateTime(prev.year, prev.month, prev.day);
                        final cDay = DateTime(cur.year, cur.month, cur.day);
                        showDateDivider = pDay != cDay;
                      }
                    }

                    return Column(
                      children: [
                        if (showDateDivider)
                          Container(
                            padding: const EdgeInsets.symmetric(vertical: 8),
                            alignment: Alignment.center,
                            child: Text(
                              formatDateSeparator(
                                parseIsoLocal(msg.time) ?? DateTime.now(),
                              ),
                              style: TextStyle(
                                fontSize: 12,
                                color: Colors.grey.shade600,
                                fontWeight: FontWeight.w600,
                              ),
                            ),
                          ),
                        if (showDivider)
                          Container(
                            padding: const EdgeInsets.symmetric(vertical: 4),
                            child: Row(
                              children: [
                                const Expanded(child: Divider(color: Colors.red)),
                                Padding(
                                  padding: const EdgeInsets.symmetric(horizontal: 8),
                                  child: Text(
                                    'Непрочитанные ($unreadCount)',
                                    style: const TextStyle(color: Colors.red, fontSize: 11),
                                  ),
                                ),
                                const Expanded(child: Divider(color: Colors.red)),
                              ],
                            ),
                          ),
                        Dismissible(
                          key: ValueKey(msg.id),
                          direction: DismissDirection.endToStart,
                          onDismissed: (_) {
                            provider.deleteMessage(msg.id);
                            widget.p2pService.deleteLocalMessage(widget.nodeAddress, msg.id);
                          },
                          background: Container(
                            color: Colors.red,
                            alignment: Alignment.centerRight,
                            padding: const EdgeInsets.only(right: 16),
                            child: const Icon(Icons.delete, color: Colors.white),
                          ),
                          child: MessageBubble(
                            message: msg,
                            onLike: () => provider.sendFeedback(msg.id, 1),
                            onDislike: () => provider.sendFeedback(msg.id, -1),
                          ),
                        ),
                      ],
                    );
                  },
                );
              },
            ),
          ),
          Container(
            padding: const EdgeInsets.all(8),
            decoration: BoxDecoration(
              color: Colors.white,
              border: Border(top: BorderSide(color: Colors.grey.shade200)),
            ),
            child: SafeArea(
              child: Row(
                children: [
                  Expanded(
                    child: Consumer<ChatProvider>(
                      builder: (_, provider, __) {
                        final blocked = provider.hasPending;
                        return TextField(
                          controller: _controller,
                          enabled: !blocked,
                          decoration: InputDecoration(
                            hintText: blocked ? 'Подождите…' : 'Сообщение...',
                            filled: true,
                            fillColor: Colors.grey.shade100,
                            border: OutlineInputBorder(
                              borderRadius: BorderRadius.circular(24),
                              borderSide: BorderSide.none,
                            ),
                            contentPadding: const EdgeInsets.symmetric(
                              horizontal: 16,
                              vertical: 10,
                            ),
                          ),
                          onSubmitted: (_) => _sendMessage(),
                        );
                      },
                    ),
                  ),
                  const SizedBox(width: 8),
                  Consumer<ChatProvider>(
                    builder: (_, provider, __) {
                      final blocked = provider.hasPending;
                      return CircleAvatar(
                        backgroundColor: blocked ? Colors.grey : Colors.green,
                        child: IconButton(
                          icon: const Icon(Icons.send, color: Colors.white, size: 20),
                          onPressed: blocked ? null : _sendMessage,
                        ),
                      );
                    },
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
      ),
    );
  }
}