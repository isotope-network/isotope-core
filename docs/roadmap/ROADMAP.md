# Дорожная карта ISOTOPE

## Текущий статус

**Версия:** v1.26.0 (E2E, подписи, UI)

Ядро — библиотека (package core). E2E-шифрование работает. Подпись контактов. Relay-стабильность. UI этап 1.1–1.3 закрыт.

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
- Экспорт API

### Мобильная версия (v1.19.0)
- libp2p через FFI (.aar)
- Стабильный PeerID
- Обработка смены сети
- NodeInfo, heartbeat
- Логирование Go → Flutter
- NSD-обнаружение

### Мобильная стабилизация (v1.21.0)
- Единый ID сообщений (дубликаты устранены)
- Non-blocking Mobile calls (ANR устранён)
- Reconnect loop
- Бейдж непрочитанных

### Foreground Service (v1.22.0)
- Foreground Service (Android)
- Battery Optimization Whitelist
- Отображение пиров через libp2p

### Relay-circuit (v1.23.0)
- Multi-address ANNOUNCE
- Relay-circuit (client.Reserve)
- Flush on reconnect (три уровня)

### E2E-шифрование (v1.24.0)
- 4.1 E2E keypair, QR v:1
- 4.2 Recipient, адресная маршрутизация
- 4.3.1 Ed25519 + X25519 ключи
- 4.3.2 isotope_contacts.json
- 4.3.3 E2E-шифрование (box.Seal, Version=2)
- 4.4 Ed25519 подпись над peerID || x25519_pub
- Backoff reconnect 1 → 30 сек
- Параллельный dial
- TTL 5 → 8 мин
- Обрезка логов 4KB
- Fix configure (PeerID из multiaddr)

### UI 1.1–1.3 (v1.25.0)
- UI 1.1: терминология (технические термины убраны)
- UI 1.2: единый вход «Добавить контакт»
- UI 1.3: пустое состояние с кнопкой

### Стабильность (v1.26.0)
- PlainText для своих E2E-сообщений
- Автоочистка старых E2E без PlainText
- Self-QR блокируется (три уровня)
- Relay: refresh reservation on reconnect
- Relay: exponential backoff
- Retry loading contacts
- Upsert nodes on alive-event

---

## В работе / Ближайшие задачи

### Приоритет 1 (сейчас)
- 🔜 **Этап 1.4 UI** — первый запуск (3 экрана)
- 🔜 **Единый источник истины для контактов** — сейчас 4 хранилища (NodeStore, _discoveredNodes, _nodesMap, _pendingNodes)

### Приоритет 2
- 🔜 **Контакт-протокол (этапы 4–8):**
  4. NSD-подтверждение (обоюдное)
  5. Запрос на контакт (как в Signal)
  6. Seed-фраза для восстановления PeerID
  7. DHT активация при 15+ узлах
  8. Локальный вес
- 🔜 **Periodic FIND** для активных контактов (закрывает второе окно ANNOUNCE)

### Приоритет 3
- 🔜 Foreground service — разное поведение на Xiaomi/Huawei
- 🔜 VPS reconnectLoop — шумит вхолостую, отключить на relay-сервере
- 🔜 DHT Provide — падает при малом числе пиров
- 🔜 Samsung Android 10 — краш
- 🔜 ANNOUNCE TTL expired — эпизодически
- 🔜 VPS memory:66 — мусор от старых сессий

### Приоритет 4 (v2.0+)
- 🔜 **Onion-маршрутизация** — защита метаданных
- 🔜 **Обфускация трафика** — после onion
- 🔜 **Padding, mixing** — временны́е паттерны
- 🔜 **BLE** — возрождение
- 🔜 **Hole punching** через /p2p-circuit/
- 🔜 **DHT на мобильном** (при 15+ узлах)

---

## Будущие версии

### v2.0 — Телефон как полноценный узел
- libp2p FFI полностью стабилен
- BLE работает
- Foreground Service + Battery Whitelist
- Hole punching через /p2p-circuit/
- DHT на мобильном (при 15+ узлах)
- Мультиплексирование транспортов
- **Условие отключения VPS:** DHT 15+ узлов + hole punching + 2-3 relay-узла

### v3.0 — ISOTOPE AI Mesh
- Распределённый инференс ИИ
- Swarm Inference: PeerRankedConsensus (Bradley–Terry, репутационные веса)
- Семантическая маршрутизация (Latent Semantic Router)
- Облегчённые модели на узлах
- Этический паспорт моделей
- Маршрутизация: ближайший узел → дата-центр

### v3.1 — Развитие AI Mesh
- (внутренняя разработка, детали не публикуются)

### v4.0 — Полная автономия
- Сеть без интернета (offline-first)
- Mesh-сообщества (isotope.zone)
- Самоорганизация без bootstrap-узлов
- Полный иммунитет (100+ узлов)
- Саморегуляция (социальный иммунитет)

---

## Уровни защиты

| Уровень | Что защищено | Статус |
|---------|--------------|--------|
| Обфускация | Маскировка трафика | ✅ v1.13 |
| E2E-шифрование | Содержимое (от relay) | ✅ v1.24 |
| Подпись | Подлинность отправителя | ✅ v1.24 |
| Метаданные | Кто с кем общается | 🔜 v2.0+ (Onion) |
| Временны́е паттерны | Тайминг, объём | 🔜 Padding, mixing |

---

## Отложено

- BLE — нестабилен на текущем стеке
- IPFS для сайта — заблокирован в России
- Samsung Android 10 — краш
- DHT Provide — при малом числе пиров
- Уведомления системы
- Оптимизация ANNOUNCE (дубли relay-адресов)

---

## Инфраструктура

### VPS bootstrap/relay
- IP: 186.246.31.176
- PeerID: QmR8u5YFdcKpM2onQvk7KV5qioai87aysi9JWLdV1LX1bi
- Bootstrap multiaddr: /ip4/186.246.31.176/tcp/9001/ws/p2p/QmR8u5YFdcKpM2onQvk7KV5qioai87aysi9JWLdV1LX1bi
- Порты: 9000 (TCP), 9001 (WS), 8081 (HTTP API)
- Роль: временная инфраструктура (relay для узлов за NAT)

### Обновление VPS
cd /root/isotope-core && git pull && cd node && go build -o isotope-node ./main
# Ctrl+C в окне VPS, затем:
cd /root/isotope && NODE_ID=bootstrap ISOTOPE_PORT=9000 ISOTOPE_HTTP_PORT=8081 ISOTOPE_ENABLE_RELAY=true /root/isotope-core/node/isotope-node

НЕ удалять /root/isotope/state/ — PeerID изменится.

### Сборка .aar
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

1. **E2E и подписи** — v1.24 (закрыто)
2. **UI и стабильность** — v1.25–v1.26 (закрыто)
3. **Контакт-протокол** — сейчас
4. **Onion, padding, mixing** — v2.0+
5. **AI Mesh** — v3.0
6. **Полная автономия** — v4.0

Каждый этап — новый уровень децентрализации.
VPS отключается, когда DHT и hole punching закроют его роль.