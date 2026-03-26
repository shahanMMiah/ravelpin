package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const RaVELPIN = "ravelPin"

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	if err != nil {
		return "", err
	}

	return string(hash), nil

}

func CheckPasswordHash(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

func MakeToken(byteAmount int) (string, error) {
	randBytes := make([]byte, byteAmount)
	rand.Read(randBytes)
	hexString := hex.EncodeToString(randBytes)

	return hexString, nil
}

func MakeJWT(userId uuid.UUID, tokenSecret string, expires time.Duration) (string, error) {

	claims := jwt.RegisteredClaims{
		Issuer:    RaVELPIN,
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expires).UTC()),
		Subject:   userId.String(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(tokenSecret))

}

func ValidateJWT(signedToken, tokenSecret string) (uuid.UUID, error) {
	token, err := jwt.ParseWithClaims(signedToken, &jwt.RegisteredClaims{}, func(token *jwt.Token) (any, error) {
		return []byte(tokenSecret), nil
	})

	if err != nil {
		return uuid.UUID{}, err
	}

	subject, err := token.Claims.GetSubject()

	if err != nil {
		return uuid.UUID{}, fmt.Errorf("subject faulure:  %s", err.Error())
	}
	id, err := uuid.Parse(subject)

	if err != nil {
		return uuid.UUID{}, fmt.Errorf("uuid parse faulure:  %s", err.Error())
	}

	return id, nil
}
