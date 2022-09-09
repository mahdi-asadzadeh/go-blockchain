package block

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/mahdi-asadzadeh/go-blockchain/utils"
)

const (
	MINING_DIFFICULTY = 3
	MINING_SENDER     = "THE BLOCKCHAIN"
	MINING_REWARD     = 1.0
	MINING_TIMER_SEC  = 20
)

type Block struct {
	nonce        int
	previousHash [32]byte
	timestamp    int64
	transactions []*Transactions
}

func (b *Block) Hash() [32]byte {
	hb, _ := json.Marshal(b)
	result := sha256.Sum256([]byte(hb))
	return result
}

func (b *Block) MarshalJSON() ([]byte, error) {
	type jsBlock struct {
		Timestamp    int64           `json:"timestamp"`
		Nonce        int             `json:"nonce"`
		PreviousHash [32]byte        `json:"previous_hash"`
		Transactions []*Transactions `json:"transactions"`
	}
	return json.Marshal(
		jsBlock{
			Timestamp:    b.timestamp,
			Nonce:        b.nonce,
			PreviousHash: b.previousHash,
			Transactions: b.transactions,
		},
	)
}

func (b *Block) Print() {
	fmt.Printf("timestamp 		%d\n", b.timestamp)
	fmt.Printf("nonce 			%d\n", b.nonce)
	fmt.Printf("previous hash 	%x\n", b.previousHash)
	for _, t := range b.transactions {
		t.Print()
	}
}

func NewBlock(nonce int, previousHash [32]byte, transactions []*Transactions) *Block {
	b := new(Block)
	b.timestamp = time.Now().UnixNano()
	b.nonce = nonce
	b.previousHash = previousHash
	b.transactions = transactions
	return b
}

type Blockchain struct {
	transactionPool   []*Transactions
	chain             []*Block
	blockchainAddress string
	port              uint16
	mux               sync.Mutex
}

func (bc *Blockchain) Print() {
	for i, block := range bc.chain {
		fmt.Printf("%s Chain %d %s\n", strings.Repeat("=", 25), i,
			strings.Repeat("=", 25))
		block.Print()
	}
	fmt.Printf("%s\n", strings.Repeat("*", 25))
	fmt.Printf("%s\n", strings.Repeat("*", 25))
	fmt.Printf("%s\n", strings.Repeat("*", 25))
}

func NewBlockchain(blockchainAddress string, port uint16) *Blockchain {
	b := &Block{}
	bc := new(Blockchain)
	bc.blockchainAddress = blockchainAddress
	bc.port = port
	bc.CreateBlock(0, b.Hash())
	return bc
}

func (bc *Blockchain) AddTransaction(sender string, recipient string, value float32, senderPublicKey *ecdsa.PublicKey, s *utils.Signature) bool {
	t := NewTransaction(sender, recipient, value)
	if sender == MINING_SENDER {
		bc.transactionPool = append(bc.transactionPool, t)
		return true
	}
	if bc.VerifyTransactionSignature(senderPublicKey, s, t) {
		// if bc.CalculateTotalAmount(sender) < value {
		// 	log.Println("ERROR: Not enough balance in a wallet")
		// 	return false
		// }
		bc.transactionPool = append(bc.transactionPool, t)
		return true
	} else {
		log.Println("ERROR: Verify Transaction")
	}
	return false
}

func (bc *Blockchain) LastBlock() *Block {
	return bc.chain[len(bc.chain)-1]
}

func (bc *Blockchain) CreateBlock(nonce int, previousHash [32]byte) *Block {
	b := NewBlock(nonce, previousHash, bc.transactionPool)
	bc.chain = append(bc.chain, b)
	bc.transactionPool = []*Transactions{}
	return b
}

type Transactions struct {
	senderBlockchainAddress    string
	recipientBlockchainAddress string
	value                      float32
}

func NewTransaction(sender string, recipient string, value float32) *Transactions {
	return &Transactions{sender, recipient, value}
}

func (t *Transactions) Print() {
	fmt.Printf("%s\n", strings.Repeat("-", 40))
	fmt.Printf(" sender_blockchain_address      %s\n", t.senderBlockchainAddress)
	fmt.Printf(" recipient_blockchain_address   %s\n", t.recipientBlockchainAddress)
	fmt.Printf(" value                          %.1f\n", t.value)
}

