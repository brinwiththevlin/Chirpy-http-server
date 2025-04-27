package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type TokenType string

const (
	TokenTypeAccess TokenType = "chirpy"
)

// Reutrns hasshed password
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// Error != nil if the password matches the hash
func CheckPasswordHash(hash, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err
}

// makes  JWT token
func MakeJWT(userid uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.RegisteredClaims{
			Issuer:    string(TokenTypeAccess),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
			Subject:   userid.String(),
		})

	return token.SignedString([]byte(tokenSecret))
}

// returns the user id if the JWT token is valid
func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (any, error) { return []byte(tokenSecret), nil })
	if err != nil {
		log.Println(err)
		return uuid.Nil, err
	}

	userid, err := token.Claims.GetSubject()
	if err != nil {
		log.Println(err)
		return uuid.Nil, err
	}

	issuer, err := token.Claims.GetIssuer()
	if err != nil {
		log.Println(err)
		return uuid.Nil, err
	}

	if issuer != string(TokenTypeAccess) {
		log.Println(err)
		return uuid.Nil, errors.New("invalid user")
	}

	id, err := uuid.Parse(userid)
	if err != nil {
		log.Println(err)
		return uuid.Nil, fmt.Errorf("invalid id %w", err)
	}
	return id, nil

}

// if request has a bearer token return it
func GetBearerToken(headers http.Header) (string, error) {
	authString := headers.Get("Authorization")
	if authString == "" {
		return "", errors.New("no authorization string")
	}
	fields := strings.Fields(authString)
	if len(fields) < 2 {
		return "", errors.New("invalid authorizasiton string format")
	}
	return strings.Join(fields[1:], " "), nil
}

//makes a new refresh token
func MakeRefreshToken() (string, error) {
	key := make([]byte, 32)
	rand.Read(key)
	str := hex.EncodeToString(key)
	return str, nil
}
