package state

import (
	"fmt"
	"sync"
)

// MPT represents a Merkle Patricia Trie
type MPT struct {
	root  Node
	cache map[string]Node
	mu    sync.RWMutex
}

// NewMPT creates a new MPT
func NewMPT() *MPT {
	return &MPT{
		root:  &EmptyNode{},
		cache: make(map[string]Node),
	}
}

// Get retrieves a value by key
func (mpt *MPT) Get(key []byte) ([]byte, error) {
	mpt.mu.RLock()
	defer mpt.mu.RUnlock()

	path, err := KeyToPath(key, true) // Is leaf
	if err != nil {
		return nil, fmt.Errorf("failed to convert key to path: %w", err)
	}

	return mpt.get(mpt.root, path)
}

// get recursively finds a value by path
func (mpt *MPT) get(node Node, path []byte) ([]byte, error) {
	if IsNil(node) {
		return nil, ErrNotFound
	}

	switch n := node.(type) {
	case *EmptyNode:
		return nil, ErrNotFound

	case *LeafNode:
		// Check if paths match
		if len(path) == 0 {
			return n.Value, nil
		}
		// Compare stored path with search path
		storedPath, err := PathToKey(n.Key)
		if err != nil {
			return nil, fmt.Errorf("failed to decode path: %w", err)
		}
		storedNibbles, err := KeyToPath(storedPath, true)
		if err != nil {
			return nil, fmt.Errorf("failed to convert to path: %w", err)
		}
		if len(storedNibbles) > len(path) {
			return nil, ErrNotFound
		}
		for i := 0; i < len(path) && i < len(storedNibbles); i++ {
			if path[i] != storedNibbles[i] {
				return nil, ErrNotFound
			}
		}
		return n.Value, nil

	case *ExtensionNode:
		// Match shared prefix
		if len(path) == 0 {
			return nil, ErrNotFound
		}
		if len(n.Path) == 0 {
			return mpt.get(n.Next, path)
		}

		// Find common prefix length
		commonLen := 0
		minLen := len(n.Path)
		if len(path) < minLen {
			minLen = len(path)
		}
		for i := 0; i < minLen; i++ {
			if n.Path[i] != path[i] {
				break
			}
			commonLen++
		}

		if commonLen == len(n.Path) {
			// Full match, continue to next node
			return mpt.get(n.Next, path[commonLen:])
		}
		// Partial match, path diverges
		return nil, ErrNotFound

	case *BranchNode:
		// Navigate through branch node
		if len(path) == 0 {
			if len(n.Value) > 0 {
				return n.Value, nil
			}
			return nil, ErrNotFound
		}

		nibble := path[0]
		if nibble >= 16 {
			return nil, ErrNotFound
		}

		return mpt.get(n.Children[nibble], path[1:])

	default:
		return nil, fmt.Errorf("unknown node type")
	}
}

// Put inserts a key-value pair
func (mpt *MPT) Put(key, value []byte) error {
	mpt.mu.Lock()
	defer mpt.mu.Unlock()

	path, err := KeyToPath(key, true) // Is leaf
	if err != nil {
		return fmt.Errorf("failed to convert key to path: %w", err)
	}

	newRoot, err := mpt.insert(mpt.root, path, value)
	if err != nil {
		return fmt.Errorf("failed to insert: %w", err)
	}

	mpt.root = newRoot
	return nil
}

// insert recursively inserts a value into the trie
func (mpt *MPT) insert(node Node, path, value []byte) (Node, error) {
	if IsNil(node) {
		// Create new leaf node
		return &LeafNode{
			Key:   path,
			Value: value,
		}, nil
	}

	switch n := node.(type) {
	case *EmptyNode:
		// Replace empty node with new leaf
		return &LeafNode{
			Key:   path,
			Value: value,
		}, nil

	case *LeafNode:
		return mpt.insertToLeaf(n, path, value)

	case *ExtensionNode:
		return mpt.insertToExtension(n, path, value)

	case *BranchNode:
		return mpt.insertToBranch(n, path, value)

	default:
		return nil, fmt.Errorf("unknown node type")
	}
}

