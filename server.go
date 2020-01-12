package main

import (
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

const (
	// MaxConnections is the Maximum number of sockets that
	// should exist at any given time
	MaxConnections = 100
	// EventBufferSize is the buffer size of the channels
	// holding events
	EventBufferSize = 10
)

// Server serves a static webpage and handles the creation,
// maintainance and communications of websockets.
type Server struct {
	lock           sync.RWMutex
	clients        map[int]*websocket.Conn
	nextClientID   int
	connections    int
	maxConnections int

	// staticAddress is the path to the root of the static
	// content to be served
	staticAddress string

	serverEvents chan ServerEvent
	clientEvents chan ClientEvent

	upgrader websocket.Upgrader
}

// ServerEvent is triggered when a command should be sent to
// all connected sockets
type ServerEvent struct {
	WSCommand string
}

// ClientEvent is when a client has messaged the server
// via a WebSocket. Usually requesting something to happen
// on the server
type ClientEvent struct {
	ClientID  int
	WsCommand string
}

// NewServer creates a new server
func NewServer(staticAddress string) (*Server, error) {
	if _, err := os.Stat(staticAddress); os.IsNotExist(err) {
		return nil, errors.Wrap(err, "couldn't find engines root directory")
	} else if err != nil {
		return nil, errors.Wrap(err, "couldn't find engines root directory")
	}
	return &Server{
		clients:        make(map[int]*websocket.Conn),
		maxConnections: MaxConnections,
		staticAddress:  staticAddress,
		serverEvents:   make(chan ServerEvent, EventBufferSize),
		clientEvents:   make(chan ClientEvent, EventBufferSize),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(*http.Request) bool { return true },
		},
	}, nil
}

// Start starts route handles for http requests, starts the
// server event system and serves a static webpage and a
// WebSocket endpoint. Once started, the server will run
// until the program is terminated.
func (s *Server) Start() error {
	// Setting up routes
	http.HandleFunc("/", s.staticHandler)
	http.HandleFunc("/ws", s.wsEndpoint)
	// Listening to server events
	go s.serverEventListener()
	// Serving content to clients
	return http.ListenAndServe(":8080", nil)
}

// TriggerEvent is used to send a command to all WebSocket connections
func (s *Server) TriggerEvent(evt ServerEvent) {
	s.serverEvents <- evt
}

// ClientEvent returns a ClientEvent when a client sends
// a message through it's WebSocket connection. Usually
// requesting something to happen on the server.
func (s *Server) ClientEvent() (ClientEvent, bool) {
	evt, ok := <-s.clientEvents
	return evt, ok
}

// Respond is used to send a message back to a client
// after a request from ClientEvent.
func (s *Server) Respond(evt ClientEvent, response string) {
	s.lock.RLock()
	client, ok := s.clients[evt.ClientID]
	if !ok {
		s.lock.RUnlock()
		return
	}
	client.WriteMessage(websocket.TextMessage, []byte(response))
	s.lock.RUnlock()
}

// staticHandler serves any of the static content
// on the server e.g. html, css, js, images
func (s *Server) staticHandler(w http.ResponseWriter, r *http.Request) {
	// Correcting path
	path := s.staticAddress + "/" + r.URL.Path[1:]
	if r.URL.Path[1:] == "" {
		path += "index.html"
	}
	// Getting file contents
	page, err := ioutil.ReadFile(path)
	if err != nil {
		http.NotFound(w, r)
	}
	// Adding header data
	var contentType string
	if strings.HasSuffix(path, ".html") {
		w.Header().Add("Content-Type", "text/html")
	} else if strings.HasSuffix(path, ".css") {
		w.Header().Add("Content-Type", "text/css")
	} else if strings.HasSuffix(path, ".js") {
		w.Header().Add("Content-Type", "text/javascript")
	}
	w.Header().Add("Content-Type", contentType)
	// Responding with content
	_, err = w.Write(page)
}

// wsEndpoint handles http requests trying to establish
// a WebSocket connection
func (s *Server) wsEndpoint(w http.ResponseWriter, r *http.Request) {
	// Checking if there are too many connections
	s.lock.RLock()
	allowConnection := s.connections < s.maxConnections
	s.lock.RUnlock()
	if !allowConnection {
		return
	}
	// Upgrading HTTP connection to WebSocket
	ws, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	// Adding reference of WebSocket to Server
	s.lock.Lock()
	clientID := s.nextClientID
	s.clients[clientID] = ws
	s.connections++
	s.nextClientID++
	s.lock.Unlock()

	// Listening to the socket
	go s.socketListener(clientID, ws)
}

// socketListener listens to WebSocket connections for
// requests from clients
func (s *Server) socketListener(clientID int, conn *websocket.Conn) {
	for {
		_, p, err := conn.ReadMessage()
		if err != nil {
			break
		}
		s.clientEvents <- ClientEvent{
			ClientID:  clientID,
			WsCommand: string(p),
		}
	}
	s.removeClient(clientID)
}

// removeClient removes a client from the
// server connection pool
func (s *Server) removeClient(clientID int) {
	// Getting reference to client
	s.lock.RLock()
	client, ok := s.clients[clientID]
	s.lock.RUnlock()
	if !ok {
		return
	}
	// Removing client
	s.lock.Lock()
	client.Close()
	delete(s.clients, clientID)
	s.connections--
	s.lock.Unlock()
}

// serverEventListener listens for any event triggered
// by TriggerEvent or other places in future versions
func (s *Server) serverEventListener() {
	for {
		evt, ok := <-s.serverEvents
		if !ok {
			return
		}
		s.commandToAll(evt.WSCommand)
	}
}

// commandToAll sends a command string to each connected
// client through their respective WebSocket
func (s *Server) commandToAll(command string) {
	s.lock.RLock()
	for _, v := range s.clients {
		v.WriteMessage(websocket.TextMessage, []byte(command))
	}
	s.lock.RUnlock()
}
