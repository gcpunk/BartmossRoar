package app

import (
	"encoding/json"
	"os"
)

type NodeSetInfo struct {
	Nodes []NodeInfo `json:"nodes"`
}

// NodeInfo associated with core/database/nodes.go
type NodeInfo struct {
	Identity []byte `json:"identity"`

	// two slices of keys, base64-encoded, for JSON serialization
	SignKey    [][]byte `json:"signKey"`
	EncryptKey [][]byte `json:"encryptKey"`
	Address    []string `json:"address"`
}

func MarshalNodeSetInfoToJSON(node NodeSetInfo) ([]byte, error) {
	return json.Marshal(node)
}

func UnmarshalJSONToNodeSetInfo(b []byte) (NodeSetInfo, error) {
	var nodeInfo NodeSetInfo

	if err := json.Unmarshal(b, &nodeInfo); err != nil {
		return NodeSetInfo{}, err
	}

	return nodeInfo, nil
}

func ReadNodeInfo(path string) (NodeSetInfo, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return NodeSetInfo{}, err
	}

	return UnmarshalJSONToNodeSetInfo(b)
}

type Config struct {
	Protocols       []string `json:"Protocols"`
	AddressesIfTCP  []string `json:"addressesIfTCP"`
	LocalPortsIfTor []int    `json:"localPortsIfTor"`
	// todo other configs here
}

func MarshalBrConfigToJSON(cfg Config) ([]byte, error) {
	return json.Marshal(cfg)
}

func UnmarshalJSONToBrConfig(b []byte) (Config, error) {
	var BrConfig Config

	if err := json.Unmarshal(b, &BrConfig); err != nil {
		return Config{}, err
	}

	return BrConfig, nil
}

func ReadBrConfig(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	return UnmarshalJSONToBrConfig(b)
}
