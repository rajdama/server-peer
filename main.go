package main

import (
	"fmt"
	"log"
	"log/slog"
	"net"
)

const defaultListenAddr = ":5001"

type Config struct {
	ListenAddress string
}

type Server struct {
	Config
	peers     map[*Peer]bool
	ln        net.Listener
	addPeerCh chan *Peer
	quitch    chan struct{}
	msgch     chan []byte
}

func NewServer(cfg Config) *Server {
	if len(cfg.ListenAddress) == 0 {
		cfg.ListenAddress = defaultListenAddr
	}
	return &Server{
		Config:    cfg,
		peers:     make(map[*Peer]bool),
		addPeerCh: make(chan *Peer),
		quitch:    make(chan struct{}),
		msgch:     make(chan []byte),
	}
}

func (s *Server) handleRawMessage(rawMsg []byte) error {
	fmt.Println(string(rawMsg))
	return nil
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.ListenAddress)
	if err != nil {
		return err
	}
	s.ln = ln
	go s.loop()

	slog.Info("server running", "listen addrress", s.ListenAddress)

	return s.acceptLoop()
}

// this loop is for handeling message from peers
func (s *Server) loop() {
	for {
		select {
		case rawMsg := <-s.msgch:
			if err := s.handleRawMessage(rawMsg); err != nil {
				slog.Error("raw message error", "err", err)
			}
			fmt.Println("raw message is", rawMsg)
		case <-s.quitch:
			return
		case peer := <-s.addPeerCh:
			s.peers[peer] = true
		}
	}
}

// this loop is responsible for connection handeling and peer initialization
func (s *Server) acceptLoop() error {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			slog.Error("accept error", "err", err)
			continue
		}
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	peer := NewPeer(conn, s.msgch)
	s.addPeerCh <- peer
	slog.Info("New peer connected", "remote_address", conn.RemoteAddr())
	if err := peer.readLoop(); err != nil {
		slog.Error("peer read error", "err", err, "remoteAddr", conn.RemoteAddr())
	}
}

func main() {
	server := NewServer(Config{})
	log.Fatal(server.Start())
}
