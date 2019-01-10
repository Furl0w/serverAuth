package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

var dbServicePort, dbServiceName string

type user struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
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

	r := mux.NewRouter()
	r.HandleFunc("/", hello).Methods("GET")
	r.HandleFunc("/test", test).Methods("GET")

	http.ListenAndServe(":"+PORT, r)
}

func hello(w http.ResponseWriter, r *http.Request) {

	fmt.Fprint(w, "Hello world !\n")
	return
}

func test(w http.ResponseWriter, r *http.Request) {

	b, err := userExist("Kappa")
	if err != nil {
		log.Println(err.Error())
	}
	if b {
		log.Println("true")
	} else {
		log.Println("false")
	}
	return
}

func userExist(name string) (bool, error) {

	request := fmt.Sprintf("http://%s:%s/user/name/%s", dbServiceName, dbServicePort, name)
	response, err := http.Get(request)
	if err != nil {
		log.Fatal(err)
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
	}
	return true, nil
}
