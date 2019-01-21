package main

import (
	"fmt"
	"log"
	jwt "serverAuth/tools/token"
	"testing"
	"time"
)

func TestJWT(t *testing.T) {
	user := user{ID: "aaaa", Email: "LOL"}
	token, err := jwt.CreateToken(user.Email, time.Duration(5))
	if err != nil {
		log.Println(err)
	}
	fmt.Printf("%s\n", token)
	email, err := jwt.ValidateToken(token, user.Email)
	if err != nil {
		log.Println(err)
	}
	fmt.Printf("%s\n", email)
}
