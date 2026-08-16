package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
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
	libp2ptls "github.com/libp2p/go-libp2p/p2p/security/tls"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/libp2p/go-libp2p/p2p/transport/websocket"
)

const protocolID = "/sbicore/1.0.0"
const syncProtocolID = "/sbicore/sync/1.0.0"
const pingProtocolID = "/sbicore/ping/1.0.0"

const OBFUSCATION_PREFIX = "[SHUF]"
const STEGO_PREFIX = "[STEGO]"
const REPLICA_PREFIX = "[REPLICA]"
const RESTORE_PREFIX = "[RESTORE]"

type Node struct {
	host             host.Host
	ethHash          string
	preHash          string
	antiHash         string
	lastSyncSent     time.Time
	lastSyncedLayers [][]float64
	layersDirty      bool
	memory           Memory
	assoc            AssocMemory
	layers           [][]float64
	msgCount         int
	nodeID           int
	stateFile        string
	mu               sync.Mutex
	lastPing         map[string]time.Time
	deadPeers        map[string]bool
	adaptive         *AdaptiveParams
	channels         *ChannelStore
}

func (n *Node) getObfuscationKey() []byte {
	hash := sha256.Sum256([]byte(n.ethHash))
	return hash[:]
}

func (n *Node) obfuscate(msg string) string {
	key := n.getObfuscationKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return msg
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return msg
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return msg
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(msg), nil)
	return OBFUSCATION_PREFIX + base64.StdEncoding.EncodeToString(ciphertext)
}

func (n *Node) deobfuscate(data string) (string, bool) {
	if !strings.HasPrefix(data, OBFUSCATION_PREFIX) {
		return data, false
	}

	encoded := strings.TrimPrefix(data, OBFUSCATION_PREFIX)
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return data, false
	}

	key := n.getObfuscationKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return data, false
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return data, false
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return data, false
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return data, false
	}

	return string(plaintext), true
}

func randomDelay(minMs, maxMs int) {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(maxMs-minMs)))
	delay := time.Duration(minMs+int(n.Int64())) * time.Millisecond
	time.Sleep(delay)
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
	buf := make([]byte, 2*1024*1024)
	nr, err := stream.Read(buf)
	if err != nil {
		log.Println("Read error:", err)
		return
	}
	msg := strings.TrimSpace(string(buf[:nr]))

	if strings.HasPrefix(msg, REPLICA_PREFIX) {
		payload := strings.TrimPrefix(msg, REPLICA_PREFIX)
		var replicaMsg Message
		if err := json.Unmarshal([]byte(payload), &replicaMsg); err == nil {
			replicaMsg.ReplicatedAt = time.Now()
			if n.memory.Add(replicaMsg) {
				log.Printf("[REPLICA] Сохранена реплика от %s: %s", replicaMsg.Sender[:8], replicaMsg.Text)
			}
		}
		return
	}

	if strings.HasPrefix(msg, RESTORE_PREFIX) {
		nodeID := strings.TrimPrefix(msg, RESTORE_PREFIX)
		replicas := n.memory.GetReplicasFor(nodeID)
		if len(replicas) > 0 {
			for _, r := range replicas {
				data, _ := json.Marshal(r)
				stream.Write([]byte(REPLICA_PREFIX + string(data) + "\n"))
			}
			log.Printf("[RESTORE] Отправлено %d реплик для узла %s", len(replicas), nodeID[:8])
		}
		return
	}

	if strings.HasPrefix(msg, STEGO_PREFIX) {
		stegoB64 := strings.TrimPrefix(msg, STEGO_PREFIX)
		wavBytes, err := base64ToWav(stegoB64)
		if err == nil && isWAV(wavBytes) {
			key := n.getObfuscationKey()
			extracted, err := extractLSB(wavBytes, key)
			if err == nil {
				if plaintext, ok := n.deobfuscate(string(extracted)); ok {
					log.Printf("[STEGO] Извлечено скрытое сообщение: %s", plaintext)
					n.processMessage(plaintext, stream.Conn().RemotePeer().String()[:8], false)
					return
				}
			}
		}
		log.Println("[STEGO] Не удалось извлечь сообщение")
		return
	}

	if plaintext, ok := n.deobfuscate(msg); ok {
		msg = plaintext
		log.Printf("[OBF] Деобфусцировано сообщение от %s", stream.Conn().RemotePeer().String()[:8])
	}

	if msg == n.ethHash {
		stream.Write([]byte("ACCEPTED\n"))
		log.Println("Accepted peer:", stream.Conn().RemotePeer())
		return
	}

	remoteID := stream.Conn().RemotePeer().String()[:8]

	if strings.HasPrefix(msg, "CHAIN:") {
		parts := strings.SplitN(msg, ":", 3)
		if len(parts) == 3 {
			nextRelays := parts[1]
			actualMsg := parts[2]
			log.Printf("[ONION] Relay-узел: следующий в цепочке: %s, сообщение: %s", nextRelays, actualMsg)

			if nextRelays != "" {
				relays := strings.Split(nextRelays, ",")
				n.sendViaRelayChain(relays, actualMsg)
			} else {
				n.processMessageRelayed(actualMsg, remoteID)
			}
			return
		}
	}

	log.Printf("[MSG] P2P сообщение от %s: %s", remoteID, msg)
	n.processMessage(msg, remoteID, false)
}

