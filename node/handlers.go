package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/gorilla/websocket"
)

// ============================================================
// CORS MIDDLEWARE
// ============================================================

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

// ============================================================
// WEBSOCKET
// ============================================================

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type wsClient struct {
	conn    *websocket.Conn
	mu      sync.Mutex
	channel string
}

type wsHub struct {
	mu      sync.RWMutex
	clients map[*wsClient]bool
}

var hub = &wsHub{
	clients: make(map[*wsClient]bool),
}

func (h *wsHub) add(c *wsClient) {
	h.mu.Lock()
	h.clients[c] = true
	h.mu.Unlock()
}

func (h *wsHub) remove(c *wsClient) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
}

func (h *wsHub) broadcast(msg interface{}) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		go func(c *wsClient) {
			c.mu.Lock()
			defer c.mu.Unlock()
			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				h.remove(c)
			}
		}(client)
	}
}

func broadcastToWS(msg interface{}) {
	hub.broadcast(msg)
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS] Upgrade failed: %v", err)
		return
	}

	client := &wsClient{conn: conn, channel: "general"}
	hub.add(client)
	defer hub.remove(client)
	defer conn.Close()

	log.Printf("[WS] Client connected, total: %d", len(hub.clients))

	allMsgs := memoryForWS.GetAll()
	total := len(allMsgs)
	start := 0
	if total > 20 {
		start = total - 20
	}
	for i := start; i < total; i++ {
		msg := map[string]interface{}{
			"type":    "message",
			"channel": "general",
			"data":    allMsgs[i],
		}
		data, _ := json.Marshal(msg)
		client.mu.Lock()
		conn.WriteMessage(websocket.TextMessage, data)
		client.mu.Unlock()
	}

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("[WS] Client disconnected: %v", err)
			break
		}

		var req map[string]interface{}
		if err := json.Unmarshal(message, &req); err != nil {
			continue
		}

		msgType, _ := req["type"].(string)

		switch msgType {
		case "subscribe":
			channelID, _ := req["channel"].(string)
			if channelID != "" && wsNode != nil {
				wsNode.subscribeToChannel(channelID)
				client.channel = channelID

				msgs := wsNode.memory.GetAll()
				for _, m := range msgs {
					if m.ChannelID == channelID {
						msg := map[string]interface{}{
							"type":    "message",
							"channel": channelID,
							"data":    m,
						}
						data, _ := json.Marshal(msg)
						client.mu.Lock()
						conn.WriteMessage(websocket.TextMessage, data)
						client.mu.Unlock()
					}
				}

				subscribedMsg := map[string]interface{}{
					"type":    "subscribed",
					"channel": channelID,
				}
				data, _ := json.Marshal(subscribedMsg)
				client.mu.Lock()
				conn.WriteMessage(websocket.TextMessage, data)
				client.mu.Unlock()
			}

		case "send":
			text, _ := req["message"].(string)
			channelID, _ := req["channel"].(string)
			if channelID == "" {
				channelID = client.channel
			}
			if text != "" && wsNode != nil {
				msgID, deliveryStatus := wsNode.processMessage(text, wsNode.host.ID().String()[:8], true, "", channelID)
				wsNode.memory.UpdateDeliveryStatus(msgID, "sent")

				statusMsg := map[string]interface{}{
					"type":           "status",
					"msgID":          msgID,
					"channel":        channelID,
					"deliveryStatus": deliveryStatus,
				}
				data, _ := json.Marshal(statusMsg)
				client.mu.Lock()
				conn.WriteMessage(websocket.TextMessage, data)
				client.mu.Unlock()
			}

		case "feedback":
			id, _ := req["id"].(string)
			score, _ := req["score"].(float64)
			if id != "" && (score == 1 || score == -1) && wsNode != nil {
				wsNode.handleFeedbackJSON(id, int(score))
			}
		}
	}
}

var wsNode *Node
var memoryForWS *Memory

// ============================================================
// HTTP-ОБРАБОТЧИКИ
// ============================================================

