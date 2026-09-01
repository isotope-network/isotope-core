import 'dart:async';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../models/message.dart';
import '../providers/chat_provider.dart';
import '../services/api_service.dart';
import '../services/ws_service.dart';
import '../services/p2p_service.dart';
import '../widgets/message_bubble.dart';
import 'connect_screen.dart';
import 'diagnostic_screen.dart';
import 'log_screen.dart';

class ChatScreen extends StatefulWidget {
  final String nodeAddress;
  final P2PService p2pService;

  const ChatScreen({
    super.key,
    required this.nodeAddress,
    required this.p2pService,
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
  Timer? _sendDelayTimer;
  int _sendCountdown = 5;
  bool _sendDelayActive = false;
  int _unreadCount = 0;

  final List<Map<String, dynamic>> _ttlOptions = [
    {'label': '10 секунд', 'value': 10},
    {'label': '1 минута', 'value': 60},
    {'label': '10 минут', 'value': 600},
    {'label': '1 час', 'value': 3600},
    {'label': '24 часа', 'value': 86400},
    {'label': '7 дней', 'value': 604800},
    {'label': '30 дней', 'value': 2592000},
    {'label': 'Вечно', 'value': 0},
  ];

  final List<int> _delayOptions = [1, 2, 3, 5, 10];

  int _selectedTtl = 86400;
  int _selectedDelay = 5;

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
    provider.configure(
      api: api,
      ws: ws,
      p2p: widget.p2pService,
      nodeIp: widget.nodeAddress,
    );
    provider.setTtl(_selectedTtl);

    // Запоминаем количество непрочитанных
    _unreadCount = widget.p2pService.getUnreadCount(widget.nodeAddress);

    final history = widget.p2pService.getHistory(widget.nodeAddress);
    for (final msg in history) {
      provider.addExternalMessage(msg);
    }

    // Сбрасываем счётчик
    widget.p2pService.resetUnread(widget.nodeAddress);

    _messageSub = widget.p2pService.onMessage.listen((data) {
      if (mounted) {
        provider.addExternalMessage(data);
        widget.p2pService.resetUnread(widget.nodeAddress);
        _scrollToBottom();
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
      }
    });

    setState(() {});
    _scrollToBottom();
  }

  void _openDiagnostics() {
    Navigator.push(
      context,
      MaterialPageRoute(
        builder: (_) => DiagnosticScreen(nodeAddress: widget.nodeAddress),
      ),
    );
  }

  void _openLogs() {
    Navigator.push(
      context,
      MaterialPageRoute(
        builder: (_) => const LogScreen(),
      ),
    );
  }

  @override
  void dispose() {
    _controller.dispose();
    _scrollController.dispose();
    _messageSub?.cancel();
    _recallSub?.cancel();
    _ttlTimer?.cancel();
    _sendDelayTimer?.cancel();
    super.dispose();
  }

  void _scrollToBottom() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (_scrollController.hasClients) {
        _scrollController.animateTo(
          _scrollController.position.maxScrollExtent,
          duration: const Duration(milliseconds: 300),
          curve: Curves.easeOut,
        );
      }
    });
  }

  void _sendMessage() {
    final text = _controller.text.trim();
    if (text.isEmpty) return;

    _controller.clear();
    _scrollToBottom();

    _sendDelayActive = true;
    _sendCountdown = _selectedDelay;
    _showSendDelaySnackbar(text);
  }

  void _showSendDelaySnackbar(String text) {
    final provider = context.read<ChatProvider>();
    final snackBarController = ScaffoldMessenger.of(context);
    final countdownNotifier = ValueNotifier<int>(_sendCountdown);

    final snackBar = SnackBar(
      duration: Duration(seconds: _selectedDelay),
      content: Row(
        children: [
          Expanded(
            child: ValueListenableBuilder<int>(
              valueListenable: countdownNotifier,
              builder: (_, count, __) {
                return Text(
                  'Отправка через $count сек...',
                  style: const TextStyle(fontSize: 12),
                );
              },
            ),
          ),
          TextButton(
            onPressed: () {
              _sendDelayTimer?.cancel();
              _sendDelayActive = false;
              _controller.text = text;
              snackBarController.hideCurrentSnackBar();
            },
            child: const Text(
              'ОТМЕНИТЬ',
              style: TextStyle(color: Colors.white, fontWeight: FontWeight.bold, fontSize: 12),
            ),
          ),
        ],
      ),
      behavior: SnackBarBehavior.floating,
      margin: EdgeInsets.only(
        bottom: MediaQuery.of(context).size.height * 0.15,
        left: 16,
        right: 16,
      ),
    );

    snackBarController.showSnackBar(snackBar);

    _sendDelayTimer = Timer.periodic(const Duration(seconds: 1), (timer) {
      if (!mounted || !_sendDelayActive) {
        timer.cancel();
        return;
      }

      _sendCountdown--;
      countdownNotifier.value = _sendCountdown;

      if (_sendCountdown <= 0) {
        timer.cancel();
        _sendDelayActive = false;
        provider.setTtl(_selectedTtl);
        provider.sendMessage(text);
        snackBarController.hideCurrentSnackBar();
        _scrollToBottom();
      }
    });
  }

  void _recallMessage(String messageId) {
    final provider = context.read<ChatProvider>();

    widget.p2pService.recallMessage(widget.nodeAddress, messageId);
    provider.deleteMessage(messageId);

    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(
        duration: Duration(seconds: 2),
        content: Text('Сообщение отозвано'),
      ),
    );
  }

  Future<void> _disconnect() async {
    if (mounted) {
      Navigator.pop(context);
    }
  }

  void _showTtlPicker() {
    showModalBottomSheet(
      context: context,
      builder: (ctx) {
        return ListView(
          shrinkWrap: true,
          children: [
            const Padding(
              padding: EdgeInsets.all(16),
              child: Text(
                'Исчезновение сообщения',
                style: TextStyle(fontWeight: FontWeight.bold, fontSize: 18),
              ),
            ),
            ..._ttlOptions.map((option) {
              final isSelected = option['value'] == _selectedTtl;
              return ListTile(
                title: Text(option['label']),
                leading: Icon(
                  isSelected ? Icons.radio_button_checked : Icons.radio_button_off,
                  color: isSelected ? Colors.green : Colors.grey,
                ),
                onTap: () {
                  setState(() {
                    _selectedTtl = option['value'];
                  });
                  Navigator.pop(ctx);
                },
              );
            }),
          ],
        );
      },
    );
  }

  void _showDelayPicker() {
    showModalBottomSheet(
      context: context,
      builder: (ctx) {
        return ListView(
          shrinkWrap: true,
          children: [
            const Padding(
              padding: EdgeInsets.all(16),
              child: Text(
                'Задержка отправки',
                style: TextStyle(fontWeight: FontWeight.bold, fontSize: 18),
              ),
            ),
            ..._delayOptions.map((delay) {
              final isSelected = delay == _selectedDelay;
              return ListTile(
                title: Text('$delay секунд'),
                leading: Icon(
                  isSelected ? Icons.radio_button_checked : Icons.radio_button_off,
                  color: isSelected ? Colors.green : Colors.grey,
                ),
                onTap: () {
                  setState(() {
                    _selectedDelay = delay;
                  });
                  Navigator.pop(ctx);
                },
              );
            }),
          ],
        );
      },
    );
  }

  @override
  Widget build(BuildContext context) {
    final ttlLabel = _ttlOptions.firstWhere(
      (o) => o['value'] == _selectedTtl,
      orElse: () => {'label': '24 часа', 'value': 86400},
    )['label'];

    return Scaffold(
      appBar: AppBar(
        title: Consumer<ChatProvider>(
          builder: (_, provider, __) {
            return Text('ИСО [${widget.nodeAddress}]');
          },
        ),
        actions: [
          IconButton(
            icon: const Icon(Icons.timer_outlined),
            onPressed: _showTtlPicker,
            tooltip: 'TTL: $ttlLabel',
          ),
          IconButton(
            icon: const Icon(Icons.hourglass_empty),
            onPressed: _showDelayPicker,
            tooltip: 'Задержка: $_selectedDelay сек',
          ),
          IconButton(
            icon: const Icon(Icons.bug_report),
            onPressed: _openDiagnostics,
            tooltip: 'Диагностика',
          ),
          IconButton(
            icon: const Icon(Icons.article_outlined),
            onPressed: _openLogs,
            tooltip: 'Журнал',
          ),
          IconButton(
            icon: const Icon(Icons.link_off),
            onPressed: _disconnect,
            tooltip: 'Отключиться',
          ),
        ],
      ),
      body: Column(
        children: [
          Container(
            width: double.infinity,
            color: Colors.blue.shade50,
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 2),
            child: Text(
              'TTL: $ttlLabel  |  Задержка: $_selectedDelay сек',
              style: const TextStyle(fontSize: 11),
              textAlign: TextAlign.center,
            ),
          ),
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
                final messages = provider.messages;

                if (messages.isEmpty) {
                  return const Center(child: Text('Нет сообщений'));
                }

                return ListView.builder(
                  controller: _scrollController,
                  itemCount: messages.length,
                  itemBuilder: (_, index) {
                    final msg = messages[index];
                    final isOwn = msg.senderType == 'own';
                    final unreadStart = messages.length - _unreadCount;
                    final showDivider = !isOwn && _unreadCount > 0 && index == unreadStart;

                    return Column(
                      children: [
                        if (showDivider)
                          Container(
                            padding: const EdgeInsets.symmetric(vertical: 4),
                            child: Row(
                              children: [
                                const Expanded(child: Divider(color: Colors.red)),
                                Padding(
                                  padding: const EdgeInsets.symmetric(horizontal: 8),
                                  child: Text(
                                    'Непрочитанные ($_unreadCount)',
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
                        if (isOwn && !msg.isExpired)
                          _recallButton(msg),
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
                    child: TextField(
                      controller: _controller,
                      decoration: InputDecoration(
                        hintText: 'Сообщение...',
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
                    ),
                  ),
                  const SizedBox(width: 8),
                  CircleAvatar(
                    backgroundColor: Colors.green,
                    child: IconButton(
                      icon: const Icon(Icons.send, color: Colors.white, size: 20),
                      onPressed: _sendMessage,
                    ),
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _recallButton(Message msg) {
    final sentTime = DateTime.tryParse(msg.time);
    if (sentTime == null) return const SizedBox.shrink();

    final elapsed = DateTime.now().difference(sentTime);
    if (elapsed.inSeconds > 30) return const SizedBox.shrink();

    return Align(
      alignment: Alignment.centerRight,
      child: Padding(
        padding: const EdgeInsets.only(top: 2),
        child: TextButton(
          onPressed: () => _recallMessage(msg.id),
          style: TextButton.styleFrom(
            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
            minimumSize: Size.zero,
            tapTargetSize: MaterialTapTargetSize.shrinkWrap,
          ),
          child: const Text(
            'Отозвать',
            style: TextStyle(fontSize: 11, color: Colors.red),
          ),
        ),
      ),
    );
  }
}