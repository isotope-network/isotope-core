# Дорожная карта ISOTOPE

## Текущий статус

**Версия:** v1.23.0 (relay-circuit, multi-address ANNOUNCE, flush on reconnect)

Ядро — библиотека (package core). Работает на десктопе (HTTP, libp2p, DHT) и на мобильных (libp2p FFI, NSD, HTTP). Связь между телефонами в разных сетях (Wi-Fi ↔ LTE) через relay на VPS.

---

## Завершено

### Ядро (v1.0–v1.18)
- P2P: libp2p + mDNS + DHT + Gossip
- Priority Gossip
- Ассоциативная память
- WebSocket + TLS (маскировка)
- Обфускация AES-GCM
- Голосовая стеганография (LSB в WAV)
- Onion Routing v2
- Исчезающие сообщения (TTL)
- Локальное шифрование (AES-256-GCM)
- Селф-хилинг (heartbeat)
- Репликация
- Самоадаптация
- Каналы с весовыми уровнями
- Весовая модель доступа
- Эмерджентное доверие
- Этический хеш: 7 заповедей
- Нейросеть: 100-мерные векторы, биграммы

### Рефакторинг (v1.18.1)
- package main → package core
- Точка входа: node/main/main.go
- Экспорт API: Config, NewNode, InitP2P, StartHTTP, StartMobile, Stop, HashText

### Мобильная версия (v1.19.0)
- libp2p через FFI (gomobile, .aar 67 МБ)
- Методы: start, send, getMessages, getPeers, getWeight, getStatus, getMultiaddrs, connectToPeer
- Стабильный PeerID (isotope_state.json.key)
- Обработка смены сети (connectivity_plus, debounce 10 сек)
- NodeInfo: PeerID, multiaddrs, lastSeen, status
- Heartbeat с счётчиком неудач (3 → dead)
- Логирование Go → Flutter
- Динамический поиск порта (8081+)
- NSD-обнаружение с PeerID и multiaddr
- HTTP-связь между телефонами
- Бейджи непрочитанных
- История сообщений

### Мобильная стабилизация (v1.21.0)
- Единый ID сообщений из Go-ядра (устранены дубликаты)
- replicateMessage() исключает отправителя
- Non-blocking Mobile calls (MainActivity.kt — устранён ANR)
- Reconnect loop (каждые 30 сек проверка пиров)
- pingPeers() — интервал сокращён до 15 сек
- Бейдж непрочитанных + линия «Непрочитанные»
- _ownMessageIds в SharedPreferences
- Проверено на реальных телефонах: reconnect после сворачивания, отсутствие ANR

### Foreground Service (v1.22.0)
- Foreground Service (Android): соединения не разрываются при сворачивании
- Battery Optimization Whitelist: запрос на исключения батареи
- Отображение пиров через libp2p (Wi-Fi / LTE)

### Relay-circuit (v1.23.0)
- Multi-address ANNOUNCE: announcedPeer — список []string
- Формат ANNOUNCE: [ANNOUNCE]\n<addr1>\n<addr2>\n[END]
- FIND отдаёт массив адресов
- ConnectToPeerWithFallback — пробует адреса по очереди
- Резервация relay-слота через client.Reserve
- GetRelayAddrs — строит relay-адрес из bootstrap
- ANNOUNCE автоматически добавляет relay-адрес
- FIND fallback — relay-адрес, если announced пуст
- VPS handleStream форвардит реплики дальше
- Flush on reconnect (три уровня: Notifiee, markPeerAlive, reconnectLoop)
- Результат: связь Wi-Fi ↔ LTE через интернет, flush за 1 сек

---

## В работе / Ближайшие задачи

### Приоритет 1 (сейчас)
- ✅ D — backoff reconnect (закрыто)
- 🔜 **Параллельный dial** в ConnectToPeerWithFallback (~30 строк, убирает задержку 5-7 сек)
- 🔜 **E2E шифрование** поверх обфускации (критично, до контакт-протокола)
- 🔜 **ANNOUNCE_TTL:** 5 мин → 8 мин (зазор 4 мин от announceLoop 4 мин)

