package httpserver

import (
	"github.com/go-chi/render"
	"net/http"
	"time"
)

type LobbyResponse struct {
	Id          string    `json:"id"`
	Name        string    `json:"name"`
	Created     time.Time `json:"created"`
	Subscribers int       `json:"subscribers"`
}

func (l LobbyResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

var _ render.Renderer = LobbyResponse{}

type CreateLobbyRequest struct {
	Name string `json:"name"`
}

func (c CreateLobbyRequest) Bind(r *http.Request) error {
	return nil
}

var _ render.Binder = CreateLobbyRequest{}
