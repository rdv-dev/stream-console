package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"time"
)

type ConsoleCommand struct {
	Command string
}

type ConsoleMessage struct {
	Message string
}

func (mm *ConsoleMessage) PrintMessage() string {
	return mm.Message
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "http://localhost:8765" {
			return true
		}
		return false
	},
}

func serveHTML(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/index.html")
}

// Sends any messages queued to be displayed to control console log
func messageHandler(conChan <-chan ConsoleMessage, ctlChan chan<- ConsoleCommand) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		readMsgChan := make(chan []byte)
		cmd := ""
		hasCmd := false
		conn, err := upgrader.Upgrade(w, r, nil)

		if err != nil {
			log.Println("Error upgrading HTTP connection to websocket:", err)
		}
		defer conn.Close()

		log.Println("Message Handler ready to forward messages to console")

		go func() {
			for {
				_, message, err := conn.ReadMessage()
				if err != nil {
					log.Println("Error reading Message Handler messages from websocket:", err)
					close(readMsgChan)
					return
				}
				readMsgChan <- message
			}
		}()

		for {
			select {
			case mm := <-conChan:
				if err := conn.WriteMessage(websocket.TextMessage, []byte(mm.PrintMessage())); err != nil {
					log.Println("Error sending message:", err)
					return
				}
			case message, ok := <-readMsgChan:
				if !ok {
					log.Println("Websocket management connection closed by client")
					return
				}
				cmd = fmt.Sprintf("%s", message)
				hasCmd = true
			}

			if hasCmd {
				select {
				case ctlChan <- ConsoleCommand{Command: cmd}:
					cmd = ""
					hasCmd = false
				default:
					time.Sleep(50 * time.Millisecond)
				}
			}
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

	consoleChan := make(chan ConsoleMessage, 50)
	controlChan := make(chan ConsoleCommand, 50)
	defer close(consoleChan)
	defer close(controlChan)

	mux := http.NewServeMux()

	mh := messageHandler(consoleChan, controlChan)

	mux.Handle("/msg", mh)
	mux.HandleFunc("/", serveHTML)

	log.Print("Management Server Started...")

	go func() {
		for {
			consoleChan <- ConsoleMessage{Message: "Hello!"}
			log.Println("Got command: " + (<-controlChan).Command)
			time.Sleep(50 * time.Millisecond)
		}
	}()

	http.ListenAndServe(":8765", mux)
}
