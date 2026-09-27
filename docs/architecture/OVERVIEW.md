# Архитектура ISOTOPE

## Обзор

ISOTOPE — инфраструктура для этичного, неуязвимого,
самообучающегося обмена данными и ИИ.

    [Пользователь] ←→ [Мессенджер / API] ←→ [Узел ISOTOPE] ←→ P2P-сеть ←→ [Другие узлы]
                            │                        │
                      [Нейросеть]              [ИИ-модель]
                      [Память]                 [Этический паспорт]
                      [Этический движок]       [Инференс]

---

## Структура ядра (v1.26.0)

Ядро ISOTOPE — **библиотека** (пакет `core`).

node/
├── *.go              # package core (ядро)
├── main/main.go      # package main (точка входа для десктопа)
└── mobile/mobile.go  # package mobile (обёртка gomobile)

### Экспортированный API ядра

package core

type Config struct {
    EthHash    string
    Transports []string
    Bootstrap  []string
    Port       int
    ListenIP   string
}

func NewNode(config Config) *Node
func InitP2P(node *Node) error
func StartHTTP(node *Node) error
func StartMobile(node *Node) error
func Stop(node *Node) error
func HashText(text string) string
func GetMultiaddrs(node *Node) []string
func ConnectToPeer(node *Node, addr string) error

### Методы Node

func (n *Node) SendMessage(text string, ttl int) (string, error)
func (n *Node) SendToPeer(recipient, text string, ttl int) (string, error)
func (n *Node) AddContact(peerID, ed25519Pub, x25519Pub, signature, name string) string
func (n *Node) GetMessages() []Message
func (n *Node) GetPeers() []string
func (n *Node) GetWeight() float64
func (n *Node) GetStatus() string

### Точка входа (десктоп)

go build -o isotope-node ./node/main
./isotope-node --config config.json

---

## Мобильная обёртка (mobile.go)

**Статус:** работает (v1.26.0)

### Методы (возвращают JSON-строки)

| Метод | Описание |
|-------|----------|
| Start(stateFile string) | Запуск узла, восстановление PeerID |
| Send(text string, ttl int) | Отправка (broadcast) |
| SendToPeer(recipient, text, ttl) | Отправка конкретному контакту (E2E) |
| GetMessages() | Список сообщений |
| GetPeers() | Список пиров |
| GetWeight() | Вес узла |
| GetStatus() | JSON-статус |
| GetMultiaddrs() | Список адресов узла |
| ConnectToPeer(addr string) | Подключение к пиру по адресу |
| ConnectToPeerWithFallback(addrs) | Подключение с перебором адресов |
| Announce(json) | Публикация адресов (массив) |
| FindPeerByID(peerID string) | Поиск пира по PeerID |
| AddContact(peerID, ed25519Pub, x25519Pub, signature, name) | Добавить контакт |
| GetMyQRData() | QR-данные (peerID, ed25519_pub, x25519_pub, signature) |

### Ключевые особенности

- **libp2p через FFI:** .aar, MethodChannel во Flutter
- **Non-blocking вызовы:** все Mobile.* обёрнуты в Thread + runOnUiThread
- **Стабильный PeerID:** приватный ключ в isotope_state.json.key
- **Смена сети:** connectivity_plus, debounce 10 сек
- **Reconnect loop:** с exponential backoff (1 → 30 сек)
- **Flush on reconnect:** три уровня (Notifiee, markPeerAlive, reconnectLoop)
- **NodeInfo:** PeerID, multiaddrs, lastSeen, status
- **Heartbeat:** 3 неудачи → dead
- **Логирование:** Go → Flutter, обрезка 4KB
- **Порты:** динамический поиск (8081+), через NSD
- **Обнаружение:** NSD с PeerID и multiaddr
- **QR = PeerID + ключи:** контакт устанавливается через FIND + fallback

### Архитектурное ограничение (Android)

- InterfaceListenAddresses недоступен (permission denied)
- Получение IP через Go невозможно. Только через Dart.
- Обход: QR через PeerID + ключи

---

## E2E-шифрование (v1.24)

### Ключи

Три ключа в системе:

| Ключ | Назначение | Тип | Файл |
|------|------------|-----|------|
| PeerID | Идентификация в libp2p | RSA 2048 | isotope_state.json.key |
| Ed25519 | Подпись контакта | Ed25519 (64/32 байта) | isotope_state.json.ed25519.key |
| X25519 | Шифрование E2E | X25519 (32 байта) | isotope_state.json.x25519.key |

Публичные ключи не хранятся — вычисляются из приватных:
- X25519: curve25519.X25519(priv, Basepoint)
- Ed25519: priv.Public()

### QR-формат (v1)

{
  "v": 1,
  "peerID": "QmX...",
  "ed25519_pub": "base64...",
  "x25519_pub": "base64...",
  "signature": ""
}

QR.v — версия формата QR. Message.Version — версия формата сообщения.

### Шифрование

