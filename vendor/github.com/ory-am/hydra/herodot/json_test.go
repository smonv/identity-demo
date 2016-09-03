package herodot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-errors/errors"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

func TestWriteError(t *testing.T) {
	var j jsonError

	h := JSON{}
	r := mux.NewRouter()
	r.HandleFunc("/do", func(w http.ResponseWriter, r *http.Request) {
		h.WriteError(context.Background(), w, r, errors.New(ErrNotFound))
	})
	ts := httptest.NewServer(r)

	resp, err := http.Get(ts.URL + "/do")
	require.Nil(t, err)

	require.Nil(t, json.NewDecoder(resp.Body).Decode(&j))
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Equal(t, j.Error, ErrNotFound.Error())
	assert.Equal(t, j.Code, ErrNotFound.Code)
	assert.NotEmpty(t, j.RequestID)
}

func TestWriteErrorCode(t *testing.T) {
	var j jsonError

	h := JSON{}
	r := mux.NewRouter()
	r.HandleFunc("/do", func(w http.ResponseWriter, r *http.Request) {
		h.WriteErrorCode(context.Background(), w, r, http.StatusBadRequest, errors.New(ErrNotFound))
	})
	ts := httptest.NewServer(r)

	resp, err := http.Get(ts.URL + "/do")
	require.Nil(t, err)

	require.Nil(t, json.NewDecoder(resp.Body).Decode(&j))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, j.Error, ErrNotFound.Error())
	assert.NotEmpty(t, j.RequestID)
}

func TestWriteJSON(t *testing.T) {
	foo := map[string]string{"foo": "bar"}

	h := JSON{}
	r := mux.NewRouter()
	r.HandleFunc("/do", func(w http.ResponseWriter, r *http.Request) {
		h.Write(context.Background(), w, r, &foo)
	})
	ts := httptest.NewServer(r)

	resp, err := http.Get(ts.URL + "/do")
	require.Nil(t, err)

	result := map[string]string{}
	require.Nil(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, foo["foo"], result["foo"])
}

func TestWriteCreatedJSON(t *testing.T) {
	foo := map[string]string{"foo": "bar"}

	h := JSON{}
	r := mux.NewRouter()
	r.HandleFunc("/do", func(w http.ResponseWriter, r *http.Request) {
		h.WriteCreated(context.Background(), w, r, "/new", &foo)
	})
	ts := httptest.NewServer(r)

	resp, err := http.Get(ts.URL + "/do")
	require.Nil(t, err)

	result := map[string]string{}
	require.Nil(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, foo["foo"], result["foo"])
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, "/new", resp.Header.Get("Location"))
}

func TestWriteCodeJSON(t *testing.T) {
	foo := map[string]string{"foo": "bar"}

	h := JSON{}
	r := mux.NewRouter()
	r.HandleFunc("/do", func(w http.ResponseWriter, r *http.Request) {
		h.WriteCode(context.Background(), w, r, 400, &foo)
	})
	ts := httptest.NewServer(r)

	resp, err := http.Get(ts.URL + "/do")
	require.Nil(t, err)

	result := map[string]string{}
	require.Nil(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, foo["foo"], result["foo"])
	assert.Equal(t, 400, resp.StatusCode)
}
