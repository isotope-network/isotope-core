package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/multiformats/go-multihash"
)

const protocolID = "/sbicore/1.0.0"
const syncProtocolID = "/sbicore/sync/1.0.0"
const dhtNamespace = "/sbicore/peers"

var bootstrapPeers = []string{
	"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
	"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
	"/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
}

var dhtCID cid.Cid

func init() {
	mh, err := multihash.Sum([]byte(dhtNamespace), multihash.SHA2_256, -1)
	if err != nil {
		panic(err)
	}
	dhtCID = cid.NewCidV1(cid.Raw, mh)
}

type Channel struct {
	ID          string
	Name        string
	Subscribers map[peer.ID]bool
	mu          sync.RWMutex
}

func (ch *Channel) subscribe(p peer.ID) {
	ch.mu.Lock()
	ch.Subscribers[p] = true
	ch.mu.Unlock()
}

func (ch *Channel) unsubscribe(p peer.ID) {
	ch.mu.Lock()
	delete(ch.Subscribers, p)
	ch.mu.Unlock()
}

func (ch *Channel) hasSubscriber(p peer.ID) bool {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	return ch.Subscribers[p]
}

func (ch *Channel) subscriberCount() int {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	return len(ch.Subscribers)
}

type Node struct {
	host             host.Host
	dht              *dht.IpfsDHT
	ethHash          string
	preHash          string
	antiHash         string
	memory           Memory
	layers           [][]float64
	msgCount         int
	nodeID           int
	stateFile        string
	mu               sync.Mutex
	lastSyncSent     time.Time
	lastSyncedLayers [][]float64
	layersDirty      bool
	channels         map[string]*Channel
	channelsMu       sync.RWMutex
}

func (n *Node) getChannel(id string) *Channel {
	n.channelsMu.Lock()
	defer n.channelsMu.Unlock()
	if ch, ok := n.channels[id]; ok {
		return ch
	}
	ch := &Channel{
		ID:          id,
		Name:        id,
		Subscribers: make(map[peer.ID]bool),
	}
	n.channels[id] = ch
	return ch
}

func (n *Node) subscribeToChannel(channelID string) {
	ch := n.getChannel(channelID)
	ch.subscribe(n.host.ID())
	log.Printf("[CHANNEL] Subscribed to %s (total: %d)", channelID, ch.subscriberCount())
}

func (n *Node) unsubscribeFromChannel(channelID string) {
	n.channelsMu.RLock()
	ch, ok := n.channels[channelID]
	n.channelsMu.RUnlock()
	if !ok {
		return
	}
	ch.unsubscribe(n.host.ID())
	log.Printf("[CHANNEL] Unsubscribed from %s", channelID)
}

func (n *Node) getChannels() []map[string]interface{} {
	n.channelsMu.RLock()
	defer n.channelsMu.RUnlock()
	result := make([]map[string]interface{}, 0, len(n.channels))
	for _, ch := range n.channels {
		if ch.hasSubscriber(n.host.ID()) {
			result = append(result, map[string]interface{}{
				"id":         ch.ID,
				"name":       ch.Name,
				"subscribers": ch.subscriberCount(),
			})
		}
	}
	return result
}

func (n *Node) HandlePeerFound(peerInfo peer.AddrInfo) {
	ctx := context.Background()
	if err := n.host.Connect(ctx, peerInfo); err != nil {
		log.Println("Connection failed:", err)
		return
	}
	log.Println("Connected to peer:", peerInfo.ID)
	go func() {
		time.Sleep(3 * time.Second)
		n.broadcastLayers()
	}()
}

func (n *Node) handleStream(stream network.Stream) {
	defer stream.Close()
	buf := make([]byte, 65536)
	nr, err := stream.Read(buf)
	if err != nil {
		log.Println("Read error:", err)
		return
	}
	data := strings.TrimSpace(string(buf[:nr]))

	if strings.HasPrefix(data, "STATUS:") {
		parts := strings.SplitN(data, ":", 3)
		if len(parts) == 3 {
			msgID := parts[1]
			status := parts[2]
			n.memory.UpdateDeliveryStatus(msgID, status)
			log.Printf("[STATUS] Статус доставки для %s: %s", msgID[:8], status)
		}
		return
	}

	var channelID, msg, incomingMsgID string
	if strings.HasPrefix(data, "CHANNEL:") {
		rest := data[len("CHANNEL:"):]
		parts := strings.SplitN(rest, ":", 3)
		if len(parts) >= 3 {
			channelID = parts[0]
			incomingMsgID = parts[1]
			msg = parts[2]
		} else if len(parts) == 2 {
			channelID = parts[0]
			msg = parts[1]
		}
	} else if idx := strings.Index(data, ":"); idx > 0 && len(data) > idx+1 {
		incomingMsgID = data[:idx]
		msg = data[idx+1:]
	} else {
		msg = data
	}

	if msg == n.ethHash {
		stream.Write([]byte("ACCEPTED\n"))
		log.Println("Accepted peer:", stream.Conn().RemotePeer())
		return
	}

	senderID := stream.Conn().RemotePeer().String()[:8]
	log.Printf("📩 P2P сообщение от %s [канал:%s]: %s", senderID, channelID, msg)

	msgID, deliveryStatus := n.processMessage(msg, senderID, false, incomingMsgID, channelID)

	statusMsg := fmt.Sprintf("STATUS:%s:%s", msgID, deliveryStatus)
	stream.Write([]byte(statusMsg))
	log.Printf("[STATUS] Отправлен статус %s для %s отправителю %s", deliveryStatus, msgID[:8], senderID)
}

