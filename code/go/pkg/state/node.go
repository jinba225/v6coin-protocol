package state

import (
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
)

const (
	// MPT node types
	NodeTypeEmpty  = 0
	NodeTypeLeaf   = 1
	NodeTypeExt    = 2
	NodeTypeBranch = 3
)

var (
	ErrNotFound    = errors.New("key not found")
	ErrInvalidNode = errors.New("invalid node")
	ErrInvalidKey  = errors.New("invalid key")
	ErrInvalidHash = errors.New("invalid hash")
)

// Node represents a node in the MPT
type Node interface {
	Type() int
	Hash() []byte
	Serialize() ([]byte, error)
}

// EmptyNode represents an empty node
type EmptyNode struct{}

func (n *EmptyNode) Type() int                  { return NodeTypeEmpty }
func (n *EmptyNode) Hash() []byte               { return emptyNodeHash }
func (n *EmptyNode) Serialize() ([]byte, error) { return []byte{0x00}, nil }

var emptyNodeHash = []byte{0x56, 0xe8, 0x1f, 0x17, 0x1b, 0xcc, 0x55, 0xa6,
	0xff, 0x83, 0x45, 0xe6, 0x92, 0xc0, 0xf8, 0x6e,
	0x06, 0xd9, 0x81, 0xa3, 0x47, 0x2d, 0x87, 0xad,
	0x6a, 0xa7, 0x90, 0x1e, 0x7c, 0x63, 0x22, 0x86,
	0x46, 0xf1, 0x1d, 0xb5, 0x9f, 0xa1, 0xd0, 0x5e,
	0x6e, 0xb3, 0x27, 0x2d, 0x27, 0x9e, 0x6b, 0xf4,
	0x7f, 0x68, 0xe3, 0x9b, 0x5e, 0x58, 0x55, 0xb1}

// LeafNode represents a leaf node (key terminates here)
type LeafNode struct {
	Key   []byte
	Value []byte
}

func (n *LeafNode) Type() int { return NodeTypeLeaf }

func (n *LeafNode) Hash() []byte {
	data, _ := n.Serialize()
	return Hash(data)
}

func (n *LeafNode) Serialize() ([]byte, error) {
	if n == nil || len(n.Key) == 0 || len(n.Value) == 0 {
		return nil, ErrInvalidNode
	}
	// Encode key: 0x80 + length
	keyData := make([]byte, 0, 1+len(n.Key))
	keyData[0] = byte(0x80 + uint(len(n.Key)))
	keyData = append(keyData, n.Key...)
	result := EncodeNode(0x20, append(keyData, n.Value...))
	return result, nil
}

// ExtensionNode represents an extension node (shared prefix)
type ExtensionNode struct {
	Path []byte // Shared prefix
	Next Node   // Next node
}

func (n *ExtensionNode) Type() int { return NodeTypeExt }

func (n *ExtensionNode) Hash() []byte {
	data, _ := n.Serialize()
	return Hash(data)
}

func (n *ExtensionNode) Serialize() ([]byte, error) {
	if n == nil || len(n.Path) == 0 || n.Next == nil {
		return nil, ErrInvalidNode
	}
	// Encode path: 0x80 + length
	pathData := make([]byte, 0, 1+len(n.Path))
	pathData[0] = byte(0x80 + uint(len(n.Path)))
	pathData = append(pathData, n.Path...)

	nextData, err := n.Next.Serialize()
	if err != nil {
		return nil, err
	}
	result := EncodeNode(0x00, append(pathData, nextData...))
	return result, nil
}

// BranchNode represents a branch node (16 children)
type BranchNode struct {
	Children [16]Node
	Value    []byte // Optional value for this node
}

func (n *BranchNode) Type() int { return NodeTypeBranch }

func (n *BranchNode) Hash() []byte {
	data, _ := n.Serialize()
	return Hash(data)
}

