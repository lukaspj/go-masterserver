package lobby

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"golang.org/x/exp/maps"
	"time"
)

var ErrNotFound = errors.New("not found")
var ErrExists = errors.New("already exists")

type RepoLobby struct {
	Id      string
	Name    string
	Created time.Time
}

type Repo interface {
	List() ([]RepoLobby, error)
	Get(id string) (RepoLobby, error)
	Add(lobby RepoLobby) (RepoLobby, error)
	Delete(id string) error
	GetMessageStream(id string) (MessageStream, error)
}

type InMemoryRepo struct {
	Lobbies map[string]RepoLobby
	streams map[string]*InMemoryMessageStream
}

func NewInMemoryRepo() *InMemoryRepo {
	return &InMemoryRepo{
		Lobbies: make(map[string]RepoLobby),
		streams: make(map[string]*InMemoryMessageStream),
	}
}

func (m *InMemoryRepo) List() ([]RepoLobby, error) {
	return maps.Values(m.Lobbies), nil
}

func (m *InMemoryRepo) Get(id string) (RepoLobby, error) {
	if l, ok := m.Lobbies[id]; ok {
		return l, nil
	}
	return RepoLobby{}, fmt.Errorf("failed to get error with id %s: %w", id, ErrNotFound)
}

func (m *InMemoryRepo) Add(lobby RepoLobby) (RepoLobby, error) {
	if lobby.Id == "" {
		lobby.Id = uuid.NewString()
	}
	if lobby.Created.IsZero() {
		lobby.Created = time.Now()
	}
	if _, ok := m.Lobbies[lobby.Id]; ok {
		return lobby, fmt.Errorf("failed to add error with id %s: %w", lobby.Id, ErrExists)
	}
	m.Lobbies[lobby.Id] = lobby
	return lobby, nil
}

func (m *InMemoryRepo) Delete(id string) error {
	if _, ok := m.Lobbies[id]; !ok {
		return ErrNotFound
	}
	delete(m.Lobbies, id)
	if stream, ok := m.streams[id]; ok {
		return stream.Close()
	}
	return nil
}

func (m *InMemoryRepo) GetMessageStream(id string) (MessageStream, error) {
	_, err := m.Get(id)
	if err != nil {
		return nil, err
	}

	if _, ok := m.streams[id]; !ok {
		m.streams[id] = NewInMemoryMessageStream()
	}

	return m.streams[id], nil
}

var _ Repo = &InMemoryRepo{}
