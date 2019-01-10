package main

import (
	"encoding/json"
	"errors"
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

type authResponse struct {
	IsAuthValid bool `json:"isAuthValid"`
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
	r.HandleFunc("e/testAuth/{nam}", test).Methods("GET")

	http.ListenAndServe(":"+PORT, r)
}

func hello(w http.ResponseWriter, r *http.Request) {

	fmt.Fprint(w, "Hello world !\n")
	return
}

func test(w http.ResponseWriter, r *http.Request) {

	params := mux.Vars(r)
	auth := false
	b, err := userExist(params["name"])
	if err != nil {
		log.Println(err.Error())
		w.WriteHeader(500)
		fmt.Fprintf(w, "Could not get access to users\n")
		return
	}
	if b {
		authPy, err := checkAuthPy()
		if err != nil {
			log.Println(err.Error())
			w.WriteHeader(500)
			fmt.Fprintf(w, "Could not get access to py\n")
			return
		}
		auth = authPy
	}
	jsonString, err := json.Marshal(authResponse{IsAuthValid: auth})
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "internal error when delivering results\n")
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	fmt.Fprintf(w, "%s", jsonString)
	return
}

func userExist(name string) (bool, error) {

	request := fmt.Sprintf("http://%s:%s/user/name/%s", dbServiceName, dbServicePort, name)
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

func checkAuthPy() (bool, error) {

	request := fmt.Sprintf("http://%s:%s/checkAuth", "localhost", "5000")
	response, err := http.Get(request)
	if err != nil {
		log.Println(err)
		return false, err
	}
	defer response.Body.Close()
	if response.StatusCode == 200 {
		var authResponse authResponse
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Println(err.Error())
			return false, err
		}
		err = json.Unmarshal(body, &authResponse)
		if err != nil {
			log.Println(err.Error())
			return false, err
		}
		return authResponse.IsAuthValid, nil
	}
	return false, errors.New("Could not reach the service")
}
