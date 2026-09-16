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

## Структура ядра (v1.23.0)

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
func (n *Node) GetMessages() []Message
func (n *Node) GetPeers() []string
func (n *Node) GetWeight() float64
func (n *Node) GetStatus() string

### Точка входа (десктоп)

go build -o isotope-node ./node/main
./isotope-node --config config.json

### Использование как библиотеки

import core "sbimain"

config := core.Config{
    Transports: []string{"ws"},
    Port:       9001,
}

node := core.NewNode(config)
core.InitP2P(node)
core.StartHTTP(node)
defer core.Stop(node)

---

## Мобильная обёртка (mobile.go)

**Статус:** работает (v1.23.0)

### Методы (возвращают JSON-строки)

| Метод | Описание |
|-------|----------|
| Start(stateFile string) | Запуск узла, восстановление PeerID |
| Send(text string, ttl int) | Отправка сообщения |
| GetMessages() | Список сообщений |
| GetPeers() | Список пиров |
| GetWeight() | Вес узла |
| GetStatus() | JSON-статус |
| GetMultiaddrs() | Список адресов узла |
| ConnectToPeer(addr string) | Подключение к пиру по адресу |
| ConnectToPeerWithFallback(addrs) | Подключение с перебором адресов |
| Announce(json) | Публикация адресов (массив) |
| FindPeerByID(peerID string) | Поиск пира по PeerID |
| JoinDHT() | Вход в DHT |
| Provide(key string) | Публикация в DHT |

### Ключевые особенности

- **libp2p через FFI:** .aar (67 МБ), MethodChannel во Flutter
- **Non-blocking вызовы:** все Mobile.* обёрнуты в Thread + runOnUiThread (MainActivity.kt) — устранён ANR
- **Стабильный PeerID:** приватный ключ в isotope_state.json.key
- **Смена сети:** connectivity_plus, debounce 10 сек, перезапуск узла
- **Reconnect loop:** каждые 30 сек проверка пиров, при 0 — переподключение к bootstrap
- **Flush on reconnect:** три уровня (Notifiee ConnectedF, markPeerAlive, reconnectLoop)
- **pingPeers():** интервал 15 сек
- **NodeInfo:** PeerID, multiaddrs, lastSeen, status
- **Heartbeat:** 3 неудачи → dead, снятие при получении сообщения
- **Логирование:** Go → Flutter, сохранение через Android Intent
- **Порты:** динамический поиск (8081+), передача через NSD
- **Обнаружение:** NSD с PeerID и multiaddr
- **QR = PeerID:** контакт устанавливается через FIND + fallback

### Архитектурное ограничение (Android)

- **InterfaceListenAddresses недоступен** — permission denied на Android
- Получение IP через Go невозможно. Всегда через Dart.
- Обход: QR через PeerID, не через локальный IP

### Единый ID сообщений (v1.21.0)

- ID генерируется один раз в SendMessage()
- processMessageWithID() принимает ID параметром
- replicateMessage() исключает отправителя (if p.String() == msg.Sender)
- Dart _addMessage() — проверка containsKey(msg.id)

### Не решено

- BLE — нестабилен, отключён
- Samsung Android 10 — вероятно, решён non-blocking вызовами
- DHT — только в ядре, не в мобильном
- Foreground Service — работает, но улучшается
- Параллельный dial — в backlog (убирает задержку 5-7 сек)

---

## Relay-circuit (v1.23.0)

**Проблема:** узлы за NAT (мобильные, CGNAT) не могут принимать входящие соединения.

**Решение:** relay через VPS.

- Узел резервирует слот на VPS (`client.Reserve`)
- VPS форвардит трафик
- Не хранит, только маршрутизирует
- ANNOUNCE автоматически добавляет relay-адрес
- FIND fallback — relay-адрес, если announced пуст

**Формат ANNOUNCE (multi-address):**

    [ANNOUNCE]
    <addr1>
    <addr2>
    [END]

- `announcedPeer` — список `[]string`, не одна строка
- FIND отдаёт массив адресов
- `ConnectToPeerWithFallback` пробует адреса по очереди

**Условие отключения VPS:**

> VPS отключается, когда одновременно выполнены три условия:
> 1. DHT покрывает 15+ узлов (иммунитет включается)
> 2. Hole punching работает для большинства NAT (CGNAT, симметричный NAT)
> 3. Есть 2-3 независимых relay-узла в сети (не на нашем VPS)

**До этого момента VPS остаётся.** Это не костыль — это стандартная инфраструктура bootstrap/relay, как в любом libp2p-проекте.

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

### Социальный иммунитет

Способность очищать среду от деструктива:

- Каждое сообщение получает вес от этического хеша
- Лайки поднимают вес. Дизлайки опускают
- Время размывает даже сильные сигналы
- Когда вес падает ниже порога — сообщение уходит в архив
- Ниже — удаляется навсегда

Это не модерация. Это коллективный иммунный ответ.
Массив пользователей сам очищает среду от того,
что противоречит ценностям сети.

Никто не решает за других. Каждый участник вносит вклад:
лайком, дизлайком, временем.

### Единство двух сторон

Технический иммунитет держит сеть живой.
Социальный иммунитет держит её чистой.

Вместе они делают ISOTOPE неубиваемой:
сеть нельзя отключить (техника)
и нельзя засорить (социум).

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
Банки, страховые, больницы, поставщики обмениваются
ответами на вопросы, не передавая сами данные.

**ИИ.**
Распределённый инференс.
Облегчённые модели работают на узлах сети.
Простые запросы обрабатываются ближайшим узлом.
Сложные — уходят в дата-центр.
Каждая модель проходит этический паспорт перед загрузкой.