func (n *Node) handleReplicaData(data string) {
	for _, line := range strings.Split(data, "\n") {
		if strings.HasPrefix(line, REPLICA_PREFIX) {
			payload := strings.TrimPrefix(line, REPLICA_PREFIX)
			var replicaMsg Message
			if err := json.Unmarshal([]byte(payload), &replicaMsg); err == nil {
				replicaMsg.ReplicatedAt = time.Now()
				if n.memory.Add(replicaMsg) {
					log.Printf("[REPLICA] Сохранена реплика от %s: %s", replicaMsg.Sender[:8], replicaMsg.Text)
				}
			}
		}
	}
}

func (n *Node) handlePingStream(stream network.Stream) {
	defer stream.Close()

	buf := make([]byte, 64)
	nr, err := stream.Read(buf)
	if err != nil {
		return
	}

	msg := strings.TrimSpace(string(buf[:nr]))
	if msg == "PING" {
		stream.Write([]byte("PONG\n"))
	}
}

func (n *Node) pingPeers() {
	go func() {
		for {
			time.Sleep(30 * time.Second)

			if n.host == nil {
				continue
			}

			peers := n.host.Network().Peers()
			for _, p := range peers {
				go func(peerID peer.ID) {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()

					s, err := n.host.NewStream(ctx, peerID, pingProtocolID)
					if err != nil {
						n.markPeerDead(peerID.String())
						return
					}
					defer s.Close()

					if _, err := s.Write([]byte("PING\n")); err != nil {
						n.markPeerDead(peerID.String())
						return
					}

					buf := make([]byte, 64)
					s.SetReadDeadline(time.Now().Add(5 * time.Second))
					nr, err := s.Read(buf)
					if err != nil || strings.TrimSpace(string(buf[:nr])) != "PONG" {
						n.markPeerDead(peerID.String())
						return
					}

					n.markPeerAlive(peerID.String())
				}(p)
			}
		}
	}()
}

func (n *Node) markPeerAlive(peerID string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.lastPing == nil {
		n.lastPing = make(map[string]time.Time)
	}
	if n.deadPeers == nil {
		n.deadPeers = make(map[string]bool)
	}

	n.lastPing[peerID] = time.Now()
	if n.deadPeers[peerID] {
		delete(n.deadPeers, peerID)
		log.Printf("[HEAL] Пир %s вернулся в сеть", peerID[:8])
	}
}

func (n *Node) markPeerDead(peerID string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.deadPeers == nil {
		n.deadPeers = make(map[string]bool)
	}

	if !n.deadPeers[peerID] {
		n.deadPeers[peerID] = true
		log.Printf("[HEAL] Пир %s помечен как мёртвый", peerID[:8])
	}
}

func (n *Node) isPeerDead(peerID string) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.deadPeers[peerID]
}

func (n *Node) getPeerWeight(peerID string) float64 {
	msgs := n.memory.GetMessagesFrom(peerID)
	if len(msgs) == 0 {
		return 0.5
	}
	total := 0.0
	for _, m := range msgs {
		total += m.Weight
	}
	return total / float64(len(msgs))
}

func (n *Node) processMessageRelayed(msg string, senderID string) {
	id := generateMsgID(msg)
	newMsg := Message{
		ID:       id,
		Text:     msg,
		Sender:   senderID,
		Time:     time.Now().Format("2006-01-02T15:04:05"),
		IsOwn:    false,
		Score:    0,
		Weight:   0.5,
		Priority: 0,
		Mode:     1,
		Relayed:  true,
	}
	if n.memory.Add(newMsg) {
		log.Printf("[ONION] Сообщение доставлено через relay и сохранено: %s", msg)
		n.replicateMessage(newMsg)
	}

	go func() {
		if err := n.saveState(); err != nil {
			log.Printf("[ERROR] Ошибка сохранения состояния: %v", err)
		}
	}()
}

