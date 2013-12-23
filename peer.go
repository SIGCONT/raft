package raft

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"sync"
)

const (
	jsonPeerPath = "peers.json"
)

// PeerStore provides an interface for persistent storage and
// retrieval of peers. We use a seperate interface than StableStore
// since the peers may need to be editted by a human operator. For example,
// in a two node cluster, the failure of either node requires human intervention
// since consensus is impossible.
type PeerStore interface {
	// Returns the list of known peers
	Peers() ([]net.Addr, error)

	// Sets the list of known peers. This is invoked when
	// a peer is added or removed
	SetPeers([]net.Addr) error
}

// StatisPeers is used to provide a static list of peers
type StaticPeers struct {
	StaticPeers []net.Addr
}

func (s *StaticPeers) Peers() ([]net.Addr, error) {
	return s.StaticPeers, nil
}

func (s *StaticPeers) SetPeers(p []net.Addr) error {
	s.StaticPeers = p
	return nil
}

// JSONPeers is used to provide peer persistence on disk in the form
// of a JSON file. This allows human operators to manipulate the file.
type JSONPeers struct {
	l     sync.Mutex
	path  string
	trans Transport
}

// NewJSONPeers creates a new JSONPeers store. Requires a transport
// to handle the serialization of network addresses
func NewJSONPeers(base string, trans Transport) *JSONPeers {
	path := filepath.Join(base, jsonPeerPath)
	store := &JSONPeers{
		path:  path,
		trans: trans,
	}
	return store
}

func (j *JSONPeers) Peers() ([]net.Addr, error) {
	j.l.Lock()
	defer j.l.Unlock()

	// Read the file
	buf, err := ioutil.ReadFile(j.path)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	// Check for no peers
	if len(buf) == 0 {
		return nil, nil
	}

	// Decode the peers
	var peerSet [][]byte
	dec := json.NewDecoder(bytes.NewReader(buf))
	if err := dec.Decode(&peerSet); err != nil {
		return nil, err
	}

	// Deserialize each peer
	var peers []net.Addr
	for _, p := range peerSet {
		peers = append(peers, j.trans.DecodePeer(p))
	}
	return peers, nil
}

func (j *JSONPeers) SetPeers(peers []net.Addr) error {
	j.l.Lock()
	defer j.l.Unlock()

	// Encode each peer
	var peerSet [][]byte
	for _, p := range peers {
		peerSet = append(peerSet, j.trans.EncodePeer(p))
	}

	// Convert to JSON
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(peerSet); err != nil {
		return err
	}

	// Write out as JSON
	return ioutil.WriteFile(j.path, buf.Bytes(), 0755)
}
