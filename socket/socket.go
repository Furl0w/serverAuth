package socket

import (
	"fmt"
	"log"

	"github.com/gorilla/websocket"
)

//Packet represent the application data received and send by the socket
type Packet struct {
	IsAuthValid bool `json:"isAuthValid"`
}

//Channel represent a connection to a client identified by his email
type Channel struct {
	Conn   *websocket.Conn
	Data   chan Packet
	Client string
}

//NewChannel is a constructor for Channel
func NewChannel(conn *websocket.Conn, client string) *Channel {
	c := Channel{
		Conn:   conn,
		Data:   make(chan Packet),
		Client: client,
	}
	return &c
}

//ChannelManager is used to managed all the channels
type ChannelManager struct {
	Channels   map[string]bool
	Register   chan *Channel
	Unregister chan *Channel
}

//Start create the go routine for registring and unregistring new channel to the manager
func (manager *ChannelManager) Start() {
	for {
		select {
		case c := <-manager.Register:
			manager.Channels[c.Client] = true
			log.Println("connection added")
			log.Println(c.Client)
		case c := <-manager.Unregister:
			if _, ok := manager.Channels[c.Client]; ok {
				c.Conn.Close()
				delete(manager.Channels, c.Client)
			}
		}
	}
}

//Send create the go routine for sending new packet from a connection
func (manager *ChannelManager) Send(channel *Channel) {
	defer channel.Conn.Close()
	for {
		select {
		case message, ok := <-channel.Data:
			if !ok {
				return
			}
			channel.Conn.WriteJSON(message)
		}
	}
}

//Receive expect a packet to come and then call the handling function for it
func (manager *ChannelManager) Receive(channel *Channel) {
	var pkt Packet
	for {
		err := channel.Conn.ReadJSON(&pkt)
		if err != nil {
			fmt.Printf("%s\n", err.Error())
			manager.Unregister <- channel
			break
		}
		fmt.Printf("RECEIVED: %t\n", pkt.IsAuthValid)
	}
}
