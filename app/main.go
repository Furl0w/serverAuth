package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"serverAuth/tools/socket"
	jwt "serverAuth/tools/token"

	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var dbServicePort, dbServiceName string

const firebaseService = "https://fcm.googleapis.com/fcm/send"

type user struct {
	ID    string `json:"id,omitempty"`
	Email string `json:"email,omitempty"`
}

type userExistsRequest struct {
	Email  string `json:"email,omitempty"`
	Exists bool   `json:"exists"`
}

type tryConnectResponse struct {
	Email  string `json:"email,omitempty"`
	Exists bool   `json:"exists"`
	Token  string `json:"token,omitempty"`
}

type connectFromTokenRequest struct {
	Email string `json:"email,omitempty"`
	Token string `json:"token,omitempty"`
}

type connectFromTokenResponse struct {
	Email       string `json:"email,omitempty"`
	IsAuthValid bool   `json:"isAuthValid"`
}

type exceptionMiddleware struct {
	Message string `json:"message,omitempty"`
}

type notificationRequest struct {
	Data notificationData `json:"data"`
	To   string           `json:"to,omitempty"`
}

type notificationData struct {
	Title string `json:"title,omitempty"`
	Body  string `json:"body,omitempty"`
}

type authAnswerRequest struct {
	Client      string `json:"client,omitempty"`
	IsAuthValid bool   `json:"isAuthValid"`
}

type userRegistrationRequest struct {
	Email      string      `json:"email,omitempty"`
	Signatures []signature `json:"signatures,omitempty"`
	Token      string      `json:"token,omitempty"`
}

type signature struct {
	Abs  []int `json:"abs"`
	Ord  []int `json:"ord"`
	Time []int `json:"time"`
}

type userRegistrationAnswer struct {
	IsRegistrationValid bool `json:"isRegistrationValid"`
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
	r.HandleFunc("/connect/{email}", validateMiddleware(connect))
	r.HandleFunc("/userExists/{email}", checkUser)
	r.HandleFunc("/tryConnect/{email}", tryConnect)
	r.HandleFunc("/connectFromToken", connectFromToken).Methods("POST")
	r.HandleFunc("/authAnswer", authAnswer).Methods("POST")
	r.HandleFunc("/register", register).Methods("POST")
	log.Fatal(http.ListenAndServeTLS(":"+PORT, "localhost.pem", "localhost-key.pem", r))
}

func checkUser(w http.ResponseWriter, r *http.Request) {

	params := mux.Vars(r)
	exists, err := userExists(params["email"])
	if err != nil {
		w.WriteHeader(400)
		log.Println(err.Error())
		return
	}
	res := userExistsRequest{Email: params["email"], Exists: exists}
	json, err := json.Marshal(res)
	if err != nil {
		w.WriteHeader(500)
		log.Println(err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "%s", string(json))
	return
}

func tryConnect(w http.ResponseWriter, r *http.Request) {

	params := mux.Vars(r)
	exists, err := userExists(params["email"])
	if err != nil {
		w.WriteHeader(400)
		log.Println(err.Error())
		return
	}
	token := ""
	if exists != false {
		token, err = jwt.CreateToken(params["email"], time.Duration(5))
		if err != nil {
			w.WriteHeader(500)
			log.Println(err.Error())
			return
		}
	}
	res := tryConnectResponse{Email: params["email"], Exists: exists, Token: token}
	json, err := json.Marshal(res)
	if err != nil {
		w.WriteHeader(500)
		log.Println(err.Error())
		return
	}
	w.Header().Add("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "%s", string(json))
	return
}

func userExists(email string) (bool, error) {

	request := fmt.Sprintf("https://%s:%s/user/email/%s", dbServiceName, dbServicePort, email)
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

func connect(w http.ResponseWriter, r *http.Request) {

	email := context.Get(r, "decoded")

	//DO NOT DO THAT IN PRODUCTION
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err.Error())
		return
	}
	c := socket.NewChannel(conn, email.(string))
	manager.Register <- c
	go manager.Receive(c)
	go manager.Send(c)
	return
}

func connectFromToken(w http.ResponseWriter, r *http.Request) {
	var connectFromTokenRequest connectFromTokenRequest
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "error when receiving stuff\n")
		log.Println(err.Error())
		return
	}
	err = json.Unmarshal(body, &connectFromTokenRequest)
	if err != nil {
		w.WriteHeader(400)
		fmt.Fprintf(w, "error when unmarshalling json\n")
		log.Println(err.Error())
		return
	}
	_, err = jwt.ValidateToken(connectFromTokenRequest.Token, connectFromTokenRequest.Email)
	if err != nil {
		w.WriteHeader(400)
		fmt.Fprintf(w, "invalid token\n")
		log.Println(err.Error())
		return
	}
	res := connectFromTokenResponse{Email: connectFromTokenRequest.Email, IsAuthValid: true}
	json, err := json.Marshal(&res)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "internal marshalling error")
		log.Println(err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "%s", json)
	return
}

