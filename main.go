package main

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

type ConsoleMessage struct {
	Message string
}

func (mm *ConsoleMessage) PrintMessage() string {
	return mm.Message
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // DEVELOLPMENT ONLY CHANGE FOR PRODUCTION
	},
}

func serveHTML(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/index.html")
}

// Sends any messages queued to be displayed to control console log
func messageHandler(conChan <-chan ConsoleMessage) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Error upgrading HTTP connection to websocket:", err)
		}
		defer conn.Close()

		log.Println("Message Handler ready to forward messages to console")

		mm := <-conChan

		for {
			if err := conn.WriteMessage(websocket.TextMessage, []byte(mm.PrintMessage())); err != nil {
				log.Println("Error sending message:", err)
			}

			mm = <-conChan
		}
	})
}

func main() {
	// channel for messages to display in browser
	// channel to send signals to control individual parts
	// module channels require a distinction, so for multiple chat bots, need to identify which one, then
	// bots will inspect the channel and remove elements that are relevant to them.

	// use HTTP to serve the main management page
	// use websockets to send incremental updates to the management page and also to handle messages from page

	consoleChan := make(chan ConsoleMessage)

	mux := http.NewServeMux()

	mh := messageHandler(consoleChan)

	mux.Handle("/msg", mh)
	mux.HandleFunc("/", serveHTML)

	log.Print("Management Server Started...")

	http.ListenAndServe(":8765", mux)
}
