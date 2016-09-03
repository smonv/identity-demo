package client

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-errors/errors"
	"github.com/julienschmidt/httprouter"
	"github.com/ory-am/common/rand/sequence"
	"github.com/ory-am/hydra/firewall"
	"github.com/ory-am/hydra/herodot"
	"github.com/ory-am/ladon"
)

type Handler struct {
	Manager Manager
	H       herodot.Herodot
	W       firewall.Firewall
}

const (
	ClientsHandlerPath = "/clients"
)

const (
	ClientsResource = "rn:hydra:clients"
	ClientResource  = "rn:hydra:clients:%s"
	Scope           = "hydra.clients"
)

func (h *Handler) SetRoutes(r *httprouter.Router) {
	r.GET(ClientsHandlerPath, h.GetAll)
	r.POST(ClientsHandlerPath, h.Create)
	r.GET(ClientsHandlerPath+"/:id", h.Get)
	r.DELETE(ClientsHandlerPath+"/:id", h.Delete)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var c Client
	var ctx = herodot.NewContext()

	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		h.H.WriteError(ctx, w, r, errors.New(err))
		return
	}

	if _, err := h.W.TokenAllowed(ctx, h.W.TokenFromRequest(r), &ladon.Request{
		Resource: ClientsResource,
		Action:   "create",
		Context: ladon.Context{
			"owner": c.Owner,
		},
	}, Scope); err != nil {
		h.H.WriteError(ctx, w, r, err)
		return
	}

	if len(c.Secret) == 0 {
		secret, err := sequence.RuneSequence(12, []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890_-.,:;$%!&/()=?+*#<>"))
		if err != nil {
			h.H.WriteError(ctx, w, r, errors.New(err))
			return
		}
		c.Secret = string(secret)
	} else if len(c.Secret) < 6 {
		h.H.WriteError(ctx, w, r, errors.New("The client secret must be at least 6 characters long"))
	}

	secret := c.Secret
	if err := h.Manager.CreateClient(&c); err != nil {
		h.H.WriteError(ctx, w, r, err)
		return
	}

	c.Secret = secret
	h.H.WriteCreated(ctx, w, r, ClientsHandlerPath+"/"+c.GetID(), &c)
}

func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var ctx = herodot.NewContext()

	if _, err := h.W.TokenAllowed(ctx, h.W.TokenFromRequest(r), &ladon.Request{
		Resource: ClientsResource,
		Action:   "get",
	}, Scope); err != nil {
		h.H.WriteError(ctx, w, r, err)
		return
	}

	c, err := h.Manager.GetClients()
	if err != nil {
		h.H.WriteError(ctx, w, r, err)
		return
	}

	for k, cc := range c {
		cc.Secret = ""
		c[k] = cc
	}

	h.H.Write(ctx, w, r, c)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var ctx = herodot.NewContext()
	var id = ps.ByName("id")

	c, err := h.Manager.GetConcreteClient(id)
	if err != nil {
		h.H.WriteError(ctx, w, r, err)
		return
	}

	if _, err := h.W.TokenAllowed(ctx, h.W.TokenFromRequest(r), &ladon.Request{
		Resource: fmt.Sprintf(ClientResource, id),
		Action:   "get",
		Context: ladon.Context{
			"owner": c.GetOwner(),
		},
	}, Scope); err != nil {
		h.H.WriteError(ctx, w, r, err)
		return
	}

	c.Secret = ""
	h.H.Write(ctx, w, r, c)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var ctx = herodot.NewContext()
	var id = ps.ByName("id")

	if _, err := h.W.TokenAllowed(ctx, h.W.TokenFromRequest(r), &ladon.Request{
		Resource: fmt.Sprintf(ClientResource, id),
		Action:   "delete",
	}, Scope); err != nil {
		h.H.WriteError(ctx, w, r, err)
		return
	}

	if err := h.Manager.DeleteClient(id); err != nil {
		h.H.WriteError(ctx, w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
