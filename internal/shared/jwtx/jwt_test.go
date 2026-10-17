package jwtx

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

func TestParseTokenValidatesPlatformClaims(t *testing.T) {
	token, err := GenerateToken("secret", 7, "alice", []int64{2}, 60)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := ParseToken(token, "secret")
	if err != nil || claims.UserID != 7 || claims.Issuer != "axiops" {
		t.Fatalf("claims=%#v err=%v", claims, err)
	}
}

func TestParseTokenRejectsWrongIssuerAndAlgorithm(t *testing.T) {
	wrongIssuer := &Claims{
		UserID: 7,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "other",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, wrongIssuer)
	signed, err := token.SignedString([]byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseToken(signed, "secret"); !errors.Is(err, ErrTokenInvalid) {
		t.Fatalf("wrong issuer err=%v", err)
	}

	wrongAlgorithm := jwt.NewWithClaims(jwt.SigningMethodHS384, &Claims{
		UserID: 7,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "axiops",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
		},
	})
	signed, err = wrongAlgorithm.SignedString([]byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseToken(signed, "secret"); !errors.Is(err, ErrTokenInvalid) {
		t.Fatalf("wrong algorithm err=%v", err)
	}
}

func TestParseTokenReportsExpiredToken(t *testing.T) {
	token, err := GenerateToken("secret", 7, "alice", nil, -1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseToken(token, "secret"); !errors.Is(err, ErrTokenExpired) {
		t.Fatalf("expired token err=%v", err)
	}
}
