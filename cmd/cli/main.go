package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/lukaspj/go-masterserver/pkg/tcp"
	"log"
	"net"
	"strings"
	"time"

	"github.com/rivo/tview"
)

func main() {
	conn, err := net.Dial("tcp", ":3001")
	if err != nil {
		log.Fatal(err)
	}

	doneChan := make(chan error)
	byteOrder := binary.LittleEndian

	app := tview.NewApplication()
	container := tview.NewFlex().SetDirection(tview.FlexRow)
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetChangedFunc(func() {
			app.Draw()
		}).
		SetScrollable(true).
		ScrollToEnd()

	logPrintf := func(format string, a ...any) {
		textView.Write([]byte(fmt.Sprintf(format, a...)))
	}

	container.AddItem(textView, 0, 1, false)

	inputField := tview.NewInputField().
		SetLabel("Write Command ").
		SetAutocompleteFunc(func(currentText string) (entries []string) {
			if strings.ContainsRune(currentText, ' ') {
				return nil
			}

			return []string{
				"list ",
				"create ",
				"join ",
				"send ",
			}
		})
	inputField.
		SetDoneFunc(func(key tcell.Key) {
			if strings.HasPrefix(inputField.GetText(), "list") {
				_, err := conn.Write([]byte{byte(tcp.LIST_LOBBIES), '\t'})
				if err != nil {
					logPrintf("[error]: %+v", err)
				}
			}
			if strings.HasPrefix(inputField.GetText(), "create") {
				name := strings.TrimSpace(inputField.GetText()[6:])
				if strings.ContainsRune(name, ' ') {
					logPrintf("Invalid Input to create")
					return
				}
				_, err := conn.Write([]byte{byte(tcp.CREATE_LOBBY)})
				if err != nil {
					logPrintf("[error]: %+v", err)
				}
				_, err = conn.Write([]byte(name))
				if err != nil {
					logPrintf("[error]: %+v", err)
				}
				_, err = conn.Write([]byte{'\t'})
				if err != nil {
					logPrintf("[error]: %+v", err)
				}
			}
			if strings.HasPrefix(inputField.GetText(), "join") {
				id := strings.TrimSpace(inputField.GetText()[4:])
				if strings.ContainsRune(id, ' ') {
					logPrintf("Invalid Input to create")
					return
				}
				_, err := conn.Write([]byte{byte(tcp.JOIN_LOBBY)})
				if err != nil {
					logPrintf("[error]: %+v", err)
				}
				_, err = conn.Write([]byte(id))
				if err != nil {
					logPrintf("[error]: %+v", err)
				}
				_, err = conn.Write([]byte{'\t'})
				if err != nil {
					logPrintf("[error]: %+v", err)
				}
			}
			if strings.HasPrefix(inputField.GetText(), "send") {
				message := strings.TrimSpace(inputField.GetText()[4:])
				if strings.ContainsRune(message, ' ') {
					logPrintf("Invalid Input to create\n")
					return
				}

				_, err := conn.Write([]byte{byte(tcp.SEND_MESSAGE)})
				if err != nil {
					logPrintf("[error]: %+v\n", err)
				}
				_, err = conn.Write([]byte(message))
				if err != nil {
					logPrintf("[error]: %+v\n", err)
				}
				_, err = conn.Write([]byte{'\t'})
				if err != nil {
					logPrintf("[error]: %+v\n", err)
				}
			}
			inputField.SetText("")
		})
	dropdown := tview.NewDropDown().SetLabel("Select an option").
		SetOptions([]string{"list"}, nil)
	dropdown.SetSelectedFunc(func(text string, index int) {
		if text == "list" {
			_, err := conn.Write([]byte{byte(tcp.LIST_LOBBIES), '\t'})
			if err != nil {
				log.Printf("[error]: %+v", err)
			}
		}
	})
	container.AddItem(inputField, 1, 1, true)

	inputContainer := tview.NewFlex()
	listLobbiesBtn := tview.NewButton("List Lobbies").SetSelectedFunc(func() {
		_, err := conn.Write([]byte{byte(tcp.LIST_LOBBIES), '\t'})
		if err != nil {
			log.Printf("[error]: %+v", err)
		}
	})
	inputContainer.AddItem(listLobbiesBtn, 0, 1, true)
	createLobbyBtn := tview.NewButton("Create Lobby").SetSelectedFunc(func() {
		_, err := conn.Write([]byte{byte(tcp.LIST_LOBBIES), '\t'})
		if err != nil {
			log.Printf("[error]: %+v", err)
		}
	})
	inputContainer.AddItem(listLobbiesBtn, 0, 1, true)
	inputContainer.AddItem(createLobbyBtn, 0, 1, false)

	go func(doneChan chan<- error) {
		reader := bufio.NewReader(conn)
		for {
			message, err := reader.ReadBytes('\t')
			if err != nil {
				logPrintf("Error! %+v\n", err)
				doneChan <- err
				return
			}

			if tcp.TCP_RESPONSE(message[0]) == tcp.LOBBY_LIST {
				logPrintf("->: LOBBY LIST\n")
				if len(message) == 2 {
					logPrintf("-->: NO LOBBIES\n")
					continue
				}

				messageReader := bytes.NewBuffer(message[:len(message)-1])
				for messageReader.Len() > 0 {
					id, err := messageReader.ReadString(0x0)
					if err != nil {
						logPrintf("-->: %+v\n", err)
						break
					}
					name, err := messageReader.ReadString(0x0)
					if err != nil {
						logPrintf("-->: %+v\n", err)
						break
					}
					t := time.Time{}
					err = t.UnmarshalBinary(messageReader.Next(15))
					if err != nil {
						logPrintf("-->: %+v\n", err)
						break
					}

					var subscribers uint32
					err = binary.Read(messageReader, byteOrder, &subscribers)
					if err != nil {
						logPrintf("-->: %+v\n", err)
						break
					}
					logPrintf("-->: ID: %s, Name: %s, Ts: %s, Subscribers: %d\n", id, name, t, subscribers)
				}
			} else if tcp.TCP_RESPONSE(message[0]) == tcp.LOBBY_CREATED {
				messageReader := bytes.NewBuffer(message)
				logPrintf("-->: MESSAGE SIZE: %d\n", messageReader.Len())
				id, err := messageReader.ReadString(0x0)
				if err != nil {
					logPrintf("-->: %+v\n", err)
					break
				}
				logPrintf("-->: ID: %s\n", id)
				messageReader.Next(1)
			} else if tcp.TCP_RESPONSE(message[0]) == tcp.LOBBY_MESSAGE {
				messageReader := bytes.NewBuffer(message[1:])
				msgType, err := messageReader.ReadString(0x0)
				if err != nil {
					logPrintf("-->: %+v\n", err)
					break
				}
				msgType = strings.TrimSpace(msgType[:len(msgType)-1])
				if msgType == "text" {
					t := time.Time{}
					err = t.UnmarshalBinary(messageReader.Next(15))
					if err != nil {
						logPrintf("-->: %+v\n", err)
						break
					}
					msg, err := messageReader.ReadString(0x0)
					if err != nil {
						logPrintf("-->: %+v\n", err)
						break
					}
					logPrintf("->: <text> %s - %s\n", t, msg)
				} else if msgType == "meta" {
					metaId, err := messageReader.ReadString(0x0)
					if err != nil {
						logPrintf("-->: %+v\n", err)
						break
					}
					metaName, err := messageReader.ReadString(0x0)
					if err != nil {
						logPrintf("-->: %+v\n", err)
						break
					}
					var metaSubscribers int32
					err = binary.Read(messageReader, byteOrder, &metaSubscribers)
					if err != nil {
						logPrintf("-->: %+v\n", err)
						break
					}
					logPrintf("->: <meta> %s, %s, %d\n", metaId, metaName, metaSubscribers)
				} else {
					logPrintf("-->: unknown message type %s\n", msgType)
				}
				messageReader.Next(1)
			} else {
				logPrintf("unknown command: %d\n", message[0])
			}
		}
	}(doneChan)

	if err := app.SetRoot(container, true).SetFocus(inputField).Run(); err != nil {
		log.Fatal(err)
	}
}