func (n *Node) processMessage(msg string, senderID string, isOwn bool, incomingMsgID string, channelID string) (string, string) {
	log.Printf("[MSG] Обработка сообщения: %s (от %s, канал:%s)", msg, senderID, channelID)

	id := incomingMsgID
	if id == "" {
		id = generateMsgID(msg)
	}

	inputVector := textToVector(msg)
	log.Printf("📊 Входной вектор: %v", inputVector[:5])

	outputVector, _ := forward(inputVector, n.layers)
	log.Printf("📤 Выходной вектор (первые 5): %v", outputVector[:5])

	answer := vectorToText(outputVector)
	if answer == "" {
		log.Printf("⚠️ Пустой ответ для сообщения: %s", msg)
	} else {
		log.Printf("💬 Сеть ответила: %s", answer)
	}

	n.mu.Lock()
	similar := n.memory.FindSimilar(msg, 0.7)
	var contextVector []float64
	if len(similar) > 0 {
		contextVector = make([]float64, VectorDim)
		totalWeight := 0.0
		for _, s := range similar {
			sv := textToVector(s.Text)
			w := s.Weight
			for i := range sv {
				contextVector[i] += sv[i] * w
			}
			totalWeight += w
		}
		for i := range contextVector {
			contextVector[i] /= totalWeight
		}
		for i := range inputVector {
			inputVector[i] = inputVector[i]*0.7 + contextVector[i]*0.3
		}
		log.Printf("[TRAIN] Найдено похожих сообщений: %d, контекст применён", len(similar))
	} else {
		log.Println("[TRAIN] Похожих сообщений не найдено, обучение без контекста")
	}
	n.layers = train(n.layers, inputVector, outputVector, inputVector, 0.01)
	n.mu.Unlock()
	n.layersDirty = true
	log.Println("[TRAIN] Обучение выполнено")

	if n.host != nil {
		go func() {
			n.broadcastLayers()
		}()
	}

	ethicsVec := hashToVector(n.ethHash)
	msgVec := textToVector(msg)
	ethicsScore := cosineSimilarity(ethicsVec, msgVec)
	initialWeight := 0.3 + ethicsScore*0.5
	log.Printf("⚖️ Этическая близость: %.2f, начальный вес: %.2f", ethicsScore, initialWeight)

	if n.preHash != "" {
		preVec := hashToVector(n.preHash)
		preScore := cosineSimilarity(preVec, msgVec)
		preWeight := 0.2 + preScore*0.6
		initialWeight = initialWeight*0.4 + preWeight*0.6
		log.Printf("🔧 Пре-хеш близость: %.2f, итоговый вес: %.2f", preScore, initialWeight)
	}

	if n.antiHash != "" {
		antiVec := hashToVector(n.antiHash)
		antiScore := cosineSimilarity(antiVec, msgVec)
		antiWeight := 0.3 + antiScore*0.5
		initialWeight = initialWeight * (1.0 - antiWeight*0.8)
		if initialWeight < 0.1 {
			initialWeight = 0.1
		}
		log.Printf("🚫 Анти-хеш близость: %.2f, итоговый вес: %.2f", antiScore, initialWeight)
	}

	deliveryStatus := "delivered"
	if initialWeight < 0.2 {
		deliveryStatus = "filtered"
	} else if initialWeight < 0.4 {
		deliveryStatus = "partial"
	}

	newMsg := Message{
		ID:             id,
		Text:           msg,
		Sender:         senderID,
		Time:           time.Now().Format("2006-01-02T15:04:05"),
		IsOwn:          isOwn,
		Score:          0,
		Weight:         initialWeight,
		DeliveryStatus: deliveryStatus,
		ChannelID:      channelID,
	}

	isNew := n.memory.Add(newMsg)
	if isNew {
		log.Printf("💾 Сообщение сохранено: %s [канал:%s]", msg, channelID)

		go broadcastToWS(map[string]interface{}{
			"type":    "message",
			"channel": channelID,
			"data":    newMsg,
		})

		n.mu.Lock()
		n.msgCount++
		if n.msgCount > 0 && n.msgCount%20 == 0 {
			newLayer := make([]float64, VectorDim)
			hashVec := hashToVector(n.ethHash)
			for i := range newLayer {
				newLayer[i] = (hashVec[i%len(hashVec)] * 0.1) + (float64(n.msgCount%10) * 0.01)
			}
			n.layers = append(n.layers, newLayer)
			n.layersDirty = true
			log.Printf("[TRAIN] Новый слой добавлен! Всего слоёв: %d", len(n.layers))
		}
		n.mu.Unlock()

		if n.host != nil {
			ch := n.getChannel(channelID)
			for _, p := range n.host.Network().Peers() {
				if p.String()[:8] == senderID {
					continue
				}
				if channelID != "" && !ch.hasSubscriber(p) {
					continue
				}
				go func(peerID peer.ID) {
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer cancel()

					s, err := n.host.NewStream(ctx, peerID, protocolID)
					if err != nil {
						return
					}
					defer s.Close()

					if channelID != "" {
						s.Write([]byte(fmt.Sprintf("CHANNEL:%s:%s:%s\n", channelID, id, msg)))
					} else {
						s.Write([]byte(id + ":" + msg + "\n"))
					}

					respBuf := make([]byte, 1024)
					nr, err := s.Read(respBuf)
					if err != nil {
						return
					}
					resp := strings.TrimSpace(string(respBuf[:nr]))
					if strings.HasPrefix(resp, "STATUS:") {
						parts := strings.SplitN(resp, ":", 3)
						if len(parts) == 3 {
							n.memory.UpdateDeliveryStatus(parts[1], parts[2])
							log.Printf("[STATUS] Статус доставки для %s: %s", parts[1][:8], parts[2])
						}
					}
				}(p)
			}
		}
	} else {
		log.Printf("⚠️ Сообщение-дубликат: %s", msg)
	}

	answerID := generateMsgID(answer)
	if answer != "" {
		answerMsg := Message{
			ID:        answerID,
			Text:      answer,
			Sender:    "🌐 Сеть",
			Time:      time.Now().Format("2006-01-02T15:04:05"),
			IsOwn:     false,
			Score:     0,
			Weight:    0.5,
			ChannelID: channelID,
		}
		if n.memory.Add(answerMsg) {
			log.Printf("💾 Ответ сети сохранён: %s", answer)
			go broadcastToWS(map[string]interface{}{
				"type":    "message",
				"channel": channelID,
				"data":    answerMsg,
			})
		}
	}

	archived := n.memory.ArchiveOld(0.15)
	if archived > 0 {
		log.Printf("[ARCHIVE] %d messages archived", archived)
	}

	if n.host != nil {
		go func() {
			if err := n.saveState(); err != nil {
				log.Printf("❌ Ошибка сохранения состояния: %v", err)
			}
		}()
	}

	log.Println("[MSG] Обработка сообщения завершена")
	return id, deliveryStatus
}

