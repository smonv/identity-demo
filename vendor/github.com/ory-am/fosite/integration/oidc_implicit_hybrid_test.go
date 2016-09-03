package integration_test

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/ory-am/fosite/compose"
	"github.com/ory-am/fosite/handler/openid"
	"github.com/ory-am/fosite/internal"
	"github.com/ory-am/fosite/token/jwt"
)

func TestOIDCImplicitFlow(t *testing.T) {
	session := &defaultSession{
		DefaultSession: &openid.DefaultSession{
			Claims: &jwt.IDTokenClaims{
				Subject: "peter",
			},
			Headers: &jwt.Headers{},
		},
	}
	f := compose.ComposeAllEnabled(new(compose.Config), fositeStore, []byte("some-secret-thats-random"), internal.MustRSAKey())
	ts := mockServer(t, f, session)
	defer ts.Close()

	oauthClient := newOAuth2Client(ts)
	fositeStore.Clients["my-client"].RedirectURIs[0] = ts.URL + "/callback"

	var state = "12345678901234567890"
	for k, c := range []struct {
		responseType string
		description  string
		nonce        string
		setup        func()
		hasToken     bool
		hasCode      bool
	}{
		{
			description:  "should pass without id token",
			responseType: "token",
			setup: func() {
				oauthClient.Scopes = []string{"fosite"}
			},
		},
		{

			responseType: "id_token%20token",
			nonce:        "1111111111111111",
			description:  "should pass id token (id_token token)",
			setup: func() {
				oauthClient.Scopes = []string{"fosite", "openid"}
			},
			hasToken: true,
		},
		{

			responseType: "token%20id_token%20code",
			nonce:        "1111111111111111",
			description:  "should pass id token (id_token token)",
			setup:        func() {},
			hasToken:     true,
			hasCode:      true,
		},
	} {
		c.setup()

		var callbackURL *url.URL
		authURL := strings.Replace(oauthClient.AuthCodeURL(state), "response_type=code", "response_type="+c.responseType, -1) + "&nonce=" + c.nonce
		client := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				callbackURL = req.URL
				return errors.New("Dont follow redirects")
			},
		}
		resp, err := client.Get(authURL)
		require.NotNil(t, err, "(%d) %s", k, c.description)

		t.Logf("Response: %s", callbackURL.String())
		fragment, err := url.ParseQuery(callbackURL.Fragment)
		require.Nil(t, err, "(%d) %s", k, c.description)

		expires, err := strconv.Atoi(fragment.Get("expires_in"))
		require.Nil(t, err, "(%d) %s", k, c.description)

		token := &oauth2.Token{
			AccessToken:  fragment.Get("access_token"),
			TokenType:    fragment.Get("token_type"),
			RefreshToken: fragment.Get("refresh_token"),
			Expiry:       time.Now().Add(time.Duration(expires) * time.Second),
		}

		if c.hasToken {
			assert.NotEmpty(t, fragment.Get("id_token"), "(%d) %s", k, c.description)
		} else {
			assert.Empty(t, fragment.Get("id_token"), "(%d) %s", k, c.description)
		}

		if c.hasCode {
			assert.NotEmpty(t, fragment.Get("code"), "(%d) %s", k, c.description)
		} else {
			assert.Empty(t, fragment.Get("code"), "(%d) %s", k, c.description)
		}

		httpClient := oauthClient.Client(oauth2.NoContext, token)
		resp, err = httpClient.Get(ts.URL + "/info")
		require.Nil(t, err, "(%d) %s", k, c.description)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode, "(%d) %s", k, c.description)
		t.Logf("Passed test case (%d) %s", k, c.description)
	}
}
