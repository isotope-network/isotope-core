package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

// ============================================================
// НЕЙРОСЕТЬ (СЛОИ)
// ============================================================

// forward — прогоняет вектор через все слои с функцией активации
func forward(input []float64, layers [][]float64) ([]float64, []float64) {
	current := make([]float64, len(input))
	copy(current, input)

	var rawOutput []float64
	for _, layer := range layers {
		if len(layer) != len(current) {
			continue
		}
		rawOutput = make([]float64, len(current))
		for i := range current {
			rawOutput[i] = current[i] * layer[i]
			if rawOutput[i] > 1.0 {
				rawOutput[i] = 1.0
			} else if rawOutput[i] < -1.0 {
				rawOutput[i] = -1.0
			}
		}
		copy(current, rawOutput)
	}
	if rawOutput == nil {
		rawOutput = current
	}
	return rawOutput, current
}

// train — градиентный спуск: корректирует веса на основе ошибки
func train(layers [][]float64, input, output, target []float64, learningRate float64) [][]float64 {
	if len(input) != len(target) || len(output) != len(target) {
		return layers
	}

	errors := make([]float64, len(output))
	for i := range output {
		errors[i] = target[i] - output[i]
	}
	log.Printf("[TRAIN] Обучение: ошибка = %v, скорость = %v", errors[:5], learningRate)

	if len(layers) > 0 {
		lastIdx := len(layers) - 1
		if len(layers[lastIdx]) == len(errors) {
			for i := range errors {
				layers[lastIdx][i] += errors[i] * learningRate
				if layers[lastIdx][i] > 1.0 {
					layers[lastIdx][i] = 1.0
				} else if layers[lastIdx][i] < -1.0 {
					layers[lastIdx][i] = -1.0
				}
			}
		}
	}
	return layers
}

// ============================================================
// СИНХРОНИЗАЦИЯ СЛОЁВ (v1.6 — gossip + троттлинг + дельта + TTL)
// ============================================================

const syncThrottleInterval = 5 * time.Second
const syncBackgroundInterval = 30 * time.Second
const syncChangeThreshold = 0.005
const gossipDefaultTTL = 3

// SyncPayload — обёртка для дельты с TTL для gossip
type SyncPayload struct {
	TTL   int          `json:"ttl"`
	Delta []DeltaLayer `json:"delta"`
}

// DeltaLayer — один слой в дельте
type DeltaLayer struct {
	Index int       `json:"index"`
	Data  []float64 `json:"data"`
}

// broadcastLayers — отправляет слои через gossip (случайным пирам)
func (n *Node) broadcastLayers() {
	n.mu.Lock()

	if n.host == nil {
		n.mu.Unlock()
		return
	}

	if time.Since(n.lastSyncSent) < syncThrottleInterval {
		n.mu.Unlock()
		return
	}

	if !n.layersDirty {
		n.mu.Unlock()
		return
	}

	var delta []DeltaLayer
	for i := range n.layers {
		if i < len(n.lastSyncedLayers) && len(n.layers[i]) == len(n.lastSyncedLayers[i]) {
			changed := false
			for j := range n.layers[i] {
				diff := n.layers[i][j] - n.lastSyncedLayers[i][j]
				if diff < 0 {
					diff = -diff
				}
				if diff > syncChangeThreshold {
					changed = true
					break
				}
			}
			if !changed {
				continue
			}
		}
		layerCopy := make([]float64, len(n.layers[i]))
		copy(layerCopy, n.layers[i])
		delta = append(delta, DeltaLayer{Index: i, Data: layerCopy})
	}

	if len(delta) == 0 {
		n.layersDirty = false
		n.mu.Unlock()
		return
	}

	n.lastSyncedLayers = make([][]float64, len(n.layers))
	for i := range n.layers {
		n.lastSyncedLayers[i] = make([]float64, len(n.layers[i]))
		copy(n.lastSyncedLayers[i], n.layers[i])
	}
	n.layersDirty = false
	n.lastSyncSent = time.Now()

	peers := n.host.Network().Peers()
	n.mu.Unlock()

	if len(peers) == 0 {
		return
	}

	fanout := int(math.Sqrt(float64(len(peers))))
	if fanout < 2 {
		fanout = 2
	}
	if fanout > len(peers) {
		fanout = len(peers)
	}

	selected := selectRandomPeers(peers, fanout)

	payload := SyncPayload{
		TTL:   gossipDefaultTTL,
		Delta: delta,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	for _, p := range selected {
		go func(peerID peer.ID) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			s, err := n.host.NewStream(ctx, peerID, syncProtocolID)
			if err != nil {
				return
			}
			defer s.Close()

			s.Write(data)
			log.Printf("[SYNC] Gossip отправлен узлу %s: %d слоёв (fanout=%d, ttl=%d)", peerID.String()[:8], len(delta), fanout, gossipDefaultTTL)
		}(p)
	}
}