// insertToLeaf handles insertion into a leaf node
func (mpt *MPT) insertToLeaf(leaf *LeafNode, path, value []byte) (Node, error) {
	if len(path) == 0 {
		// Replace value at this leaf
		return &LeafNode{
			Key:   leaf.Key,
			Value: value,
		}, nil
	}

	// Find common prefix between existing key and new path
	commonLen := CommonPrefixLength(leaf.Key, path)

	if commonLen == 0 {
		// No common prefix, create branch node
		newBranch := &BranchNode{
			Children: [16]Node{},
			Value:    []byte{},
		}

		// Insert existing leaf into branch
		if len(leaf.Key) > 0 {
			firstNibble := leaf.Key[0]
			if firstNibble < 16 {
				newBranch.Children[firstNibble], _ = mpt.insert(&EmptyNode{}, leaf.Key[1:], leaf.Value)
			}
		}

		// Insert new value into branch
		if len(path) > 0 {
			firstNibble := path[0]
			if firstNibble < 16 {
				newBranch.Children[firstNibble], _ = mpt.insert(&EmptyNode{}, path[1:], value)
			}
		}

		return newBranch, nil
	}

	// Common prefix > 0, need to create extension node
	if commonLen == len(leaf.Key) {
		// New key is extension of existing key
		newExt := &ExtensionNode{
			Path: leaf.Key,
			Next: &LeafNode{
				Key:   path[commonLen:],
				Value: value,
			},
		}
		return newExt, nil
	}

	if commonLen == len(path) {
		// Existing key is extension of new key
		newExt := &ExtensionNode{
			Path: path,
			Next: &LeafNode{
				Key:   leaf.Key[commonLen:],
				Value: value,
			},
		}
		return newExt, nil
	}

	// Both keys diverge at commonLen
	newExt := &ExtensionNode{
		Path: path[:commonLen],
		Next: &BranchNode{
			Children: [16]Node{},
			Value:    []byte{},
		},
	}

	// Insert existing key
	if commonLen < len(leaf.Key) {
		existingNibble := leaf.Key[commonLen]
		if existingNibble < 16 {
			newExt.Next.(*BranchNode).Children[existingNibble], _ = mpt.insert(&EmptyNode{}, leaf.Key[commonLen+1:], leaf.Value)
		}
	}

	// Insert new key
	if commonLen < len(path) {
		newNibble := path[commonLen]
		if newNibble < 16 {
			newExt.Next.(*BranchNode).Children[newNibble], _ = mpt.insert(&EmptyNode{}, path[commonLen+1:], value)
		}
	}

	return newExt, nil
}

// insertToExtension handles insertion into an extension node
func (mpt *MPT) insertToExtension(ext *ExtensionNode, path, value []byte) (Node, error) {
	if len(path) == 0 {
		// Replace extension node with leaf
		return &LeafNode{
			Key:   ext.Path,
			Value: value,
		}, nil
	}

	// Find common prefix
	commonLen := CommonPrefixLength(ext.Path, path)

	if commonLen == len(ext.Path) {
		// Full match, insert into next node
		newNext, err := mpt.insert(ext.Next, path[commonLen:], value)
		if err != nil {
			return nil, err
		}
		return &ExtensionNode{
			Path: ext.Path,
			Next: newNext,
		}, nil
	}

	// Paths diverge, need to convert extension to branch
	newBranch := &BranchNode{
		Children: [16]Node{},
		Value:    []byte{},
	}

	// Insert old path
	if len(ext.Path) > 0 {
		oldNibble := ext.Path[0]
		if oldNibble < 16 {
			oldValue := []byte(nil)
			if leafNode, ok := ext.Next.(*LeafNode); ok {
				oldValue = leafNode.Value
			}
			newBranch.Children[oldNibble], _ = mpt.insert(ext.Next, ext.Path[1:], oldValue)
		}
	}

	// Insert new path
	if len(path) > 0 {
		newNibble := path[0]
		if newNibble < 16 {
			newBranch.Children[newNibble], _ = mpt.insert(&EmptyNode{}, path[1:], value)
		}
	}

	return newBranch, nil
}

// insertToBranch handles insertion into a branch node
func (mpt *MPT) insertToBranch(branch *BranchNode, path, value []byte) (Node, error) {
	if len(path) == 0 {
		// Replace value at this branch
		newBranch := CloneNode(branch).(*BranchNode)
		newBranch.Value = value
		return newBranch, nil
	}

	nibble := path[0]
	if nibble >= 16 {
		return nil, ErrInvalidKey
	}

	// Insert into appropriate child
	newChild, err := mpt.insert(branch.Children[nibble], path[1:], value)
	if err != nil {
		return nil, err
	}

	// Create new branch with updated child
	newBranch := &BranchNode{
		Children: [16]Node{},
		Value:    append([]byte(nil), branch.Value...),
	}
	for i := 0; i < 16; i++ {
		newBranch.Children[i] = branch.Children[i]
	}
	newBranch.Children[nibble] = newChild

	return newBranch, nil
}

// Delete removes a key from the trie
func (mpt *MPT) Delete(key []byte) error {
	mpt.mu.Lock()
	defer mpt.mu.Unlock()

	path, err := KeyToPath(key, true) // Is leaf
	if err != nil {
		return fmt.Errorf("failed to convert key to path: %w", err)
	}

	newRoot, err := mpt.delete(mpt.root, path)
	if err != nil {
		return fmt.Errorf("failed to delete: %w", err)
	}

	mpt.root = newRoot
	return nil
}