- box.Seal (X25519 + XSalsa20-Poly1305)
- Формат payload: nonce(24) || ciphertext
- Version = 2 — E2E-сообщения
- Version = 0 — история (без E2E)

### Подпись (v1.24)

- Ed25519 подпись над peerID || x25519_pub
- Проверка при добавлении контакта
- Флаг verified: сигнал UI, не пропуск
- verified: false — контакт не проверен, отправка разрешена (если есть x25519)
- Нет x25519 — отправка блокируется

### Контакты

Файл: isotope_contacts.json (рядом со state)

{
  "v": 1,
  "contacts": [
    {
      "peerID": "QmX...",
      "ed25519_pub": "base64...",
      "x25519_pub": "base64...",
      "signature": "",
      "verified": false,
      "name": "",
      "added_at": "..."
    }
  ]
}

### PlainText (v1.25)

- Message.PlainText — открытый текст для UI
- Свои E2E: Text = шифротекст, PlainText = открытый
- Входящие: расшифровка на лету
- UI показывает PlainText для isOwn && Version == 2
- Автоочистка: удаление своих E2E без PlainText при первом запуске (.e2e_cleanup флаг)

---

## Relay-circuit (v1.23+)

**Проблема:** узлы за NAT (мобильные, CGNAT) не могут принимать входящие соединения.

**Решение:** relay через VPS.

- Узел резервирует слот на VPS (client.Reserve)
- VPS форвардит трафик
- Не хранит, только маршрутизирует
- ANNOUNCE автоматически добавляет relay-адрес
- FIND fallback — relay-адрес

**Обновление резервации (v1.25):**
- Notifiee ConnectedF — немедленный refresh при reconnect
- relayLoop — каждые 30 секунд с проверкой существования
- Exponential backoff 2 → 30 сек для retry
- Throttle 10 сек (не дёргать на каждом reconnect)

**Условие отключения VPS:**

VPS отключается, когда одновременно выполнены три условия:
1. DHT покрывает 15+ узлов (иммунитет включается)
2. Hole punching работает для большинства NAT
3. Есть 2-3 независимых relay-узла в сети (не на нашем VPS)

---

## Иммунитет

Иммунитет — способность сети сохранять себя
без центрального управления.

У иммунитета две стороны: техническая и социальная.

### Технический иммунитет

Способность восстанавливать связь:

- Три уровня защиты:
  Notifiee ConnectedF, markPeerAlive (dead → alive), reconnectLoop
- Репликация: на 2+ живых узла
- Селф-хилинг: heartbeat каждые 30 сек
- Автоматический fallback: локальный IP → relay
- Exponential backoff для reconnect

### Социальный иммунитет

Способность очищать среду от деструктива:

- Каждое сообщение получает вес от этического хеша
- Лайки поднимают вес. Дизлайки опускают
- Время размывает даже сильные сигналы
- Когда вес падает ниже порога — сообщение уходит в архив
- Ниже — удаляется навсегда

Это не модерация. Это коллективный иммунный ответ.

### Единство двух сторон

Технический иммунитет держит сеть живой.
Социальный иммунитет держит её чистой.

Вместе они делают ISOTOPE неубиваемой.

### Иммунитет — функция масштаба

| Узлов | Технический | Социальный |
|-------|-------------|------------|
| 2-3 | Защищённый канал | Личное доверие |
| 5-10 | Ранний консенсус | Первые оценки |
| 15-20 | Иммунитет включается | Коллективная очистка |
| 50+ | Стабильный иммунитет | Устойчивые нормы |
| 100+ | Неубиваемая сеть | Саморегуляция |

---

## Три столпа ISOTOPE

**Данные.**
Запрос-ответ без раскрытия.

**ИИ.**
Распределённый инференс.

**Люди.**
Мессенджер как точка входа.

Фундамент — **Сеть**:
P2P, этический хеш, иммунитет, самообучение.

---

## Ключевые компоненты

### 1. P2P-сеть (libp2p)

- Транспорт: TCP + WebSocket + TLS
- Обнаружение: mDNS + DHT Kademlia + NSD (мобильный)
- Синхронизация: Gossip-протокол
- Маршрутизация: Onion Routing v2 (4-5 relay, по весу > 0.7)
- Relay-circuit: для узлов за NAT (v1.23+)
- Маскировка: WebSocket + TLS + обфускация AES-GCM
- Приоритеты: Priority Gossip
- Память: ассоциативная
- Восстановление: репликация на 2 живых узла
- Адаптация: самонастройка порогов
- Reconnect: exponential backoff
- Flush: три уровня

### 2. Этический движок (нейросеть)

- Вектор: 100 измерений (VectorDim = 100)
- Слои: растут каждые 20 сообщений
- Обучение: градиентный спуск, lr адаптивный
- Контекст: косинусное расстояние > 0.7
- Хеш: семь универсальных заповедей

### 3. Память

