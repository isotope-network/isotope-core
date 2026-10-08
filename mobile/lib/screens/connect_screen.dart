// mobile/lib/screens/connect_screen.dart
import 'dart:async';
import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';
import 'package:qr_flutter/qr_flutter.dart';
import 'package:shared_preferences/shared_preferences.dart';
import '../models/node_info.dart';
import '../models/message.dart';
import '../services/mdns_service.dart';
import '../services/p2p_service.dart';
import '../services/libp2p_service.dart';
import '../services/log_service.dart';
import '../services/ethics_service.dart';
import '../services/identity_service.dart';
import '../services/network_service.dart';
import '../services/permission_service.dart';
import '../providers/chat_provider.dart';
import '../utils/time_format.dart';
import '../widgets/requests_section.dart';
import 'chat_screen.dart';
import 'log_screen.dart';
import 'qr_scan_screen.dart';
import 'settings_screen.dart';

const String DEFAULT_BOOTSTRAP_ADDR = '/ip4/186.246.31.176/tcp/9001/ws/p2p/QmR8u5YFdcKpM2onQvk7KV5qioai87aysi9JWLdV1LX1bi';
const String BOOTSTRAP_PEER_ID = 'QmR8u5YFdcKpM2onQvk7KV5qioai87aysi9JWLdV1LX1bi';

/// Вид подзаголовка контакта в списке.
/// - draft: есть черновик сообщения (зелёный).
/// - request: verified && !confirmed (оранжевый, «Отправить запрос?»).
/// - normal: последнее сообщение (серый).
enum ContactSubtitleKind { draft, request, normal }

/// Префикс для QR-кодов ISOTOPE — чтобы отличать от чужих QR.
const String ISOTOPE_QR_PREFIX = 'isotope:';

class ConnectScreen extends StatefulWidget {
  /// Если задан — при старте экрана автоматически выполнится действие:
  /// 'scan'   — открыть сканер QR.
  /// 'showQR' — показать мой QR.
  /// null     — обычный старт.
  final String? initialAction;

  const ConnectScreen({super.key, this.initialAction});

  @override
  State<ConnectScreen> createState() => _ConnectScreenState();
}

class _ConnectScreenState extends State<ConnectScreen> {
  final TextEditingController _manualController = TextEditingController(text: 'http://192.168.31.203:8081');
  final TextEditingController _bootstrapController = TextEditingController(text: DEFAULT_BOOTSTRAP_ADDR);
  final TextEditingController _multiaddrController = TextEditingController();
  final NetworkService _networkService = NetworkService();

  List<NodeInfo> _discoveredNodes = [];
  List<NodeInfo> _pendingNodes = [];
  List<String> _logs = [];
  bool _scanning = false;
  bool _connecting = false;
  bool _findingNetwork = false;
  String? _error;
  String? _status;
  String? _localIp;
  bool _serverStarted = false;
  String _localMultiaddr = '';
  String _bootstrapPeers = DEFAULT_BOOTSTRAP_ADDR;
  StreamSubscription? _nodeSub;
  StreamSubscription? _ipSub;
  StreamSubscription? _helloAckSub;
  VoidCallback? _chatListener;
  Timer? _coreLogsTimer;
  final Set<String> _coreLogsSeen = {};
  static const int _maxCoreLogsSeen = 5000;

  /// Кеш статуса верификации контактов по PeerID.
  /// Заполняется при сканировании QR и при загрузке контактов из ядра.
  final Map<String, bool> _verifiedContacts = {};

  /// Кеш статуса подтверждения контактов по PeerID.
  /// confirmed = true: обе стороны подтвердили ([CONTACT_ACCEPT] получен).
  /// Заполняется при загрузке контактов из ядра.
  final Map<String, bool> _confirmedContacts = {};

  /// PeerID, которым уже отправили [CONTACT_REQUEST] в этой сессии.
  /// Set в памяти (MVP). При перезапуске сбрасывается.
  final Set<String> _requestSent = {};

  /// PeerID удалённых контактов. Фильтр — не показывать в UI.
  /// Загружается из Go при старте. Обновляется при удалении / QR-возврате.
  final Set<String> _deletedPeers = {};

  String _myPeerId = '';
  bool _announced = false;

  /// Ключ для RequestsSection — чтобы перезагрузить список запросов.
  final GlobalKey<RequestsSectionState> _requestsKey = GlobalKey<RequestsSectionState>();

  /// Флаг: initialAction уже обработан — не повторяем.
  bool _initialActionHandled = false;

