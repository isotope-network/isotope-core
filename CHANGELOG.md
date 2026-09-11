# История изменений ISOTOPE

## v1.21.0 (2026-09-11)

### Исправлено
- Дубликаты сообщений (корень в Go-ядре):
  - processMessageWithID() — принимает ID параметром, не генерирует
  - SendMessage() передаёт свой ID в processMessageWithID()
  - replicateMessage() исключает отправителя: if p.String() == msg.Sender { continue }
  - Dart _addMessage() — простая проверка containsKey(msg.id)
- Бейдж непрочитанных и линия «Непрочитанные»:
  - _ownMessageIds сохраняется в SharedPreferences (own_message_ids)
  - unreadSnapshot — снимок до обнуления, используется для линии
  - setChatOpen(false) вызывается в ConnectScreen после Navigator.pop
  - loadMessages() загружает все сообщения (свои и входящие)
- Зависание UI на медленных телефонах (ANR):
  - Все вызовы Mobile.* в MainActivity.kt обёрнуты в Thread { ... } + runOnUiThread
  - Обёрнуты: start, stop, send, getMessages, getPeers, getStatus, getMultiaddrs, connectToPeer, joinDHT, findPeer, findPeersViaNetwork, provide, getDHTInfo

### Добавлено
- Reconnect loop + keepalive (node/node.go):
  - reconnectLoop() — каждые 30 сек проверяет len(Network().Peers())
  - Если 0 — переподключается к bootstrap + ExchangePeers
  - pingPeers() — интервал сокращён с 30 до 15 секунд
- Проверено: peers 2 → 0 (5 мин свёрнутыми) → 2 (30-60 сек после разворачивания)

### Проверено на реальных телефонах
- P2P-сообщения через VPS relay
- Бейдж непрочитанных
- Линия «Непрочитанные»
- История (свои/входящие, загрузка после перезапуска)
- Прокрутка истории
- Reconnect после сворачивания
- Отсутствие ANR на медленном телефоне

### Инфраструктура
- VPS bootstrap/relay:
  - IP: 186.246.31.176
  - PeerID: QmNmr3YqGD9uKpPCx7W86t7Tc3vrBJF1GbmTAzDQ25Sskx
  - Bootstrap multiaddr: /ip4/186.246.31.176/tcp/9001/ws/p2p/QmNmr3YqGD9uKpPCx7W86t7Tc3vrBJF1GbmTAzDQ25Sskx
  - Порты: 9000 (TCP), 9001 (WS), 8081 (HTTP API)
- ВАЖНО: не удалять /root/isotope/state/ — PeerID изменится, телефоны потеряют связь

---

## v1.19.0 (2026-09-01)

### Добавлено
- Рефакторинг ядра: package main → package core
- main/main.go — точка входа для десктопа
- mobile/mobile.go — обёртка для gomobile
- libp2p через FFI (.aar 67 МБ, подключён к Flutter через MethodChannel)
- Методы мобильной обёртки: start, send, getMessages, getPeers, getWeight, getStatus, getMultiaddrs, connectToPeer
- GetMultiaddrs() и ConnectToPeer() в ядро
- ListenIP в Config
- Стабильный PeerID: приватный ключ в isotope_state.json.key, восстановление при перезапуске
- Обработка смены сети: connectivity_plus, debounce 10 сек, перезапуск узла с новым IP, приоритет Wi-Fi
- Модель NodeInfo: PeerID, multiaddrs, lastSeen, status
- Хранение списка узлов в SharedPreferences
- Heartbeat с счётчиком неудач (3 → dead), снятие dead при получении сообщения
- Кнопка «Обновить» — полная очистка и пересканирование
- Логирование: перехват логов Go-ядра и передача в Flutter
- Кнопка «Сохранить журнал» через Android Intent
- Динамический поиск свободного порта (8081+), порт передаётся через NSD

### Что работает
- HTTP-связь между телефонами
- libp2p P2P-соединение
- NSD-обнаружение с PeerID и multiaddr
- Отправка/приём сообщений
- Бейджи непрочитанных
- История сообщений
- Стабильный PeerID
- Обработка смены сети

---

## v1.18.1 (2026-08-28)

### Добавлено
- Рефакторинг ядра: package main → package core
- Точка входа: node/main/main.go (package main)
- Экспорт публичного API: Config, NewNode, InitP2P, StartHTTP, StartMobile, Stop
- HashText — экспортирован
- Транспорты настраиваются через Config
- Подготовка структуры для gomobile (node/mobile/mobile.go)

### Изменено
- Ядро теперь — библиотека, а не исполняемый файл
- Dockerfile: сборка из node/main
- docker-compose.yml: 5 узлов с новой структурой
- Makefile: команды для библиотеки + CLI
- go test ./... — ok

### Миграция
Для запуска узла:

go build -o isotope-node ./node/main
./isotope-node --config config.json