func authAnswer(w http.ResponseWriter, r *http.Request) {

	var authAnswer authAnswerRequest
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "error when receiving stuff\n")
		log.Println(err.Error())
		return
	}
	err = json.Unmarshal(body, &authAnswer)
	if err != nil {
		w.WriteHeader(400)
		fmt.Fprintf(w, "error when unmarshalling json\n")
		log.Println(err.Error())
		return
	}
	expirationLimit := time.Duration(131400)
	token, err := jwt.CreateToken(authAnswer.Client, expirationLimit)
	if err != nil {
		w.WriteHeader(400)
		fmt.Fprintf(w, "error when creating token\n")
		log.Println(err.Error())
	}
	if channel, ok := manager.Channels[authAnswer.Client]; ok {
		channel.Data <- socket.Packet{IsAuthValid: authAnswer.IsAuthValid, Token: token}
	}
	return
}

func sendNotification(token string) {

	data := notificationRequest{Data: notificationData{Title: "FunConnect", Body: "Connect to your app"}, To: token}
	body, err := json.Marshal(data)
	if err != nil {
		log.Println(err.Error())
		return
	}
	req, err := http.NewRequest("POST", firebaseService, bytes.NewReader(body))
	if err != nil {
		log.Println(err.Error())
		return
	}

	//Secret key
	key := ""

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("key=%s", key))

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err.Error())
	}
	defer resp.Body.Close()
	return
}

func register(w http.ResponseWriter, r *http.Request) {

	var userRequest userRegistrationRequest
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "error when receiving stuff\n")
		log.Println(err.Error())
		return
	}
	err = json.Unmarshal(body, &userRequest)
	if err != nil {
		w.WriteHeader(400)
		fmt.Fprintf(w, "error when unmarshalling json\n")
		log.Println(err.Error())
		return
	}
	exists, err := userExists(userRequest.Email)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "internal error")
		log.Println(err.Error())
		return
	}
	if exists != false {
		w.WriteHeader(400)
		fmt.Fprintf(w, "user already exists")
		return
	}
	registration := registerUser(userRequest)
	if registration != true {
		w.WriteHeader(500)
		fmt.Fprintf(w, "error during registration\n")
		return
	}
	answer, err := json.Marshal(userRegistrationAnswer{IsRegistrationValid: registration})
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "error during answer\n")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	fmt.Fprintf(w, "%s", string(answer))
	return
}

func registerUser(user userRegistrationRequest) bool {

	request := fmt.Sprintf("https://%s:%s/user", dbServiceName, dbServicePort)
	body, err := json.Marshal(user)
	if err != nil {
		log.Println(err.Error())
		return false
	}
	req, err := http.NewRequest("POST", request, bytes.NewReader(body))
	if err != nil {
		log.Println(err.Error())
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err.Error())
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return false
	}
	return true
}

func validateMiddleware(next http.HandlerFunc) http.HandlerFunc {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		keys := r.URL.Query()
		token := keys.Get("token")
		if token != "" {
			mail, err := jwt.ValidateToken(token, params["email"])
			if err != nil {
				w.WriteHeader(400)
				json.NewEncoder(w).Encode(exceptionMiddleware{Message: err.Error()})
				return
			}
			if mail != "" {
				context.Set(r, "decoded", mail)
				next(w, r)
			} else {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(400)
				json.NewEncoder(w).Encode(exceptionMiddleware{Message: "Invalid authorization token"})
				return
			}
		}
	})
}
