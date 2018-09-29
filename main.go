package main

import (
	"encoding/json"
	"flag"
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

var addr = flag.String("addr", ":8080", "http service address")

type chatRoom struct {
	clients     map[*client]bool
	toBroadcast chan []byte
	joining     chan *client
	leaving     chan *client
}

type client struct {
	username  string
	userID    string
	socket    *websocket.Conn
	forClient chan []byte
}

type message struct {
	Username string `json:"username,omitempty"`
	UserID   string `json:"userID,omitempty"`
	Text     string `json:"text,omitempty"`
	Type     string `json:"type,omitempty"`
}

func (cr *chatRoom) broadcast(jmsg []byte, ignore *client) {
	for cli := range cr.clients {
		if cli != ignore {
			select {
			case cli.forClient <- jmsg:
			default:
				go func() {
					cr.leaving <- cli
				}()
			}
		}
	}
}

func (cr *chatRoom) run() {
	for {
		select {
		// joining clients are added to map of clients
		case cli := <-cr.joining:
			cr.clients[cli] = true
			jsonMsg, _ := json.Marshal(&message{Username: "SYSTEM", Text: cli.username + " has connected."})
			cr.broadcast(jsonMsg, nil)
		case cli := <-cr.leaving:
			if _, ok := cr.clients[cli]; ok {
				close(cli.forClient)
				delete(cr.clients, cli)
				jsonMsg, _ := json.Marshal(&message{Username: "SYSTEM", Text: cli.username + " has disconnected."})
				cr.broadcast(jsonMsg, nil)
			}
		case msg := <-cr.toBroadcast:
			cr.broadcast(msg, nil)
		}
	}
}

func (cli *client) read() {
	defer func() {
		room.leaving <- cli
		cli.socket.Close()
	}()

	for {
		_, msgIn, err := cli.socket.ReadMessage()
		if err != nil {
			room.leaving <- cli
			cli.socket.Close()
			break
		}
		msg := message{UserID: cli.userID}
		json.Unmarshal(msgIn, &msg)

		if msg.Type == "join" {
			cli.username = msg.Username
			room.joining <- cli
		} else {
			jmsg, _ := json.Marshal(&msg)
			room.toBroadcast <- jmsg
		}
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
		userID:    id.String(),
		socket:    conn,
		forClient: make(chan []byte),
	}

	go cli.read()
	go cli.write()
}

func logMux(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		log.Println(req.RemoteAddr, req.Method, req.URL)
		handler.ServeHTTP(res, req)
	})
}

func main() {
	flag.Parse()
	fmt.Println("Starting server...")
	go room.run()
	http.Handle("/", http.FileServer(http.Dir("./")))
	http.HandleFunc("/ws", serveConn)
	http.ListenAndServe(*addr, logMux(http.DefaultServeMux))
}
