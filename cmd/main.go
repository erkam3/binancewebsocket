package main

import (
	"nanoseccasestudy/relay"
	"nanoseccasestudy/top150"
	"time"
)

var CurrentTop150 map[string]chan bool

// main starts the WebSocket server and updates the top 150 every 10 minutes
func main() {
	CurrentTop150 = make(map[string]chan bool)
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	go relay.WebSocketServer(&CurrentTop150)
	updateTop150()
	for range ticker.C {
		updateTop150()
	}
}

// updateTop150 updates the top 150 symbols and closes channels for symbols that are no longer in the top 150
func updateTop150() {
	top150, err := top150.GetTop150()
	if err != nil {
		panic(err)
	}
	for symbol := range top150 {
		if CurrentTop150[symbol] == nil {
			CurrentTop150[symbol] = make(chan bool)
		}
	}
	for symbol := range CurrentTop150 {
		if !top150[symbol] {
			CurrentTop150[symbol] <- true
			delete(CurrentTop150, symbol)
		}
	}
}
