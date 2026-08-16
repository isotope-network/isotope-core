package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ============================================================
// HTTP-ОБРАБОТЧИКИ
// ============================================================

// handleSend — обрабатывает POST-запросы с JSON {"message": "текст", "priority": 100, "mode": 1, "ttl": 60}
func (n *Node) handleSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Message  string `json:"message"`
		Priority int    `json:"priority"`
		Mode     int    `json:"mode"`
		TTL      int    `json:"ttl"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Message == "" {
		http.Error(w, "missing message field", http.StatusBadRequest)
		return
	}

	if req.Mode < 0 || req.Mode > 2 {
		req.Mode = 0
	}

	var expiresAt time.Time
	if req.TTL > 0 {
		expiresAt = time.Now().Add(time.Duration(req.TTL) * time.Second)
	}

	log.Printf("[MSG] HTTP запрос на /send: %s (priority=%d, mode=%d, ttl=%d)", req.Message, req.Priority, req.Mode, req.TTL)

	if req.Mode > 0 {
		n.processMessageWithModeAndTTL(req.Message, n.host.ID().String()[:8], true, req.Mode, expiresAt)
	} else {
		n.processMessageWithTTL(req.Message, n.host.ID().String()[:8], true, expiresAt)
	}

	if n.host != nil {
		go func() {
			for _, p := range n.host.Network().Peers() {
				randomDelay(5, 25)
				obfuscated := n.obfuscate(req.Message)
				ctx := context.Background()
				s, err := n.host.NewStream(ctx, p, protocolID)
				if err != nil {
					continue
				}
				fmt.Fprintf(s, "%s\n", obfuscated)
				s.Close()
			}
		}()
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(fmt.Sprintf(`{"status":"ok","mode":%d,"ttl":%d}`, req.Mode, req.TTL)))
}

// handleSendStego — отправляет сообщение, спрятанное в WAV
func (n *Node) handleSendStego(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Message string `json:"message"`
		WAV     string `json:"wav"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Message == "" || req.WAV == "" {
		http.Error(w, "missing message or wav field", http.StatusBadRequest)
		return
	}

	wavBytes, err := base64ToWav(req.WAV)
	if err != nil {
		http.Error(w, "invalid wav base64", http.StatusBadRequest)
		return
	}

	if !isWAV(wavBytes) {
		http.Error(w, "invalid WAV format", http.StatusBadRequest)
		return
	}

	key := n.getObfuscationKey()
	encrypted := n.obfuscate(req.Message)
	encryptedBytes := []byte(encrypted)

	wavWithData, err := embedLSB(wavBytes, encryptedBytes, key)
	if err != nil {
		http.Error(w, fmt.Sprintf("embed failed: %v", err), http.StatusBadRequest)
		return
	}

	stegoMsg := "[STEGO]" + wavToBase64(wavWithData)

	n.processMessageWithTTL(req.Message, n.host.ID().String()[:8], true, time.Time{})

	if n.host != nil {
		go func() {
			for _, p := range n.host.Network().Peers() {
				randomDelay(10, 50)
				ctx := context.Background()
				s, err := n.host.NewStream(ctx, p, protocolID)
				if err != nil {
					continue
				}
				fmt.Fprintf(s, "%s\n", stegoMsg)
				s.Close()
			}
		}()
	}

	log.Printf("[STEGO] Сообщение спрятано в WAV и отправлено: %s", req.Message)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok","mode":3}`))
}

// handleCreateChannel — создаёт канал
func (n *Node) handleCreateChannel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		http.Error(w, "missing name field", http.StatusBadRequest)
		return
	}

	id := generateMsgID(req.Name)
	owner := n.host.ID().String()[:8]
	ch := n.channels.Create(id, req.Name, owner)

	log.Printf("[CHANNEL] Создан канал %s: %s", id[:8], req.Name)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(fmt.Sprintf(`{"id":"%s","name":"%s","owner":"%s"}`, id, req.Name, owner)))
	_ = ch
}

// handleGetChannels — список каналов
func (n *Node) handleGetChannels(w http.ResponseWriter, r *http.Request) {
	channels := n.channels.GetAll()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(channels)
}

// handleSendToChannel — отправить сообщение в канал
func (n *Node) handleSendToChannel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	// ID канала из URL: /channels/{id}/messages
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	channelID := parts[2]

	ch := n.channels.Get(channelID)
	if ch == nil {
		http.Error(w, "channel not found", http.StatusNotFound)
		return
	}

	senderWeight := n.getPeerWeight(n.host.ID().String()[:8])

	var req struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Message == "" {
		http.Error(w, "missing message field", http.StatusBadRequest)
		return
	}

	msg := Message{
		ID:      generateMsgID(req.Message),
		Text:    req.Message,
		Sender:  n.host.ID().String()[:8],
		Time:    time.Now().Format("2006-01-02T15:04:05"),
		IsOwn:   true,
		Score:   0,
		Weight:  senderWeight,
		Created: time.Now(),
	}

	ch.AddMessage(msg)
	log.Printf("[CHANNEL] Сообщение в канал %s: %s", channelID[:8], req.Message)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

