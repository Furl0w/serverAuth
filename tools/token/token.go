package token

import (
	"fmt"
	"log"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

type clientClaims struct {
	Email string `json:"email"`
	jwt.StandardClaims
}

//CreateToken creates a token from an email and a time duration. The token will expire in the expiration provided minutes from now.
//
//TODO -> change the email for a user structure for passing parameters to the client
func CreateToken(email string, expiration time.Duration, secret []byte) (string, error) {
	newClaims := clientClaims{email, jwt.StandardClaims{ExpiresAt: time.Now().Add(time.Minute * expiration).Unix()}}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &newClaims)
	tokenString, err := token.SignedString(secret)
	if err != nil {
		log.Println(err)
		return "", err
	}
	return tokenString, nil
}

//ValidateToken validate a token by : validating the signing method is HMAC,
//validating the key, checking if the token is not expired and checking that the client match the email in the token
func ValidateToken(tokenString string, client string, secret []byte) (string, error) {
	token, _ := jwt.ParseWithClaims(tokenString, &clientClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("There was an error")
		}
		return secret, nil
	})
	if claims, ok := token.Claims.(*clientClaims); ok && token.Valid {
		if claims.Email == client && time.Unix(claims.StandardClaims.ExpiresAt, 0).After(time.Now()) {
			return client, nil
		}
	}
	return "", fmt.Errorf("invalid token")
}
