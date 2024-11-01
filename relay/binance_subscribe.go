package relay

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/gorilla/websocket"
)

type Message struct {
	Symbol  string `json:"s"`
	BestBid string `json:"b"`
	BestAsk string `json:"a"`
}

type subscribeMessage struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
}

// tryReconnect closes the connection and attempts to reconnect to the WebSocket
// Arguments:
//
//	conn: the WebSocket connection to close
//	url: the URL to reconnect to
//
// Returns:
//
//	*websocket.Conn: the reconnected WebSocket connection
func tryReconnect(conn *websocket.Conn, url string) *websocket.Conn {
	conn.Close()
	newConn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Fatal("Error reconnecting to WebSocket:", err)
	}
	return newConn
}

// SubscribeToBinanceWebSocket subscribes to a Binance WebSocket for a given symbol
// Arguments:
//
//	symbol: the symbol to subscribe to
//	quit: a channel to signal when to unsubscribe
func SubscribeToBinanceWebSocket(symbol string, quit chan bool) {
	var LastMessage Message
	lowerCaseSymbol := strings.ToLower(symbol)
	url := fmt.Sprintf("wss://fstream.binance.com/ws/%s@bookTicker", lowerCaseSymbol)
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Fatal("Error connecting to WebSocket:", err)
	}
	defer conn.Close()
	subscribe, err := json.Marshal(subscribeMessage{
		Method: "SUBSCRIBE",
		Params: []string{symbol + "@bookTicker"},
	})
	if err != nil {
		log.Fatal("Error marshalling subscribe message:", err)
	}
	conn.WriteMessage(websocket.TextMessage, subscribe)
	log.Println("Subscribed to", symbol)

	for {
		select {
		case <-quit:
			ClosePair(symbol)
			log.Println("Unsubscribed from", symbol)
			return
		default:
			messageType, p, err := conn.ReadMessage()
			if err != nil {
				log.Println("Error reading message:", err)
				conn = tryReconnect(conn, url)
				continue
			}
			if messageType == websocket.PingMessage {
				if err := conn.WriteMessage(websocket.PongMessage, p); err != nil {
					log.Println("Error writing pong message:", err)
				}
				continue
			}
			var message Message
			if err := json.Unmarshal(p, &message); err != nil {
				log.Println("Error unmarshalling message:", err)
				continue
			}
			if message == LastMessage {
				continue
			}
			go PushMessage(symbol, message)
		}
	}

}