**Люди.**
Мессенджер как точка входа.
Общение без цензуры, блокировок и слежки.
Узел пользователя автоматически участвует в работе сети.

Фундамент — **Сеть**:
P2P, этический хеш, иммунитет, самообучение.

---

## Ключевые компоненты

### 1. P2P-сеть (libp2p)

- **Транспорт:** TCP + WebSocket + TLS
- **Обнаружение:** mDNS (локалка) + DHT Kademlia (WAN) + NSD (мобильный)
- **Синхронизация:** Gossip-протокол (TTL=3, fanout=√N)
- **Маршрутизация:** Onion Routing v2 (4-5 relay, выбор по весу > 0.7)
- **Relay-circuit:** для узлов за NAT (v1.23.0)
- **Маскировка:** WebSocket + TLS + обфускация AES-GCM
- **Приоритеты:** Priority Gossip — TTL зависит от веса узла
- **Память:** ассоциативная
- **Восстановление:** репликация на 2 живых узла (без отправителя)
- **Адаптация:** самонастройка порогов и интервалов
- **Reconnect:** reconnectLoop() каждые 30 сек, pingPeers() каждые 15 сек
- **Flush:** три уровня (Notifiee, markPeerAlive, reconnectLoop)

### 2. Этический движок (нейросеть)

- **Вектор:** 100 измерений (VectorDim = 100)
- **Слои:** растут каждые 20 сообщений
- **Обучение:** градиентный спуск, lr адаптивный
- **Контекст:** поиск похожих (косинусное расстояние > 0.7)
- **Хеш:** семь универсальных заповедей

### 3. Память

- **Тип:** взвешенная, FIFO с архивом
- **Вес:** от этического хеша + preHash/antiHash + оценки
- **Старение:** -0.01/час
- **Архивация:** вес < порог (адаптивный)
- **Удаление:** вес < порог (адаптивный)
- **Исчезновение:** TTL (60 сек, 3600 сек, вечно)
- **Шифрование:** AES-256-GCM (ISOTOPE_STATE_PASSWORD)
- **Репликация:** на 2 случайных живых узла

### 4. Каналы

- **Channel, ChannelStore**
- Пороги доступа: full=0.3, comment=0.5, vote=0.7
- Эндпоинты: POST/GET /channels, POST/GET /channels/{id}/messages
- Доступ зависит от веса узла

### 5. Безопасность

- **Анонимность:** Onion Routing v2 (4-5 relay)
- **Маскировка:** WebSocket + TLS + обфускация
- **Стеганография:** LSB в WAV (голос пользователя)
- **Селф-хилинг:** heartbeat каждые 30 сек
- **Стабильный ID:** приватный ключ в state/ (десктоп), isotope_state.json.key (мобильный)

### 6. Весовая модель доступа

Вес узла — универсальный пропуск.

| Вес | Права |
|-----|-------|
| 0–0.3 | Читать (тизер) |
| 0.3–0.5 | Полный доступ |
| 0.5–0.7 | Комментировать, модерировать |
| 0.7–1.0 | Голосовать, relay, реплики |

[Подробнее →](WEIGHT_ACCESS_MODEL.md)

### 7. Самоадаптация

- node/adapt.go: сбор метрик
- avgWeight, lowWeightRatio, highWeightRatio
- Фоновая адаптация каждые 5 минут
- Адаптивный learningRate, порог архивации, очистки

### 8. ИИ-слой (ISOTOPE AI Mesh)

- **Модели:** облегчённые версии на узлах
- **Инференс:** простые запросы — ближайший узел
- **Маршрутизация:** сложные запросы — дата-центр
- **Этический паспорт:** проверка модели перед загрузкой
- **Статус:** v3.0+
- **E2E шифрование:** в плане (до контакт-протокола)

### 9. API

- **REST:** /send, /messages, /status, /health, /feedback
- **Каналы:** /channels, /channels/{id}/messages
- **WebSocket:** /ws (двунаправленный канал)
- **Фильтры:** /setprehash, /setantihash
- **Стего:** /send_stego

---

## Инфраструктура

### VPS bootstrap/relay (v1.23.0)

- IP: 186.246.31.176
- PeerID: QmR8u5YFdcKpM2onQvk7KV5qioai87aysi9JWLdV1LX1bi
- Bootstrap multiaddr: /ip4/186.246.31.176/tcp/9001/ws/p2p/QmR8u5YFdcKpM2onQvk7KV5qioai87aysi9JWLdV1LX1bi
- Порты: 9000 (TCP), 9001 (WS), 8081 (HTTP API)
- Код на коммите: 6b223c0

⚠️ НЕ удалять /root/isotope/state/ — PeerID изменится, телефоны потеряют связь.

### Обновление VPS

cd /root/isotope-core && git pull && cd node && go build -o isotope-node ./main
# Ctrl+C в окне VPS, затем:
cd /root/isotope && NODE_ID=bootstrap ISOTOPE_PORT=9000 ISOTOPE_HTTP_PORT=8081 ISOTOPE_ENABLE_RELAY=true /root/isotope-core/node/isotope-node

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

## Принципы

1. **Децентрализация.** Нет сервера, нет единой точки отказа
2. **Этический иммунитет.** Сеть отличает добро от зла математически
3. **Самообучение.** Сообщество учит сеть через оценки
4. **Приватность.** Данные не покидают устройство без согласия
5. **Неуязвимость.** Сеть нельзя заблокировать, отключить или взломать
6. **Унификация.** Каждый механизм — кирпич для множества применений
7. **Правка в корне.** Не костыли, а исправление причины
8. **Открытость без наивности.** Публичное — для сообщества, внутреннее — для команды

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