### Приоритет 2
- 🔜 Контакт-протокол (этапы 1–8):
  1. Терминология (человеческие слова: контакт, связь, ключ, имя)
  2. QR = PeerID (работает, нужно UI-оформление)
  3. Ссылка (isotope:<peerID>)
  4. NSD
  5. Запрос
  6. Seed-фраза (обсудить отдельно)
  7. DHT 15+
  8. Локальный вес

### Приоритет 3
- 🔜 BLE — возрождение
- 🔜 Samsung Android 10 — краш (вероятно, решён non-blocking вызовами)
- 🔜 DHT Provide — падает при малом числе пиров
- 🔜 Уведомления системы (не только бейдж)
- 🔜 Foreground Service — надёжнее

---

## Будущие версии

### v2.0 — Телефон как полноценный узел
- libp2p FFI полностью стабилен
- BLE работает
- Foreground Service + Battery Whitelist
- Hole punching через /p2p-circuit/
- DHT на мобильном (при 15+ узлах)
- Мультиплексирование транспортов (Wi-Fi, BLE, мобильная сеть)
- **Условие отключения VPS:** когда одновременно выполнены:
  1. DHT покрывает 15+ узлов (иммунитет включается)
  2. Hole punching работает для большинства NAT (CGNAT, симметричный NAT)
  3. Есть 2-3 независимых relay-узла в сети (не на нашем VPS)

### v3.0 — ISOTOPE AI Mesh
- Распределённый инференс ИИ
- Swarm Inference: PeerRankedConsensus (Bradley–Terry, репутационные веса)
- Семантическая маршрутизация (Latent Semantic Router)
- Облегчённые модели на узлах
- Этический паспорт моделей
- Маршрутизация запросов: ближайший узел → дата-центр

### v3.1 — Развитие AI Mesh
- (внутренняя разработка, детали не публикуются)

### v4.0 — Полная автономия
- Сеть без интернета (offline-first)
- Mesh-сообщества (isotope.zone)
- Самоорганизация без bootstrap-узлов
- Полный иммунитет (100+ узлов)
- Саморегуляция (социальный иммунитет)

---

## Отложено

- BLE — нестабилен на текущем стеке, вернуться после v2.0
- IPFS для сайта — заблокирован в России, не работает с Cloudflare
- Samsung Android 10 — краш при запуске
- DHT Provide — при малом числе пиров падает (нужно 15+)
- Уведомления системы
- Оптимизация ANNOUNCE (убрать дубли relay-адресов)

---

## Инфраструктура

### VPS bootstrap/relay (v1.23.0)
- IP: 186.246.31.176
- PeerID: QmR8u5YFdcKpM2onQvk7KV5qioai87aysi9JWLdV1LX1bi
- Bootstrap multiaddr: /ip4/186.246.31.176/tcp/9001/ws/p2p/QmR8u5YFdcKpM2onQvk7KV5qioai87aysi9JWLdV1LX1bi
- Порты: 9000 (TCP), 9001 (WS), 8081 (HTTP API)
- Код на коммите: 6b223c0
- Роль: временная инфраструктура (relay для узлов за NAT)

### Обновление VPS
cd /root/isotope-core && git pull && cd node && go build -o isotope-node ./main
# Ctrl+C в окне VPS, затем:
cd /root/isotope && NODE_ID=bootstrap ISOTOPE_PORT=9000 ISOTOPE_HTTP_PORT=8081 ISOTOPE_ENABLE_RELAY=true /root/isotope-core/node/isotope-node

⚠️ НЕ удалять /root/isotope/state/ — PeerID изменится, телефоны потеряют связь.

### Сборка .aar (только если менялся Go-код)
cd D:\isotope\node
del isotope.aar
gomobile bind -target=android -androidapi 21 -ldflags "-checklinkname=0" -o isotope.aar ./mobile
copy /y isotope.aar D:\isotope\mobile\android\app\libs\

### Сборка APK
cd D:\isotope\mobile
flutter clean
flutter pub get
flutter build apk --debug

---

## Стратегия

1. **Мобильная стабилизация** — v1.21–v1.23
2. **E2E + контакт-протокол** — сейчас
3. **Foreground Service и офлайн** — v2.0
4. **AI Mesh** — v3.0
5. **Полная автономия** — v4.0

Каждый этап — это новый уровень децентрализации.
VPS отключается, когда DHT и hole punching закроют его роль.