// handleGetChannelMessages — получить сообщения канала с учётом веса
func (n *Node) handleGetChannelMessages(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	channelID := parts[2]

	ch := n.channels.Get(channelID)
	if ch == nil {
		http.Error(w, "channel not found", http.StatusNotFound)
		return
	}

	weight := n.getPeerWeight(n.host.ID().String()[:8])
	messages := ch.GetVisibleMessages(weight)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

// handleMessages — возвращает все сообщения в формате JSON (новые сверху)
func (n *Node) handleMessages(w http.ResponseWriter, r *http.Request) {
	msgs := n.memory.GetAll()
	reversed := make([]Message, len(msgs))
	for i := range msgs {
		reversed[len(msgs)-1-i] = msgs[i]
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reversed)
}

// handleStatus — возвращает статус узла
func (n *Node) handleStatus(w http.ResponseWriter, r *http.Request) {
	n.mu.Lock()
	defer n.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(fmt.Sprintf(`{"id":"%s","peers":%d,"memory":%d,"layers":%d}`,
		n.host.ID().String(),
		len(n.host.Network().Peers()),
		n.memory.Count(),
		len(n.layers),
	)))
}

// handleHealth — возвращает метрики здоровья узла
func (n *Node) handleHealth(w http.ResponseWriter, r *http.Request) {
	n.mu.Lock()
	defer n.mu.Unlock()

	allWeights := []float64{}
	for _, layer := range n.layers {
		allWeights = append(allWeights, layer...)
	}

	totalNeurons := len(allWeights)
	avgWeight := 0.0
	if totalNeurons > 0 {
		sum := 0.0
		for _, w := range allWeights {
			sum += w
		}
		avgWeight = sum / float64(totalNeurons)
	}

	stddev := 0.0
	if totalNeurons > 0 {
		variance := 0.0
		for _, w := range allWeights {
			variance += (w - avgWeight) * (w - avgWeight)
		}
		variance /= float64(totalNeurons)
		stddev = sqrt(variance)
	}

	peers := []string{}
	for _, p := range n.host.Network().Peers() {
		peers = append(peers, p.String()[:8])
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write([]byte(fmt.Sprintf(
		`{"layers":%d,"neurons":%d,"avgWeight":%f,"stddev":%f,"peers":%v}`,
		len(n.layers), totalNeurons, avgWeight, stddev, peers,
	)))
}

// handleLayersPage — отдаёт HTML-страницу для просмотра слоёв
func (n *Node) handleLayersPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "layers.html")
}

// handleLayersAPI — возвращает слои в формате JSON
func (n *Node) handleLayersAPI(w http.ResponseWriter, r *http.Request) {
	n.mu.Lock()
	defer n.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(n.layers)
}

// ============================================================
// ОБРАТНАЯ СВЯЗЬ + ОБУЧЕНИЕ
// ============================================================

// handleFeedback — обрабатывает запросы /feedback?id=...&score=1
func (n *Node) handleFeedback(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	scoreStr := r.URL.Query().Get("score")
	if id == "" || scoreStr == "" {
		http.Error(w, "missing id or score", http.StatusBadRequest)
		return
	}

	score, err := strconv.Atoi(scoreStr)
	if err != nil || (score != 1 && score != -1) {
		http.Error(w, "score must be 1 or -1", http.StatusBadRequest)
		return
	}

	n.mu.Lock()

	var targetMsg *Message
	for i := range n.memory.messages {
		if n.memory.messages[i].ID == id {
			targetMsg = &n.memory.messages[i]
			break
		}
	}
	if targetMsg == nil {
		n.mu.Unlock()
		http.Error(w, "message not found", http.StatusNotFound)
		return
	}

	if targetMsg.Archived {
		if score == 1 {
			n.memory.RestoreFromArchive(id)
			log.Printf("[FEEDBACK] Сообщение %s восстановлено из архива!", id[:8])
			for i := range n.memory.messages {
				if n.memory.messages[i].ID == id {
					targetMsg = &n.memory.messages[i]
					break
				}
			}
		} else {
			n.mu.Unlock()
			log.Printf("[FEEDBACK] Сообщение %s в архиве, дизлайк проигнорирован", id[:8])
			w.Write([]byte("OK\n"))
			return
		}
	}

	targetMsg.Score = score
	if score == 1 {
		targetMsg.Weight += 0.15
		if targetMsg.Weight > 1.0 {
			targetMsg.Weight = 1.0
		}
		log.Printf("[FEEDBACK] 👍 Лайк: вес сообщения %s увеличен до %.2f", id[:8], targetMsg.Weight)
	} else {
		targetMsg.Weight -= 0.15
		if targetMsg.Weight < 0.0 {
			targetMsg.Weight = 0.0
		}
		log.Printf("[FEEDBACK] 👎 Дизлайк: вес сообщения %s уменьшен до %.2f", id[:8], targetMsg.Weight)
	}

	inputVector := textToVector(targetMsg.Text)
	outputVector, _ := forward(inputVector, n.layers)
	ethicsVec := hashToVector(n.ethHash)

	if score == 1 {
		learningRate := 0.02 * targetMsg.Weight
		n.layers = train(n.layers, inputVector, outputVector, ethicsVec, learningRate)
		log.Printf("[FEEDBACK] [TRAIN] Этическое обучение (лайк): lr=%.4f, weight=%.2f", learningRate, targetMsg.Weight)
	} else {
		learningRate := 0.02 * targetMsg.Weight
		invertedInput := make([]float64, len(inputVector))
		for i := range inputVector {
			invertedInput[i] = -inputVector[i]
		}
		n.layers = train(n.layers, inputVector, outputVector, invertedInput, learningRate)
		log.Printf("[FEEDBACK] [TRAIN] Этическое обучение (дизлайк): lr=%.4f, weight=%.2f", learningRate, targetMsg.Weight)
	}

	if targetMsg.Weight < 0.15 {
		targetMsg.Archived = true
		log.Printf("[FEEDBACK] Сообщение %s отправлено в архив (вес %.2f)", id[:8], targetMsg.Weight)
	}

	n.mu.Unlock()

	if err := n.saveState(); err != nil {
		log.Printf("[ERROR] Ошибка сохранения состояния: %v", err)
	}

	w.Write([]byte("OK\n"))
}

