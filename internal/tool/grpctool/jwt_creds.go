package grpctool

import (
	"context"
	"time"

	"github.com/dgrijalva/jwt-go/v4"
)

const (
	jwtValidFor  = 5 * time.Second
	jwtNotBefore = 5 * time.Second
)

type JwtCredentials struct {
	Secret   []byte
	Audience string
	Issuer   string
	Insecure bool
}

func (c *JwtCredentials) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	now := time.Now()
	claims := jwt.StandardClaims{
		Audience:  jwt.ClaimStrings{c.Audience},
		ExpiresAt: jwt.At(now.Add(jwtValidFor)),
		IssuedAt:  jwt.At(now),
		Issuer:    c.Issuer,
		NotBefore: jwt.At(now.Add(-jwtNotBefore)),
	}
	signedClaims, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).
		SignedString(c.Secret)
	if err != nil {
		return nil, err
	}
	return map[string]string{
		MetadataAuthorization: "Bearer " + signedClaims,
	}, nil
}

func (c *JwtCredentials) RequireTransportSecurity() bool {
	return !c.Insecure
}