func (n *Node) handleSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Message   string `json:"message"`
		ChannelID string `json:"channel"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Message == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "missing message field"})
		return
	}

	log.Printf("[MSG] HTTP запрос на /send: %s [канал:%s]", req.Message, req.ChannelID)
	msgID, deliveryStatus := n.processMessage(req.Message, n.host.ID().String()[:8], true, "", req.ChannelID)
	n.memory.UpdateDeliveryStatus(msgID, "sent")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":         "ok",
		"msgID":          msgID,
		"deliveryStatus": deliveryStatus,
	})
}

func (n *Node) handleMessages(w http.ResponseWriter, r *http.Request) {
	channelID := r.URL.Query().Get("channel")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 50
	offset := 0

	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 && v <= 200 {
			limit = v
		}
	}
	if offsetStr != "" {
		if v, err := strconv.Atoi(offsetStr); err == nil && v >= 0 {
			offset = v
		}
	}

	allMsgs := n.memory.GetAll()

	if channelID != "" {
		var filtered []Message
		for _, msg := range allMsgs {
			if msg.ChannelID == channelID {
				filtered = append(filtered, msg)
			}
		}
		allMsgs = filtered
	}
	total := len(allMsgs)

	start := offset
	end := offset + limit
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	page := allMsgs[start:end]

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"messages": page,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
		"channel":  channelID,
	})
}

func (n *Node) handleChannels(w http.ResponseWriter, r *http.Request) {
	channels := n.getChannels()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"channels": channels,
	})
}

func (n *Node) handleStatus(w http.ResponseWriter, r *http.Request) {
	n.mu.Lock()
	defer n.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":     n.host.ID().String(),
		"peers":  len(n.host.Network().Peers()),
		"memory": n.memory.Count(),
		"layers": len(n.layers),
	})
}

func (n *Node) handleLayersPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "layers.html")
}

func (n *Node) handleLayersAPI(w http.ResponseWriter, r *http.Request) {
	n.mu.Lock()
	defer n.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(n.layers)
}

func (n *Node) handleFeedbackJSON(id string, score int) {
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
		return
	}

	if targetMsg.Archived {
		if score == 1 {
			n.memory.RestoreFromArchive(id)
			for i := range n.memory.messages {
				if n.memory.messages[i].ID == id {
					targetMsg = &n.memory.messages[i]
					break
				}
			}
		} else {
			n.mu.Unlock()
			return
		}
	}

	targetMsg.Score = score
	if score == 1 {
		targetMsg.Weight += 0.15
		if targetMsg.Weight > 1.0 {
			targetMsg.Weight = 1.0
		}
	} else {
		targetMsg.Weight -= 0.15
		if targetMsg.Weight < 0.0 {
			targetMsg.Weight = 0.0
		}
	}

	inputVector := textToVector(targetMsg.Text)
	outputVector, _ := forward(inputVector, n.layers)
	ethicsVec := hashToVector(n.ethHash)

	if score == 1 {
		learningRate := 0.02 * targetMsg.Weight
		n.layers = train(n.layers, inputVector, outputVector, ethicsVec, learningRate)
	} else {
		learningRate := 0.02 * targetMsg.Weight
		invertedInput := make([]float64, len(inputVector))
		for i := range inputVector {
			invertedInput[i] = -inputVector[i]
		}
		n.layers = train(n.layers, inputVector, outputVector, invertedInput, learningRate)
	}

	if targetMsg.Weight < 0.15 {
		targetMsg.Archived = true
	}

	n.mu.Unlock()
	n.saveState()
}

func (n *Node) handleFeedback(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	scoreStr := r.URL.Query().Get("score")
	if id == "" || scoreStr == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "missing id or score"})
		return
	}

	score, err := strconv.Atoi(scoreStr)
	if err != nil || (score != 1 && score != -1) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "score must be 1 or -1"})
		return
	}

	n.handleFeedbackJSON(id, score)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (n *Node) handleSetPreHash(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "only POST allowed"})
		return
	}
	var req struct {
		PreHash string `json:"prehash"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
		return
	}
	n.mu.Lock()
	n.preHash = req.PreHash
	n.mu.Unlock()
	go n.saveState()
	log.Printf("🔧 Пре-хеш установлен: %s", req.PreHash)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "prehash": req.PreHash})
}

func (n *Node) handleSetAntiHash(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "only POST allowed"})
		return
	}
	var req struct {
		AntiHash string `json:"antihash"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
		return
	}
	n.mu.Lock()
	n.antiHash = req.AntiHash
	n.mu.Unlock()
	go n.saveState()
	log.Printf("🚫 Анти-хеш установлен: %s", req.AntiHash)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "antihash": req.AntiHash})
}

func (n *Node) handleGetHashes(w http.ResponseWriter, r *http.Request) {
	n.mu.Lock()
	defer n.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"prehash":  n.preHash,
		"antihash": n.antiHash,
	})
}

func (n *Node) handleHealth(w http.ResponseWriter, r *http.Request) {
	n.mu.Lock()
	defer n.mu.Unlock()

	var allWeights []float64
	for _, layer := range n.layers {
		allWeights = append(allWeights, layer...)
	}
	avg := 0.0
	if len(allWeights) > 0 {
		for _, w := range allWeights {
			avg += w
		}
		avg /= float64(len(allWeights))
	}
	variance := 0.0
	if len(allWeights) > 0 {
		for _, w := range allWeights {
			d := w - avg
			variance += d * d
		}
		variance /= float64(len(allWeights))
	}
	stdDev := sqrt(variance)

	peerList := make([]string, 0)
	for _, p := range n.host.Network().Peers() {
		peerList = append(peerList, p.String()[:16])
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"layers":    len(n.layers),
		"stddev":    stdDev,
		"peers":     peerList,
		"neurons":   len(allWeights),
		"avgWeight": avg,
	})
}