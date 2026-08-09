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

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
)

const protocolID = "/sbicore/1.0.0"
const syncProtocolID = "/sbicore/sync/1.0.0"

type Node struct {
	host              host.Host
	ethHash           string
	preHash           string
	antiHash          string
	lastSyncSent      time.Time
	lastSyncedLayers  [][]float64
	layersDirty       bool
	memory            Memory
	layers            [][]float64
	msgCount          int
	nodeID            int
	stateFile         string
	mu                sync.Mutex
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
	buf := make([]byte, 1024)
	nr, err := stream.Read(buf)
	if err != nil {
		log.Println("Read error:", err)
		return
	}
	msg := strings.TrimSpace(string(buf[:nr]))
	if msg == n.ethHash {
		stream.Write([]byte("ACCEPTED\n"))
		log.Println("Accepted peer:", stream.Conn().RemotePeer())
		return
	}
	log.Printf("[MSG] P2P сообщение от %s: %s", stream.Conn().RemotePeer().String()[:8], msg)
	n.processMessage(msg, stream.Conn().RemotePeer().String()[:8], false)
}

func (n *Node) processMessage(msg string, senderID string, isOwn bool) {
	log.Printf("[MSG] Обработка сообщения: %s (от %s)", msg, senderID)

	inputVector := textToVector(msg)
	log.Printf("[MSG] Входной вектор (первые 5): %v", inputVector[:5])

	outputVector, _ := forward(inputVector, n.layers)
	log.Printf("[MSG] Выходной вектор (первые 5): %v", outputVector[:5])

	answer := vectorToText(outputVector)
	if answer == "" {
		log.Printf("[MSG] Пустой ответ для сообщения: %s", msg)
	} else {
		log.Printf("[MSG] Сеть ответила: %s", answer)
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
	n.layersDirty = true
	n.mu.Unlock()
	log.Println("[TRAIN] Обучение выполнено")

	go func() {
		n.broadcastLayers()
	}()

	// Этическая близость
	ethicsVec := hashToVector(n.ethHash)
	msgVec := textToVector(msg)
	ethicsScore := cosineSimilarity(ethicsVec, msgVec)
	initialWeight := 0.3 + ethicsScore*0.5
	log.Printf("[ETHICS] Этическая близость: %.2f, начальный вес: %.2f", ethicsScore, initialWeight)

	// Пре-хеш пользователя
	if n.preHash != "" {
		preVec := hashToVector(n.preHash)
		preScore := cosineSimilarity(preVec, msgVec)
		preWeight := 0.2 + preScore*0.6
		initialWeight = initialWeight*0.4 + preWeight*0.6
		log.Printf("[PREHASH] Пре-хеш близость: %.2f, итоговый вес: %.2f", preScore, initialWeight)
	}

	// Анти-хеш
	if n.antiHash != "" {
		antiVec := hashToVector(n.antiHash)
		antiScore := cosineSimilarity(antiVec, msgVec)
		antiWeight := 0.3 + antiScore*0.5
		initialWeight = initialWeight * (1.0 - antiWeight*0.8)
		if initialWeight < 0.1 {
			initialWeight = 0.1
		}
		log.Printf("[ANTIHASH] Анти-хеш близость: %.2f, итоговый вес: %.2f", antiScore, initialWeight)
	}

	// Приоритет сообщения (Priority Gossip)
	priority := 0
	if isOwn {
		nodeWeight := 0.5
		activeMsgs := n.memory.GetActiveMessages(0.5)
		if len(activeMsgs) > 0 {
			totalWeight := 0.0
			for _, m := range activeMsgs {
				totalWeight += m.Weight
			}
			nodeWeight = totalWeight / float64(len(activeMsgs))
		}
		if nodeWeight > 0.7 {
			priority = 100
		}
	}

	id := generateMsgID(msg)
	newMsg := Message{
		ID:       id,
		Text:     msg,
		Sender:   senderID,
		Time:     time.Now().Format("2006-01-02T15:04:05"),
		IsOwn:    isOwn,
		Score:    0,
		Weight:   initialWeight,
		Priority: priority,
	}
	if n.memory.Add(newMsg) {
		log.Printf("[MSG] Сообщение сохранено: %s (priority=%d)", msg, priority)
		n.mu.Lock()
		n.msgCount++
		if n.msgCount > 0 && n.msgCount%20 == 0 {
			newLayer := make([]float64, VectorDim)
			hashVec := hashToVector(n.ethHash)
			for i := range newLayer {
				if i < len(hashVec) {
					newLayer[i] = (hashVec[i] * 0.1) + (float64(n.msgCount%10) * 0.01)
				}
			}
			n.layers = append(n.layers, newLayer)
			log.Printf("[TRAIN] Новый слой добавлен! Всего слоёв: %d", len(n.layers))
		}
		n.mu.Unlock()
	} else {
		log.Printf("[MSG] Сообщение-дубликат: %s", msg)
	}

	answerID := generateMsgID(answer)
	if answer != "" {
		answerMsg := Message{
			ID:       answerID,
			Text:     answer,
			Sender:   "🌐 Сеть",
			Time:     time.Now().Format("2006-01-02T15:04:05"),
			IsOwn:    false,
			Score:    0,
			Weight:   0.5,
			Priority: 0,
		}
		if n.memory.Add(answerMsg) {
			log.Printf("[MSG] Ответ сети сохранён: %s", answer)
		}
	}

	archived := n.memory.ArchiveOld(0.15)
	if archived > 0 {
		log.Printf("[ARCHIVE] %d messages archived", archived)
	}

	go func() {
		if err := n.saveState(); err != nil {
			log.Printf("[ERROR] Ошибка сохранения состояния: %v", err)
		} else {
			log.Println("[STATE] Состояние сохранено")
		}
	}()

	// Форвардинг с приоритетом
	if !isOwn {
		ttl := 3
		if priority > 50 {
			ttl = 6
		}
		for _, p := range n.host.Network().Peers() {
			if p.String()[:8] == senderID {
				continue
			}
			go func(peerID peer.ID) {
				ctx := context.Background()
				s, err := n.host.NewStream(ctx, peerID, protocolID)
				if err != nil {
					return
				}
				defer s.Close()
				fmt.Fprintf(s, "%s\n", msg)
			}(p)
		}
		_ = ttl
	}
	log.Println("[MSG] Обработка сообщения завершена")
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
		Messages []Message  `json:"messages"`
		Layers   [][]float64 `json:"layers"`
		MsgCount int        `json:"msgCount"`
		PreHash  string     `json:"preHash"`
		AntiHash string     `json:"antiHash"`
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
	if n.preHash != "" {
		log.Printf("[PREHASH] Загружен пре-хеш: %s", n.preHash)
	}
	if n.antiHash != "" {
		log.Printf("[ANTIHASH] Загружен анти-хеш: %s", n.antiHash)
	}
	log.Printf("[STATE] Загружено состояние: %d сообщений, %d слоёв", len(state.Messages), len(state.Layers))
	return nil
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
	log.Printf("[INIT] Узел %d, файл состояния: %s", nodeID, n.stateFile)

	if err := n.loadState(); err != nil {
		log.Println("[INIT] Состояние не найдено или повреждено, начинаем с нуля")
	} else {
		log.Println("[INIT] Состояние успешно загружено")
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
	n.host.SetStreamHandler(protocolID, n.handleStream)
	n.host.SetStreamHandler(syncProtocolID, n.handleSyncStream)

	mdnsService := mdns.NewMdnsService(n.host, "sbicore", n)
	if err := mdnsService.Start(); err != nil {
		log.Fatal(err)
	}
	log.Println("[INIT] Node started with ID:", host.ID())
	log.Println("[INIT] Listening on:", host.Addrs())

	if len(n.layers) == 0 {
		n.layers = append(n.layers, make([]float64, VectorDim))
		hashVec := hashToVector(n.ethHash)
		for i := range n.layers[0] {
			if i < len(hashVec) {
				n.layers[0][i] = hashVec[i] * 0.1
			}
		}
		log.Println("[INIT] Initial layer created with love vector")
	}

	go func() {
		for {
			time.Sleep(1 * time.Hour)
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
	http.HandleFunc("/send", n.handleSend)
	http.HandleFunc("/messages", n.handleMessages)
	http.HandleFunc("/status", n.handleStatus)
	http.HandleFunc("/health", n.handleHealth)
	http.HandleFunc("/layers", n.handleLayersPage)
	http.HandleFunc("/api/layers", n.handleLayersAPI)
	http.HandleFunc("/feedback", n.handleFeedback)
	http.HandleFunc("/setprehash", n.handleSetPreHash)
	http.HandleFunc("/setantihash", n.handleSetAntiHash)
	http.HandleFunc("/gethashes", n.handleGetHashes)
	http.HandleFunc("/ws", n.handleWebSocket)

	go func() {
		log.Println("[INIT] HTTP server listening on :8081")
		if err := http.ListenAndServe(":8081", nil); err != nil {
			log.Fatal(err)
		}
	}()
	select {}
}