// handleSetPreHash — устанавливает пользовательский пре-хеш
func (n *Node) handleSetPreHash(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "only POST allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		PreHash string `json:"prehash"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	n.mu.Lock()
	n.preHash = req.PreHash
	n.mu.Unlock()
	go n.saveState()
	log.Printf("[PREHASH] Пре-хеш установлен: %s", req.PreHash)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(fmt.Sprintf(`{"prehash":"%s"}`, req.PreHash)))
}

// handleSetAntiHash — устанавливает пользовательский анти-хеш
func (n *Node) handleSetAntiHash(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "only POST allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		AntiHash string `json:"antihash"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	n.mu.Lock()
	n.antiHash = req.AntiHash
	n.mu.Unlock()
	go n.saveState()
	log.Printf("[ANTIHASH] Анти-хеш установлен: %s", req.AntiHash)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(fmt.Sprintf(`{"antihash":"%s"}`, req.AntiHash)))
}

// handleGetHashes — возвращает текущие preHash и antiHash
func (n *Node) handleGetHashes(w http.ResponseWriter, r *http.Request) {
	n.mu.Lock()
	defer n.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(fmt.Sprintf(`{"prehash":"%s","antihash":"%s"}`, n.preHash, n.antiHash)))
}

// ============================================================
// WEBSOCKET
// ============================================================

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var wsClients = make(map[*websocket.Conn]bool)
var wsMu sync.Mutex

// handleWebSocket — обрабатывает WebSocket-соединения
func (n *Node) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("[WS] Upgrade error:", err)
		return
	}
	defer conn.Close()

	wsMu.Lock()
	wsClients[conn] = true
	wsMu.Unlock()
	defer func() {
		wsMu.Lock()
		delete(wsClients, conn)
		wsMu.Unlock()
	}()

	log.Println("[WS] Client connected")

	msgs := n.memory.GetAll()
	start := 0
	if len(msgs) > 20 {
		start = len(msgs) - 20
	}
	for _, msg := range msgs[start:] {
		data, _ := json.Marshal(msg)
		conn.WriteMessage(websocket.TextMessage, data)
	}

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("[WS] Client disconnected:", err)
			break
		}

		var req struct {
			Type     string `json:"type"`
			Message  string `json:"message"`
			Priority int    `json:"priority"`
			Mode     int    `json:"mode"`
			TTL      int    `json:"ttl"`
		}
		if err := json.Unmarshal(message, &req); err != nil {
			continue
		}

		if req.Type == "send" && req.Message != "" {
			if req.Mode < 0 || req.Mode > 2 {
				req.Mode = 0
			}
			var expiresAt time.Time
			if req.TTL > 0 {
				expiresAt = time.Now().Add(time.Duration(req.TTL) * time.Second)
			}
			if req.Mode > 0 {
				n.processMessageWithModeAndTTL(req.Message, n.host.ID().String()[:8], true, req.Mode, expiresAt)
			} else {
				n.processMessageWithTTL(req.Message, n.host.ID().String()[:8], true, expiresAt)
			}

			if n.host != nil {
				go func() {
					for _, p := range n.host.Network().Peers() {
						randomDelay(5, 25)
						obfuscated := n.obfuscate(req.Message)
						ctx := context.Background()
						s, err := n.host.NewStream(ctx, p, protocolID)
						if err != nil {
							continue
						}
						fmt.Fprintf(s, "%s\n", obfuscated)
						s.Close()
					}
				}()
			}

			conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf(`{"type":"status","status":"sent","mode":%d,"ttl":%d}`, req.Mode, req.TTL)))
		}
	}
}

// broadcastToWS — рассылает сообщение всем WebSocket-клиентам
func (n *Node) broadcastToWS(msg Message) {
	wsMu.Lock()
	defer wsMu.Unlock()

	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	for conn := range wsClients {
		go func(c *websocket.Conn) {
			c.WriteMessage(websocket.TextMessage, data)
		}(conn)
	}
}