func (t *Transactions) MarshalJSON() ([]byte, error) {
	type jsTrans struct {
		Sender    string  `json:"sender_blockchain_address"`
		Recipient string  `json:"recipient_blockchain_address"`
		Value     float32 `json:"value"`
	}
	return json.Marshal(
		jsTrans{
			Sender:    t.senderBlockchainAddress,
			Recipient: t.recipientBlockchainAddress,
			Value:     t.value,
		},
	)
}

func (bc *Blockchain) CreateTransaction(sender string, recipient string, value float32, senderPublicKey *ecdsa.PublicKey, s *utils.Signature) bool {
	isTransacted := bc.AddTransaction(sender, recipient, value, senderPublicKey, s)
	return isTransacted
}

func (bc *Blockchain) CopyTransactionPool() []*Transactions {
	transactions := make([]*Transactions, 0)
	for _, t := range bc.transactionPool {
		transactions = append(transactions, NewTransaction(t.senderBlockchainAddress,
			t.recipientBlockchainAddress,
			t.value))
	}

	return transactions
}

func (bc *Blockchain) ValidProof(nonce int, previousHash [32]byte, transactions []*Transactions, difficulty int) bool {
	zeros := strings.Repeat("0", difficulty)
	guessBlock := Block{nonce, previousHash, 0, transactions}
	guessHashStr := fmt.Sprintf("%x", guessBlock.Hash())
	return guessHashStr[:difficulty] == zeros
}

func (bc *Blockchain) ProofOfWork() int {
	transactions := bc.CopyTransactionPool()
	previousHash := bc.LastBlock().Hash()
	nonce := 0
	for !bc.ValidProof(nonce, previousHash, transactions, MINING_DIFFICULTY) {
		nonce += 1
	}
	return nonce
}

func (bc *Blockchain) Mining() bool {
	bc.mux.Lock()
	defer bc.mux.Unlock()
	if len(bc.transactionPool) == 0 {
		return false
	}
	bc.AddTransaction(MINING_SENDER, bc.blockchainAddress, MINING_REWARD, nil, nil)
	nonce := bc.ProofOfWork()
	previousHash := bc.LastBlock().Hash()
	bc.CreateBlock(nonce, previousHash)
	return true
}

func (bc *Blockchain) StartMining() {
	bc.Mining()
	_ = time.AfterFunc(time.Second*MINING_TIMER_SEC, bc.StartMining)
}

func (bc *Blockchain) MarshalJSON() ([]byte, error) {
	type jsonBChain struct {
		Blocks []*Block `json:"chains"`
	}
	return json.Marshal(jsonBChain{
		Blocks: bc.chain,
	})
}

func (bc *Blockchain) CalculateTotalAmount(blockchainAddress string) float32 {
	var totalAmount float32 = 0.0
	for _, b := range bc.chain {
		for _, t := range b.transactions {
			value := t.value

			if blockchainAddress == t.recipientBlockchainAddress {
				totalAmount += value
			}

			if blockchainAddress == t.senderBlockchainAddress {
				totalAmount -= value
			}
		}
	}
	return totalAmount
}

func (bc *Blockchain) VerifyTransactionSignature(senderPublicKey *ecdsa.PublicKey, s *utils.Signature, t *Transactions) bool {
	m, _ := json.Marshal(t)
	h := sha256.Sum256([]byte(m))
	return ecdsa.Verify(senderPublicKey, h[:], s.R, s.S)
}

func (bc *Blockchain) TransactionPool() []*Transactions {
	return bc.transactionPool
}

type AmountResponse struct {
	Amount float32 `json:"amount"`
}

type TransactionRequest struct {
	SenderBlockchainAddress    *string  `json:"sender_blockchain_address"`
	RecipientBlockchainAddress *string  `json:"recipient_blockchain_address"`
	SenderPublicKey            *string  `json:"sender_public_key"`
	Value                      *float32 `json:"value"`
	Signature                  *string  `json:"signature"`
}

func (tr *TransactionRequest) Validate() bool {
	if tr.SenderBlockchainAddress == nil ||
		tr.RecipientBlockchainAddress == nil ||
		tr.SenderPublicKey == nil ||
		tr.Value == nil ||
		tr.Signature == nil {
		return false
	}
	return true
}
