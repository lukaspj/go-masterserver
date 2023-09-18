package lobby

import (
	"context"
	"golang.org/x/exp/maps"
	"sync"
	"time"
)

type MessageType string

const (
	TextMessageType MessageType = "text"
	MetaMessageType MessageType = "meta"
)

type TextMessage struct {
	Content string    `json:"content"`
	Created time.Time `json:"created"`
}

type MetaMessage struct {
	Name        string `json:"name"`
	Id          string `json:"id"`
	Subscribers int    `json:"subscribers"`
}

type Message struct {
	Type MessageType `json:"type"`
	Text TextMessage `json:"text"`
	Meta MetaMessage `json:"meta"`
}

type MessageStream interface {
	Publish(ctx context.Context, msg Message) error
	Subscribe(ctx context.Context) <-chan Message
	SubscriberCount() int
}

var inMemoryMessageHistorySize = 10
var inMemorySubscriberBufferSize = 20

type InMemoryMessageStream struct {
	messageHistory []Message
	input          chan Message
	subscribers    map[chan Message]any
	subscribersMu  sync.Mutex
}

type InMemoryMessageStreamSubscription struct {
	cb func(str string) error
}

func (s *InMemoryMessageStream) SubscriberCount() int {
	return len(s.subscribers)
}

func (s *InMemoryMessageStream) Publish(ctx context.Context, msg Message) error {
	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()

	s.input <- msg
	return nil
}

func (s *InMemoryMessageStream) Subscribe(ctx context.Context) <-chan Message {
	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()

	stream := make(chan Message, inMemorySubscriberBufferSize)
	s.subscribers[stream] = true
	for _, msg := range s.messageHistory {
		stream <- msg
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				s.subscribersMu.Lock()
				delete(s.subscribers, stream)
				close(stream)
				s.subscribersMu.Unlock()
				return
			}
		}
	}()

	return stream
}

var _ MessageStream = &InMemoryMessageStream{}

func (s *InMemoryMessageStream) Close() error {
	close(s.input)
	return nil
}

func NewInMemoryMessageStream() *InMemoryMessageStream {

	messageStream := &InMemoryMessageStream{
		messageHistory: make([]Message, 0, inMemoryMessageHistorySize),
		input:          make(chan Message),
		subscribers:    make(map[chan Message]any),
	}

	go func() {
		for {
			select {
			case msg := <-messageStream.input:

				messageStream.subscribersMu.Lock()

				// only the last n entries are kept
				n := inMemoryMessageHistorySize // n > 0 and small
				if len(messageStream.messageHistory) >= n {
					copy(messageStream.messageHistory, messageStream.messageHistory[len(messageStream.messageHistory)-n+1:])
					messageStream.messageHistory = messageStream.messageHistory[:n-1]
				}
				messageStream.messageHistory = append(messageStream.messageHistory, msg)

				for _, subscription := range maps.Keys(messageStream.subscribers) {
					subscription <- msg
				}

				messageStream.subscribersMu.Unlock()
			}
		}
	}()

	return messageStream
}
