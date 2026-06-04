package auth

import "github.com/golang-jwt/jwt/v5"
import "github.com/google/uuid"
import(
	"time"
	"log"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

type TokenType string

const (TokenTypeAccess TokenType = "chirpy-access") 

func MakeJWT(userID uuid.UUID, tokenSecret string) (string, error) {
	mySigningKey := []byte(tokenSecret)
	rc := jwt.RegisteredClaims{
		Issuer: string(TokenTypeAccess),
		IssuedAt: jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Hour * 1)),
		Subject: userID.String(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, rc)
	return token.SignedString(mySigningKey)
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	rc := jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenString, &rc, func(token *jwt.Token) (interface{}, error) {
		return []byte(tokenSecret), nil
	})
	if err != nil {
		log.Printf("Error validating token %v: %v\n", tokenString, err)
		return uuid.Nil, err
	}
	userIdStr, err := token.Claims.GetSubject()
	if err != nil {
		return uuid.Nil, err
	}
	issuer, err := token.Claims.GetIssuer()
	if err != nil {
		return uuid.Nil, err
	}
	if issuer != string(TokenTypeAccess) {
		return uuid.Nil, errors.New("Invalid issuer")
	}
	id, err := uuid.Parse(userIdStr)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid user ID: %v", err)
	}
	return id, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	bearer := headers.Get("Authorization")
	if bearer == "" {
		log.Printf("No autorization found")
		return bearer, errors.New("No authorization found")
	}
	parts := strings.Fields(bearer)
	if len(parts) == 2 && parts[0] == "Bearer" {
		log.Printf("Bearer token : %v\n", parts[1])
		return parts[1], nil
	}
	return "", errors.New("Invalid auth header format")
}
