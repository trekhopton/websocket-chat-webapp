package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	uuid "github.com/satori/go.uuid"
)

var room = chatRoom{
	clients:     make(map[*client]bool),
	toBroadcast: make(chan []byte),
	joining:     make(chan *client),
	leaving:     make(chan *client),
}

type chatRoom struct {
	clients     map[*client]bool
	toBroadcast chan []byte
	joining     chan *client
	leaving     chan *client
}

type client struct {
	id        string
	socket    *websocket.Conn
	forClient chan []byte
}

type message struct {
	Sender    string `json:"sender,omitempty"`
	Recipient string `json:"recipient,omitempty"`
	Text      string `json:"content,omitempty"`
}

func (cr *chatRoom) broadcast(jmsg []byte, ignore *client) {
	for cli := range cr.clients {
		if cli != ignore {
			cli.forClient <- jmsg
		}
	}
}

func (cr *chatRoom) run() {
	for {
		select {
		// joining clients are added to map of clients
		case cli := <-cr.joining:
			cr.clients[cli] = true
			jsonMsg, _ := json.Marshal(&message{Text: "/A user has connected."})
			cr.broadcast(jsonMsg, cli)
		case cli := <-cr.leaving:
			if _, ok := cr.clients[cli]; ok {
				close(cli.forClient)
				delete(cr.clients, cli)
				jsonMsg, _ := json.Marshal(&message{Text: "/A user has disconnected."})
				cr.broadcast(jsonMsg, cli)
			}
		case msg := <-cr.toBroadcast:
			for cli := range cr.clients {
				select {
				case cli.forClient <- msg:
				default: // go routine, cli should be put into leaving?
					close(cli.forClient)
					delete(cr.clients, cli)
				}
			}
		}
	}
}

func (cli *client) read() {
	defer func() {
		room.leaving <- cli
		cli.socket.Close()
	}()

	for {
		_, msgTxt, err := cli.socket.ReadMessage()
		if err != nil {
			room.leaving <- cli
			cli.socket.Close()
			break
		}
		jmsg, _ := json.Marshal(&message{Sender: cli.id, Text: string(msgTxt)})
		room.toBroadcast <- jmsg
	}
}

func (cli *client) write() {
	defer func() {
		cli.socket.Close()
	}()

	for {
		select {
		case msg, ok := <-cli.forClient:
			if !ok {
				cli.socket.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			cli.socket.WriteMessage(websocket.TextMessage, msg)
		}
	}
}

func serveConn(res http.ResponseWriter, req *http.Request) {
	upgrader := &websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	conn, err := upgrader.Upgrade(res, req, nil)

	if err != nil {
		http.NotFound(res, req)
		return
	}

	id, _ := uuid.NewV4()
	cli := &client{
		id:        id.String(),
		socket:    conn,
		forClient: make(chan []byte),
	}

	room.joining <- cli

	go cli.read()
	go cli.write()
}

func serveSite(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.ServeFile(w, r, "index.html")
}

func main() {
	fmt.Println("Starting application...")
	go room.run()
	http.HandleFunc("/", serveSite)
	http.HandleFunc("/ws", serveConn)
	http.ListenAndServe(":8765", nil)
}
