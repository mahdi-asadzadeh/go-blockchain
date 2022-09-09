package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"path"
	"strconv"

	"github.com/mahdi-asadzadeh/go-blockchain/block"
	"github.com/mahdi-asadzadeh/go-blockchain/utils"
	"github.com/mahdi-asadzadeh/go-blockchain/wallet"
)

const tempDir = "wallet_server/templates"

type WalletServer struct {
	port    uint16
	gateway string
}

func (wal *WalletServer) Port() uint16 {
	return wal.port
}

func (wal *WalletServer) Gateway() string {
	return wal.gateway
}

func NewWalletServer(port uint16, gateway string) *WalletServer {
	return &WalletServer{port: port, gateway: gateway}
}

type Student struct {
	Id   int
	Name string //exported field since it begins with a capital letter
}

func (wal *WalletServer) Index(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		tempAddress := path.Join("templates", "index.html")
		fmt.Println(tempAddress)
		templates, err := template.ParseFiles(tempAddress)
		fmt.Println(err)
		templates.Execute(w, "")
	default:
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (wal *WalletServer) Wallet(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPost:
		w.Header().Add("Content-Type", "application/json")
		wall := wallet.NewWallet()
		m, _ := wall.MarshalJSON()
		io.WriteString(w, string(m[:]))
	default:
		w.WriteHeader(http.StatusBadRequest)
	}
}
func (ws *WalletServer) CreateTransaction(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPost:
		decoder := json.NewDecoder(req.Body)
		var t wallet.TransactionRequest
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
		privateKey := utils.PrivateKeyFromString(*t.SenderPrivateKey, publicKey)
		value, err := strconv.ParseFloat(*t.Value, 32)
		if err != nil {
			log.Println("ERROR: parse error")
			io.WriteString(w, "fail")
			return
		}
		value32 := float32(value)

		w.Header().Add("Content-Type", "application/json")

		transaction := wallet.NewTransaction(privateKey, publicKey,
			*t.SenderBlockchainAddress, *t.RecipientBlockchainAddress, value32)
		signature := transaction.GenerateSignature()
		signatureStr := signature.String()

		bt := &block.TransactionRequest{
			t.SenderBlockchainAddress,
			t.RecipientBlockchainAddress,
			t.SenderPublicKey,
			&value32, &signatureStr,
		}
		m, _ := json.Marshal(bt)
		buf := bytes.NewBuffer(m)

		resp, _ := http.Post(ws.Gateway()+"/transactions", "application/json", buf)
		if resp.StatusCode == 201 {
			io.WriteString(w, "success")
			return
		}
		io.WriteString(w, "fail")
	default:
		w.WriteHeader(http.StatusBadRequest)
		log.Println("ERROR: Invalid HTTP Method")
	}
}

func (wal *WalletServer) Amount(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		blockchain_address := req.URL.Query().Get("address")

		newReq, _ := http.NewRequest("GET", wal.Gateway()+"/amount", nil)
		q := newReq.URL.Query()
		q.Add("address", blockchain_address)
		newReq.URL.RawQuery = q.Encode()

		client := http.Client{}
		res, err := client.Do(newReq)
		if err != nil {
			io.WriteString(w, err.Error())
			return
		}
		if res.StatusCode == 200 {
			var resAmount wallet.AmountRsponse
			decoder := json.NewDecoder(res.Body)
			err = decoder.Decode(&resAmount)
			if err != nil {
				io.WriteString(w, err.Error())
				return
			}
			m, _ := json.Marshal(struct {
				Amount float32 `json:"amount"`
			}{
				Amount: resAmount.Amount,
			})

			io.WriteString(w, string(m[:]))
		} else {
			io.WriteString(w, string(res.StatusCode))
		}
	default:
		io.WriteString(w, "Invalid request method!")
	}
}

func (wal *WalletServer) Run() {
	http.HandleFunc("/", wal.Index)
	http.HandleFunc("/amount", wal.Amount)
	http.HandleFunc("/wallet", wal.Wallet)
	http.HandleFunc("/transaction", wal.CreateTransaction)
	if err := http.ListenAndServe(":"+strconv.Itoa(int(wal.Port())), nil); err != nil {
		log.Fatal(err)
	}
}

func main() {
	port := flag.Uint("port", 5000, "TCP Port Number for Blockchain Server")
	gateway := flag.String("gateway", "http://localhost:8081", "Blockchain Gateway")
	flag.Parse()

	app := NewWalletServer(uint16(*port), *gateway)
	app.Run()

}