func (n *Node) replicateMessage(msg Message) {
	if n.host == nil {
		return
	}

	msg.ReplicatedFrom = msg.Sender
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	peers := n.host.Network().Peers()
	var alive []peer.ID
	for _, p := range peers {
		if !n.isPeerDead(p.String()) {
			alive = append(alive, p)
		}
	}

	if len(alive) == 0 {
		return
	}

	perm := make([]int, len(alive))
	for i := range perm {
		perm[i] = i
	}
	for i := len(perm) - 1; i > 0; i-- {
		j, _ := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		perm[i], perm[int(j.Int64())] = perm[int(j.Int64())], perm[i]
	}

	replicaCount := 2
	if len(alive) < replicaCount {
		replicaCount = len(alive)
	}

	for i := 0; i < replicaCount; i++ {
		go func(peerID peer.ID) {
			randomDelay(10, 30)
			ctx := context.Background()
			s, err := n.host.NewStream(ctx, peerID, protocolID)
			if err != nil {
				return
			}
			defer s.Close()
			fmt.Fprintf(s, "%s%s\n", REPLICA_PREFIX, string(data))
		}(alive[perm[i]])
	}

	log.Printf("[REPLICA] Сообщение реплицировано на %d узлов", replicaCount)
}

func (n *Node) requestRestore() {
	if n.host == nil {
		return
	}

	myID := n.host.ID().String()[:8]
	peers := n.host.Network().Peers()

	for _, p := range peers {
		go func(peerID peer.ID) {
			randomDelay(100, 500)
			ctx := context.Background()
			s, err := n.host.NewStream(ctx, peerID, protocolID)
			if err != nil {
				return
			}
			defer s.Close()

			fmt.Fprintf(s, "%s%s\n", RESTORE_PREFIX, myID)

			buf := make([]byte, 2*1024*1024)
			s.SetReadDeadline(time.Now().Add(5 * time.Second))
			nr, _ := s.Read(buf)
			if nr > 0 {
				response := strings.TrimSpace(string(buf[:nr]))
				n.handleReplicaData(response)
			}
		}(p)
	}
	log.Printf("[RESTORE] Запрошены реплики у %d соседей", len(peers))
}

func (n *Node) processMessageWithTTL(msg string, senderID string, isOwn bool, expiresAt time.Time) {
	n.processMessageInternal(msg, senderID, isOwn, expiresAt)
}

func (n *Node) processMessageWithModeAndTTL(msg string, senderID string, isOwn bool, mode int, expiresAt time.Time) {
	if mode == 0 || n.host == nil {
		n.processMessageInternal(msg, senderID, isOwn, expiresAt)
		return
	}

	relayCount := 4
	if mode == 2 {
		relayCount = 5
		delay := 10 + time.Duration(time.Now().UnixNano()%50)*time.Second
		log.Printf("[ONION] Скрытый режим: задержка %v", delay)
		time.Sleep(delay)
	}

	relays := n.selectRelays(relayCount)
	if len(relays) < relayCount {
		log.Printf("[ONION] Недостаточно доверенных пиров: нужно %d, есть %d. Отправляю напрямую.", relayCount, len(relays))
		n.processMessageInternal(msg, senderID, isOwn, expiresAt)
		return
	}

	log.Printf("[ONION] Анонимная цепочка из %d relay: %s → ... → получатель", len(relays), senderID)
	n.sendViaRelayChain(relays, msg)
}

func (n *Node) processMessageWithMode(msg string, senderID string, isOwn bool, mode int) {
	n.processMessageWithModeAndTTL(msg, senderID, isOwn, mode, time.Time{})
}

func (n *Node) selectRelays(count int) []string {
	peers := n.host.Network().Peers()

	var trusted []peer.ID
	var fallback []peer.ID
	for _, p := range peers {
		if n.isPeerDead(p.String()) {
			continue
		}
		weight := n.getPeerWeight(p.String())
		if weight > 0.7 {
			trusted = append(trusted, p)
		} else {
			fallback = append(fallback, p)
		}
	}

	if len(trusted) >= count {
		perm := make([]int, len(trusted))
		for i := range perm {
			perm[i] = i
		}
		for i := len(perm) - 1; i > 0; i-- {
			j, _ := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
			perm[i], perm[int(j.Int64())] = perm[int(j.Int64())], perm[i]
		}
		result := make([]string, count)
		for i := 0; i < count; i++ {
			result[i] = trusted[perm[i]].String()
		}
		return result
	}

	selected := make([]string, 0)
	for _, p := range trusted {
		selected = append(selected, p.String())
	}
	perm := make([]int, len(fallback))
	for i := range perm {
		perm[i] = i
	}
	for i := len(perm) - 1; i > 0; i-- {
		j, _ := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		perm[i], perm[int(j.Int64())] = perm[int(j.Int64())], perm[i]
	}
	for i := 0; len(selected) < count && i < len(fallback); i++ {
		selected = append(selected, fallback[perm[i]].String())
	}
	return selected
}

