package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multihash"
)

// DHTNode — обёртка над Kademlia DHT
type DHTNode struct {
	dht     *dht.IpfsDHT
	host    host.Host
	mode    dht.ModeOpt
	started bool
}

// NewDHT — создаёт DHT-узел
// mode: dht.ModeServer (desktop/bootstrap) или dht.ModeClient (mobile)
func NewDHT(h host.Host, mode dht.ModeOpt) (*DHTNode, error) {
	if h == nil {
		return nil, fmt.Errorf("host is nil")
	}

	kdht, err := dht.New(
		context.Background(),
		h,
		dht.Mode(mode),
		dht.ProtocolPrefix("/isotope/kad"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create DHT: %w", err)
	}

	return &DHTNode{
		dht:     kdht,
		host:    h,
		mode:    mode,
		started: false,
	}, nil
}

// JoinDHT — подключается к bootstrap-пирам и запускает DHT
func (dn *DHTNode) JoinDHT(bootstrapPeers []string) error {
	if dn == nil || dn.dht == nil {
		return fmt.Errorf("DHT not initialized")
	}

	// Подключаемся к bootstrap-пирам
	var connected int
	for _, addrStr := range bootstrapPeers {
		addrStr = strings.TrimSpace(addrStr)
		if addrStr == "" {
			continue
		}

		peerInfo, err := peer.AddrInfoFromString(addrStr)
		if err != nil {
			log.Printf("[DHT] Invalid bootstrap addr %s: %v", addrStr, err)
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err = dn.host.Connect(ctx, *peerInfo)
		cancel()
		if err != nil {
			log.Printf("[DHT] Failed to connect to bootstrap %s: %v", addrStr, err)
			continue
		}
		connected++
		log.Printf("[DHT] Connected to bootstrap: %s", peerInfo.ID.String()[:16])
	}

	if connected == 0 && len(bootstrapPeers) > 0 {
		return fmt.Errorf("failed to connect to any bootstrap peer")
	}

	// Bootstrap DHT (только если есть bootstrap-пиры)
	if connected > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := dn.dht.Bootstrap(ctx); err != nil {
			return fmt.Errorf("DHT bootstrap failed: %w", err)
		}
	}

	dn.started = true
	log.Printf("[DHT] DHT started (mode=%v, bootstraps=%d)", dn.mode, connected)

	// Фоновая поддержка routing table (только для server-режима)
	if dn.mode == dht.ModeServer {
		go dn.refreshLoop()
	}

	// Анонсируем себя
	go func() {
		time.Sleep(2 * time.Second)
		if err := dn.Provide(); err != nil {
			log.Printf("[DHT] Initial provide failed: %v", err)
		}
	}()

	return nil
}

// Provide — анонсирует PeerID узла в DHT
func (dn *DHTNode) Provide() error {
	if dn == nil || dn.dht == nil || !dn.started {
		return fmt.Errorf("DHT not started")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Создаём CID из PeerID
	mh, err := multihash.Sum([]byte(dn.host.ID().String()), multihash.SHA2_256, -1)
	if err != nil {
		return fmt.Errorf("failed to create multihash: %w", err)
	}

	c := cid.NewCidV1(cid.Raw, mh)

	if err := dn.dht.Provide(ctx, c, true); err != nil {
		return fmt.Errorf("provide failed: %w", err)
	}

	log.Printf("[DHT] Provided self to DHT: %s", dn.host.ID().String()[:16])
	return nil
}

// FindPeer — ищет multiaddr по PeerID
func (dn *DHTNode) FindPeer(peerID string) ([]peer.AddrInfo, error) {
	if dn == nil || dn.dht == nil || !dn.started {
		return nil, fmt.Errorf("DHT not started")
	}

	pid, err := peer.Decode(peerID)
	if err != nil {
		return nil, fmt.Errorf("invalid peer ID: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	peerInfo, err := dn.dht.FindPeer(ctx, pid)
	if err != nil {
		return nil, fmt.Errorf("find peer failed: %w", err)
	}

	log.Printf("[DHT] Found peer %s with %d addrs", peerID[:16], len(peerInfo.Addrs))
	return []peer.AddrInfo{peerInfo}, nil
}

// refreshLoop — фоновая поддержка routing table (только desktop/bootstrap)
func (dn *DHTNode) refreshLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		if !dn.started {
			return
		}

		err := dn.dht.RefreshRoutingTable()
		if err != nil {
			log.Printf("[DHT] Routing table refresh failed: %v", err)
		} else {
			log.Printf("[DHT] Routing table refreshed")
		}
	}
}

// Close — корректное завершение DHT
func (dn *DHTNode) Close() error {
	if dn == nil || dn.dht == nil {
		return nil
	}

	dn.started = false
	return dn.dht.Close()
}

// GetDHTInfo — возвращает информацию о DHT
func (dn *DHTNode) GetDHTInfo() string {
	if dn == nil || dn.dht == nil {
		return `{"started":false}`
	}

	rt := dn.dht.RoutingTable()
	peers := rt.ListPeers()

	peersList := make([]string, 0, len(peers))
	for _, p := range peers {
		peersList = append(peersList, p.String()[:16])
	}

	info := map[string]interface{}{
		"started":  dn.started,
		"mode":     fmt.Sprintf("%v", dn.mode),
		"peer_id":  dn.host.ID().String(),
		"rt_size":  len(peers),
		"rt_peers": peersList,
	}

	data, _ := json.Marshal(info)
	return string(data)
}

// IsStarted — проверяет, запущен ли DHT
func (dn *DHTNode) IsStarted() bool {
	return dn != nil && dn.started
}