// delete recursively removes a key from the trie
func (mpt *MPT) delete(node Node, path []byte) (Node, error) {
	if IsNil(node) {
		return nil, ErrNotFound
	}

	switch n := node.(type) {
	case *EmptyNode:
		return nil, ErrNotFound

	case *LeafNode:
		if len(path) == 0 {
			// Found the key to delete
			return &EmptyNode{}, nil
		}
		return nil, ErrNotFound

	case *ExtensionNode:
		if len(path) == 0 {
			// Delete this extension node
			return &EmptyNode{}, nil
		}
		commonLen := CommonPrefixLength(n.Path, path)
		if commonLen == len(n.Path) {
			// Continue to next node
			newNext, err := mpt.delete(n.Next, path[commonLen:])
			if err != nil {
				return nil, err
			}
			if IsNil(newNext) {
				return &EmptyNode{}, nil
			}
			return &ExtensionNode{
				Path: n.Path,
				Next: newNext,
			}, nil
		}
		return n, ErrNotFound

	case *BranchNode:
		// Create a copy of the branch node
		newBranch := &BranchNode{
			Children: [16]Node{},
			Value:    append([]byte(nil), n.Value...),
		}
		for i := 0; i < 16; i++ {
			newBranch.Children[i] = n.Children[i]
		}

		if len(path) == 0 {
			// Clear value at this branch
			newBranch.Value = []byte{}
			return newBranch, nil
		}

		nibble := path[0]
		if nibble >= 16 {
			return nil, ErrInvalidKey
		}

		newChild, err := mpt.delete(newBranch.Children[nibble], path[1:])
		if err != nil {
			return nil, err
		}

		newBranch.Children[nibble] = newChild
		return newBranch, nil

	default:
		return nil, fmt.Errorf("unknown node type")
	}
}

// Root returns the root node
func (mpt *MPT) Root() Node {
	mpt.mu.RLock()
	defer mpt.mu.RUnlock()
	return mpt.root
}

// RootHash returns the hash of the root node
func (mpt *MPT) RootHash() []byte {
	mpt.mu.RLock()
	defer mpt.mu.RUnlock()

	return mpt.root.Hash()
}

// String returns a string representation of the MPT
func (mpt *MPT) String() string {
	mpt.mu.RLock()
	defer mpt.mu.RUnlock()

	return NodeToString(mpt.root, "")
}

// Clear clears the trie
func (mpt *MPT) Clear() {
	mpt.mu.Lock()
	defer mpt.mu.Unlock()

	mpt.root = &EmptyNode{}
	mpt.cache = make(map[string]Node)
}

// Size returns the number of key-value pairs in the trie
func (mpt *MPT) Size() int {
	mpt.mu.RLock()
	defer mpt.mu.RUnlock()

	return mpt.count(mpt.root)
}

// count recursively counts key-value pairs
func (mpt *MPT) count(node Node) int {
	if IsNil(node) {
		return 0
	}

	switch n := node.(type) {
	case *EmptyNode:
		return 0
	case *LeafNode:
		return 1
	case *ExtensionNode:
		return mpt.count(n.Next)
	case *BranchNode:
		count := 0
		if len(n.Value) > 0 {
			count = 1
		}
		for i := 0; i < 16; i++ {
			count += mpt.count(n.Children[i])
		}
		return count
	default:
		return 0
	}
}

// Iterator creates a new iterator for the MPT
func (mpt *MPT) Iterator() *MPTIterator {
	return &MPTIterator{
		mpt:   mpt,
		stack: make([]*nodeWithKey, 0, 32),
		path:  make([]byte, 0, 128),
	}
}

// nodeWithKey pairs a node with its path
type nodeWithKey struct {
	node Node
	path []byte
}

// MPTIterator iterates over the MPT
type MPTIterator struct {
	mpt   *MPT
	stack []*nodeWithKey
	path  []byte
}

// Next moves to the next key-value pair
func (it *MPTIterator) Next() (key, value []byte, err error) {
	for len(it.stack) > 0 {
		// Pop node from stack
		n := len(it.stack) - 1
		it.stack = it.stack[:n]
		current := it.stack[n]

		it.path = it.path[:len(it.path)-len(current.path)]
		it.path = append(it.path, current.path...)

		// Process node
		switch node := current.node.(type) {
		case *LeafNode:
			key, err := PathToKey(node.Key)
			if err != nil {
				return nil, nil, err
			}
			return key, node.Value, nil

		case *ExtensionNode:
			// Push next node onto stack
			it.stack = append(it.stack, &nodeWithKey{
				node: node.Next,
				path: node.Path,
			})
			continue

		case *BranchNode:
			// Push children onto stack (in reverse order)
			for i := 15; i >= 0; i-- {
				if node.Children[i] != nil {
					it.stack = append(it.stack, &nodeWithKey{
						node: node.Children[i],
						path: []byte{byte(i)},
					})
				}
			}
			if len(node.Value) > 0 {
				// Push value as leaf
				it.stack = append(it.stack, &nodeWithKey{
					node: &LeafNode{
						Key:   []byte{},
						Value: node.Value,
					},
					path: []byte{},
				})
			}
			continue

		default:
			return nil, nil, nil
		}
	}

	return nil, nil, nil
}

// Reset resets the iterator to the beginning
func (it *MPTIterator) Reset() {
	it.stack = make([]*nodeWithKey, 0, 32)
	it.path = make([]byte, 0, 128)
}