func migrateTime(t string) string {
	if strings.Contains(t, "T") {
		return t
	}
	if len(t) == 8 && t[2] == ':' {
		return time.Now().Format("2006-01-02") + "T" + t
	}
	return time.Now().Format("2006-01-02T15:04:05")
}

func (n *Node) loadState() error {
	data, err := os.ReadFile(n.stateFile)
	if err != nil {
		return err
	}
	var state struct {
		Messages []Message   `json:"messages"`
		Layers   [][]float64 `json:"layers"`
		MsgCount int         `json:"msgCount"`
		PreHash  string      `json:"preHash"`
		AntiHash string      `json:"antiHash"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}
	for i := range state.Messages {
		state.Messages[i].Time = migrateTime(state.Messages[i].Time)
	}
	for _, msg := range state.Messages {
		n.memory.Add(msg)
	}
	n.layers = state.Layers
	n.msgCount = state.MsgCount
	n.preHash = state.PreHash
	n.antiHash = state.AntiHash
	n.layersDirty = true

	archivePath := fmt.Sprintf("state/archive_node%d.json", n.nodeID)
	if err := n.memory.LoadArchive(archivePath); err != nil {
	}

	if n.preHash != "" {
		log.Printf("🔧 Загружен пре-хеш: %s", n.preHash)
	}
	if n.antiHash != "" {
		log.Printf("🚫 Загружен анти-хеш: %s", n.antiHash)
	}
	log.Printf("Загружено состояние: %d сообщений, %d слоёв", len(state.Messages), len(state.Layers))
	return nil
}

func (n *Node) announceToDHT() {
	if n.dht == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := n.dht.Provide(ctx, dhtCID, true); err != nil {
		log.Printf("[DHT] Announce failed: %v", err)
		return
	}
	log.Println("[DHT] Announced self as provider")
}

func (n *Node) findPeersLoop() {
	for {
		time.Sleep(60 * time.Second)
		if n.dht == nil {
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		providers, err := n.dht.FindProviders(ctx, dhtCID)
		cancel()
		if err != nil {
			continue
		}

		for _, p := range providers {
			if p.ID == n.host.ID() {
				continue
			}
			if n.host.Network().Connectedness(p.ID) == network.Connected {
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			err := n.host.Connect(ctx, p)
			cancel()
			if err != nil {
				continue
			}
			log.Printf("[DHT] Connected to peer: %s", p.ID.String()[:8])
			go func() {
				time.Sleep(2 * time.Second)
				n.broadcastLayers()
			}()
		}
	}
}

func (n *Node) start() {
	nodeIDStr := os.Getenv("NODE_ID")
	if nodeIDStr == "" {
		nodeIDStr = "1"
	}
	nodeID, err := strconv.Atoi(nodeIDStr)
	if err != nil {
		nodeID = 1
	}
	n.nodeID = nodeID
	n.stateFile = fmt.Sprintf("state/state_node%d.json", nodeID)
	log.Printf("Узел %d, файл состояния: %s", nodeID, n.stateFile)

	if err := n.loadState(); err != nil {
		log.Println("Состояние не найдено или повреждено, начинаем с нуля")
	} else {
		log.Println("Состояние успешно загружено")
	}

	priv, _, err := crypto.GenerateKeyPair(crypto.RSA, 2048)
	if err != nil {
		log.Fatal(err)
	}
	host, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/9000"),
		libp2p.Identity(priv),
	)
	if err != nil {
		log.Fatal(err)
	}
	n.host = host

	wsNode = n
	memoryForWS = &n.memory

	n.channels = make(map[string]*Channel)
	n.subscribeToChannel("general")

	n.host.SetStreamHandler(protocolID, n.handleStream)
	n.host.SetStreamHandler(syncProtocolID, n.handleSyncStream)

	kdht, err := dht.New(context.Background(), host)
	if err != nil {
		log.Printf("[DHT] Failed to create DHT: %v", err)
	} else {
		n.dht = kdht
		if err := n.dht.Bootstrap(context.Background()); err != nil {
			log.Printf("[DHT] Bootstrap failed: %v", err)
		} else {
			log.Println("[DHT] Bootstrap started")
		}
	}

	n.startSyncBackground()

	mdnsService := mdns.NewMdnsService(n.host, "sbicore", n)
	if err := mdnsService.Start(); err != nil {
		log.Fatal(err)
	}
	log.Println("Node started with ID:", host.ID())
	log.Println("Listening on:", host.Addrs())

	if n.dht != nil {
		go func() {
			time.Sleep(5 * time.Second)
			n.announceToDHT()
		}()
		go n.findPeersLoop()
	}

	if len(n.layers) == 0 {
		n.layers = append(n.layers, make([]float64, VectorDim))
		hashVec := hashToVector(n.ethHash)
		for i := range n.layers[0] {
			if i < len(hashVec) {
				n.layers[0][i] = hashVec[i] * 0.1
			}
		}
		log.Println("Initial layer created with love vector")
	}

	go func() {
		for {
			time.Sleep(1 * time.Hour)
			archivePath := fmt.Sprintf("state/archive_node%d.json", n.nodeID)
			if err := n.memory.SaveArchive(archivePath); err != nil {
				log.Printf("[MEMORY] Failed to save archive: %v", err)
			}
			removed := n.memory.PurgeDead(0.1)
			if removed > 0 {
				log.Printf("[PURGE] removed %d dead messages", removed)
				n.saveState()
			}
		}
	}()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})
	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/send", corsMiddleware(n.handleSend))
	http.HandleFunc("/messages", corsMiddleware(n.handleMessages))
	http.HandleFunc("/status", corsMiddleware(n.handleStatus))
	http.HandleFunc("/layers", corsMiddleware(n.handleLayersPage))
	http.HandleFunc("/api/layers", corsMiddleware(n.handleLayersAPI))
	http.HandleFunc("/feedback", corsMiddleware(n.handleFeedback))
	http.HandleFunc("/setprehash", corsMiddleware(n.handleSetPreHash))
	http.HandleFunc("/setantihash", corsMiddleware(n.handleSetAntiHash))
	http.HandleFunc("/gethashes", corsMiddleware(n.handleGetHashes))
	http.HandleFunc("/health", corsMiddleware(n.handleHealth))
	http.HandleFunc("/channels", corsMiddleware(n.handleChannels))

	go func() {
		log.Println("HTTP server listening on :8081")
		if err := http.ListenAndServe(":8081", nil); err != nil {
			log.Fatal(err)
		}
	}()
	select {}
}