// handleSyncStream — принимает слои, усредняет, пересылает дальше (gossip)
func (n *Node) handleSyncStream(stream network.Stream) {
	defer stream.Close()

	data, err := io.ReadAll(stream)
	if err != nil {
		log.Println("Sync read error:", err)
		return
	}

	var payload SyncPayload
	if err := json.Unmarshal(data, &payload); err == nil && len(payload.Delta) > 0 {
		n.mu.Lock()
		for _, dl := range payload.Delta {
			if dl.Index < len(n.layers) {
				if len(dl.Data) == len(n.layers[dl.Index]) {
					for j := range dl.Data {
						n.layers[dl.Index][j] = (n.layers[dl.Index][j] + dl.Data[j]) / 2.0
					}
				}
			} else {
				for len(n.layers) <= dl.Index {
					n.layers = append(n.layers, make([]float64, VectorDim))
				}
				n.layers[dl.Index] = dl.Data
			}
		}
		n.layersDirty = true

		ttl := payload.TTL - 1
		peers := n.host.Network().Peers()
		senderID := stream.Conn().RemotePeer()

		if ttl > 0 && len(peers) > 1 {
			fanout := int(math.Sqrt(float64(len(peers))))
			if fanout < 2 {
				fanout = 2
			}
			if fanout > len(peers) {
				fanout = len(peers)
			}

			var candidates []peer.ID
			for _, p := range peers {
				if p != senderID {
					candidates = append(candidates, p)
				}
			}

			if len(candidates) > 0 {
				selected := selectRandomPeers(candidates, fanout)
				payload.TTL = ttl
				fwdData, _ := json.Marshal(payload)

				for _, p := range selected {
					go func(peerID peer.ID) {
						ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
						defer cancel()

						s, err := n.host.NewStream(ctx, peerID, syncProtocolID)
						if err != nil {
							return
						}
						defer s.Close()

						s.Write(fwdData)
						log.Printf("[SYNC] Gossip forwarded to %s: ttl=%d", peerID.String()[:8], ttl)
					}(p)
				}
			}
		}

		n.mu.Unlock()
		log.Printf("[SYNC] Gossip delta received from %s: %d layers (ttl=%d)", senderID.String()[:8], len(payload.Delta), payload.TTL)
		return
	}

	var delta []DeltaLayer
	if err := json.Unmarshal(data, &delta); err == nil && len(delta) > 0 {
		n.mu.Lock()
		for _, dl := range delta {
			if dl.Index < len(n.layers) {
				if len(dl.Data) == len(n.layers[dl.Index]) {
					for j := range dl.Data {
						n.layers[dl.Index][j] = (n.layers[dl.Index][j] + dl.Data[j]) / 2.0
					}
				}
			} else {
				for len(n.layers) <= dl.Index {
					n.layers = append(n.layers, make([]float64, VectorDim))
				}
				n.layers[dl.Index] = dl.Data
			}
		}
		n.layersDirty = true
		n.mu.Unlock()
		log.Printf("[SYNC] Дельта получена от %s: %d слоёв (старый формат)", stream.Conn().RemotePeer().String()[:8], len(delta))
		return
	}

	var peerLayers [][]float64
	if err := json.Unmarshal(data, &peerLayers); err != nil {
		log.Println("Sync unmarshal error:", err)
		return
	}

	n.mu.Lock()
	for i := range peerLayers {
		if i < len(n.layers) {
			if len(peerLayers[i]) == len(n.layers[i]) {
				for j := range peerLayers[i] {
					n.layers[i][j] = (n.layers[i][j] + peerLayers[i][j]) / 2.0
				}
			}
		} else {
			n.layers = append(n.layers, peerLayers[i])
		}
	}
	n.layersDirty = true
	n.mu.Unlock()
	log.Printf("[SYNC] Полные слои получены от %s: %d слоёв", stream.Conn().RemotePeer().String()[:8], len(peerLayers))
}

// selectRandomPeers — выбирает случайные k пиров из списка
func selectRandomPeers(peers []peer.ID, k int) []peer.ID {
	if k >= len(peers) {
		return peers
	}
	perm := rand.Perm(len(peers))
	result := make([]peer.ID, k)
	for i := 0; i < k; i++ {
		result[i] = peers[perm[i]]
	}
	return result
}

// startSyncBackground — фоновый цикл синхронизации
func (n *Node) startSyncBackground() {
	go func() {
		for {
			time.Sleep(syncBackgroundInterval)
			n.mu.Lock()
			if n.layersDirty {
				n.mu.Unlock()
				n.broadcastLayers()
			} else {
				n.mu.Unlock()
			}
		}
	}()
}