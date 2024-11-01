package relay

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type Client struct {
	Conn          *websocket.Conn
	Subscriptions map[string]bool
	Lock          sync.Mutex
}

type ClientMessage struct {
	Method string   `json:"method"`
	Pairs  []string `json:"pairs"`
}

var subscribedPairs *map[string]chan bool
var subscribedClients map[string]map[*Client]bool

// PushMessage sends a message to all clients subscribed to a given symbol
// Arguments:
//
//	symbol: the symbol to send the message for
//	message: the message to send
func PushMessage(symbol string, message interface{}) {
	messageJSON, err := json.Marshal(message)
	if err != nil {
		log.Println("Error marshalling binance message:", err)
		return
	}
	for client := range subscribedClients[symbol] {
		client.Lock.Lock()
		client.Conn.WriteMessage(websocket.TextMessage, messageJSON)
		client.Lock.Unlock()
	}
}

// ClosePair closes the connection for all clients subscribed to a given symbol
// Arguments:
//
//	symbol: the symbol to close the connection for
func ClosePair(symbol string) {
	for client := range subscribedClients[symbol] {
		client.UnsubscribeFromPair(symbol, true)
	}
	delete(subscribedClients, symbol)
}

// SubscribeToPair subscribes a client to a given symbol if the symbol is in the top 150.
// If the symbol has not been subscribed to before, a new goroutine is started to subscribe to the Binance WebSocket.
// Arguments:
//
//	pair: the symbol to subscribe to
//
// Returns:
//
//	bool: true if the symbol has been subscribed to, false otherwise
func (client *Client) SubscribeToPair(pair string) bool {
	if (*subscribedPairs)[pair] == nil {
		client.Conn.WriteMessage(websocket.TextMessage, []byte("Invalid pair: "+pair))
		delete(*subscribedPairs, pair)
		return false
	}
	if subscribedClients[pair] == nil {
		subscribedClients[pair] = make(map[*Client]bool)
		go SubscribeToBinanceWebSocket(pair, (*subscribedPairs)[pair])
	}
	subscribedClients[pair][client] = true
	return true
}

// UnsubscribeFromPair unsubscribes a client from a given symbol.
// If the unsubscription is caused by the symbol no longer being in the top 150,
// a message is sent to the client.
// If the client is the last client subscribed to the symbol, the goroutine for the Binance WebSocket is stopped.
// Arguments:
//
//	pair: the symbol to unsubscribe from
//	isAuto: true if the client is unsubscribing automatically, false otherwise
func (client *Client) UnsubscribeFromPair(pair string, isAuto bool) {
	delete(client.Subscriptions, pair)
	delete(subscribedClients[pair], client)
	if len(subscribedClients[pair]) == 0 {
		delete(subscribedClients, pair)
		(*subscribedPairs)[pair] <- true
	}
	if isAuto {
		client.Conn.WriteMessage(websocket.TextMessage, []byte("Unsubscribed from "+pair+" since pair is no longer in top 150"))
	}
}

// handleConnections handles incoming WebSocket connections
func handleConnections(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading to WebSocket:", err)
		return
	}
	client := &Client{
		Conn:          conn,
		Subscriptions: make(map[string]bool),
	}
	go handleClient(client)
}

// handleClient handles incoming messages from a client
func handleClient(client *Client) {
	log.Println("Client connected")
	defer func() {
		client.Conn.Close()
		for pair := range client.Subscriptions {
			client.UnsubscribeFromPair(pair, false)
		}
		log.Println("Client disconnected")
	}()
	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			log.Println("Error reading message:", err)
			break
		}
		var clientMessage ClientMessage
		err = json.Unmarshal(message, &clientMessage)
		if err != nil {
			client.Conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprint("Error unmarshalling message:", err)))
			continue
		}
		if clientMessage.Method == "SUBSCRIBE" {
			for _, pair := range clientMessage.Pairs {
				if client.Subscriptions[pair] {
					delete(client.Subscriptions, pair)
				} else {
					delete(client.Subscriptions, pair)
					if !client.SubscribeToPair(pair) {
						client.Conn.WriteMessage(websocket.TextMessage, []byte("Invalid pair: "+pair))
					}
				}
			}
			for pair := range client.Subscriptions {
				client.UnsubscribeFromPair(pair, false)
				delete(client.Subscriptions, pair)
			}
			for _, pair := range clientMessage.Pairs {
				client.Subscriptions[pair] = true
			}
		}
	}
}

// WebSocketServer is a goroutine that starts a WebSocket server on port 8080
func WebSocketServer(CurrentTop150 *map[string]chan bool) {
	subscribedClients = make(map[string]map[*Client]bool)
	subscribedPairs = CurrentTop150
	http.HandleFunc("/ws", handleConnections)
	log.Println("WebSocket server started on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
