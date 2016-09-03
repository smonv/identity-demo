package openid

import (
	"net/http"

	"fmt"

	"github.com/ory-am/fosite"
	"github.com/ory-am/fosite/handler/oauth2"
	"github.com/ory-am/fosite/token/jwt"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

type OpenIDConnectHybridHandler struct {
	AuthorizeImplicitGrantTypeHandler *oauth2.AuthorizeImplicitGrantTypeHandler
	AuthorizeExplicitGrantHandler     *oauth2.AuthorizeExplicitGrantHandler
	IDTokenHandleHelper               *IDTokenHandleHelper
	ScopeStrategy                     fosite.ScopeStrategy

	Enigma *jwt.RS256JWTStrategy
}

func (c *OpenIDConnectHybridHandler) HandleAuthorizeEndpointRequest(ctx context.Context, req *http.Request, ar fosite.AuthorizeRequester, resp fosite.AuthorizeResponder) error {
	if len(ar.GetResponseTypes()) < 2 {
		return nil
	}

	if !(ar.GetResponseTypes().Matches("token", "id_token", "code") || ar.GetResponseTypes().Matches("token", "code")) {
		return nil
	}

	if !ar.GetClient().GetResponseTypes().Has("token", "code") {
		return errors.Wrap(fosite.ErrInvalidGrant, "The client is not allowed to use the code and token response type")
	} else if ar.GetResponseTypes().Matches("id_token") && !ar.GetClient().GetResponseTypes().Has("id_token") {
		return errors.Wrap(fosite.ErrInvalidGrant, "The client is not allowed to use the id_token response type")
	}

	sess, ok := ar.GetSession().(Session)
	if !ok {
		return errors.Wrap(ErrInvalidSession, "")
	}

	client := ar.GetClient()
	for _, scope := range ar.GetRequestedScopes() {
		if !c.ScopeStrategy(client.GetScopes(), scope) {
			return errors.Wrap(fosite.ErrInvalidScope, fmt.Sprintf("The client is not allowed to request scope %s", scope))
		}
	}

	claims := sess.IDTokenClaims()
	if ar.GetResponseTypes().Has("code") {
		if !ar.GetClient().GetGrantTypes().Has("authorization_code") {
			return errors.Wrap(fosite.ErrInvalidGrant, "The client is not allowed to use the authorization_code grant type")
		}

		code, signature, err := c.AuthorizeExplicitGrantHandler.AuthorizeCodeStrategy.GenerateAuthorizeCode(ctx, ar)
		if err != nil {
			return errors.Wrap(fosite.ErrServerError, err.Error())
		} else if err := c.AuthorizeExplicitGrantHandler.AuthorizeCodeGrantStorage.CreateAuthorizeCodeSession(ctx, signature, ar); err != nil {
			return errors.Wrap(fosite.ErrServerError, err.Error())
		}

		resp.AddFragment("code", code)
		resp.AddFragment("state", ar.GetState())
		ar.SetResponseTypeHandled("code")

		hash, err := c.Enigma.Hash([]byte(resp.GetFragment().Get("code")))
		if err != nil {
			return err
		}
		claims.CodeHash = hash[:c.Enigma.GetSigningMethodLength()/2]
	}

	if ar.GetResponseTypes().Has("token") {
		if !ar.GetClient().GetGrantTypes().Has("implicit") {
			return errors.Wrap(fosite.ErrInvalidGrant, "The client is not allowed to use the implicit grant type")
		} else if err := c.AuthorizeImplicitGrantTypeHandler.IssueImplicitAccessToken(ctx, req, ar, resp); err != nil {
			return errors.Wrap(err, err.Error())
		}
		ar.SetResponseTypeHandled("token")

		hash, err := c.Enigma.Hash([]byte(resp.GetFragment().Get("access_token")))
		if err != nil {
			return err
		}
		claims.AccessTokenHash = hash[:c.Enigma.GetSigningMethodLength()/2]
	}

	if !ar.GetGrantedScopes().Has("openid") {
		return nil
	}

	if err := c.IDTokenHandleHelper.IssueImplicitIDToken(ctx, req, ar, resp); err != nil {
		return errors.Wrap(err, err.Error())
	} else if err := c.IDTokenHandleHelper.IssueImplicitIDToken(ctx, req, ar, resp); err != nil {
		return errors.Wrap(err, err.Error())
	}

	// there is no need to check for https, because implicit flow does not require https
	// https://tools.ietf.org/html/rfc6819#section-4.4.2

	ar.SetResponseTypeHandled("id_token")
	return nil
}
