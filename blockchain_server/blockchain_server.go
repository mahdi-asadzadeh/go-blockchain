package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/mahdi-asadzadeh/go-blockchain/block"
	"github.com/mahdi-asadzadeh/go-blockchain/utils"
	"github.com/mahdi-asadzadeh/go-blockchain/wallet"
)

var cache map[string]*block.Blockchain = make(map[string]*block.Blockchain)

type BlockchainServer struct {
	port uint16
}

func NewBlockchainServer(port uint16) *BlockchainServer {
	return &BlockchainServer{port}
}

func (bcs *BlockchainServer) Port() uint16 {
	return bcs.port
}

func (bcs *BlockchainServer) GetBlockchain() *block.Blockchain {
	bc, ok := cache["blockchain"]
	if !ok {
		minersWallet := wallet.NewWallet()
		bc := block.NewBlockchain(minersWallet.BlockchainAddress(), bcs.Port())
		cache["blockchain"] = bc
		log.Printf("private_key %s", minersWallet.PrivateKeyStr())
		log.Printf("public_key %s", minersWallet.PublicKeyStr())
		log.Printf("blockchain_address %s", minersWallet.BlockchainAddress())
	}
	return bc
}

func (bcs *BlockchainServer) GetChain(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		w.Header().Add("Content-Type", "application/json")
		bc := bcs.GetBlockchain()
		m, _ := bc.MarshalJSON()
		io.WriteString(w, string(m[:]))
	default:
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (bcs *BlockchainServer) Transactions(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		w.Header().Add("Content-Type", "application/json")
		bc := bcs.GetBlockchain()
		transactions := bc.TransactionPool()
		m, _ := json.Marshal(struct {
			Transactions []*block.Transactions `json:"transactions"`
			Length       int                   `json:"length"`
		}{
			Transactions: transactions,
			Length:       len(transactions),
		})
		io.WriteString(w, string(m[:]))

	case http.MethodPost:
		decoder := json.NewDecoder(req.Body)
		var t block.TransactionRequest
		err := decoder.Decode(&t)
		if err != nil {
			log.Printf("ERROR: %v", err)
			io.WriteString(w, "fail")
			return
		}
		if !t.Validate() {
			log.Println("ERROR: missing field(s)")
			io.WriteString(w, "fail")
			return
		}
		publicKey := utils.PublicKeyFromString(*t.SenderPublicKey)
		signature := utils.SignatureFromString(*t.Signature)
		bc := bcs.GetBlockchain()
		isCreated := bc.CreateTransaction(*t.SenderBlockchainAddress,
			*t.RecipientBlockchainAddress, *t.Value, publicKey, signature)

		w.Header().Add("Content-Type", "application/json")
		var m string
		if !isCreated {
			w.WriteHeader(http.StatusBadRequest)
			m = "fail"
		} else {
			w.WriteHeader(http.StatusCreated)
			m = "success"
		}
		io.WriteString(w, m)
	default:
		log.Println("ERROR: Invalid HTTP Method")
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (bcs *BlockchainServer) StartMine(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		bcs.GetBlockchain().StartMining()

		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "Success")
	default:
		w.WriteHeader(http.StatusBadRequest)

	}
}

func (bcs *BlockchainServer) Amount(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		blocchain_address := req.URL.Query().Get("address")
		amount := bcs.GetBlockchain().CalculateTotalAmount(blocchain_address)
		fmt.Println(blocchain_address)
		amountRes := block.AmountResponse{Amount: amount}
		fmt.Println(amountRes)

		m, _ := json.Marshal(amountRes)
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, string(m[:]))
	default:
		io.WriteString(w, "Invalid request method!")
	}

}

func (bcs *BlockchainServer) Run() {
	http.HandleFunc("/", bcs.GetChain)
	http.HandleFunc("/amount", bcs.Amount)
	http.HandleFunc("/mine/start", bcs.StartMine)
	http.HandleFunc("/transactions", bcs.Transactions)
	if err := http.ListenAndServe(":"+strconv.Itoa(int(bcs.Port())), nil); err != nil {
		log.Fatal(err)
	}
}

func main() {
	port := flag.Uint("port", 5000, "TCP Port Number for Blockchain Server")
	flag.Parse()
	app := NewBlockchainServer(uint16(*port))
	app.Run()
}
