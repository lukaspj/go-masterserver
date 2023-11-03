package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/render"
	"github.com/lukaspj/go-masterserver/pkg/lobby"
	"io"
	"net/http"
	"nhooyr.io/websocket"
)

type Server struct {
	LobbyService *lobby.Service
}

func NewServer(service *lobby.Service) *Server {
	return &Server{
		LobbyService: service,
	}
}

func (s *Server) ListenAndServe() error {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Use(cors.AllowAll().Handler)

	r.Get("/lobby", s.listLobbiesHandler)
	r.Post("/lobby", s.createLobbyHandler)
	r.Route("/lobby/{lobbyId}", func(r chi.Router) {
		r.Get("/", s.subscribeHandler)
		r.Post("/", s.publishHandler)
		r.Delete("/", s.deleteLobbyHandler)
	})

	return http.ListenAndServe(":3000", r)
}

type SocketConnection struct {
	conn *websocket.Conn
}

func (sc SocketConnection) WriteMessage(ctx context.Context, message lobby.Message) error {
	bytes, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return sc.conn.Write(ctx, websocket.MessageText, bytes)
}

func (s *Server) subscribeHandler(w http.ResponseWriter, r *http.Request) {
	lobbyId := chi.URLParam(r, "lobbyId")

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		s.LobbyService.Logf("%v", err)
		return
	}
	defer conn.Close(websocket.StatusInternalError, "")

	ctx := conn.CloseRead(r.Context())
	err = s.LobbyService.Subscribe(ctx, lobbyId, SocketConnection{conn})
	if errors.Is(err, context.Canceled) {
		return
	}
	if websocket.CloseStatus(err) == websocket.StatusNormalClosure ||
		websocket.CloseStatus(err) == websocket.StatusGoingAway {
		return
	}
	if err != nil {
		s.LobbyService.Logf("%v", err)
		return
	}
}

func (s *Server) publishHandler(w http.ResponseWriter, r *http.Request) {
	lobbyId := chi.URLParam(r, "lobbyId")

	body := http.MaxBytesReader(w, r.Body, 8192)
	msg, err := io.ReadAll(body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusRequestEntityTooLarge), http.StatusRequestEntityTooLarge)
		return
	}

	s.LobbyService.Publish(r.Context(), lobbyId, msg)

	render.Status(r, http.StatusAccepted)
}

func (s *Server) deleteLobbyHandler(w http.ResponseWriter, r *http.Request) {
	lobbyId := chi.URLParam(r, "lobbyId")

	err := s.LobbyService.Delete(r.Context(), lobbyId)
	if errors.Is(err, lobby.ErrNotFound) {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	render.Status(r, http.StatusOK)
}

func (s *Server) createLobbyHandler(w http.ResponseWriter, r *http.Request) {
	data := CreateLobbyRequest{}
	if err := render.Bind(r, &data); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	lobbyId, err := s.LobbyService.Create(r.Context(), data.Name)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	render.JSON(w, r, lobbyId)
}

func (s *Server) listLobbiesHandler(w http.ResponseWriter, r *http.Request) {
	lobbies, err := s.LobbyService.List(r.Context())
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	render.JSON(w, r, lobbies)
}
