package main

import (
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/rdv-dev/stream-console/TwitchEventSub"
	"github.com/rdv-dev/stream-console/Types"
	"log"
	"net/http"
	"time"
)

// This is a handle that an HTTP handler uses to forward data to the console it is connected to
type ConsoleHandle struct {
	outMessages []string
	inMessages  []string
	readMsgChan chan []byte
	ConsoleType Types.ConsoleType
	conn        *websocket.Conn
	Active      bool
}

func newConsoleHandle(conn *websocket.Conn, messageType Types.ConsoleType) *ConsoleHandle {

	return &ConsoleHandle{
		readMsgChan: make(chan []byte, 25),
		conn:        conn,
		ConsoleType: messageType,
		Active:      false}
}

func (c *ConsoleHandle) ReadMessage() {
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			log.Println("Error reading Message Handler messages from websocket:", err)
			c.Close()
			return
		}
		c.readMsgChan <- message
	}
}

func (c *ConsoleHandle) WriteMessage(message string) error {
	err := c.conn.WriteMessage(websocket.TextMessage, []byte(message))

	if err != nil {
		log.Println("Error sending message:", err)
		c.Close()
		return errors.New(fmt.Sprintf("Error sending message: %s\n", err))
	}
	return nil
}

func (c *ConsoleHandle) Close() {
	c.conn.Close()
	close(c.readMsgChan)
}

type ConsoleState struct {
	consoleChannel chan Types.ConsoleMessage
	controlChannel chan Types.SystemCommand
	consoleBacklog []*Types.ConsoleMessage
	handles        []*ConsoleHandle
	numHandles     int
}

func (c *ConsoleState) HasHandles() bool {
	return c.numHandles > 0
}

func (c *ConsoleState) Register(ch *ConsoleHandle) {
	if ch == nil {
		log.Println("Attempting to register nil Console Handle, skipping.")
		return
	}

	ch.Active = true

	c.handles = append(c.handles, ch)
	c.numHandles++

}

func (c *ConsoleState) WriteMessage(message Types.ConsoleMessage) {
	if c.numHandles == 0 {
		c.consoleBacklog = append(c.consoleBacklog, &message)
		return
	}

	if message.Source == Types.ConsoleTypeAll {
		for i := range c.handles {
			if c.handles[i].Active {
				err := c.handles[i].WriteMessage(message.PrintMessage())
				if err != nil {
					log.Println(err)
					c.handles[i].Active = false
					c.numHandles--
				}
			}
		}
	}
}

func (c *ConsoleState) Close() {
	close(c.consoleChannel)
	close(c.controlChannel)
	for i := range c.handles {
		c.handles[i].Close()
	}
	c.numHandles = 0

	c.handles = make([]*ConsoleHandle, 0)
}

func serveHTML(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/ServerManagementMain.html")
}

// Sends any messages queued to be displayed to control console log
func messageHandler(state *ConsoleState) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		upgrader := websocket.Upgrader{
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

		conn, err := upgrader.Upgrade(w, r, nil)

		if err != nil {
			log.Fatal("Error upgrading HTTP connection to websocket:", err)
			return
		}

		myHandle := newConsoleHandle(conn, Types.ConsoleTypeAll)
		if myHandle == nil {
			log.Fatal("Failed to set up websocket")
			return
		}
		state.Register(myHandle)

		log.Println("Message Handler ready to forward messages to console")

		go myHandle.ReadMessage()

	})
}

func main() {
	// channel for messages to display in browser
	// channel to send signals to control individual parts
	// module channels require a distinction, so for multiple chat bots, need to identify which one, then
	// bots will inspect the channel and remove elements that are relevant to them.

	// use HTTP to serve the main management page
	// use websockets to send incremental updates to the management page and also to handle messages from page

	mainState := &ConsoleState{
		consoleChannel: make(chan Types.ConsoleMessage, 50),
		controlChannel: make(chan Types.SystemCommand, 50),
		consoleBacklog: make([]*Types.ConsoleMessage, 0),
		handles:        make([]*ConsoleHandle, 0),
		numHandles:     0}

	incomingCommands := make(chan *Types.SystemCommand, 50)
	outgoingCommands := make(chan *Types.SystemCommand, 50)

	mux := http.NewServeMux()

	mh := messageHandler(mainState)

	mux.Handle("/msg", mh)
	mux.HandleFunc("/", serveHTML)

	defer mainState.Close()

	log.Print("Management Server Started...")

	go func() {
		//i := 0
		for {
			//mainState.consoleChannel <- ConsoleMessage{Message: "Hello!", Source: ConsoleTypeAll}

			//mm := <-mainState.consoleChannel
			mainState.WriteMessage(Types.ConsoleMessage{Message: "Hello!", Source: Types.ConsoleTypeAll})

			time.Sleep(time.Second)

			//i++
			//if i == len(mainState.handles) {
			//	i = 0
			//}
		}
	}()

	go func() {
		i := 0
		queue := make([]*Types.SystemCommand, 0)
		for {
			if mainState.HasHandles() {
				select {
				case message, ok := <-mainState.handles[i].readMsgChan:
					if !ok {
						log.Println("Websocket management connection closed by client")
						return
					}

					//mainState.controlChannel <- SystemCommand{Command: cmd}
					//log.Println("Got command: " + (<-mainState.controlChannel).Command)
					log.Println("Got command: " + fmt.Sprintf("%s", message))
					queue = append(queue, &Types.SystemCommand{Command: fmt.Sprintf("%s", message), Source: Types.SystemMain})

				case command, ok := <-incomingCommands:
					if !ok {
						log.Println("Websocket management connection closed by client")
						return
					}
					log.Printf("Got command from somewhere %s\n", command.Command)

				default:
					time.Sleep(25 * time.Millisecond)
				}

				i++
				if i == len(mainState.handles) {
					i = 0
				}
			}
		}
	}()

	TwitchEventSub.Setup(outgoingCommands, incomingCommands)

	http.ListenAndServe(":8765", mux)
}