func (n *BranchNode) Serialize() ([]byte, error) {
	if n == nil {
		return nil, ErrInvalidNode
	}

	childrenData := make([]byte, 0, 17)
	for i := 0; i < 16; i++ {
		if n.Children[i] == nil {
			childrenData = append(childrenData, emptyNodeHash...)
		} else {
			childData, err := n.Children[i].Serialize()
			if err != nil {
				return nil, err
			}
			childrenData = append(childrenData, childData...)
		}
	}

	if len(n.Value) > 0 {
		childrenData = append(childrenData, n.Value...)
	} else {
		childrenData = append(childrenData, emptyNodeHash...)
	}

	result := EncodeNode(0x10, childrenData)
	return result, nil
}

// EncodeKey encodes a key with length prefix
func EncodeKey(key []byte) []byte {
	if len(key) <= 55 {
		// Length prefix: 0x80 + length
		prefix := byte(0x80 + len(key))
		return append([]byte{prefix}, key...)
	}
	// Long key: 0xb7 + length bytes + key
	length := len(key)
	prefix := 0xb7
	lengthData := make([]byte, 0)
	for length > 0 {
		lengthData = append([]byte{byte(length & 0xff)}, lengthData...)
		length >>= 8
	}
	return append([]byte{byte(prefix)}, append(lengthData, key...)...)
}

// EncodeNode encodes a node with type prefix
func EncodeNode(nodeType byte, data []byte) []byte {
	if len(data) <= 55 {
		// Length prefix: 0xC0 + nodeType + length
		prefix := byte(0xC0) + nodeType + byte(uint(len(data)))
		return append([]byte{prefix}, data...)
	}
	// Long data: 0xF7 + nodeType + length bytes + data
	length := len(data)
	prefix := byte(0xF7) + nodeType
	lengthData := make([]byte, 0)
	for length > 0 {
		lengthData = append([]byte{byte(length & 0xff)}, lengthData...)
		length >>= 8
	}
	return append([]byte{byte(prefix)}, append(lengthData, data...)...)
}

// DecodeKey decodes a key from RLP encoding
func DecodeKey(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, nil
	}

	prefix := data[0]
	if prefix < 0x80 {
		return data, nil
	} else if prefix < 0xb8 {
		// Single byte length
		length := int(prefix - 0x80)
		if len(data)-1 < length {
			return nil, errors.New("invalid key length")
		}
		return data[1 : 1+length], nil
	} else if prefix < 0xc0 {
		// Multi-byte length
		lengthLength := int(prefix - 0xb7)
		if len(data)-1 < lengthLength {
			return nil, errors.New("invalid key length")
		}
		length := 0
		for i := 0; i < lengthLength; i++ {
			length = length<<8 | int(data[1+i])
		}
		if len(data)-1-lengthLength < length {
			return nil, errors.New("invalid key length")
		}
		return data[1+lengthLength : 1+lengthLength+length], nil
	}

	return nil, errors.New("invalid key prefix")
}

// NibblesToBytes converts nibbles (4-bit values) to bytes
func NibblesToBytes(nibbles []byte) []byte {
	bytes := make([]byte, (len(nibbles)+1)/2)
	for i := 0; i < len(nibbles); i++ {
		if i%2 == 0 {
			bytes[i/2] = nibbles[i] << 4
		} else {
			bytes[i/2] |= nibbles[i]
		}
	}
	return bytes
}

// BytesToNibbles converts bytes to nibbles (4-bit values)
func BytesToNibbles(data []byte) []byte {
	nibbles := make([]byte, len(data)*2)
	for i := 0; i < len(data); i++ {
		nibbles[2*i] = data[i] >> 4
		nibbles[2*i+1] = data[i] & 0x0f
	}
	return nibbles
}

// NibblesToHex converts nibbles to hex string
func NibblesToHex(nibbles []byte) string {
	hexStr := ""
	for _, n := range nibbles {
		hexStr += fmt.Sprintf("%x", n)
	}
	return hexStr
}

// Hash computes SHA3-256 hash (Keccak-256)
func Hash(data []byte) []byte {
	// TODO: Implement Keccak-256
	// For now, use SHA-256 as placeholder
	h := SHA3_256(data)
	return h[:]
}

// SHA3_256 computes SHA3-256 hash
func SHA3_256(data []byte) [32]byte {
	// TODO: Implement SHA3-256
	// For now, use a simple hash as placeholder
	var h [32]byte

	// Simple hash for testing
	if len(data) == 0 {
		return h
	}

	// Mix bytes into hash
	for i := 0; i < len(data); i++ {
		h[i%32] ^= data[i]
		h[i%32] = (h[i%32] << 3) | (h[i%32] >> 5)
	}

	return h
}

