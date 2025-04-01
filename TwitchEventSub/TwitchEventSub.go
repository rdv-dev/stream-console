package TwitchEventSub

import (
	"github.com/rdv-dev/stream-console/Types"
	"log"
	"time"
)

func Setup(ReceiveChannel <-chan *Types.SystemCommand, SendChannel chan<- *Types.SystemCommand) {
	// Recieves a Receive and Send channel to communicate with main
	// load username, api key
	// login to twitch
	// set variables ready for processing

}

func HandleReceive(ReceiveChannel <-chan *Types.SystemCommand) {
	for {
		select {
		case cmd, ok := <-ReceiveChannel:
			if !ok {
				log.Println("Websocket management connection closed by client")
				return
			}

			log.Printf("Got command %s from source %s\n", cmd.Command, cmd.Source)
		default:
			time.Sleep(25 * time.Millisecond)
		}
	}
}
