package p2p

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"mychain/internal/chain"
)

type Node struct {
	selfURL string
	peers   []string
	bc      *chain.Blockchain
}

func NewNode(selfURL string, peers []string, bc *chain.Blockchain) *Node {
	return &Node{
		selfURL: selfURL,
		peers:   peers,
		bc:      bc,
	}
}

func (n *Node) postTo(url string, body []byte) error {
	resp, err := http.Post(url+"/sync", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	fmt.Println("post to", url+"/sync", ":", result["status"])
	return nil
}

func (n *Node) syncFromPeer(peer string) error {
	resp, err := http.Get(peer + "/blocks")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("peer returned: %d", resp.StatusCode)
	}

	var blocks []*chain.Block
	if err := json.NewDecoder(resp.Body).Decode(&blocks); err != nil {
		return fmt.Errorf("decode blocks: %w", err)
	}

	body, err := json.Marshal(map[string]any{
		"blocks":     blocks,
		"difficulty": n.bc.GetDifficulty(),
	})
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	return n.postTo(n.selfURL, body)
}

func (n *Node) SyncFromPeers() {
	for _, peer := range n.peers {
		if peer == n.selfURL {
			continue
		}
		if err := n.syncFromPeer(peer); err != nil {
			fmt.Println("sync from", peer, "failed:", err)
		}
	}
}

func (n *Node) Broadcast() {
	blocks := n.bc.GetBlocks()
	difficulty := n.bc.GetDifficulty()

	body, err := json.Marshal(map[string]any{
		"blocks":     blocks,
		"difficulty": difficulty,
	})
	if err != nil {
		return
	}

	for _, peer := range n.peers {
		if peer == n.selfURL {
			continue
		}
		go n.postTo(peer, body)
	}
}

func (n *Node) StartPeriodicSync(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			n.SyncFromPeers()
		}
	}()
}