// HashToString converts hash to hex string
func HashToString(hash []byte) string {
	if hash == nil {
		return ""
	}
	return hex.EncodeToString(hash)
}

// StringToHash converts hex string to hash
func StringToHash(s string) []byte {
	if s == "" {
		return nil
	}
	hash, err := hex.DecodeString(s)
	if err != nil {
		return nil
	}
	return hash
}

// PathToKey converts a path to a key (removes nibble prefixes)
func PathToKey(path []byte) ([]byte, error) {
	if len(path) == 0 {
		return nil, nil
	}

	// Remove leading nibbles if present
	if path[0] >= 0x10 {
		path = path[1:]
	}

	return NibblesToBytes(path), nil
}

// KeyToPath converts a key to a path with nibble prefix
func KeyToPath(key []byte, isLeaf bool) ([]byte, error) {
	nibbles := BytesToNibbles(key)

	if isLeaf {
		// Add terminator nibble
		nibbles = append(nibbles, 0x10)
	}

	return nibbles, nil
}

// IsNil checks if a node is nil or empty
func IsNil(node Node) bool {
	if node == nil {
		return true
	}
	return node.Type() == NodeTypeEmpty
}

// IsEqual compares two nodes for equality
func IsEqual(a, b Node) bool {
	if IsNil(a) && IsNil(b) {
		return true
	}
	if IsNil(a) || IsNil(b) {
		return false
	}

	return reflect.DeepEqual(a, b)
}

// CloneNode creates a deep copy of a node
func CloneNode(node Node) Node {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *LeafNode:
		return &LeafNode{
			Key:   append([]byte(nil), n.Key...),
			Value: append([]byte(nil), n.Value...),
		}
	case *ExtensionNode:
		return &ExtensionNode{
			Path: append([]byte(nil), n.Path...),
			Next: CloneNode(n.Next),
		}
	case *BranchNode:
		newNode := &BranchNode{
			Value: append([]byte(nil), n.Value...),
		}
		for i := 0; i < 16; i++ {
			newNode.Children[i] = CloneNode(n.Children[i])
		}
		return newNode
	case *EmptyNode:
		return &EmptyNode{}
	default:
		return nil
	}
}

// NodeToString converts a node to string representation
func NodeToString(node Node, indent string) string {
	if node == nil {
		return indent + "nil"
	}

	switch n := node.(type) {
	case *EmptyNode:
		return indent + "Empty"
	case *LeafNode:
		return indent + fmt.Sprintf("Leaf[key=%x, value=%x]", n.Key, n.Value)
	case *ExtensionNode:
		return indent + fmt.Sprintf("Ext[path=%x]\n%s", n.Path, NodeToString(n.Next, indent+"  "))
	case *BranchNode:
		str := indent + "Branch[\n"
		for i := 0; i < 16; i++ {
			if n.Children[i] != nil {
				str += indent + fmt.Sprintf("  %d: %s\n", i, NodeToString(n.Children[i], ""))
			}
		}
		if len(n.Value) > 0 {
			str += indent + fmt.Sprintf("  value: %x\n", n.Value)
		}
		str += indent + "]"
		return str
	default:
		return indent + fmt.Sprintf("Unknown[type=%d]", node.Type())
	}
}

// CommonPrefixLength finds the length of common prefix
func CommonPrefixLength(a, b []byte) int {
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}

	for i := 0; i < minLen; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return minLen
}

// SplitKey splits a key at position
func SplitKey(key []byte, pos int) ([]byte, []byte) {
	if pos <= 0 {
		return []byte{}, key
	}
	if pos >= len(key) {
		return key, []byte{}
	}
	return key[:pos], key[pos:]
}

// ConcatKeys concatenates multiple keys
func ConcatKeys(keys ...[]byte) []byte {
	totalLen := 0
	for _, key := range keys {
		totalLen += len(key)
	}

	result := make([]byte, 0, totalLen)
	for _, key := range keys {
		result = append(result, key...)
	}
	return result
}

// HashEqual compares two hashes for equality
func HashEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
