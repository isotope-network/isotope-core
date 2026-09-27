# История изменений ISOTOPE

## v1.26.0 (2026-09-27)

### Добавлено
- PlainText в Message: свои E2E-сообщения сохраняются открытым текстом для UI
- Автоочистка: удаление своих E2E-сообщений без PlainText при первом запуске
- Флаг .e2e_cleanup рядом со state
- Memory.Remove — с чисткой seen
- Relay: refresh reservation on reconnect + 30s loop check
- Relay: exponential backoff для reservation retry (2 → 30 сек)
- UI: отображение plainText для своих E2E-сообщений

### Исправлено
- Свой QR блокируется (три уровня: UI + ядро + автоочистка)
- UI: retry loading contacts from core (гонка с Go startup)
- Upsert nodes on alive-event — статус меняется с unknown на alive

### Коммиты
- d0e9744 — own QR blocked, PlainText for own E2E messages, auto-cleanup
- 3235060 — relay: refresh reservation on reconnect + 30s loop check
- 882c998 — relay: exponential backoff for reservation retry
- abc5f01 — UI: display plainText for own E2E messages
- 6e1d1b9 — UI: retry loading contacts from core
- 2f35fb1 — UI: upsert nodes on alive-event

### Следующий шаг
- Этап 1.4 UI — первый запуск (3 экрана)
- Контакт-протокол (этапы 4–8)
- Единый источник истины для контактов

---

## v1.25.0 (2026-09-25)

### Добавлено
- UI 1.1 — терминология: технические термины убраны из интерфейса
- UI 1.2 — единый вход «Добавить контакт», настройки в bottom sheet
- UI 1.3 — пустое состояние с кнопкой действия

### Коммиты
- 5ebe222 — UI 1.1: remove technical terms from UI
- 4940998 — UI 1.2: single add-contact entry, settings bottom sheet
- 988fe17 — UI 1.3: empty state with action button

### Следующий шаг
- Этап 1.4 — первый запуск
- Продолжение контакт-протокола

---

## v1.24.0 (2026-09-23)

### Добавлено
- E2E 4.1 — генерация ключей, QR v:1 с e2e_pub
- E2E 4.2 — поле Recipient, адресная маршрутизация
- E2E 4.3.1 — Ed25519 + X25519, раздельные ключи, curve25519.X25519
- E2E 4.3.2 — isotope_contacts.json с флагом verified
- E2E 4.3.3 — шифрование Text через box.Seal, Version=2
- E2E 4.4 — Ed25519 подпись над peerID || x25519_pub, флаг verified
- Backoff reconnect: 1 → 30 сек, сброс при успехе
- Параллельный dial в ConnectToPeerWithFallback
- ANNOUNCE TTL 5 → 8 мин, удаление disconnected сразу

### Исправлено
- Логи: обрезка длинных строк (4KB max)
- Извлечение PeerID из multiaddr в configure

### Коммиты
- e54d056 — E2E 4.1: keypair generation, QR v:1
- 9c2fc8b — E2E 4.2: Recipient field, address-based routing
- dc82e81 — E2E 4.3.1: Ed25519+X25519 keypairs, QR v:1
- f3aaeab — E2E 4.3.2: isotope_contacts.json
- 3a3b859 — E2E 4.3.3: encrypt Text via box.Seal, Version=2
- 39c97c5 — E2E 4.4: Ed25519 signature, verified flag
- 1d2e3d3 — reconnect: exponential backoff
- 57cfeaf — connect: parallel dial
- 87524bd — announce: TTL 5→8 min
- 1a44715 — logs: truncate long lines
- b225a22 — fix: extract PeerID from multiaddr in configure

### Уровни защиты (после v1.24.0)
- Содержимое от перехвата — E2E
- Содержимое от relay — E2E
- Подлинность отправителя — подпись

### Следующий шаг
- Метаданные — Onion (v2.0+)
- Контакт-протокол

---

## v1.23.0 (2026-09-17)

### Добавлено
- Multi-address ANNOUNCE (Слой A):
  - announcedPeer — список []string
  - Формат ANNOUNCE: [ANNOUNCE]\n<addr1>\n<addr2>\n[END]
  - FIND отдаёт массив адресов
  - ConnectToPeerWithFallback — перебор адресов
- Relay-circuit (Слой B):
  - Резервация relay-слота через client.Reserve
  - GetRelayAddrs — relay-адрес из bootstrap
  - ANNOUNCE автоматически добавляет relay-адрес
  - FIND fallback — relay-адрес
  - VPS handleStream форвардит реплики
- Flush on reconnect (три уровня):
  - Notifiee ConnectedF
  - markPeerAlive → dead → alive
  - reconnectLoop
  - Результат: flush за 1 сек вместо 3-4 мин

### Изменено
- VPS PeerID: QmNmr3Yq... → QmR8u5YF...
- Условие отключения VPS: DHT 15+ узлов + hole punching + 2-3 relay-узла

### Коммиты
- 9b5c6c2 — multi-address ANNOUNCE + relay-circuit
- 6b223c0 — flush on libp2p ConnectedF

---

## v1.22.0 (2026-09-13)

### Добавлено
- Foreground Service (Android)
- Battery Optimization Whitelist
- Отображение пиров через libp2p

### Коммиты
- b706f97 — Foreground Service (Android)
- 716a48f — Battery Optimization Whitelist
- ccec4be — Отображение пиров через libp2p

