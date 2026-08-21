package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestParseRejectsTokenWithWrongAudience(t *testing.T) {
	claims := Claims{RegisteredClaims: jwt.RegisteredClaims{
		Issuer: "r2-image-admin", Audience: jwt.ClaimStrings{"other-service"},
		IssuedAt: jwt.NewNumericDate(time.Now()), ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("test-jwt-secret-with-at-least-thirty-two-characters"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Parse("test-jwt-secret-with-at-least-thirty-two-characters", token); err == nil {
		t.Fatal("expected audience validation failure")
	}
}
