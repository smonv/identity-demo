package jwk

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"

	"crypto/x509"

	"github.com/go-errors/errors"
	"github.com/square/go-jose"
)

type ECDSA521Generator struct{}

func (g *ECDSA521Generator) Generate(id string) (*jose.JsonWebKeySet, error) {
	key, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	if err != nil {
		return nil, errors.Errorf("Could not generate key because %s", err)
	}

	return &jose.JsonWebKeySet{
		Keys: []jose.JsonWebKey{
			{
				Key:          key,
				KeyID:        ider("private", id),
				Certificates: []*x509.Certificate{},
			},
			{
				Key:          &key.PublicKey,
				KeyID:        ider("public", id),
				Certificates: []*x509.Certificate{},
			},
		},
	}, nil
}
