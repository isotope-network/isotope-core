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

// DHTNode — равноправный DHT-узел (без Server/Client разделения)
type DHTNode struct {
	dht     *dht.IpfsDHT
	host    host.Host
	started bool
}

// NewDHT — создаёт DHT-узел
func NewDHT(h host.Host) (*DHTNode, error) {
	if h == nil {
		return nil, fmt.Errorf("host is nil")
	}

	kdht, err := dht.New(
		context.Background(),
		h,
		dht.ProtocolPrefix("/isotope/kad"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create DHT: %w", err)
	}

	return &DHTNode{
		dht:     kdht,
		host:    h,
		started: false,
	}, nil
}

// JoinDHT — подключается к известным узлам и запускает DHT
func (dn *DHTNode) JoinDHT(bootstrapPeers []string) error {
	if dn == nil || dn.dht == nil {
		return fmt.Errorf("DHT not initialized")
	}

	// Подключаемся к известным узлам
	var connected int
	for _, addrStr := range bootstrapPeers {
		addrStr = strings.TrimSpace(addrStr)
		if addrStr == "" {
			continue
		}

		peerInfo, err := peer.AddrInfoFromString(addrStr)
		if err != nil {
			log.Printf("[DHT] Invalid addr %s: %v", addrStr, err)
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err = dn.host.Connect(ctx, *peerInfo)
		cancel()
		if err != nil {
			log.Printf("[DHT] Failed to connect to %s: %v", addrStr, err)
			continue
		}
		connected++
		log.Printf("[DHT] Connected to known peer: %s", peerInfo.ID.String()[:16])
	}

	// Bootstrap DHT (только если есть к кому)
	if connected > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := dn.dht.Bootstrap(ctx); err != nil {
			log.Printf("[DHT] Bootstrap warning: %v", err)
		}
		
		// Обновить routing table после подключения
		dn.RefreshOnDemand()
	}

	dn.started = true
	log.Printf("[DHT] DHT started (peers=%d)", connected)

	// Анонсируем себя (с задержкой и повторными попытками)
	go func() {
		for i := 0; i < 5; i++ {
			time.Sleep(time.Duration(2+i*2) * time.Second)
			if err := dn.Provide(); err != nil {
				log.Printf("[DHT] Provide attempt %d failed: %v", i+1, err)
			} else {
				log.Printf("[DHT] Provided self to DHT (attempt %d)", i+1)
				break
			}
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

// RefreshOnDemand — обновление routing table по требованию
func (dn *DHTNode) RefreshOnDemand() {
	if dn == nil || dn.dht == nil || !dn.started {
		return
	}

	err := dn.dht.RefreshRoutingTable()
	if err != nil {
		log.Printf("[DHT] Refresh on demand failed: %v", err)
	} else {
		log.Printf("[DHT] Routing table refreshed (on demand)")
	}
}

// GetRoutingTableSize — размер routing table
func (dn *DHTNode) GetRoutingTableSize() int {
	if dn == nil || dn.dht == nil {
		return 0
	}
	rt := dn.dht.RoutingTable()
	return rt.Size()
}

// IsDHTActive — DHT активен, если таблица достаточно большая
func (dn *DHTNode) IsDHTActive() bool {
	return dn.GetRoutingTableSize() >= 10
}

// SaveRoutingTable — сериализует routing table для state
func (dn *DHTNode) SaveRoutingTable() []byte {
	if dn == nil || dn.dht == nil {
		return []byte("[]")
	}

	rt := dn.dht.RoutingTable()
	peers := rt.ListPeers()

	var addrs []string
	for _, p := range peers {
		peerInfo := dn.host.Peerstore().PeerInfo(p)
		for _, addr := range peerInfo.Addrs {
			addrs = append(addrs, addr.String()+"/p2p/"+p.String())
		}
	}

	data, _ := json.Marshal(addrs)
	return data
}

// LoadRoutingTable — восстанавливает routing table из state
func (dn *DHTNode) LoadRoutingTable(data []byte) error {
	if dn == nil || dn.dht == nil {
		return fmt.Errorf("DHT not initialized")
	}

	var addrs []string
	if err := json.Unmarshal(data, &addrs); err != nil {
		return err
	}

	for _, addr := range addrs {
		peerInfo, err := peer.AddrInfoFromString(addr)
		if err != nil {
			continue
		}
		dn.host.Peerstore().AddAddrs(peerInfo.ID, peerInfo.Addrs, time.Hour*24)
	}

	log.Printf("[DHT] Loaded %d known peers from routing table", len(addrs))
	return nil
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
		"peer_id":  dn.host.ID().String(),
		"rt_size":  len(peers),
		"rt_peers": peersList,
		"dht_active": dn.IsDHTActive(),
	}

	data, _ := json.Marshal(info)
	return string(data)
}

// IsStarted — проверяет, запущен ли DHT
func (dn *DHTNode) IsStarted() bool {
	return dn != nil && dn.started
}