  @override
  void initState() {
    super.initState();
    LogService.log('=== ConnectScreen initState ===');

    _initAsync();

    _checkSavedNode();
    _listenToP2P();
    _startServer();
    _startNetworkMonitoring();

    _coreLogsTimer = Timer.periodic(const Duration(seconds: 5), (_) {
      _pullCoreLogs();
    });
    Future.delayed(const Duration(seconds: 3), () => _pullCoreLogs());

    // Слушаем [CONTACT_HELLO_ACK] — ChatProvider уже сделал AddContact +
    // sendContactRequest. Здесь только UI-обновление.
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      final chatProvider = context.read<ChatProvider>();

      chatProvider.helloAckStream.listen((ack) {
        if (!mounted) return;
        LogService.log('ConnectScreen: helloAck для ${ack.peerID}');
        chatProvider.loadPeerNames();
        setState(() {});
      });

      // Слушаем [CONTACT_REQUEST] — push-событие.
      // Go сохранил запрос, RequestsSection сам обновит список.
      // Здесь — SnackBar с уведомлением.
      chatProvider.contactRequestStream.listen((peerID) {
        if (!mounted) return;
        LogService.log('ConnectScreen: входящий запрос от $peerID');
        final short = peerID.length > 12 ? peerID.substring(0, 12) : peerID;
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text('Входящий запрос от $short'),
            action: SnackBarAction(
              label: 'Открыть',
              onPressed: () {
                if (mounted) setState(() {});
              },
            ),
          ),
        );
      });

      // Слушаем [CONTACT_ACCEPT] — наш запрос принят.
      // Go обновил contact.confirmed. Перечитываем контакты.
      chatProvider.contactAcceptStream.listen((peerID) {
        if (!mounted) return;
        LogService.log('ConnectScreen: [CONTACT_ACCEPT] от $peerID — перечитать контакты');
        _loadContactsFromCore();
        chatProvider.loadPeerNames();
      });

      // Слушаем peerSeen — при отправке сообщения обновляем status → alive.
      // Устраняет асимметрию: входящие обновляли статус, исходящие — нет.
      chatProvider.peerSeenStream.listen((peerID) {
        if (!mounted || peerID.isEmpty) return;
        final index = _discoveredNodes.indexWhere((n) => n.peerID == peerID);
        if (index < 0) return;
        final node = _discoveredNodes[index];
        if (node.status == NodeStatus.alive) return;
        setState(() {
          _discoveredNodes[index] = node.copyWith(
            status: NodeStatus.alive,
            lastSeen: DateTime.now(),
          );
        });
        LogService.log('ConnectScreen: peerSeen $peerID → alive');
      });

      // Слушаем openChat — тап по уведомлению → открыть чат с peerID.
      chatProvider.openChatStream.listen((peerID) {
        if (!mounted || peerID.isEmpty) return;
        LogService.log('ConnectScreen: openChat для $peerID');
        final index = _discoveredNodes.indexWhere((n) => n.peerID == peerID);
        if (index >= 0) {
          _connectToNode(_discoveredNodes[index]);
        } else {
          _findAndConnectByPeerId(peerID);
        }
      });
    });

    // Обработка initialAction — после первого кадра,
    // когда UI готов к показу диалогов/snackbar.
    if (widget.initialAction != null) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        _handleInitialAction();
      });
    }
  }

  Future<void> _handleInitialAction() async {
    if (_initialActionHandled) return;
    _initialActionHandled = true;

    // Небольшая пауза — чтобы ConnectScreen успел отрисоваться
    // и контекст был доступен для Navigator / showDialog.
    await Future.delayed(const Duration(milliseconds: 300));
    if (!mounted) return;

    switch (widget.initialAction) {
      case 'scan':
        _scanQR();
        break;
      case 'showQR':
        _showMyQR();
        break;
      default:
        break;
    }
  }

  Future<void> _initAsync() async {
    await _loadBootstrapPeers();

    if (!mounted) return;

    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) {
        final chatProvider = context.read<ChatProvider>();
        final p2p = context.read<P2PService>();
        chatProvider.setP2P(p2p);
        chatProvider.initialize(bootstrapPeers: _bootstrapPeers);
        _checkBatteryOptimization();

        _scheduleAnnounce();
        _loadContactsFromCore();
      }
    });
  }

  /// Загружает контакты из ядра и кеширует их verified-статус.
  Future<void> _loadContactsFromCore() async {
    // Загружаем список удалённых (для фильтра).
    try {
      final deleted = await LibP2PService.getDeletedPeers();
      if (mounted && deleted.isNotEmpty) {
        _deletedPeers.clear();
        _deletedPeers.addAll(deleted);
      }
    } catch (_) {}

    // Retry: Go-ядро может стартовать позже Flutter-экрана.
    // До 5 попыток с интервалом 2 сек.
    for (int attempt = 0; attempt < 5; attempt++) {
      try {
        final contacts = await LibP2PService.getContacts();
        if (!mounted) return;
        if (contacts.isNotEmpty) {
          for (final c in contacts) {
            final peerID = c['peerID'] as String? ?? '';
            final verified = c['verified'] as bool? ?? false;
            final confirmed = c['confirmed'] as bool? ?? false;
            if (peerID.isNotEmpty) {
              _verifiedContacts[peerID] = verified;
              _confirmedContacts[peerID] = confirmed;
            }
          }
          if (mounted) {
            await context.read<ChatProvider>().loadPeerNames();
          }
          setState(() {});
          LogService.log('ConnectScreen: загружено контактов из ядра: ${_verifiedContacts.length} (попытка ${attempt + 1})');
          return;
        }
        // Пусто — попробуем через 2 сек (ядро ещё стартует).
        await Future.delayed(const Duration(seconds: 2));
      } catch (e) {
        LogService.log('ConnectScreen: ошибка загрузки контактов (попытка ${attempt + 1}): $e');
        await Future.delayed(const Duration(seconds: 2));
      }
    }
    LogService.log('ConnectScreen: не удалось загрузить контакты после 5 попыток');
  }

  void _scheduleAnnounce() {
    Future.delayed(const Duration(seconds: 5), () async {
      if (!mounted) return;

      for (int i = 0; i < 6; i++) {
        if (_localIp != null && _localIp!.isNotEmpty && _myPeerId.isNotEmpty) {
          break;
        }
        if (_myPeerId.isEmpty) {
          try {
            final status = await LibP2PService.getStatus();
            _myPeerId = status['id'] as String? ?? '';
          } catch (_) {}
        }
        await Future.delayed(const Duration(seconds: 5));
      }

      if (_localIp != null && _localIp!.isNotEmpty && _myPeerId.isNotEmpty) {
        _sendAnnounce();
      } else {
        LogService.log('ANNOUNCE: не дождались _localIp или PeerID (ip=$_localIp, peer=$_myPeerId)');
      }
    });
  }

  /// Отправить ANNOUNCE на bootstrap (список multiaddr).
  Future<void> _sendAnnounce() async {
    if (_announced) return;
    if (_localIp == null || _localIp!.isEmpty) return;
    if (_myPeerId.isEmpty) return;

    final multiaddrs = <String>[
      '/ip4/$_localIp/tcp/9001/ws/p2p/$_myPeerId',
    ];

    try {
      final result = await LibP2PService.announce(multiaddrs);
      if (result.containsKey('error')) {
        LogService.log('ANNOUNCE: ошибка: ${result['error']}');
        return;
      }
      _announced = true;
      LogService.log('ANNOUNCE: отправлено ${multiaddrs.length} адресов');
    } catch (e) {
      LogService.log('ANNOUNCE: исключение: $e');
    }
  }

  Future<void> _pullCoreLogs() async {
    try {
      final coreLogs = await LibP2PService.getCoreLogs();
      for (final line in coreLogs) {
        if (line.isEmpty) continue;
        if (_coreLogsSeen.contains(line)) continue;
        _coreLogsSeen.add(line);
        LogService.log('CORE: $line');
      }
      // Ограничиваем Set — защита от роста
      if (_coreLogsSeen.length > _maxCoreLogsSeen) {
        _coreLogsSeen.clear();
      }
    } catch (_) {}
  }

  void _syncDiscoveredNodesFromP2P() {
    try {
      final p2p = context.read<P2PService>();
      int changed = 0;
      for (final node in p2p.discoveredNodes) {
        if (node.peerID == BOOTSTRAP_PEER_ID) continue;
        if (node.peerID.isNotEmpty && _deletedPeers.contains(node.peerID)) continue; // удалён
        // Upsert: если уже есть — ЗАМЕНИТЬ (обновить статус), не игнорировать.
        final idx = _discoveredNodes.indexWhere((n) => n.key == node.key);
        if (idx >= 0) {
          _discoveredNodes[idx] = node;
        } else {
          _discoveredNodes.add(node);
        }
        changed++;
      }
      if (changed > 0 && mounted) {
        LogService.log('ConnectScreen: синхронизировано контактов: $changed');
        setState(() {
          _status = 'Контактов: ${_discoveredNodes.length}';
        });
      }
    } catch (e) {
      LogService.log('ConnectScreen: sync error: $e');
    }
  }

  Future<void> _checkBatteryOptimization() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final alreadyAsked = prefs.getBool('battery_opt_asked') ?? false;
      if (alreadyAsked) return;

      final ignoring = await LibP2PService.isIgnoringBatteryOptimizations();
      if (ignoring) {
        await prefs.setBool('battery_opt_asked', true);
        return;
      }

      if (!mounted) return;

      await showDialog(
        context: context,
        barrierDismissible: false,
        builder: (ctx) => AlertDialog(
          title: const Text('Фоновая работа'),
          content: const Text(
            'ISOTOPE — это P2P-сеть. Чтобы сообщения приходили, когда приложение свёрнуто, '
            'разрешите работу в фоне.\n\n'
            'Если системное окно не открылось — зайдите в:\n'
            'Настройки → Батарея → Запуск приложений → ISOTOPE\n'
            'и включите «Автозапуск» + «Работа в фоне».\n\n'
            'Или нажмите «Настройки приложения» ниже.',
          ),
          actions: [
            TextButton(
              onPressed: () async {
                await prefs.setBool('battery_opt_asked', true);
                if (ctx.mounted) Navigator.pop(ctx);
              },
              child: const Text('Позже'),
            ),
            TextButton(
              onPressed: () async {
                if (ctx.mounted) Navigator.pop(ctx);
                await LibP2PService.requestIgnoreBatteryOptimizations();
              },
              child: const Text('Разрешить'),
            ),
            TextButton(
              onPressed: () async {
                await prefs.setBool('battery_opt_asked', true);
                if (ctx.mounted) Navigator.pop(ctx);
                await PermissionService.openAppSettings();
              },
              child: const Text('Настройки приложения'),
            ),
          ],
        ),
      );
    } catch (e) {
      LogService.log('Battery opt check: $e');
    }
  }

  Future<void> _loadBootstrapPeers() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final saved = prefs.getString('bootstrap_peers') ?? DEFAULT_BOOTSTRAP_ADDR;
      _bootstrapPeers = saved;
      _bootstrapController.text = saved;
      if (mounted) setState(() {});
    } catch (e) {
      LogService.log('Bootstrap: ERROR: $e');
    }
  }

  Future<void> _saveBootstrapPeers(String value) async {
    try {
      final prefs = await SharedPreferences.getInstance();
      await prefs.setString('bootstrap_peers', value.trim());
      _bootstrapPeers = value.trim();
    } catch (e) {
      LogService.log('Bootstrap: ERROR: $e');
    }
  }

  bool _isBootstrapPeer(String addr) {
    return addr.contains(BOOTSTRAP_PEER_ID);
  }

  Future<void> _findPeersViaNetwork() async {
    setState(() {
      _findingNetwork = true;
      _status = 'Поиск контактов через сеть...';
      _error = null;
    });

    try {
      final response = await LibP2PService.findPeersViaNetwork();
      if (response.containsKey('error')) {
        setState(() {
          _error = 'Ошибка поиска: ${response['error']}';
          _status = null;
        });
        return;
      }

      final addrs = response['addrs'] as List<dynamic>? ?? [];
      for (final addr in addrs) {
        final addrStr = addr.toString();

        if (_isBootstrapPeer(addrStr)) {
          continue;
        }

        final parts = addrStr.split('/');
        final ipIndex = parts.indexOf('ip4');
        if (ipIndex >= 0 && ipIndex + 1 < parts.length) {
          final ip = parts[ipIndex + 1];
          if (ip != _localIp && !ip.startsWith('127.')) {
            final node = NodeInfo(
              peerID: addrStr.contains('/p2p/') ? addrStr.split('/p2p/').last : '',
              knownMultiaddrs: [addrStr],
              lastSeen: DateTime.now(),
              status: NodeStatus.alive,
            );
            _addNode(node);
          }
        }
      }

      setState(() {
        _status = 'Контактов: ${_discoveredNodes.length}';
        _error = null;
      });
    } catch (e) {
      setState(() {
        _error = 'Ошибка: $e';
        _status = null;
      });
    } finally {
      if (mounted) {
        setState(() => _findingNetwork = false);
      }
    }
  }

  void _startNetworkMonitoring() {
    _networkService.startMonitoring();
    _ipSub = _networkService.onIpChanged.listen((newIp) {
      if (newIp != null && newIp != _localIp) {
        final p2p = context.read<P2PService>();
        if (!p2p.canRestart) return;
        p2p.markRestart();
        _handleNetworkChange(newIp);
      }
    });
  }

  Future<void> _handleNetworkChange(String newIp) async {
    setState(() {
      _localIp = newIp;
      _status = 'Смена сети: $newIp';
    });

    final p2p = context.read<P2PService>();
    await p2p.stopAnnounce();
    p2p.stopNsdDiscovery();

    final shortId = newIp.replaceAll('.', '');
    await p2p.announceNative('ISOTOPE-$shortId', newIp);
    p2p.startNsdDiscovery();

    _announced = false;
    _sendAnnounce();

    setState(() {
      _status = 'Телефон-узел: $newIp:8081';
      _error = null;
    });
  }

  Future<void> _startServer() async {
    final p2p = context.read<P2PService>();

    if (!_serverStarted) {
      final ip = await p2p.startHttpServer();
      _serverStarted = true;
      if (mounted) {
        setState(() {
          _localIp = ip;
          if (ip != null) _status = 'Телефон-узел: $ip:8081';
        });
      }
      await Future.delayed(const Duration(seconds: 5));
    }

    for (final node in _pendingNodes) {
      _addNode(node);
    }
    _pendingNodes.clear();

    final shortId = _localIp?.replaceAll('.', '') ?? 'node';
    await p2p.announceNative('ISOTOPE-$shortId', _localIp ?? '');
    p2p.startNsdDiscovery();

    _syncDiscoveredNodesFromP2P();

    Future.delayed(const Duration(seconds: 10), () {
      if (mounted) p2p.startNsdDiscovery();
    });
  }

  void _listenToP2P() {
    final p2p = context.read<P2PService>();
    final chatProvider = context.read<ChatProvider>();

    _chatListener = () {
      if (mounted) setState(() {});
    };
    chatProvider.addListener(_chatListener!);

    _nodeSub = p2p.onNodeFound.listen((node) {
      if (node.peerID == BOOTSTRAP_PEER_ID) return;

      if (_localIp == null) {
        // Upsert: если узел уже в pending — ЗАМЕНИТЬ, не игнорировать.
        // Иначе alive теряется, если сначала был unknown от _loadNodes.
        final idx = _pendingNodes.indexWhere((n) => n.key == node.key);
        if (idx >= 0) {
          _pendingNodes[idx] = node;
        } else {
          _pendingNodes.add(node);
        }
      } else {
        _addNode(node);
      }
    });

    p2p.onLog.listen((log) {
      if (mounted) {
        setState(() => _logs.add(log));
      }
    });

    p2p.onUnread.listen((senderIp) {
      if (mounted) setState(() {});
    });
  }

  void _addNode(NodeInfo node) {
    if (_localIp == null) return;
    if (node.peerID == BOOTSTRAP_PEER_ID) return;
    if (node.peerID.isNotEmpty && _deletedPeers.contains(node.peerID)) return; // удалён — не показываем
    if (node.currentAddress.startsWith('BLE:')) return;
    if (node.currentAddress.contains(_localIp!)) return;
    if (node.currentAddress.startsWith('127.') || node.currentAddress.startsWith('localhost')) return;

    if (mounted) {
      final existingIndex = _discoveredNodes.indexWhere((n) => n.key == node.key);
      if (existingIndex >= 0) {
        _discoveredNodes[existingIndex] = node;
      } else {
        _discoveredNodes.add(node);
      }
      setState(() {
        _connecting = false;
        _scanning = false;
        _error = null;
        _status = 'Контактов: ${_discoveredNodes.length}';
      });
    }
  }

  Future<void> _checkSavedNode() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final saved = prefs.getString('node_address');
      LogService.log('Saved node address: ${saved ?? "нет"}');
    } catch (e) {
      LogService.log('Saved: ERROR: $e');
    }
  }

  Future<void> _scanNearby() async {
    final ok = await PermissionService.request(PermissionService.nearby);
    if (!ok) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: const Text('Разрешите Bluetooth и геолокацию, чтобы искать рядом'),
            action: SnackBarAction(
              label: 'Настройки',
              onPressed: () => PermissionService.openAppSettings(),
            ),
          ),
        );
      }
      return;
    }

    final p2p = context.read<P2PService>();
    setState(() {
      _scanning = true;
      _discoveredNodes.clear();
      _status = 'Поиск контактов рядом...';
    });

    p2p.stopNsdDiscovery();
    p2p.startNsdDiscovery();

    final shortId = _localIp?.replaceAll('.', '') ?? 'node';
    p2p.announceNative('ISOTOPE-$shortId', _localIp ?? '');

    _syncDiscoveredNodesFromP2P();

    Future.delayed(const Duration(seconds: 3), () {
      if (mounted) {
        setState(() {
          _scanning = false;
          _status = _discoveredNodes.isEmpty ? 'Нет контактов' : 'Контактов: ${_discoveredNodes.length}';
        });
      }
    });
  }

  Future<void> _connectToNode(NodeInfo node) async {
    final p2p = context.read<P2PService>();
    setState(() => _connecting = true);

    var clean = node.currentAddress.trim();
    clean = clean.replaceFirst('http://', '');
    clean = clean.replaceFirst('https://', '');
    clean = clean.replaceFirst('ws://', '');
    clean = clean.replaceFirst('wss://', '');

    final prefs = await SharedPreferences.getInstance();
    await prefs.setString('node_address', clean);

    if (mounted) {
      setState(() => _connecting = false);

      await Navigator.push(
        context,
        MaterialPageRoute(
          builder: (_) => ChatScreen(
            nodeAddress: clean,
            p2pService: p2p,
            contactName: _displayName(node),
          ),
        ),
      );

      if (mounted) {
        final chatProvider = context.read<ChatProvider>();
        chatProvider.setChatOpen(false, '');
        LogService.log('ConnectScreen: возврат из чата → setChatOpen(false)');
        setState(() {});
      }
    }
  }

  void _showNodeAddresses(NodeInfo node) {
    showModalBottomSheet(
      context: context,
      builder: (ctx) {
        return ListView(
          shrinkWrap: true,
          children: [
            Padding(
              padding: const EdgeInsets.all(16),
              child: Text('Адреса контакта', style: TextStyle(fontWeight: FontWeight.bold, fontSize: 18)),
            ),
            ...node.addresses.map((addr) {
              final isCurrent = addr == node.currentAddress;
              return ListTile(
                leading: Icon(isCurrent ? Icons.check_circle : Icons.circle_outlined, color: isCurrent ? Colors.green : Colors.grey),
                title: Text(addr),
                subtitle: isCurrent ? Text('Текущий') : null,
                onTap: () {
                  Navigator.pop(ctx);
                  _connectToNode(NodeInfo(peerID: node.peerID, knownMultiaddrs: [addr], lastSeen: DateTime.now()));
                },
              );
            }),
          ],
        );
      },
    );
  }

  /// Долгий тап на контакте — bottom sheet: Открыть чат, Переименовать, Удалить.
  Future<void> _showContactActions(NodeInfo node) async {
    try {
    final peerID = node.peerID;
    if (peerID.isEmpty) return;

    final displayName = _displayName(node);
    final action = await showModalBottomSheet<String>(
      context: context,
      builder: (ctx) => SafeArea(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Padding(
              padding: const EdgeInsets.all(16),
              child: Text(
                displayName,
                style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 16),
                overflow: TextOverflow.ellipsis,
              ),
            ),
            const Divider(height: 1),
            ListTile(
              leading: const Icon(Icons.chat_bubble_outline),
              title: const Text('Открыть чат'),
              onTap: () => Navigator.pop(ctx, 'chat'),
            ),
            ListTile(
              leading: const Icon(Icons.edit_outlined),
              title: const Text('Переименовать'),
              onTap: () => Navigator.pop(ctx, 'rename'),
            ),
            ListTile(
              leading: const Icon(Icons.delete_outline, color: Colors.red),
              title: const Text('Удалить', style: TextStyle(color: Colors.red)),
              onTap: () => Navigator.pop(ctx, 'delete'),
            ),
            const SizedBox(height: 8),
          ],
        ),
      ),
    );

    if (!mounted || action == null) return;

    switch (action) {
      case 'chat':
        await _connectToNode(node);
        break;
      case 'rename':
        await _renameContact(node);
        break;
      case 'delete':
        await _deleteContact(node);
        break;
    }
    } catch (e) {
      LogService.log('ContactActions: EXCEPTION: $e');
    }
  }

  /// Диалог «Переименовать контакт». Предзаполнено, текст выделен.
  Future<void> _renameContact(NodeInfo node) async {
    final peerID = node.peerID;
    if (peerID.isEmpty) return;

    final current = _displayName(node);
    final controller = TextEditingController(text: current);
    controller.selection = TextSelection(
      baseOffset: 0,
      extentOffset: current.length,
    );

    final newName = await showDialog<String>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Переименовать контакт'),
        content: TextField(
          controller: controller,
          autofocus: true,
          maxLength: 100,
          decoration: const InputDecoration(
            hintText: 'Имя контакта',
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, null),
            child: const Text('Отмена'),
          ),
          TextButton(
            onPressed: () {
              Navigator.pop(ctx, controller.text.trim());
            },
            child: const Text('Сохранить'),
          ),
        ],
      ),
    );

    if (!mounted || newName == null || newName == current) return;

    final result = await LibP2PService.renameContact(
      peerID: peerID,
      localName: newName,
    );
    if (result.containsKey('error')) {
      LogService.log('RenameContact: ошибка: ${result['error']}');
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Не удалось переименовать: ${result['error']}')),
        );
      }
      return;
    }
    LogService.log('RenameContact: $peerID → "$newName"');
    await _loadContactsFromCore();
    if (mounted) {
      context.read<ChatProvider>().setPeerName(peerID, newName);
      setState(() {});
    }
  }

  /// Диалог «Удалить контакт?». У собеседника остаётся.
  Future<void> _deleteContact(NodeInfo node) async {
    final peerID = node.peerID;
    if (peerID.isEmpty) return;

    final displayName = _displayName(node);
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text('Удалить «$displayName»?'),
        content: Text(
          'Контакт и вся переписка будут удалены.\n'
          'Если $displayName напишет снова — вы не увидите его сообщения.\n'
          'Чтобы вернуть — попросите новый QR.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: const Text('Отмена'),
          ),
          TextButton(
            onPressed: () => Navigator.pop(ctx, true),
            child: const Text('Удалить', style: TextStyle(color: Colors.red)),
          ),
        ],
      ),
    );

    if (!mounted || ok != true) return;

    final result = await LibP2PService.removeContact(peerID: peerID);
    if (result.containsKey('error')) {
      final err = result['error']?.toString() ?? '';
      // "contact not found" — контакт уже нет в Go-хранилище.
      // Это не ошибка для UI — просто удаляем из локального списка.
      if (!err.contains('not found')) {
        LogService.log('RemoveContact: ошибка: $err');
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text('Не удалось удалить: $err')),
          );
        }
        return;
      }
      LogService.log('RemoveContact: $peerID отсутствует в Go — удаляю локально');
    } else {
      LogService.log('RemoveContact: $peerID удалён в Go');
    }

    // Удаляем переписку из ChatProvider.
    if (mounted) {
      final chatProvider = context.read<ChatProvider>();
      await chatProvider.removePeerMessages(peerID);
    }

    if (mounted) {
      setState(() {
        _discoveredNodes.removeWhere((n) => n.peerID == peerID);
        _verifiedContacts.remove(peerID);
        _confirmedContacts.remove(peerID);
        _requestSent.remove(peerID);
        _deletedPeers.add(peerID); // фильтр — не показывать после перезапуска
      });
    }
  }
  /// Настройки — открывает экран SettingsScreen.
  /// Передаёт колбэки для системных пунктов (логика остаётся здесь).
  void _showSettingsDialog() {
    Navigator.push(
      context,
      MaterialPageRoute(
        builder: (_) => SettingsScreen(
          onConnection: _showConnectionDialog,
          onManualCode: _showManualCodeDialog,
        ),
      ),
    );
  }

  

  /// Подключение — диалог с адресом подключения (bootstrap).
  void _showConnectionDialog() {
    showDialog(
      context: context,
      builder: (ctx) {
        return AlertDialog(
          title: const Text('Подключение'),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(
                controller: _bootstrapController,
                decoration: const InputDecoration(
                  hintText: '/ip4/.../tcp/9001/ws/p2p/Qm...',
                  labelText: 'Адрес подключения',
                ),
                maxLines: 3,
                minLines: 1,
              ),
              const SizedBox(height: 8),
              const Text('Оставьте пустым, если не нужен.', style: TextStyle(fontSize: 12, color: Colors.grey)),
            ],
          ),
          actions: [
            TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('Отмена')),
            TextButton(
              onPressed: () {
                _saveBootstrapPeers(_bootstrapController.text);
                Navigator.pop(ctx);
                if (mounted) setState(() {});
              },
              child: const Text('Сохранить'),
            ),
          ],
        );
      },
    );
  }

  /// Ввод кода контакта вручную — для продвинутых (отладка).
  void _showManualCodeDialog() {
    showDialog(
      context: context,
      builder: (ctx) {
        return AlertDialog(
          title: const Text('Ввести код контакта'),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(
                controller: _multiaddrController,
                decoration: const InputDecoration(
                  hintText: 'Qm... или /ip4/.../p2p/Qm...',
                  labelText: 'Код контакта',
                ),
                maxLines: 3,
                minLines: 1,
              ),
              const SizedBox(height: 8),
              const Text(
                'Попросите друга показать код.',
                style: TextStyle(fontSize: 12, color: Colors.grey),
              ),
            ],
          ),
          actions: [
            TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('Отмена')),
            TextButton(
              onPressed: () {
                final addr = _multiaddrController.text.trim();
                Navigator.pop(ctx);
                if (addr.isNotEmpty) {
                  if (addr.startsWith('/')) {
                    _connectViaMultiaddr(addr);
                  } else {
                    _findAndConnectByPeerId(addr);
                  }
                }
              },
              child: const Text('Подключиться'),
            ),
          ],
        );
      },
    );
  }

  Future<void> _showMyQR() async {
    try {
      // Этап 4.1: пробуем получить новый формат QR (JSON с E2E-ключом).
      // Fallback на старый формат — если что-то не так.
      String qrData = '';

      final jsonData = await LibP2PService.getMyQRData();
      if (jsonData.isNotEmpty && !jsonData.contains('"error"') && jsonData.startsWith('{')) {
        qrData = '$ISOTOPE_QR_PREFIX$jsonData';
        LogService.log('QR: используется формат v:1 (JSON с E2E)');
      } else {
        // Fallback: старый формат — только PeerID.
        if (_myPeerId.isEmpty) {
          final status = await LibP2PService.getStatus();
          _myPeerId = status['id'] as String? ?? '';
        }
        if (_myPeerId.isEmpty) {
          if (mounted) {
            ScaffoldMessenger.of(context).showSnackBar(
              const SnackBar(content: Text('Код ещё не готов, подождите')),
            );
          }
          return;
        }
        qrData = '$ISOTOPE_QR_PREFIX$_myPeerId';
        LogService.log('QR: используется старый формат v:0 (только PeerID)');
      }

      if (!mounted) return;
      showDialog(
        context: context,
        builder: (ctx) {
          return AlertDialog(
            title: const Text('Мой QR (контакт)'),
            content: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Container(
                  width: 220,
                  height: 220,
                  color: Colors.white,
                  padding: const EdgeInsets.all(8),
                  child: CustomPaint(
                    painter: QrPainter(
                      data: qrData,
                      version: QrVersions.auto,
                      errorCorrectionLevel: QrErrorCorrectLevel.M,
                      emptyColor: Colors.white,
                    ),
                  ),
                ),
                const SizedBox(height: 12),
                const Text(
                  'Покажите этот QR другу',
                  style: TextStyle(fontSize: 13, fontWeight: FontWeight.bold),
                ),
                const SizedBox(height: 6),
                const Text(
                  'Ваш код для добавления',
                  style: TextStyle(fontSize: 11, color: Colors.grey),
                ),
              ],
            ),
            actions: [
              TextButton(
                onPressed: () {
                  Clipboard.setData(ClipboardData(text: qrData));
                  Navigator.pop(ctx);
                  ScaffoldMessenger.of(context).showSnackBar(
                    const SnackBar(content: Text('Код скопирован')),
                  );
                },
                child: const Text('Копировать'),
              ),
              TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('Закрыть')),
            ],
          );
        },
      );
    } catch (e) {
      LogService.log('QR show error: $e');
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Ошибка: $e')),
        );
      }
    }
  }

  Future<void> _scanQR() async {
    try {
      final ok = await PermissionService.request(PermissionService.camera);
      if (!ok) {
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(
              content: const Text('Разрешите камеру, чтобы сканировать QR'),
              action: SnackBarAction(
                label: 'Настройки',
                onPressed: () => PermissionService.openAppSettings(),
              ),
            ),
          );
        }
        return;
      }

      final result = await Navigator.push(
        context,
        MaterialPageRoute(builder: (_) => const QRScanScreen()),
      );
      if (result == null || result is! String || result.isEmpty) return;

      var code = result.trim();

      if (code.startsWith(ISOTOPE_QR_PREFIX)) {
        code = code.substring(ISOTOPE_QR_PREFIX.length);
      }

      String peerId = '';
      String ed25519Pub = '';
      String x25519Pub = '';
      String signature = '';
      bool readEnabled = true;
      String displayNameFromQR = '';

      // Этап 4.3: новый формат — JSON с версией.
      if (code.startsWith('{')) {
        try {
          final json = jsonDecode(code) as Map<String, dynamic>;
          final v = json['v'] as int? ?? 0;
          if (v >= 1) {
            peerId = (json['peerID'] as String?) ?? '';
            ed25519Pub = (json['ed25519_pub'] as String?) ?? '';
            x25519Pub = (json['x25519_pub'] as String?) ?? '';
            signature = (json['signature'] as String?) ?? '';
            // read_enabled — опционально. Дефолт true (обратная совместимость).
            readEnabled = (json['read_enabled'] as bool?) ?? true;
            // display_name — представление владельца QR.
            displayNameFromQR = (json['display_name'] as String?) ?? '';
            LogService.log('QR: распознан формат v:$v, peerID=$peerId, ed25519=${ed25519Pub.isNotEmpty ? "есть" : "нет"}, x25519=${x25519Pub.isNotEmpty ? "есть" : "нет"}, signature=${signature.isNotEmpty ? "есть" : "нет"}, display_name=$displayNameFromQR, read_enabled=$readEnabled');
          }
        } catch (e) {
          LogService.log('QR: ошибка парсинга JSON: $e');
        }
      }
      // Обратная совместимость: старый формат — только PeerID.
      if (peerId.isEmpty) {
        if (code.contains('/p2p/')) {
          code = code.split('/p2p/').last;
        }
        code = code.trim();
        final peerIdMatch = RegExp(r'[A-Za-z0-9]+').firstMatch(code);
        if (peerIdMatch != null) {
          peerId = peerIdMatch.group(0)!;
          LogService.log('QR: распознан старый формат, peerID=$peerId');
        }
      }

      if (peerId.isEmpty) {
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text('Не удалось распознать код')),
          );
        }
        return;
      }

      // Проверка «свой QR» — берём PeerID напрямую из Go,
      // не полагаемся на _myPeerId (может быть пуст до первого _sendAnnounce).
      String myPeerIdNow = _myPeerId;
      if (myPeerIdNow.isEmpty) {
        try {
          final status = await LibP2PService.getStatus();
          myPeerIdNow = status['id'] as String? ?? '';
          _myPeerId = myPeerIdNow;
        } catch (_) {}
      }
      if (myPeerIdNow.isNotEmpty && peerId == myPeerIdNow) {
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text('Это ваш собственный код')),
          );
        }
        return;
      }

      // Пользователь явно добавляет контакт — снять с «удалённых».
      if (_deletedPeers.contains(peerId)) {
        try {
          await LibP2PService.removeFromDeleted(peerID: peerId);
          _deletedPeers.remove(peerId);
        } catch (_) {}
      }

      // Сохраняем контакт (если есть ключи).
      bool contactVerified = false;
      if (ed25519Pub.isNotEmpty && x25519Pub.isNotEmpty) {
        final saveResult = await LibP2PService.addContact(
          peerID: peerId,
          ed25519Pub: ed25519Pub,
          x25519Pub: x25519Pub,
          signature: signature,
          localName: '',
          remoteName: displayNameFromQR,
          readEnabled: readEnabled,
        );
        if (saveResult.containsKey('error')) {
          LogService.log('QR: не удалось сохранить контакт: ${saveResult['error']}');
        } else {
          contactVerified = saveResult['verified'] == true;
          _verifiedContacts[peerId] = contactVerified;
          _confirmedContacts[peerId] = false;
          LogService.log('QR: контакт сохранён peerID=$peerId, verified=$contactVerified, confirmed=false');
          if (mounted) {
            ScaffoldMessenger.of(context).showSnackBar(
              SnackBar(
                content: Text(contactVerified
                    ? 'Контакт проверен'
                    : 'Контакт не проверен'),
              ),
            );
          }
        }
      }

      await _findAndConnectByPeerId(peerId);
    } catch (e) {
      LogService.log('QRScan: ERROR: $e');
      setState(() {
        _error = 'Ошибка: $e';
        _connecting = false;
      });
    }
  }

  /// Универсальный поиск по PeerID + подключение через список multiaddr.
  Future<void> _findAndConnectByPeerId(String peerId) async {
    setState(() {
      _connecting = true;
      _status = 'Поиск контакта...';
      _error = null;
    });

    try {
      final findResult = await LibP2PService.findPeerByID(peerId);
      if (findResult.containsKey('error')) {
        setState(() {
          _error = 'Не удалось найти: ${findResult['error']}';
          _status = null;
          _connecting = false;
        });
        return;
      }

      final rawAddrs = findResult['multiaddrs'];
      final multiaddrs = <String>[];
      if (rawAddrs is List) {
        for (final a in rawAddrs) {
          final s = a.toString().trim();
          if (s.isNotEmpty) multiaddrs.add(s);
        }
      }

      if (multiaddrs.isEmpty) {
        setState(() {
          _error = 'Адрес не найден';
          _status = null;
          _connecting = false;
        });
        return;
      }

      final connectResult = await LibP2PService.connectToPeerWithFallback(multiaddrs);
      if (connectResult.containsKey('error')) {
        setState(() {
          _error = 'Ошибка подключения: ${connectResult['error']}';
          _status = null;
          _connecting = false;
        });
        return;
      }

      final usedAddr = connectResult['used'] as String? ?? multiaddrs.first;
      final shortId = peerId.length > 12 ? peerId.substring(0, 12) : peerId;

      final node = NodeInfo(
        peerID: peerId,
        knownMultiaddrs: [usedAddr],
        lastSeen: DateTime.now(),
        status: NodeStatus.alive,
      );
      _addNode(node);

      setState(() {
        _status = 'Подключено к контакту $shortId';
        _error = null;
        _connecting = false;
      });
    } catch (e) {
      LogService.log('FindAndConnect: ERROR: $e');
      setState(() {
        _error = 'Ошибка: $e';
        _connecting = false;
      });
    }
  }

  Future<void> _connectViaMultiaddr(String multiaddr) async {
    final clean = multiaddr.trim();
    try {
      final response = await LibP2PService.connectToPeer(clean);
      if (response.containsKey('error')) {
        setState(() => _error = 'Ошибка подключения: ${response['error']}');
        return;
      }
      setState(() {
        _status = 'Подключено к контакту';
        _error = null;
      });
    } catch (e) {
      setState(() => _error = 'Ошибка: $e');
    }
  }

  /// Добавить контакт — bottom sheet. Три опции.
  void _showAddContactDialog() {
    showModalBottomSheet(
      context: context,
      builder: (ctx) {
        return SafeArea(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              const Padding(
                padding: EdgeInsets.all(16),
                child: Text(
                  'Добавить контакт',
                  style: TextStyle(fontWeight: FontWeight.bold, fontSize: 18),
                ),
              ),
              ListTile(
                leading: const Icon(Icons.qr_code_scanner),
                title: const Text('Сканировать QR-код'),
                subtitle: const Text('Друг показывает код, вы сканируете'),
                onTap: () {
                  Navigator.pop(ctx);
                  _scanQR();
                },
              ),
              ListTile(
                leading: const Icon(Icons.qr_code),
                title: const Text('Показать мой QR-код'),
                subtitle: const Text('Друг сканирует ваш код'),
                onTap: () {
                  Navigator.pop(ctx);
                  _showMyQR();
                },
              ),
              ListTile(
                leading: const Icon(Icons.wifi_find),
                title: const Text('Найти рядом'),
                subtitle: const Text('Поиск контактов в той же сети'),
                onTap: () {
                  Navigator.pop(ctx);
                  _scanNearby();
                },
              ),
              const SizedBox(height: 8),
            ],
          ),
        );
      },
    );
  }

  void _openLogs() {
    Navigator.push(context, MaterialPageRoute(builder: (_) => const LogScreen()));
  }

  /// Перезагружает RequestsSection после accept/reject.
  Future<void> _reloadRequests() async {
    await _requestsKey.currentState?.loadRequests();
    _syncDiscoveredNodesFromP2P();
  }

  String _displayName(NodeInfo node) {
    try {
      final peerID = node.peerID;
      if (peerID.isEmpty) {
        return node.currentAddress;
      }
      final name = context.read<ChatProvider>().nameFor(peerID);
      if (name.isEmpty) return peerID;
      return name;
    } catch (_) {
      return 'Контакт';
    }
  }

  String _lastMessagePreview(NodeInfo node, ChatProvider chatProvider) {
    try {
      final peerID = node.peerID;
      if (peerID.isEmpty) return 'Контакт ISOTOPE';
      final last = chatProvider.getLastMessageForPeer(peerID);
      if (last == null) {
        switch (node.status) {
          case NodeStatus.unknown:
            return 'Не проверен';
          case NodeStatus.alive:
            return 'Контакт ISOTOPE';
          case NodeStatus.dead:
            return 'Недоступен';
        }
      }
      if (last.isVoice) {
        return '🎤 Голосовое сообщение';
      }
      if (last.isPhoto) {
        return '📷 Фото';
      }
      if (last.isFile) {
        final name = last.fileName.isNotEmpty ? last.fileName : 'Файл';
        final short = name.length > 30 ? '${name.substring(0, 30)}...' : name;
        return '📎 $short';
      }
      final raw = last.displayText;
      final text = raw.length > 40 ? '${raw.substring(0, 40)}...' : raw;
      return text;
    } catch (_) {
      return 'Ошибка';
    }
  }

  String _lastMessageTime(NodeInfo node, ChatProvider chatProvider) {
    try {
      final peerID = node.peerID;
      if (peerID.isEmpty) return '';

      final last = chatProvider.getLastMessageForPeer(peerID);
      if (last == null) return '';

      return formatShortTime(last.time);
    } catch (_) {
      return '';
    }
  }

  IconData _getNodeIcon(NodeStatus status) {
    switch (status) {
      case NodeStatus.unknown:
        return Icons.help_outline;
      case NodeStatus.alive:
        return Icons.router;
      case NodeStatus.dead:
        return Icons.wifi_off;
    }
  }

  Color? _getNodeIconColor(NodeStatus status) {
    switch (status) {
      case NodeStatus.unknown:
        return Colors.grey.shade500;
      case NodeStatus.alive:
        return null;
      case NodeStatus.dead:
        return Colors.grey;
    }
  }

  Color? _getNodeTextColor(NodeStatus status) {
    switch (status) {
      case NodeStatus.unknown:
        return Colors.grey.shade700;
      case NodeStatus.alive:
        return null;
      case NodeStatus.dead:
        return Colors.grey;
    }
  }

  /// Иконка верификации контакта (4.4).
  /// verified: true → галочка, verified: false → предупреждение.
  Widget? _verifiedBadge(String peerID) {
    if (peerID.isEmpty) return null;
    final verified = _verifiedContacts[peerID];
    if (verified == null) return null;
    if (verified) {
      return const Icon(Icons.verified, size: 16, color: Colors.green);
    }
    return const Icon(Icons.warning_amber_rounded, size: 16, color: Colors.orange);
  }

  /// Возвращает вид подзаголовка (для цвета и действия).
  ContactSubtitleKind _contactSubtitleKind(NodeInfo node, ChatProvider chatProvider) {
    final peerID = node.peerID;
    if (peerID.isEmpty) return ContactSubtitleKind.normal;

    // Черновик — высший приоритет. Моё незавершённое действие.
    if (chatProvider.hasDraftFor(peerID)) {
      return ContactSubtitleKind.draft;
    }

    final verified = _verifiedContacts[peerID] ?? false;
    final confirmed = _confirmedContacts[peerID] ?? false;
    if (verified && !confirmed && !_requestSent.contains(peerID)) {
      return ContactSubtitleKind.request;
    }

    return ContactSubtitleKind.normal;
  }

  /// Возвращает текст подзаголовка.
  /// Приоритет: черновик → «Отправить запрос?» / «Ожидание» → последнее сообщение.
  String? _contactStatusSubtitle(NodeInfo node, ChatProvider chatProvider) {
    final peerID = node.peerID;
    if (peerID.isEmpty) return _lastMessagePreview(node, chatProvider);

    // Черновик — начало текста.
    if (chatProvider.hasDraftFor(peerID)) {
      final draft = chatProvider.draftFor(peerID);
      final short = draft.length > 30 ? '${draft.substring(0, 30)}...' : draft;
      return 'Черновик: «$short»';
    }

    final verified = _verifiedContacts[peerID] ?? false;
    final confirmed = _confirmedContacts[peerID] ?? false;

    if (verified && !confirmed) {
      if (_requestSent.contains(peerID)) {
        return 'Запрос отправлен. Ожидание...';
      }
      return 'Не подтверждён. Отправить запрос?';
    }

    return _lastMessagePreview(node, chatProvider);
  }

  /// Тап на subtitle: если контакт не подтверждён — показать диалог отправки запроса.
  /// Возвращает true, если обработано (тап не должен открывать чат).
  Future<bool> _onSubtitleTap(NodeInfo node) async {
    final peerID = node.peerID;
    if (peerID.isEmpty) return false;

    final verified = _verifiedContacts[peerID] ?? false;
    final confirmed = _confirmedContacts[peerID] ?? false;
    if (!verified || confirmed) return false;
    if (_requestSent.contains(peerID)) return true;

    // Диалог «Как вас представить?» — разовое представление для этого контакта.
    // Предзаполнено: MyDisplayName (если задано) или PeerID. Текст выделен.
    final myDisplayName = await LibP2PService.getMyDisplayName();
    final shortPeerID = peerID.length > 12 ? peerID.substring(0, 12) : peerID;
    final preset = myDisplayName.isNotEmpty ? myDisplayName : shortPeerID;

    if (!mounted) return true;
    final displayName = await _showDisplayNameDialog(preset);
    if (displayName == null) {
      // Отмена — не отправляем.
      return true;
    }
    // Bootstrap-handshake: ChatProvider сохранит displayName и шлёт [CONTACT_HELLO].
    // После [CONTACT_HELLO_ACK] ChatProvider отправит [CONTACT_REQUEST] с этим именем.
    final chatProvider = context.read<ChatProvider>();
    final result = await chatProvider.startContactHandshake(peerID, displayName);
    if (result.containsKey('error')) {
      LogService.log('ContactHello: ошибка: ${result['error']}');
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Не удалось отправить: ${result['error']}')),
        );
      }
      return true;
    }

    setState(() {
      _requestSent.add(peerID);
    });

    // Таймаут 30 сек: если ACK не пришёл — сбрасываем _requestSent.
    Timer(const Duration(seconds: 30), () {
      if (!mounted) return;
      if (_confirmedContacts[peerID] == true) return; // уже подтверждён
      setState(() {
        _requestSent.remove(peerID);
      });
      LogService.log('ContactHello: timeout для $peerID');
    });

    LogService.log('ContactHello: отправлен $peerID (name="$displayName")');
    return true;
  }

  /// Диалог «Как вас представить?». Предзаполнено, текст выделен.
  /// Возвращает введённое имя или null (отмена).
  Future<String?> _showDisplayNameDialog(String preset) async {
    final controller = TextEditingController(text: preset);
    controller.selection = TextSelection(
      baseOffset: 0,
      extentOffset: preset.length,
    );

    return showDialog<String>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Как вас представить?'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            TextField(
              controller: controller,
              autofocus: true,
              maxLength: 100,
              decoration: const InputDecoration(
                hintText: 'Например: Иван, коллега по работе',
              ),
            ),
            const SizedBox(height: 8),
            const Text(
              'Например: Иван, коллега по работе,\n'
              'Пётр, мы договаривались на ремонт',
              style: TextStyle(fontSize: 11, color: Colors.grey),
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, null),
            child: const Text('Отмена'),
          ),
          TextButton(
            onPressed: () {
              final text = controller.text.trim();
              Navigator.pop(ctx, text.isEmpty ? preset : text);
            },
            child: const Text('Отправить'),
          ),
        ],
      ),
    );
  }
  @override
  void dispose() {
    final chatProvider = context.read<ChatProvider>();
    if (_chatListener != null) {
      chatProvider.removeListener(_chatListener!);
    }
    _nodeSub?.cancel();
    _ipSub?.cancel();
    _helloAckSub?.cancel();
    _coreLogsTimer?.cancel();
    _networkService.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('ISOTOPE'),
        actions: [
          IconButton(
            icon: const Icon(Icons.person_add),
            onPressed: _showAddContactDialog,
            tooltip: 'Добавить контакт',
          ),
          IconButton(
            icon: const Icon(Icons.settings),
            onPressed: _showSettingsDialog,
            tooltip: 'Настройки',
          ),
          IconButton(
            icon: const Icon(Icons.article_outlined),
            onPressed: _openLogs,
            tooltip: 'Журнал',
          ),
        ],
      ),
      body: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            if (_localIp != null)
              Container(
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(color: Colors.green.shade50, borderRadius: BorderRadius.circular(8)),
                child: Text('📍 Ваш адрес: $_localIp:8081', style: const TextStyle(fontWeight: FontWeight.bold), textAlign: TextAlign.center),
              ),
            if (_status != null)
              Container(
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(color: Colors.blue.shade50, borderRadius: BorderRadius.circular(8)),
                child: Row(
                  children: [
                    if (_connecting || _scanning || _findingNetwork) const SizedBox(width: 16, height: 16, child: CircularProgressIndicator(strokeWidth: 2)),
                    const SizedBox(width: 8),
                    Expanded(child: Text(_status!)),
                  ],
                ),
              ),
            if (_error != null)
              Container(
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(color: Colors.orange.shade50, borderRadius: BorderRadius.circular(8)),
                child: Text(_error!, style: const TextStyle(color: Colors.orange)),
              ),
            const SizedBox(height: 16),
            RequestsSection(
              key: _requestsKey,
              onAccepted: () {
                _loadContactsFromCore();
                _reloadRequests();
              },
              onRejected: () {
                _reloadRequests();
              },
            ),
            const Text('Контакты:', style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold)),
            const SizedBox(height: 8),
            Expanded(
              flex: 2,
              child: Consumer<ChatProvider>(
                builder: (_, chatProvider, __) {
                  if (_discoveredNodes.isEmpty) {
                    return Center(
                      child: Padding(
                        padding: const EdgeInsets.symmetric(horizontal: 24),
                        child: Column(
                          mainAxisAlignment: MainAxisAlignment.center,
                          children: [
                            Icon(Icons.group_outlined, size: 64, color: Colors.grey.shade400),
                            const SizedBox(height: 16),
                            const Text(
                              'Пока никого нет',
                              style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
                            ),
                            const SizedBox(height: 8),
                            Text(
                              'Добавьте первый контакт —\n'
                              'и сможете общаться без цензуры и слежки.',
                              style: TextStyle(fontSize: 13, color: Colors.grey.shade700),
                              textAlign: TextAlign.center,
                            ),
                            const SizedBox(height: 20),
                            FilledButton.icon(
                              onPressed: _showAddContactDialog,
                              icon: const Icon(Icons.person_add),
                              label: const Text('Добавить контакт'),
                            ),
                          ],
                        ),
                      ),
                    );
                  }

                  return ListView.builder(
                    itemCount: _discoveredNodes.length,
                    itemBuilder: (context, index) {
                      final node = _discoveredNodes[index];
                      final unread = chatProvider.unreadFor(node.peerID);
                      final displayName = _displayName(node);
                      final subtitleText = _contactStatusSubtitle(node, chatProvider) ?? '';
                      final lastTime = _lastMessageTime(node, chatProvider);
                      final badge = _verifiedBadge(node.peerID);
                      final kind = _contactSubtitleKind(node, chatProvider);
                      final subtitleColor = kind == ContactSubtitleKind.draft
                          ? Colors.green
                          : kind == ContactSubtitleKind.request
                              ? Colors.orange
                              : Colors.grey.shade700;
                      final subtitleWeight = kind == ContactSubtitleKind.normal
                          ? FontWeight.normal
                          : FontWeight.w600;

                      return InkWell(
                        onLongPress: () => _showContactActions(node),
                        onTap: node.status == NodeStatus.dead || _connecting
                            ? null
                            : () => _connectToNode(node),
                        child: ListTile(
                          leading: Icon(
                            _getNodeIcon(node.status),
                            color: _getNodeIconColor(node.status),
                          ),
                          title: Row(
                            children: [
                              Flexible(
                                child: Text(
                                  displayName,
                                  style: TextStyle(color: _getNodeTextColor(node.status)),
                                  overflow: TextOverflow.ellipsis,
                                ),
                              ),
                              if (badge != null) ...[
                                const SizedBox(width: 4),
                                badge,
                              ],
                            ],
                          ),
                          subtitle: kind == ContactSubtitleKind.request
                              ? GestureDetector(
                                  onTap: () => _onSubtitleTap(node),
                                  child: Text(
                                    subtitleText,
                                    maxLines: 1,
                                    overflow: TextOverflow.ellipsis,
                                    style: TextStyle(
                                      fontSize: 13,
                                      color: subtitleColor,
                                      fontWeight: subtitleWeight,
                                    ),
                                  ),
                                )
                              : Text(
                                  subtitleText,
                                  maxLines: 1,
                                  overflow: TextOverflow.ellipsis,
                                  style: TextStyle(
                                    fontSize: 13,
                                    color: subtitleColor,
                                    fontWeight: subtitleWeight,
                                  ),
                                ),
                          trailing: Column(
                            mainAxisAlignment: MainAxisAlignment.center,
                            crossAxisAlignment: CrossAxisAlignment.end,
                            children: [
                              if (lastTime.isNotEmpty)
                                Text(
                                  lastTime,
                                  style: const TextStyle(fontSize: 11, color: Colors.grey),
                                ),
                              if (unread > 0)
                                Container(
                                  margin: const EdgeInsets.only(top: 4),
                                  padding: const EdgeInsets.all(6),
                                  decoration: BoxDecoration(color: Colors.red, borderRadius: BorderRadius.circular(12)),
                                  child: Text('$unread', style: const TextStyle(color: Colors.white, fontSize: 12)),
                                ),
                            ],
                          ),
                        ),
                      );
                    },
                  );
                },
              ),
            ),
            const SizedBox(height: 8),
            const Text('Журнал:', style: TextStyle(fontSize: 14, fontWeight: FontWeight.bold)),
            const SizedBox(height: 4),
            Expanded(
              flex: 2,
              child: Container(
                padding: const EdgeInsets.all(8),
                decoration: BoxDecoration(color: Colors.black87, borderRadius: BorderRadius.circular(8)),
                child: ListView.builder(
                  itemCount: _logs.length,
                  itemBuilder: (context, index) {
                    return Text(_logs[index], style: const TextStyle(color: Colors.greenAccent, fontSize: 11));
                  },
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
// mobile/lib/screens/connect_screen.dart