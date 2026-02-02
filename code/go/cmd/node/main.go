package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/jinba225/v6coin-protocol/pkg/blockchain"
	"github.com/jinba225/v6coin-protocol/pkg/consensus"
	"github.com/jinba225/v6coin-protocol/pkg/p2p"
	"github.com/jinba225/v6coin-protocol/pkg/rpc"
	"github.com/jinba225/v6coin-protocol/pkg/state"
	"github.com/jinba225/v6coin-protocol/pkg/tx"
	"github.com/jinba225/v6coin-protocol/pkg/wallet"
)

const (
	Version = "0.1.0"

	// Default configuration
	DefaultBindAddress   = "0.0.0.0"
	DefaultBindPort      = 38901
	DefaultDataDir       = "/tmp/v6coin"
	DefaultNetworkPrefix = "2001:db8::"
	DefaultMaxPeers      = 50

	// Genesis block
	GenesisHeight    = 0
	GenesisTimestamp = uint64(1704067200) // 2024-01-01 00:00:00 UTC
	GenesisValidator = "genesis"
)

var (
	node             *Node
	DefaultSeedPeers = []string{
		"2001:db8::1:1:38901",
	}
)

// Node represents a V6Coin full node
type Node struct {
	config  *Config
	context context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup

	// Core components
	chain           *blockchain.BlockChain
	consensusEngine *consensus.ConsensusEngine
	txPool          *tx.TxPool
	wallet          *wallet.Wallet
	p2pNetwork      *p2p.ConnectionManager
	rpcServer       *rpc.Server

	// State
	stateDB        state.StateDB
	forkHandler    *blockchain.ForkHandler
	blockValidator *blockchain.BlockValidator

	// Channels
	blockCh   chan *consensus.Block
	txCh      chan *consensus.Transaction
	syncReqCh chan *SyncRequest

	// Status
	isRunning   bool
	peerCount   int
	chainHeight uint64
	syncing     bool

	mu sync.RWMutex
}

// SyncRequest represents a sync request
type SyncRequest struct {
	PeerID      string
	StartHeight uint64
	EndHeight   uint64
}

// Config is node configuration
type Config struct {
	BindAddress   string
	BindPort      int
	DataDir       string
	NetworkPrefix string
	MaxPeers      int
	SeedPeers     []string
	WalletFile    string
	Mnemonic      string
	Password      string
	IsValidator   bool
	RPCHost       string
	RPCPort       int
}

// NewNode creates a new node instance
func NewNode(config *Config) (*Node, error) {
	ctx, cancel := context.WithCancel(context.Background())

	node := &Node{
		config:    config,
		context:   ctx,
		cancel:    cancel,
		blockCh:   make(chan *consensus.Block, 100),
		txCh:      make(chan *consensus.Transaction, 1000),
		syncReqCh: make(chan *SyncRequest, 10),
		isRunning: false,
		syncing:   false,
	}

	// Initialize state database (in-memory for now)
	node.stateDB = state.NewMemoryStateDB()

	// Create blockchain
	genesisBlock := createGenesisBlock()
	blockchainConfig := &blockchain.BlockChainConfig{
		GenesisBlock: genesisBlock,
		StateDB:      node.stateDB,
		Validator:    blockchain.NewBlockValidator(),
	}

	var err error
	node.chain, err = blockchain.NewBlockChain(blockchainConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create blockchain: %w", err)
	}

	// Create fork handler
	node.forkHandler = blockchain.NewForkHandler(node.chain)

	// Create consensus engine
	node.consensusEngine = consensus.NewConsensusEngine()

	// Create transaction pool
	node.txPool = tx.NewTxPool(node.stateDB)

	// Create wallet
	if config.Mnemonic != "" || config.WalletFile != "" {
		walletConfig := &wallet.WalletConfig{
			DataDir:       config.DataDir,
			Mnemonic:      config.Mnemonic,
			Password:      config.Password,
			NetworkPrefix: config.NetworkPrefix,
		}
		node.wallet, err = wallet.NewWallet(walletConfig)
		if err != nil {
			log.Printf("Warning: failed to create wallet: %v", err)
		}
	}

	// Create P2P network
	maxConnections := config.MaxPeers
	if maxConnections == 0 {
		maxConnections = DefaultMaxPeers
	}
	node.p2pNetwork = p2p.NewConnectionManager(maxConnections)

	return node, nil
}

