package httpserver

import (
	"github.com/go-chi/render"
	"github.com/lukaspj/go-masterserver/pkg/lobby"
)

func MapLobbyToResponse(l lobby.Lobby) LobbyResponse {
	return LobbyResponse{
		Id:          l.Id,
		Name:        l.Name,
		Created:     l.Created,
		Subscribers: l.Subscribers,
	}
}

func MapLobbiesToResponseRenderer(ls []lobby.Lobby) []render.Renderer {
	lobbyResponses := make([]render.Renderer, len(ls), len(ls))
	for i, l := range ls {
		lobbyResponses[i] = MapLobbyToResponse(l)
	}
	return lobbyResponses
}