func (n *Node) sendViaRelayChain(relays []string, msg string) {
	if len(relays) == 0 {
		return
	}

	obfuscated := n.obfuscate(msg)
	chainMsg := fmt.Sprintf("CHAIN:%s:%s", strings.Join(relays[1:], ","), obfuscated)

	relayPeer, err := peer.Decode(relays[0])
	if err != nil {
		log.Printf("[ONION] Invalid relay peer ID %s: %v", relays[0][:8], err)
		return
	}

	randomDelay(10, 50)

	ctx := context.Background()
	s, err := n.host.NewStream(ctx, relayPeer, protocolID)
	if err != nil {
		log.Printf("[ONION] Failed to create stream to first relay %s: %v", relays[0][:8], err)
		return
	}
	defer s.Close()

	if _, err := s.Write([]byte(chainMsg + "\n")); err != nil {
		log.Printf("[ONION] Failed to write to first relay %s: %v", relays[0][:8], err)
		return
	}
	log.Printf("[ONION] Сообщение отправлено через цепочку из %d relay (обфусцировано)", len(relays))
}

func (n *Node) processMessage(msg string, senderID string, isOwn bool) {
	n.processMessageInternal(msg, senderID, isOwn, time.Time{})
}

func (n *Node) processMessageInternal(msg string, senderID string, isOwn bool, expiresAt time.Time) {
	log.Printf("[MSG] Обработка сообщения: %s (от %s)", msg, senderID)

	if !isOwn && n.host != nil {
		n.assoc.AddAssociation(senderID, n.host.ID().String()[:8], msg)
	}

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

	lr := 0.01
	if n.adaptive != nil {
		lr = n.adaptive.LearningRate
	}
	n.layers = train(n.layers, inputVector, outputVector, inputVector, lr)
	n.layersDirty = true
	n.mu.Unlock()
	log.Println("[TRAIN] Обучение выполнено")

	go func() {
		n.broadcastLayers()
	}()

	ethicsVec := hashToVector(n.ethHash)
	msgVec := textToVector(msg)
	ethicsScore := cosineSimilarity(ethicsVec, msgVec)
	initialWeight := 0.3 + ethicsScore*0.5
	log.Printf("[ETHICS] Этическая близость: %.2f, начальный вес: %.2f", ethicsScore, initialWeight)

	if n.preHash != "" {
		preVec := hashToVector(n.preHash)
		preScore := cosineSimilarity(preVec, msgVec)
		preWeight := 0.2 + preScore*0.6
		initialWeight = initialWeight*0.4 + preWeight*0.6
		log.Printf("[PREHASH] Пре-хеш близость: %.2f, итоговый вес: %.2f", preScore, initialWeight)
	}

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
		ID:        id,
		Text:      msg,
		Sender:    senderID,
		Time:      time.Now().Format("2006-01-02T15:04:05"),
		IsOwn:     isOwn,
		Score:     0,
		Weight:    initialWeight,
		Priority:  priority,
		Mode:      0,
		ExpiresAt: expiresAt,
	}
	if n.memory.Add(newMsg) {
		log.Printf("[MSG] Сообщение сохранено: %s (priority=%d)", msg, priority)
		if !expiresAt.IsZero() {
			log.Printf("[MSG] Сообщение исчезнет через %v", time.Until(expiresAt))
		}
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

		n.replicateMessage(newMsg)
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
			Mode:     0,
		}
		if n.memory.Add(answerMsg) {
			log.Printf("[MSG] Ответ сети сохранён: %s", answer)
			n.replicateMessage(answerMsg)
		}
	}

	threshold := 0.15
	if n.adaptive != nil {
		threshold = n.adaptive.ArchiveThreshold
	}
	archived := n.memory.ArchiveOld(threshold)
	if archived > 0 {
		log.Printf("[ARCHIVE] %d messages archived (threshold=%.2f)", archived, threshold)
	}

	go func() {
		if err := n.saveState(); err != nil {
			log.Printf("[ERROR] Ошибка сохранения состояния: %v", err)
		} else {
			log.Println("[STATE] Состояние сохранено")
		}
	}()

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