// Start starts the node
func (n *Node) Start() error {
	n.mu.Lock()
	if n.isRunning {
		n.mu.Unlock()
		return fmt.Errorf("node is already running")
	}
	n.isRunning = true
	n.mu.Unlock()

	log.Printf("Starting V6Coin Node v%s", Version)
	log.Printf("Network prefix: %s", n.config.NetworkPrefix)
	log.Printf("Bind address: %s:%d", n.config.BindAddress, n.config.BindPort)
	log.Printf("Data directory: %s", n.config.DataDir)

	// Start background workers
	n.wg.Add(1)
	go n.blockWorker()

	n.wg.Add(1)
	go n.transactionWorker()

	n.wg.Add(1)
	go n.syncWorker()

	n.wg.Add(1)
	go n.consensusWorker()

	// Start peer discovery
	if len(n.config.SeedPeers) > 0 {
		n.wg.Add(1)
		go n.peerDiscovery()
	}

	// Update status
	n.updateStatus()

	n.wg.Add(1)
	go func() {
		defer n.wg.Done()
		rpcServer := rpc.NewServer(&rpc.Config{
			Host: n.config.RPCHost,
			Port: n.config.RPCPort,
		})
		if err := rpcServer.Start(n.context); err != nil {
			log.Printf("RPC server error: %v", err)
		}
	}()

	log.Println("V6Coin node started successfully")
	return nil
}

// Stop stops the node
func (n *Node) Stop() {
	n.mu.Lock()
	if !n.isRunning {
		n.mu.Unlock()
		return
	}
	n.isRunning = false
	n.mu.Unlock()

	log.Println("Stopping V6Coin node...")

	// Cancel context
	n.cancel()

	// Wait for workers to finish
	n.wg.Wait()

	// Close channels
	close(n.blockCh)
	close(n.txCh)
	close(n.syncReqCh)

	log.Println("V6Coin node stopped")
}

// blockWorker processes incoming blocks
func (n *Node) blockWorker() {
	defer n.wg.Done()

	for {
		select {
		case <-n.context.Done():
			return
		case block := <-n.blockCh:
			if err := n.processBlock(block); err != nil {
				log.Printf("Error processing block: %v", err)
			}
		}
	}
}

// transactionWorker processes incoming transactions
func (n *Node) transactionWorker() {
	defer n.wg.Done()

	for {
		select {
		case <-n.context.Done():
			return
		case tx := <-n.txCh:
			if err := n.processTransaction(tx); err != nil {
				log.Printf("Error processing transaction: %v", err)
			}
		}
	}
}

// syncWorker handles blockchain synchronization
func (n *Node) syncWorker() {
	defer n.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-n.context.Done():
			return
		case <-ticker.C:
			if err := n.syncChain(); err != nil {
				log.Printf("Error syncing chain: %v", err)
			}
		case req := <-n.syncReqCh:
			if err := n.handleSyncRequest(req); err != nil {
				log.Printf("Error handling sync request: %v", err)
			}
		}
	}
}

// consensusWorker runs consensus logic
func (n *Node) consensusWorker() {
	defer n.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-n.context.Done():
			return
		case <-ticker.C:
			if n.config.IsValidator && !n.syncing {
				if err := n.proposeBlock(); err != nil {
					log.Printf("Error proposing block: %v", err)
				}
			}
		}
	}
}

// peerDiscovery discovers and connects to peers
func (n *Node) peerDiscovery() {
	defer n.wg.Done()

	for _, seed := range n.config.SeedPeers {
		addr, err := net.ResolveTCPAddr("tcp", seed)
		if err != nil {
			log.Printf("Failed to resolve seed node %s: %v", seed, err)
			continue
		}

		peerID := addr.String()
		log.Printf("Connecting to seed node: %s (peer ID: %s)", seed, peerID)
		// TODO: Implement actual connection
	}

	// Periodically discover peers
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-n.context.Done():
			return
		case <-ticker.C:
			n.discoverPeers()
		}
	}
}

