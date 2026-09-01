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

## Структура ядра (v1.19.0)

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

**Статус:** работает (v1.19.0)

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

### Ключевые особенности

- **libp2p через FFI:** .aar (67 МБ), MethodChannel во Flutter
- **Стабильный PeerID:** приватный ключ в isotope_state.json.key
- **Смена сети:** connectivity_plus, debounce 10 сек, перезапуск узла
- **NodeInfo:** PeerID, multiaddrs, lastSeen, status
- **Heartbeat:** 3 неудачи → dead, снятие при получении сообщения
- **Логирование:** Go → Flutter, сохранение через Android Intent
- **Порты:** динамический поиск (8081+), передача через NSD
- **Обнаружение:** NSD с PeerID и multiaddr

### Не решено

- BLE — нестабилен, отключён
- Samsung Android 10 — краш при запуске
- DHT — только в ядре, не в мобильном

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
- **Маршрутизация:** Onion Routing v2 (цепочка из 4-5 relay-пиров, выбор по весу > 0.7)
- **Маскировка:** WebSocket + TLS + обфускация AES-GCM
- **Приоритеты:** Priority Gossip — TTL зависит от веса узла
- **Память:** ассоциативная — узлы запоминают, кто у кого что спрашивал
- **Восстановление:** репликация на 2 случайных живых узла
- **Адаптация:** самонастройка порогов и интервалов

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

### 9. API

- **REST:** /send, /messages, /status, /health, /feedback
- **Каналы:** /channels, /channels/{id}/messages
- **WebSocket:** /ws (двунаправленный канал)
- **Фильтры:** /setprehash, /setantihash
- **Стего:** /send_stego

---

## Принципы

1. **Децентрализация.** Нет сервера, нет единой точки отказа
2. **Этический иммунитет.** Сеть отличает добро от зла математически
3. **Самообучение.** Сообщество учит сеть через оценки
4. **Приватность.** Данные не покидают устройство без согласия
5. **Неуязвимость.** Сеть нельзя заблокировать, отключить или взломать
6. **Унификация.** Каждый механизм — кирпич для множества применений

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