# Дорожная карта ISOTOPE

## Текущий статус

**Версия:** v1.28.0 (право на забвение)

Ядро — библиотека (package core). Контакт-протокол работает. Система имён. Статусы. Таймер. TTL с двумя режимами. FLAG_SECURE. Удаление контакта.

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
- Единый ID сообщений
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
- UI 1.1: терминология
- UI 1.2: единый вход «Добавить контакт»
- UI 1.3: пустое состояние с кнопкой

### Стабильность (v1.26.0)
- PlainText для своих E2E-сообщений
- Автоочистка старых E2E без PlainText
- Self-QR блокируется
- Relay: refresh reservation on reconnect
- Relay: exponential backoff
- Retry loading contacts
- Upsert nodes on alive-event

### Контакт-протокол и имена (v1.27.0)
- Bootstrap-handshake: [HELLO] → [ACK] → [REQUEST] → [ACCEPT]
- Открытые сервисные — через bootstrap
- E2E — circuit → bootstrap fallback
- tempContacts
- Push через messageHook
- Симметрия: confirmed: true
- Система имён: Name / RemoteName / PeerID
- MyDisplayName в Settings
- QR содержит display_name
- Диалог «Как вас представить?»
- Профиль → «Ваше имя»
- Долгий тап → Открыть / Переименовать / Удалить
- Предупреждение о безопасности

### TTL и право на забвение (v1.28.0)
- Периоды: 10s / 30s / 1m / 5m / 15m / 30m / 1h / 4h / 24h / never
- Режимы: hard / after_read
- [TTL_UPDATE] (Type=8) — авто-hard
- Fallback 48 часов
- DeleteExpired в cleanupLoop (1 мин)
- FLAG_SECURE: TTL 10s–1m
- Бейдж + превью синхронно
- Удаление контакта: тихий отказ
- isotope_deleted.json
- Per-chat / per-peer: сообщения, черновики, непрочитанные
- [READ] только для текущего чата
- UI: настройки TTL, оранжевые периоды, умный формат времени

---

## В работе / Ближайшие задачи

### Приоритет 1 (сейчас)
- 🔜 **connect_screen — единый источник имени** (убрать _contactNames, _displayName → chatProvider.nameFor)
- 🔜 **[PROFILE_UPDATE]** (Type=9) — смена read_enabled / display_name без QR
- 🔜 **Батч [READ]** — массив msg_id в одном вызове

### Приоритет 2
- 🔜 **Circuit direct при 15+ узлах**
- 🔜 Foreground service — разное поведение Xiaomi/Huawei
- 🔜 VPS reconnectLoop — шумит вхолостую
- 🔜 messageStatus — рост. Чистка при удалении / архивации

### Приоритет 3
- 🔜 DHT Provide — падает при малом числе пиров
- 🔜 Samsung Android 10 — краш
- 🔜 ANNOUNCE TTL expired — эпизодически
- 🔜 VPS memory:66 — мусор

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
- Swarm Inference: PeerRankedConsensus
- Семантическая маршрутизация
- Облегчённые модели на узлах
- Этический паспорт моделей

### v3.1 — Развитие AI Mesh
- (внутренняя разработка, детали не публикуются)

### v4.0 — Полная автономия
- Сеть без интернета (offline-first)
- Mesh-сообщества (isotope.zone)
- Самоорганизация без bootstrap
- Полный иммунитет (100+ узлов)
- Саморегуляция

---

## Уровни защиты

| Уровень | Что защищено | Статус |
|---------|--------------|--------|
| Обфускация | Маскировка трафика | ✅ v1.13 |
| E2E-шифрование | Содержимое | ✅ v1.24 |
| Подпись | Подлинность | ✅ v1.24 |
| Метаданные | Кто с кем | 🔜 v2.0+ |
| Временны́е паттерны | Тайминг | 🔜 |

---

## Отложено

- BLE — нестабилен
- IPFS для сайта — заблокирован
- Samsung Android 10 — краш
- DHT Provide — при малом числе пиров
- Уведомления системы
- Оптимизация ANNOUNCE
- История имён
- Разные имена у одного пользователя
- Голос / видео (v2.0+)
- Групповые чаты (G4)

---

## Инфраструктура

### VPS bootstrap/relay
- IP: 186.246.31.176
- PeerID: QmR8u5YFdcKpM2onQvk7KV5qioai87aysi9JWLdV1LX1bi
- Bootstrap: /ip4/186.246.31.176/tcp/9001/ws/p2p/QmR8u5YFdcKpM2onQvk7KV5qioai87aysi9JWLdV1LX1bi
- Порты: 9000 (TCP), 9001 (WS), 8081 (HTTP API)
- Роль: временная инфраструктура

### Телефоны (тестовые)
- Xiaomi: QmTtUkwXmx9vnUSTLCfdPZn6MNaPkFbxTMHjofFzHMfUz5
- Huawei: QmWyqjGzRQS2T2j4vGtet2j1M4hqDhJD99BPoG6QvKZBv6

### Обновление VPS
pkill -f isotope-node
sleep 1
pgrep -f isotope-node          # пусто
cd /root/isotope-core && git pull
cd /root/isotope-core/node && go build -o isotope-node ./main
nohup env NODE_ID=bootstrap ISOTOPE_PORT=9000 ISOTOPE_HTTP_PORT=8081 ISOTOPE_ENABLE_RELAY=true ./isotope-node > isotope.log 2>&1 &
sleep 3
pgrep -f isotope-node          # один, новый
curl -s http://127.0.0.1:8081/status

НЕ удалять /root/isotope/state/

### Сборка .aar
cd /d D:\isotope\node
del isotope.aar
gomobile bind -target=android -androidapi 21 -ldflags "-checklinkname=0" -o isotope.aar ./mobile
copy /y isotope.aar D:\isotope\mobile\android\app\libs\

### Сборка APK
cd /d D:\isotope\mobile
flutter build apk --debug

**Порядок:** правка → проверка → коммит → CI → .aar (если Go) → APK → установка → VPS → тест.

---

## Стратегия

1. **E2E и подписи** — v1.24 (закрыто)
2. **UI и стабильность** — v1.25–v1.26 (закрыто)
3. **Контакты и идентификация** — v1.27 (закрыто)
4. **Право на забвение** — v1.28 (закрыто)
5. **Мелкие UX + инфраструктура** — сейчас
6. **Onion, padding, mixing** — v2.0+
7. **AI Mesh** — v3.0
8. **Полная автономия** — v4.0

Каждый этап — новый уровень децентрализации.
VPS отключается, когда DHT и hole punching закроют его роль.