// processBlock processes an incoming block
func (n *Node) processBlock(block *consensus.Block) error {
	log.Printf("Processing block at height %d", block.Header.Height)

	// Check for fork
	fork, err := n.forkHandler.CheckForFork(block)
	if err != nil {
		return fmt.Errorf("fork check failed: %w", err)
	}

	// If fork detected, resolve it
	if fork != nil {
		log.Printf("Fork detected at height %d, resolving...", block.Header.Height)
		if err := n.forkHandler.ResolveFork(fork); err != nil {
			return fmt.Errorf("fork resolution failed: %w", err)
		}
		log.Printf("Fork resolved, new chain head at height %d", n.chain.GetChainHead().Header.Height)
	}

	// Add block to chain
	if err := n.chain.AddBlock(block); err != nil {
		return fmt.Errorf("failed to add block: %w", err)
	}

	log.Printf("Block added to chain at height %d", block.Header.Height)

	// Update consensus engine
	n.consensusEngine.UpdateContribution(block.Header.ValidatorID, func(c *consensus.NodeContribution) {
		c.OnlineTime += time.Second * 10
	})

	// Update status
	n.updateStatus()

	return nil
}

// processTransaction processes an incoming transaction
func (n *Node) processTransaction(tx *consensus.Transaction) error {
	log.Printf("Processing transaction from %s to %s", tx.From, tx.To)

	// Add to transaction pool
	if err := n.txPool.AddTransaction(tx); err != nil {
		return fmt.Errorf("failed to add transaction to pool: %w", err)
	}

	return nil
}

// proposeBlock proposes a new block (validator only)
func (n *Node) proposeBlock() error {
	currentHeight := n.chain.GetCurrentHeight()
	blockHeight := currentHeight + 1

	log.Printf("Proposing block at height %d", blockHeight)

	// Get pending transactions
	pendingTxs := n.txPool.GetPendingTransactions(100)

	// Calculate block reward
	blockReward := consensus.CalculateBlockReward(blockHeight)

	// Get chain head
	chainHead := n.chain.GetChainHead()

	// Create block
	block := consensus.NewBlock(blockHeight, chainHead.Hash(), pendingTxs)

	log.Printf("Block proposed: height=%d, txs=%d, reward=%d", blockHeight, len(pendingTxs), blockReward)

	// Process our own block
	if err := n.processBlock(block); err != nil {
		return fmt.Errorf("failed to process proposed block: %w", err)
	}

	return nil
}

// syncChain synchronizes the blockchain
func (n *Node) syncChain() error {
	n.mu.RLock()
	currentHeight := n.chain.GetCurrentHeight()
	peerCount := n.peerCount
	n.mu.RUnlock()

	// Find best peer to sync from
	peers := n.p2pNetwork.GetPeers()
	if len(peers) == 0 {
		return nil
	}

	// In a real implementation, we would query peers for their heights
	// and sync from the best peer
	log.Printf("Syncing chain at height %d with %d peers", currentHeight, peerCount)

	return nil
}

// handleSyncRequest handles a sync request from a peer
func (n *Node) handleSyncRequest(req *SyncRequest) error {
	log.Printf("Handling sync request from peer %s for blocks %d-%d", req.PeerID, req.StartHeight, req.EndHeight)

	// In a real implementation, we would send requested blocks
	// to the peer
	return nil
}

// discoverPeers discovers new peers
func (n *Node) discoverPeers() {
	// In a real implementation, we would query peers for their peer lists
	// and try to connect to new peers
	peers := n.p2pNetwork.GetPeers()
	log.Printf("Discovering peers, currently connected to %d peers", len(peers))
}

// updateStatus updates node status
func (n *Node) updateStatus() {
	n.mu.Lock()
	n.chainHeight = n.chain.GetCurrentHeight()
	peerIDs := n.p2pNetwork.GetConnectedPeers()
	n.peerCount = len(peerIDs)
	n.mu.Unlock()
}