Для использования как библиотеки:

import core "sbimain"

---

## v1.18 (2026-08-16)

### Добавлено
- Каналы с весовыми уровнями (G4): Channel, ChannelStore
- Пороги доступа: full=0.3, comment=0.5, vote=0.7
- Эндпоинты: POST/GET /channels, POST/GET /channels/{id}/messages
- Доступ зависит от веса узла

---

## v1.17 (2026-08-16)

### Добавлено
- Самоадаптация: node/adapt.go
- Сбор метрик: avgWeight, lowWeightRatio, highWeightRatio
- Правила: порог архивации, интервал синхронизации
- Фоновая адаптация каждые 5 минут
- Адаптивный learningRate
- Адаптивный порог архивации и очистки

---

## v1.16 (2026-08-16)

### Добавлено
- Onion Routing v2: цепочка из 4 relay (анонимный режим)
- Цепочка из 5 relay + задержка (скрытый режим)
- Выбор relay по весу > 0.7
- Fallback на обычных пиров
- getPeerWeight — средний вес сообщений пира

---

## v1.15 (2026-08-16)

### Добавлено
- Репликация сообщений на 2 случайных живых узла
- Поля ReplicatedFrom, ReplicatedAt
- Восстановление при старте через [RESTORE] и [REPLICA]
- Стабильный PeerID: приватный ключ в state/private_key_N.bin

---

## v1.14 (2026-08-16)

### Добавлено
- Голосовая стеганография: LSB-встраивание в WAV
- node/stego.go: embedLSB, extractLSB
- Случайное распределение через seed от ключа
- node/stego_test.go: 5 юнит-тестов
- Эндпоинт POST /send_stego
- [STEGO] префикс в handleStream
- Буфер увеличен до 2 МБ

---

## v1.13 (2026-08-16)

### Добавлено
- Обфускация трафика: AES-GCM с префиксом [SHUF]
- Случайные задержки 5-50 мс
- Дедупликация: форвардинг только из handleSend
- Лимиты ресурсов: 0.5 CPU, 512 MiB на узел

---

## v1.12 (2026-08-16)

### Добавлено
- Селф-хилинг: heartbeat каждые 30 сек
- Обнаружение мёртвых пиров: 5 сек без PONG
- Relay выбирает только живых
- Автоперезапуск: restart: unless-stopped

---

## v1.11 (2026-08-16)

### Добавлено
- Исчезающие сообщения (TTL): вечно, 60 сек, 3600 сек
- ExpiresAt в структуре Message
- Фоновая очистка раз в 60 секунд
- Локальное шифрование state: AES-256-GCM
- Пароль из ENV: ISOTOPE_STATE_PASSWORD

---

## v1.10 (2026-08-15)

### Добавлено
- Onion Routing v1: три режима анонимности
- mode=0: обычный (прямое соединение)
- mode=1: анонимный (цепочка из 2 relay-пиров)
- mode=2: скрытый (3 relay-пира + задержка 10-60 сек)
- selectRelays: случайный выбор пиров
- sendViaRelayChain: отправка через цепочку
- Поле Relayed в структуре Message

---

## v1.9 (2026-08-14)

### Добавлено
- 5 узлов в docker-compose
- Документация: три столпа ISOTOPE
- README (EN + RU) под новую концепцию
- docs/FAQ.md: 20 вопросов экспертов
- Весовая модель доступа

---

## v1.8 (2026-08-14)

### Добавлено
- Ассоциативная память: узлы запоминают, кто у кого что спрашивал
- Децентрализованный bootstrap: mDNS, DHT, вручную через ENV

---

## v1.7 (2026-08-14)

### Добавлено
- WebSocket + TLS: трафик неотличим от HTTPS
- Новый универсальный этический хеш: семь заповедей, собранных из всех учений

---

## v1.6 (2026-08-11)

### Добавлено
- Priority Gossip: поле Priority в Message
- TTL форвардинга зависит от приоритета
- Приоритет от узлов с весом > 0.7

---

## v1.5 (2026-07-27)

### Добавлено
- REST API для внешних клиентов
- WebSocket для реального времени
- Пагинация для /messages
- CORS middleware на всех эндпоинтах
- Мобильное приложение (Flutter, базовая версия)
- Статусы доставки с форвардингом
- Защита памяти (лимит 10 000 сообщений, архив)

### Изменено
- Синхронизация слоёв через gossip-протокол
- Метрики логирования: английские метки [MSG], [TRAIN], [SYNC], [FEEDBACK]

---

## v1.4 (2026-07-19)

### Добавлено
- 100-мерные векторы (VectorDim = 100)
- Биграммы в textToVector
- Марковские цепочки для русских ответов
- Кнопки preHash/antiHash в дашборде
- Мониторинг здоровья сети
- 55 автотестов (позже расширено до 67)

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