# Дорожная карта ISOTOPE

## Текущий статус

**Версия:** v1.27.0 (контакты и идентификация)

Ядро — библиотека (package core). Контакт-протокол работает. Система имён. Статусы. Таймер. Relay-стабильность.

---

## Завершено

### Ядро (v1.0–v1.18)
- P2P: libp2p + mDNS + DHT + Gossip
- Priority Gossip
- Ассоциативная память
- WebSocket + TLS
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

### Контакт-протокол и имена (v1.27.0)
- Bootstrap-handshake: [HELLO] → [ACK] → [REQUEST] → [ACCEPT]
- Открытые сервисные (HELLO, ACK) — всегда через bootstrap
- E2E (REQUEST, ACCEPT) — circuit → bootstrap fallback
- tempContacts — временные в памяти для расшифровки
- Push через messageHook (не polling)
- Симметрия: обе стороны confirmed: true
- Система имён: Name / RemoteName / PeerID
- MyDisplayName в Settings
- QR содержит display_name
- Диалог «Как вас представить?» (предзаполнено + выделено)
- Профиль → «Ваше имя»
- Долгий тап → Открыть / Переименовать / Удалить
- Предупреждение о безопасности
- Статусы: ✓ / ✓✓ / ✓🔒 / ✓✓ (цвет)
- Таймер отправки: 0/3/5/10 сек

---

## В работе / Ближайшие задачи

### Приоритет 1 (сейчас)
- 🔜 **[PROFILE_UPDATE]** — для смены MyDisplayName
- 🔜 **Батч [READ]** — сейчас по одному
- 🔜 **TTL для tempContacts** (5 минут)
- 🔜 **Чистка messageStatus** при удалении

### Приоритет 2
- 🔜 **Circuit direct при 15+ узлах**
- 🔜 Foreground service — разное поведение на Xiaomi/Huawei
- 🔜 VPS reconnectLoop — шумит вхолостую
- 🔜 Полупрозрачность pending-сообщения
- 🔜 TTL в настройках — перенести из chat_screen.dart
- 🔜 Черновик UI — тап по 📝 → возврат в поле

### Приоритет 3
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
- Swarm Inference: PeerRankedConsensus (Bradley–Terry)
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

### Телефоны (тестовые)
- Xiaomi: QmT5sgdagJnN3imUAn2NW2vaQ1jJNJhkh67ThUwLqWaL4P
- Huawei: QmSFnPtYPeh1xEeh5ZMyaCApqP3fQcTYoenYsgD8Q3opAq

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
3. **Контакты и идентификация** — v1.27 (закрыто)
4. **Мелкие UX + инфраструктура** — сейчас
5. **Onion, padding, mixing** — v2.0+
6. **AI Mesh** — v3.0
7. **Полная автономия** — v4.0

Каждый этап — новый уровень децентрализации.
VPS отключается, когда DHT и hole punching закроют его роль.