// GetStatus returns current node status
func (n *Node) GetStatus() map[string]interface{} {
	n.mu.RLock()
	defer n.mu.RUnlock()

	status := map[string]interface{}{
		"version":       Version,
		"running":       n.isRunning,
		"chainHeight":   n.chainHeight,
		"peerCount":     n.peerCount,
		"syncing":       n.syncing,
		"isValidator":   n.config.IsValidator,
		"bindAddress":   n.config.BindAddress,
		"bindPort":      n.config.BindPort,
		"networkPrefix": n.config.NetworkPrefix,
	}

	if n.wallet != nil {
		accountCount := n.wallet.GetAccountCount()
		status["accountCount"] = accountCount
	}

	return status
}

// createGenesisBlock creates a genesis block
func createGenesisBlock() *consensus.Block {
	return &consensus.Block{
		Header: &consensus.BlockHeader{
			Version:       0x0100,
			Height:        GenesisHeight,
			PrevBlockHash: []byte{},
			MerkleRoot:    []byte{},
			Timestamp:     GenesisTimestamp,
			ValidatorID:   GenesisValidator,
			StateRoot:     []byte{},
			Signature:     []byte{},
		},
		Transactions:  []*consensus.Transaction{},
		ValidatorSigs: [][]byte{},
	}
}

func main() {
	var (
		showVersion   bool
		command       string
		bindAddress   string
		bindPort      int
		dataDir       string
		networkPrefix string
		mnemonic      string
		password      string
		isValidator   bool
		rpcHost       string
		rpcPort       int
	)

	flag.BoolVar(&showVersion, "version", false, "Show version information")
	flag.StringVar(&bindAddress, "bind", DefaultBindAddress, "Bind address")
	flag.IntVar(&bindPort, "port", DefaultBindPort, "Bind port (default 38901)")
	flag.StringVar(&dataDir, "datadir", DefaultDataDir, "Data directory")
	flag.StringVar(&networkPrefix, "network", DefaultNetworkPrefix, "Network prefix")
	flag.StringVar(&mnemonic, "mnemonic", "", "Recover wallet from mnemonic")
	flag.StringVar(&password, "password", "", "Wallet password")
	flag.BoolVar(&isValidator, "validator", false, "Run as validator")
	flag.StringVar(&rpcHost, "rpc-host", "0.0.0.0", "RPC server bind address")
	flag.IntVar(&rpcPort, "rpc-port", 9090, "RPC server port")
	flag.Parse()

	args := flag.Args()
	if len(args) > 0 {
		command = args[0]
	}

	if showVersion {
		fmt.Printf("V6Coin Node v%s\n", Version)
		return
	}

	// Create configuration
	config := &Config{
		BindAddress:   bindAddress,
		BindPort:      bindPort,
		DataDir:       dataDir,
		NetworkPrefix: networkPrefix,
		MaxPeers:      DefaultMaxPeers,
		SeedPeers:     DefaultSeedPeers,
		Mnemonic:      mnemonic,
		Password:      password,
		IsValidator:   isValidator,
		RPCHost:       rpcHost,
		RPCPort:       rpcPort,
	}

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Create node
	var err error
	node, err = NewNode(config)
	if err != nil {
		log.Fatalf("Failed to create node: %v", err)
	}

	go func() {
		sig := <-sigCh
		log.Printf("Received signal: %v", sig)
		node.Stop()
		os.Exit(0)
	}()

	// Handle commands
	switch command {
	case "start":
		if err := node.Start(); err != nil {
			log.Fatalf("Failed to start node: %v", err)
		}

		// Keep running
		node.wg.Wait()

	case "stop":
		if node != nil {
			node.Stop()
		} else {
			log.Fatal("Node is not running")
		}

	case "status":
		if node == nil {
			// Try to load from config
			node, _ = NewNode(config)
		}

		status := node.GetStatus()
		fmt.Println("V6Coin Node Status:")
		for k, v := range status {
			fmt.Printf("  %s: %v\n", k, v)
		}

	default:
		fmt.Printf("V6Coin Node v%s\n", Version)
		fmt.Println("\nUsage: node [command] [options]")
		fmt.Println("\nCommands:")
		fmt.Println("  start    Start node")
		fmt.Println("  stop     Stop node")
		fmt.Println("  status   Show node status")
		fmt.Println("\nOptions:")
		flag.PrintDefaults()
		os.Exit(1)
	}
}
