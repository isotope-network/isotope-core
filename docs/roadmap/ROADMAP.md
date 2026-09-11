# Дорожная карта ISOTOPE

## Текущий статус

**Версия:** v1.21.0 (мобильная стабилизация)

Ядро — библиотека (package core). Работает на десктопе (HTTP, libp2p, DHT) и на мобильных (libp2p FFI, NSD, HTTP). Reconnect loop, единый ID сообщений, non-blocking вызовы.

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

---

## В работе / Ближайшие задачи

### Мобильное приложение
- ✅ Базовая версия — работает
- ✅ Reconnect loop — работает
- 🔜 **Этап 2: Foreground Service (Android)** — не даёт системе убивать соединения в свёрнутом приложении
- 🔜 Этап 3: Battery Optimization Whitelist — попросить пользователя добавить ISOTOPE в исключения
- 🔜 Отображение пиров через libp2p в ConnectScreen (сейчас только NSD)
- 🔜 BLE — отложен, нестабилен
- 🔜 DHT в мобильном
- 🔜 Круговой циферблат TTL — UI-улучшение
- 🔜 PWA + F-Droid

### Архитектурное
- 🔜 Hole punching через интернет (/p2p-circuit/) — телефон А → телефон Б через VPS
- 🔜 DHT announce — телефоны публикуют relay-адреса, FindPeer возвращает /p2p-circuit/

### Ядро
- 🔜 Образная стеганография
- 🔜 Морфинг трафика
- 🔜 Улучшение DHT (стабильность)

### Гигиена
- 🔜 Очистка старых дубликатов в state (наследие старых версий)
- 🔜 Упрощение «Отозвать» — recall best effort, без ожидания 30 сек

---

## Будущие версии

### v2.0 — Телефон как полноценный узел
- libp2p FFI полностью стабилен
- BLE работает
- Foreground Service + Battery Whitelist
- Hole punching через /p2p-circuit/
- DHT на мобильном
- Мультиплексирование транспортов (Wi-Fi, BLE, мобильная сеть)

### v3.0 — ISOTOPE AI Mesh
- Распределённый инференс ИИ
- Облегчённые модели на узлах
- Этический паспорт моделей
- Маршрутизация запросов: ближайший узел → дата-центр

### v4.0 — Полная автономия
- Сеть без интернета (offline-first)
- Mesh-сообщества (isotope.zone)
- Самоорганизация без bootstrap-узлов
- Полный иммунитет (100+ узлов)

---

## Отложено

- BLE — нестабилен на текущем стеке, вернуться после v2.0
- IPFS для сайта — заблокирован в России, не работает с Cloudflare
- Samsung Android 10 — вероятно, решён non-blocking вызовами (проверить)

---

## Инфраструктура

### VPS bootstrap/relay
- IP: 186.246.31.176
- PeerID: QmNmr3YqGD9uKpPCx7W86t7Tc3vrBJF1GbmTAzDQ25Sskx
- Bootstrap multiaddr: /ip4/186.246.31.176/tcp/9001/ws/p2p/QmNmr3YqGD9uKpPCx7W86t7Tc3vrBJF1GbmTAzDQ25Sskx
- Порты: 9000 (TCP), 9001 (WS), 8081 (HTTP API)

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

1. **Мобильная стабилизация** — сейчас (v1.21)
2. **Foreground Service и офлайн** — v2.0
3. **AI Mesh** — v3.0
4. **Полная автономия** — v4.0

Каждый этап — это новый уровень децентрализации.