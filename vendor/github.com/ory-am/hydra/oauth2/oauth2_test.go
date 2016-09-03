package oauth2_test

import (
	"fmt"
	"net/http/httptest"
	"net/url"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-errors/errors"
	"github.com/julienschmidt/httprouter"
	"github.com/ory-am/fosite"
	"github.com/ory-am/fosite/compose"
	"github.com/ory-am/fosite/hash"
	hc "github.com/ory-am/hydra/client"
	"github.com/ory-am/hydra/internal"
	"github.com/ory-am/hydra/jwk"
	. "github.com/ory-am/hydra/oauth2"
	"github.com/ory-am/hydra/pkg"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

var hasher = &hash.BCrypt{}

var store = &internal.FositeMemoryStore{
	Manager: &hc.MemoryManager{
		Clients: map[string]hc.Client{},
		Hasher:  hasher,
	},
	AuthorizeCodes: make(map[string]fosite.Requester),
	IDSessions:     make(map[string]fosite.Requester),
	AccessTokens:   make(map[string]fosite.Requester),
	Implicit:       make(map[string]fosite.Requester),
	RefreshTokens:  make(map[string]fosite.Requester),
}

var keyManager = &jwk.MemoryManager{}
var keyGenerator = &jwk.RS256Generator{}

var fc = &compose.Config{}
var handler = &Handler{
	OAuth2: compose.Compose(
		fc,
		store,
		&compose.CommonStrategy{
			CoreStrategy:               compose.NewOAuth2HMACStrategy(fc, []byte("some super secret secret")),
			OpenIDConnectTokenStrategy: compose.NewOpenIDConnectStrategy(pkg.MustRSAKey()),
		},
		compose.OAuth2AuthorizeExplicitFactory,
		compose.OAuth2AuthorizeImplicitFactory,
		compose.OAuth2ClientCredentialsGrantFactory,
		compose.OAuth2RefreshTokenGrantFactory,
		compose.OpenIDConnectExplicit,
		compose.OpenIDConnectHybrid,
		compose.OpenIDConnectImplicit,
	),
	Consent: &DefaultConsentStrategy{
		Issuer:                   "http://hydra.localhost",
		KeyManager:               keyManager,
		DefaultChallengeLifespan: time.Hour,
		DefaultIDTokenLifespan:   time.Hour * 24,
	},
	ForcedHTTP: true,
}

var router = httprouter.New()
var ts *httptest.Server
var oauthConfig *oauth2.Config
var oauthClientConfig *clientcredentials.Config

func init() {
	keys, err := keyGenerator.Generate("")
	pkg.Must(err, "")
	keyManager.AddKeySet(ConsentChallengeKey, keys)

	keys, err = keyGenerator.Generate("")
	pkg.Must(err, "")
	keyManager.AddKeySet(ConsentEndpointKey, keys)
	ts = httptest.NewServer(router)

	handler.SetRoutes(router)
	store.Manager.(*hc.MemoryManager).Clients["app"] = hc.Client{
		ID:            "app",
		Secret:        "secret",
		RedirectURIs:  []string{ts.URL + "/callback"},
		ResponseTypes: []string{"id_token", "code", "token"},
		GrantTypes:    []string{"implicit", "refresh_token", "authorization_code", "password", "client_credentials"},
		Scope:         "hydra",
	}

	c, _ := url.Parse(ts.URL + "/consent")
	handler.ConsentURL = *c

	h, _ := hasher.Hash([]byte("secret"))
	store.Manager.(*hc.MemoryManager).Clients["app-client"] = hc.Client{
		ID:            "app-client",
		Secret:        string(h),
		RedirectURIs:  []string{ts.URL + "/callback"},
		ResponseTypes: []string{"id_token", "code", "token"},
		GrantTypes:    []string{"implicit", "refresh_token", "authorization_code", "password", "client_credentials"},
		Scope:         "hydra",
	}

	oauthConfig = &oauth2.Config{
		ClientID:     "app-client",
		ClientSecret: "secret",
		Endpoint: oauth2.Endpoint{
			AuthURL:  ts.URL + "/oauth2/auth",
			TokenURL: ts.URL + "/oauth2/token",
		},
		RedirectURL: ts.URL + "/callback",
		Scopes:      []string{"hydra"},
	}

	oauthClientConfig = &clientcredentials.Config{
		ClientID:     "app-client",
		ClientSecret: "secret",
		TokenURL:     ts.URL + "/oauth2/token",
		Scopes:       []string{"hydra"},
	}
}

func signConsentToken(claims jwt.MapClaims) (string, error) {
	token := jwt.New(jwt.SigningMethodRS256)
	token.Claims = claims

	keys, err := keyManager.GetKey(ConsentEndpointKey, "private")
	if err != nil {
		return "", errors.New(err)
	}
	rsaKey, err := jwk.ToRSAPrivate(jwk.First(keys.Keys))
	if err != nil {
		return "", err
	}

	var signature, encoded string
	if encoded, err = token.SigningString(); err != nil {
		return "", errors.New(err)
	} else if signature, err = token.Method.Sign(encoded, rsaKey); err != nil {
		return "", errors.New(err)
	}

	return fmt.Sprintf("%s.%s", encoded, signature), nil
}
