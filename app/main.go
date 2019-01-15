package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"serverAuth/socket"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var dbServicePort, dbServiceName string

type user struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type authResponse struct {
	Client      string `json:"client,omitempty"`
	IsAuthValid bool   `json:"isAuthValid"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var manager = socket.ChannelManager{
	Channels:   make(map[string]*socket.Channel),
	Register:   make(chan *socket.Channel),
	Unregister: make(chan *socket.Channel),
}

func main() {
	var PORT string
	if PORT = os.Getenv("PORT"); PORT == "" {
		PORT = "3030"
	}
	if dbServicePort = os.Getenv("DB_SERVICE_PORT"); dbServicePort == "" {
		dbServicePort = "3031"
	}
	if dbServiceName = os.Getenv("DB_SERVICE_NAME"); dbServiceName == "" {
		dbServiceName = "localhost"
	}

	go manager.Start()

	r := mux.NewRouter()
	r.HandleFunc("/connect/{email}", connect)
	r.HandleFunc("/authAnswer", authAnswer).Methods("POST")

	http.ListenAndServe(":"+PORT, r)
}

func userExist(email string) (bool, error) {

	request := fmt.Sprintf("http://%s:%s/user/email/%s", dbServiceName, dbServicePort, email)
	response, err := http.Get(request)
	if err != nil {
		log.Println(err)
		return false, err
	}
	defer response.Body.Close()
	if response.StatusCode == 200 {
		var userRequest []user
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Println(err.Error())
			return false, err
		}
		err = json.Unmarshal(body, &userRequest)
		if err != nil {
			log.Println(err.Error())
			return false, err
		}
		if len(userRequest) != 1 {
			return false, nil
		}
		return true, nil
	}
	return false, errors.New("Could not reach the service")
}

//c.Data <- packet{IsAuthValid: false}

func connect(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	//DO NOT DO THAT IN PRODUCTION
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err.Error())
		return
	}
	c := socket.NewChannel(conn, params["email"])
	manager.Register <- c
	go manager.Receive(c)
	go manager.Send(c)
	return
}

func authAnswer(w http.ResponseWriter, r *http.Request) {

	var authResponse authResponse
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "error when receiving stuff\n")
		log.Println(err.Error())
		return
	}
	err = json.Unmarshal(body, &authResponse)
	if err != nil {
		w.WriteHeader(400)
		fmt.Fprintf(w, "error when unmarshalling json\n")
		log.Println(err.Error())
		return
	}
	if channel, ok := manager.Channels[authResponse.Client]; ok {
		channel.Data <- socket.Packet{IsAuthValid: authResponse.IsAuthValid}
	}
	return
}