func (n *Node) loadBootstrapPeers() []string {
	var peers []string

	if envPeers := os.Getenv("ISOTOPE_BOOTSTRAP_PEERS"); envPeers != "" {
		for _, p := range strings.Split(envPeers, ",") {
			if p = strings.TrimSpace(p); p != "" {
				peers = append(peers, p)
			}
		}
	}

	if data, err := os.ReadFile("bootstrap.txt"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if line = strings.TrimSpace(line); line != "" {
				peers = append(peers, line)
			}
		}
	}

	return peers
}

func (n *Node) loadState() error {
	data, err := n.loadStateData()
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
	n.adaptive = NewAdaptiveParams()
	n.channels = NewChannelStore()
	log.Printf("[INIT] Узел %d, файл состояния: %s", nodeID, n.stateFile)

	if err := n.loadState(); err != nil {
		log.Println("[INIT] Состояние не найдено или повреждено, начинаем с нуля")
	} else {
		log.Println("[INIT] Состояние успешно загружено")
	}

	var priv crypto.PrivKey
	keyBytes, err := n.loadPrivateKey()
	if err != nil {
		priv, _, err = crypto.GenerateKeyPair(crypto.RSA, 2048)
		if err != nil {
			log.Fatal(err)
		}
		keyBytes, _ = crypto.MarshalPrivateKey(priv)
		n.savePrivateKey(keyBytes)
		log.Printf("[INIT] Сгенерирован новый стабильный ключ узла")
	} else {
		priv, err = crypto.UnmarshalPrivateKey(keyBytes)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("[INIT] Загружен стабильный ключ узла")
	}

	host, err := libp2p.New(
		libp2p.ListenAddrStrings(
			"/ip4/0.0.0.0/tcp/9000",
			"/ip4/0.0.0.0/tcp/9001/ws",
		),
		libp2p.Identity(priv),
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Transport(websocket.New),
		libp2p.Security(libp2ptls.ID, libp2ptls.New),
	)
	if err != nil {
		log.Fatal(err)
	}
	n.host = host
	n.host.SetStreamHandler(protocolID, n.handleStream)
	n.host.SetStreamHandler(syncProtocolID, n.handleSyncStream)
	n.host.SetStreamHandler(pingProtocolID, n.handlePingStream)

	mdnsService := mdns.NewMdnsService(n.host, "sbicore", n)
	if err := mdnsService.Start(); err != nil {
		log.Fatal(err)
	}
	log.Println("[INIT] Node started with ID:", host.ID())
	log.Println("[INIT] Listening on:", host.Addrs())

	bootstrapPeers := n.loadBootstrapPeers()
	for _, addr := range bootstrapPeers {
		go func(addr string) {
			peerInfo, err := peer.AddrInfoFromString(addr)
			if err != nil {
				log.Printf("[BOOTSTRAP] Invalid peer addr %s: %v", addr, err)
				return
			}
			ctx := context.Background()
			if err := n.host.Connect(ctx, *peerInfo); err != nil {
				log.Printf("[BOOTSTRAP] Failed to connect to %s: %v", addr, err)
				return
			}
			log.Printf("[BOOTSTRAP] Connected to %s", addr)
		}(addr)
	}

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

	n.pingPeers()
	n.StartAdaptation()

	go func() {
		time.Sleep(5 * time.Second)
		n.requestRestore()
	}()

	go func() {
		for {
			time.Sleep(60 * time.Second)
			removed := n.memory.DeleteExpired()
			if removed > 0 {
				log.Printf("[EXPIRE] Удалено истёкших сообщений: %d", removed)
				n.saveState()
			}
		}
	}()

	go func() {
		for {
			time.Sleep(24 * time.Hour)
			n.assoc.CleanupOldAssociations(7)
			log.Println("[ASSOC] Старые ассоциации очищены")
		}
	}()

	go func() {
		for {
			time.Sleep(1 * time.Hour)
			threshold := 0.1
			if n.adaptive != nil {
				threshold = n.adaptive.ArchiveThreshold - 0.05
			}
			removed := n.memory.PurgeDead(threshold)
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
	http.HandleFunc("/send_stego", n.handleSendStego)
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
	http.HandleFunc("/channels", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			n.handleCreateChannel(w, r)
		} else {
			n.handleGetChannels(w, r)
		}
	})
	http.HandleFunc("/channels/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			n.handleSendToChannel(w, r)
		} else {
			n.handleGetChannelMessages(w, r)
		}
	})

	go func() {
		log.Println("[INIT] HTTP server listening on :8081")
		if err := http.ListenAndServe(":8081", nil); err != nil {
			log.Fatal(err)
		}
	}()
	select {}
}