---

## v1.21.0 (2026-09-11)

### Исправлено
- Дубликаты сообщений (единый ID из Go)
- Бейдж непрочитанных + линия «Непрочитанные»
- ANR на медленных телефонах (non-blocking Mobile calls)
- Reconnect loop + keepalive (15 сек ping)

### Коммиты
- ec0c59e — pass message ID from SendMessage
- 919967d — do not replicate back to sender
- c06d8a8 — non-blocking Mobile calls + reconnect loop

---

## v1.19.0 (2026-09-01)

### Добавлено
- Рефакторинг ядра: package main → package core
- main/main.go — точка входа для десктопа
- mobile/mobile.go — обёртка gomobile
- libp2p через FFI (.aar 67 МБ, MethodChannel)
- Стабильный PeerID (isotope_state.json.key)
- Обработка смены сети (connectivity_plus)
- NodeInfo, heartbeat, dead-статус
- Логирование Go → Flutter
- Динамический поиск порта (8081+)
- NSD-обнаружение с PeerID и multiaddr

---

## v1.18.1 (2026-08-28)

### Добавлено
- Рефакторинг ядра: package main → package core
- Точка входа: node/main/main.go
- Экспорт публичного API: Config, NewNode, InitP2P, StartHTTP, StartMobile, Stop
- HashText — экспортирован

---

## v1.18 (2026-08-16)

### Добавлено
- Каналы с весовыми уровнями (G4): Channel, ChannelStore
- Пороги доступа: full=0.3, comment=0.5, vote=0.7
- Эндпоинты: POST/GET /channels

---

## v1.17 (2026-08-16)

### Добавлено
- Самоадаптация: node/adapt.go
- Метрики: avgWeight, lowWeightRatio, highWeightRatio
- Фоновая адаптация каждые 5 минут

---

## v1.16 (2026-08-16)

### Добавлено
- Onion Routing v2: цепочка из 4-5 relay
- Выбор relay по весу > 0.7
- getPeerWeight

---

## v1.15 (2026-08-16)

### Добавлено
- Репликация на 2 случайных живых узла
- Поля ReplicatedFrom, ReplicatedAt
- Стабильный PeerID: приватный ключ в state/

---

## v1.14 (2026-08-16)

### Добавлено
- Голосовая стеганография: LSB в WAV
- node/stego.go: embedLSB, extractLSB
- Эндпоинт POST /send_stego

---

## v1.13 (2026-08-16)

### Добавлено
- Обфускация: AES-GCM с префиксом [SHUF]
- Случайные задержки 5-50 мс
- Дедупликация: форвардинг только из handleSend

---

## v1.12 (2026-08-16)

### Добавлено
- Селф-хилинг: heartbeat каждые 30 сек
- Обнаружение мёртвых пиров: 5 сек без PONG
- Автоперезапуск

---

## v1.11 (2026-08-16)

### Добавлено
- Исчезающие сообщения (TTL): вечно, 60 сек, 3600 сек
- ExpiresAt в Message
- Локальное шифрование state: AES-256-GCM

---

## v1.10 (2026-08-15)

### Добавлено
- Onion Routing v1: три режима анонимности
- mode=0 обычный, mode=1 анонимный, mode=2 скрытый

---

## v1.9 (2026-08-14)

### Добавлено
- 5 узлов в docker-compose
- Документация: три столпа ISOTOPE
- README (EN + RU) под новую концепцию
- docs/FAQ.md: 20 вопросов

---

## v1.8 (2026-08-14)

### Добавлено
- Ассоциативная память
- Децентрализованный bootstrap: mDNS, DHT, вручную через ENV

---

## v1.7 (2026-08-14)

### Добавлено
- WebSocket + TLS: трафик неотличим от HTTPS
- Новый универсальный этический хеш: семь заповедей

---

## v1.6 (2026-08-11)

### Добавлено
- Priority Gossip: поле Priority в Message
- TTL форвардинга зависит от приоритета

---

## v1.5 (2026-07-27)

### Добавлено
- REST API для внешних клиентов
- WebSocket для реального времени
- Пагинация для /messages
- CORS middleware
- Мобильное приложение (Flutter, базовая версия)
- Защита памяти (лимит 10 000 сообщений, архив)

---

## v1.4 (2026-07-19)

### Добавлено
- 100-мерные векторы (VectorDim = 100)
- Биграммы в textToVector
- Марковские цепочки для русских ответов
- Кнопки preHash/antiHash в дашборде
- Мониторинг здоровья сети
- 55 автотестов (позже 67)

---

## v1.3 (2026-07-19)

### Добавлено
- Персональные фильтры: preHash и antiHash
- Эндпоинты /setprehash и /setantihash

---

## v1.2 (2026-07-19)

### Добавлено
- Этическое обучение нейросети через лайки/дизлайки

---

## v1.1 (2026-07-19)

### Добавлено
- Этический иммунитет: начальный вес зависит от близости к хешу
- Семь заповедей как этический хеш

---

## v1.0 (2026-07-16)

### Первый стабильный релиз
- P2P-сеть на libp2p + mDNS (3 узла)
- Нейросеть: 10 нейронов на слой
- Взвешенная память
- Чат с дашбордом
- 37 успешных тестов

---

**ISOTOPE эволюционирует.**
**Каждая версия — шаг к неуязвимой связи.**