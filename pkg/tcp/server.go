package tcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"github.com/lukaspj/go-masterserver/pkg/lobby"
	"log"
	"net"
	"strings"
)

type Server struct {
	LobbyService *lobby.Service
}

func NewServer(service *lobby.Service) *Server {
	return &Server{
		LobbyService: service,
	}
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	tcpListener, err := net.Listen("tcp", ":3001")
	if err != nil {
		return err
	}

	defer tcpListener.Close()

	for {
		conn, err := tcpListener.Accept()
		if err != nil {
			return err
		}

		s.subscribe(ctx, conn, s.LobbyService)
	}
}

type Subscriber struct {
	Conn         net.Conn
	LobbyService *lobby.Service
	byteOrder    binary.ByteOrder
	lobbyId      string
}

type TCP_COMMAND byte
type TCP_RESPONSE byte

const (
	CLIENT_ERROR TCP_COMMAND = iota + 1
	LIST_LOBBIES
	CREATE_LOBBY
	JOIN_LOBBY
	SEND_MESSAGE
)

const (
	SERVER_ERROR TCP_RESPONSE = iota + 1
	LOBBY_LIST
	LOBBY_CREATED
	LOBBY_MESSAGE
)

func (s *Subscriber) Listen(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	for {
		netData, err := bufio.NewReader(s.Conn).ReadBytes('\t')
		if err != nil {
			log.Printf("failed to read bytes for subscriber due to: %+v", err)
			break
		}

		if TCP_COMMAND(netData[0]) == LIST_LOBBIES {
			log.Printf("list lobbies received")
			err = s.listLobbies(ctx)
		} else if TCP_COMMAND(netData[0]) == CREATE_LOBBY {
			log.Printf("create lobby received")
			err = s.createLobby(ctx, netData[1:])
		} else if TCP_COMMAND(netData[0]) == JOIN_LOBBY {
			log.Printf("join lobby received")
			err = s.joinLobby(ctx, netData[1:])
		} else if TCP_COMMAND(netData[0]) == SEND_MESSAGE {
			log.Printf("send message received")
			err = s.sendMessage(ctx, netData[1:])
		} else {
			log.Printf("unknown command")
		}

		if err != nil {
			log.Printf("error occured while parsing received data from client: %+v", err)
		}
	}
	cancel()
}

func (s *Subscriber) listLobbies(ctx context.Context) error {
	list, err := s.LobbyService.List(context.Background())
	if err != nil {
		return err
	}

	resp := new(bytes.Buffer)

	err = binary.Write(resp, s.byteOrder, []byte{
		byte(LOBBY_LIST),
	})
	if err != nil {
		return err
	}

	for _, l := range list {
		err = binary.Write(resp, s.byteOrder, []byte(l.Id))
		err = binary.Write(resp, s.byteOrder, byte(0x0))
		err = binary.Write(resp, s.byteOrder, []byte(l.Name))
		err = binary.Write(resp, s.byteOrder, byte(0x0))
		timeBytes, err := l.Created.MarshalBinary()
		if err != nil {
			return err
		}
		err = binary.Write(resp, s.byteOrder, timeBytes)
		err = binary.Write(resp, s.byteOrder, uint32(l.Subscribers))
	}

	err = binary.Write(resp, s.byteOrder, byte('\t'))

	_, err = s.Conn.Write(resp.Bytes())
	if err != nil {
		return err
	}
	return nil
}

func (s *Subscriber) createLobby(ctx context.Context, data []byte) error {
	name := strings.TrimSpace(string(data))
	if name == "" {
		return fmt.Errorf("invalid input, cannot be empty name")
	}
	lobbyId, err := s.LobbyService.Create(ctx, name)
	if err != nil {
		return err
	}

	resp := new(bytes.Buffer)

	err = binary.Write(resp, s.byteOrder, []byte{
		byte(LOBBY_CREATED),
	})
	if err != nil {
		return err
	}

	err = binary.Write(resp, s.byteOrder, []byte(lobbyId))
	if err != nil {
		return err
	}
	err = binary.Write(resp, s.byteOrder, byte(0x0))
	if err != nil {
		return err
	}

	err = binary.Write(resp, s.byteOrder, byte('\t'))
	if err != nil {
		return err
	}

	_, err = s.Conn.Write(resp.Bytes())
	if err != nil {
		return err
	}

	return nil
}

func (s *Subscriber) joinLobby(ctx context.Context, data []byte) error {
	lobbyId := strings.TrimSpace(string(data))
	s.lobbyId = lobbyId
	go func() {
		err := s.LobbyService.Subscribe(ctx, lobbyId, s)
		if err != nil {
			log.Printf("failed to subscribe due to %+v", err)
		}
	}()
	return nil
}

func (s *Subscriber) WriteMessage(ctx context.Context, msg lobby.Message) error {
	resp := new(bytes.Buffer)

	err := binary.Write(resp, s.byteOrder, []byte{
		byte(LOBBY_MESSAGE),
	})
	if err != nil {
		return err
	}

	_, err = resp.Write([]byte(msg.Type))
	if err != nil {
		return err
	}
	err = binary.Write(resp, s.byteOrder, byte(0x0))
	if err != nil {
		return err
	}

	switch msg.Type {
	case lobby.TextMessageType:
		created, err := msg.Text.Created.MarshalBinary()
		if err != nil {
			return err
		}
		err = binary.Write(resp, s.byteOrder, created)
		if err != nil {
			return err
		}
		err = binary.Write(resp, s.byteOrder, []byte(msg.Text.Content))
		err = binary.Write(resp, s.byteOrder, byte(0x0))
		if err != nil {
			return err
		}
		break
	case lobby.MetaMessageType:
		err = binary.Write(resp, s.byteOrder, []byte(msg.Meta.Id))
		err = binary.Write(resp, s.byteOrder, byte(0x0))
		if err != nil {
			return err
		}
		err = binary.Write(resp, s.byteOrder, []byte(msg.Meta.Name))
		err = binary.Write(resp, s.byteOrder, byte(0x0))
		if err != nil {
			return err
		}
		err = binary.Write(resp, s.byteOrder, int32(msg.Meta.Subscribers))
		if err != nil {
			return err
		}
		break
	}

	err = binary.Write(resp, s.byteOrder, byte('\t'))
	if err != nil {
		return err
	}

	_, err = s.Conn.Write(resp.Bytes())
	if err != nil {
		return err
	}

	return nil
}

func (s *Subscriber) sendMessage(ctx context.Context, data []byte) error {
	s.LobbyService.Publish(ctx, s.lobbyId, []byte(strings.TrimSpace(string(data))))
	return nil
}

func (s *Server) subscribe(ctx context.Context, conn net.Conn, service *lobby.Service) {
	sub := &Subscriber{conn, service, binary.LittleEndian, ""}
	go sub.Listen(ctx)
}