- Тип: взвешенная, FIFO с архивом
- Вес: от этического хеша + preHash/antiHash + оценки
- Старение: -0.01/час
- Архивация: вес < порог (адаптивный)
- Удаление: вес < порог (адаптивный)
- Исчезновение: TTL (60 сек, 3600 сек, вечно)
- Шифрование: AES-256-GCM
- Репликация: на 2 случайных живых узла
- PlainText: открытый текст для UI (свои E2E)

### 4. Каналы

- Channel, ChannelStore
- Пороги доступа: full=0.3, comment=0.5, vote=0.7
- Эндпоинты: POST/GET /channels, POST/GET /channels/{id}/messages

### 5. Безопасность

- Анонимность: Onion Routing v2
- Маскировка: WebSocket + TLS + обфускация
- Стеганография: LSB в WAV
- Селф-хилинг: heartbeat
- Стабильный ID: state/ (десктоп), isotope_state.json.key (мобильный)
- E2E: X25519 + box.Seal
- Подпись: Ed25519

### 6. Весовая модель доступа

| Вес | Права |
|-----|-------|
| 0–0.3 | Читать (тизер) |
| 0.3–0.5 | Полный доступ |
| 0.5–0.7 | Комментировать, модерировать |
| 0.7–1.0 | Голосовать, relay, реплики |

### 7. Самоадаптация

- node/adapt.go: сбор метрик
- avgWeight, lowWeightRatio, highWeightRatio
- Фоновая адаптация каждые 5 минут

### 8. ИИ-слой (ISOTOPE AI Mesh)

- Модели: облегчённые версии на узлах
- Инференс: простые запросы — ближайший узел
- Маршрутизация: сложные запросы — дата-центр
- Этический паспорт: проверка модели
- Статус: v3.0+

### 9. API

- REST: /send, /messages, /status, /health, /feedback
- Каналы: /channels
- WebSocket: /ws
- Фильтры: /setprehash, /setantihash
- Стего: /send_stego

---

## Уровни защиты

| Уровень | Что защищено | Статус |
|---------|--------------|--------|
| Обфускация | Маскировка трафика | ✅ v1.13 |
| E2E-шифрование | Содержимое (от relay) | ✅ v1.24 |
| Подпись | Подлинность отправителя | ✅ v1.24 |
| Метаданные | Кто с кем общается | 🔜 v2.0+ (Onion) |
| Временны́е паттерны | Тайминг, объём | 🔜 Padding, mixing |

**Сейчас:** содержимое защищено, подлинность проверена, relay видит только шифротекст.

**Позже:** защита метаданных (Onion), защита паттернов (padding, mixing).

---

## Инфраструктура

### VPS bootstrap/relay

- IP: 186.246.31.176
- PeerID: QmR8u5YFdcKpM2onQvk7KV5qioai87aysi9JWLdV1LX1bi
- Bootstrap multiaddr: /ip4/186.246.31.176/tcp/9001/ws/p2p/QmR8u5YFdcKpM2onQvk7KV5qioai87aysi9JWLdV1LX1bi
- Порты: 9000 (TCP), 9001 (WS), 8081 (HTTP API)

НЕ удалять /root/isotope/state/ — PeerID изменится, телефоны потеряют связь.

### Обновление VPS

cd /root/isotope-core && git pull && cd node && go build -o isotope-node ./main
# Ctrl+C в окне VPS, затем:
cd /root/isotope && NODE_ID=bootstrap ISOTOPE_PORT=9000 ISOTOPE_HTTP_PORT=8081 ISOTOPE_ENABLE_RELAY=true /root/isotope-core/node/isotope-node

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

## Принципы

1. **Децентрализация.** Нет сервера, нет единой точки отказа
2. **Этический иммунитет.** Сеть отличает добро от зла математически
3. **Самообучение.** Сообщество учит сеть через оценки
4. **Приватность.** Данные не покидают устройство без согласия
5. **Неуязвимость.** Сеть нельзя заблокировать, отключить или взломать
6. **Унификация.** Каждый механизм — кирпич для множества применений
7. **Правка в корне.** Не костыли, а исправление причины
8. **Открытость без наивности.** Публичное — для сообщества, внутреннее — для команды
9. **Эмерджентное доверие.** Verified — сигнал, не пропуск
10. **Приватность по умолчанию.** E2E для каждого сообщения

---

## Масштаб и иммунитет

Иммунная система ISOTOPE включается при 15+ узлах.
До этого сеть работает как защищённый P2P-протокол.

[Шкала иммунитета →](IMMUNITY_SCALE.md)

---

## Подробнее

- [P2P-сеть](P2P_NETWORK.md)
- [Этический движок](ETHICS_ENGINE.md)
- [Весовая модель доступа](WEIGHT_ACCESS_MODEL.md)
- [Токеномика](TOKENOMICS.md)
- [B2B-обмен данными](B2B_DATA_EXCHANGE.md)
- [Самоадаптация](AUTONOMOUS_ADAPTATION.md)
- [Шкала иммунитета](IMMUNITY_SCALE.md)