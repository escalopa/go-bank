package token

import (
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v4"
)

const minSecretKeyLen = 32

type JWTMaker struct {
	secretKey string
}

func NewJWTMaker(secretKey string) (Maker, error) {
	if len(secretKey) < minSecretKeyLen {
		return nil, fmt.Errorf("secretKet len is less than the min value %d", minSecretKeyLen)
	}

	return &JWTMaker{secretKey}, nil
}

func (jwtMaker *JWTMaker) CreateToken(username string) (string, *Payload, error) {
	payload, err := NewPayload(username, AccessTokenExpiration)
	if err != nil {
		return "", payload, err
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS512, payload)
	token, err := jwtToken.SignedString([]byte(jwtMaker.secretKey))
	return token, payload, err
}

func (JWTMaker *JWTMaker) CreateRefreshToken(username string) (string, *Payload, error) {
	return "", nil, nil
}

func (jwtMaker *JWTMaker) VerifyToken(token string) (*Payload, error) {
	keyFunc := func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodECDSA); ok {
			return nil, ErrTokenInvalid
		}

		return []byte(jwtMaker.secretKey), nil
	}

	jwtToken, err := jwt.ParseWithClaims(token, &Payload{}, keyFunc)
	if err != nil {
		verr, ok := err.(*jwt.ValidationError)
		if ok && errors.Is(verr, ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}

	payload, ok := jwtToken.Claims.(*Payload)
	if !ok {
		return nil, ErrTokenInvalid
	}

	return payload, nil
}
