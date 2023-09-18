package lobby

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"golang.org/x/time/rate"

	"nhooyr.io/websocket"
)

type Lobby struct {
	Id          string
	Name        string
	Created     time.Time
	Subscribers int
}

// Service enables broadcasting to a set of subscribers.
type Service struct {
	// publishLimiter controls the rate limit applied to the Publish endpoint.
	//
	// Defaults to one Publish every 100ms with a burst of 8.
	publishLimiter *rate.Limiter

	// Logf controls where logs are sent.
	// Defaults to log.Printf.
	Logf func(f string, v ...interface{})

	repo Repo
}

// NewService constructs a chatServer with the defaults.
func NewService() *Service {
	cs := &Service{
		Logf:           log.Printf,
		publishLimiter: rate.NewLimiter(rate.Every(time.Millisecond*100), 8),
		repo:           NewInMemoryRepo(),
	}

	return cs
}

// Subscribe subscribes the given WebSocket to all broadcast messages.
// It creates a subscriber with a buffered msgs chan to give some room to slower
// connections and then registers the subscriber. It then listens for all messages
// and writes them to the WebSocket. If the context is cancelled or
// an error occurs, it returns and deletes the subscription.
//
// It uses CloseRead to keep reading from the connection to process control
// messages and cancel the context if the connection drops.
func (ls *Service) Subscribe(id string, ctx context.Context, conn *websocket.Conn) error {
	ctx = conn.CloseRead(ctx)
	messageStream, err := ls.repo.GetMessageStream(id)
	if err != nil {
		return err
	}

	lobby, err := ls.repo.Get(id)

	messageChan := messageStream.Subscribe(ctx)

	bytes, err := json.Marshal(Message{
		Type: MetaMessageType,
		Meta: MetaMessage{
			Name:        lobby.Name,
			Id:          lobby.Id,
			Subscribers: messageStream.SubscriberCount(),
		},
	})

	writeTimeout(ctx, time.Second*5, conn, bytes)

	for msg := range messageChan {
		bytes, err = json.Marshal(msg)
		if err != nil {
			return err
		}
		err = writeTimeout(ctx, time.Second*5, conn, bytes)
		if err != nil {
			return err
		}
	}

	// Handle: conn.Close(websocket.StatusPolicyViolation, "connection too slow to keep up with messages")

	return ctx.Err()
}

// Publish publishes the msg to all subscribers.
// It never blocks and so messages to slow subscribers
// are dropped.
func (ls *Service) Publish(ctx context.Context, id string, msg []byte) {
	ls.publishLimiter.Wait(ctx)

	stream, err := ls.repo.GetMessageStream(id)
	if err != nil {
		return
	}

	stream.Publish(ctx, Message{
		Type: TextMessageType,
		Text: TextMessage{
			Content: string(msg),
			Created: time.Now(),
		},
	})
}

func (ls *Service) Delete(_ context.Context, id string) error {
	return ls.repo.Delete(id)
}

func (ls *Service) Create(ctx context.Context, name string) (string, error) {
	lobby, err := ls.repo.Add(RepoLobby{Name: name})
	return lobby.Id, err
}

func (ls *Service) List(context.Context) ([]Lobby, error) {
	repoLobbies, err := ls.repo.List()
	if err != nil {
		return nil, err
	}

	lobbies := make([]Lobby, len(repoLobbies), len(repoLobbies))
	for idx, repoLobby := range repoLobbies {
		stream, err := ls.repo.GetMessageStream(repoLobby.Id)
		if err != nil {
			return nil, err
		}

		lobby := Lobby{
			Id:          repoLobby.Id,
			Name:        repoLobby.Name,
			Created:     repoLobby.Created,
			Subscribers: stream.SubscriberCount(),
		}
		lobbies[idx] = lobby
	}

	return lobbies, nil
}

func writeTimeout(ctx context.Context, timeout time.Duration, c *websocket.Conn, msg []byte) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return c.Write(ctx, websocket.MessageText, msg)
}
