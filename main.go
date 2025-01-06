package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"
)

const PRODNET = "tcp4"
const PRODADDR = "127.0.0.1:8080"
const CONN_LIMIT = 10000

type ServerStatus int

const (
	Stopped ServerStatus = iota
	Running
)

type UserMsg struct {
	Action PlayerAction `json:"action"`
}

type ServerMsg struct {
	player Player
}

type Server struct {
	Version  uint8
	Status   ServerStatus
	Listener net.Listener
	GS       GameState
	ConnPool map[*net.Conn]uint32
	Logger   *slog.Logger
	mu       sync.RWMutex
}

func (s *Server) ConnHandler(c net.Conn, actions chan PlayerAction) {
	defer s.closeConn(c)

	s.mu.Lock()
	id := s.GS.makePlayer()
	s.ConnPool[&c] = id
	s.mu.Unlock()

	dec := json.NewDecoder(c)
	var msg UserMsg

	for {
		err := dec.Decode(&msg)
		if err != nil {
			if err == io.EOF {
				continue
			}
			s.Logger.Error("SERVER - Failed to decode", "err", err)
		}
		s.Logger.Info("SERVER - Received message", "msg", msg)
	}
}

func (s *Server) closeConn(c net.Conn) {
	s.mu.Lock()
	err := c.Close()
	s.mu.Unlock()
	if err != nil {
		s.Logger.Error("SERVER - Failed to clean up TCP connection", "err", err)
		os.Exit(1)
	}
	s.Logger.Info("SERVER - Cleaned up TCP connection")
	s.mu.Lock()
	_, prs := s.ConnPool[&c]
	if prs {
		delete(s.ConnPool, &c)
	}
	s.mu.Unlock()
}

func (s *Server) setupGameState(version uint8) GameState {
	gBullets := make([]*Bullet, 0)
	gPlayers := make([]*Player, 0)
	gIDtoPlayer := make(map[uint32]*Player)

	gameState := GameState{
		version,
		gPlayers,
		gBullets,
		gIDtoPlayer,
	}
	return gameState
}

func (s *Server) gameLoop(actions chan PlayerAction, done chan bool) {
	ticker := time.NewTicker(16 * time.Millisecond)

	for {
		select {
		case <-done:
			ticker.Stop()
			return
		case <-actions:
			// handle action
		case t := <-ticker.C:
			// game loop logic
			t.Add(1)
		}
	}
}

func (s *Server) setupListener(network string, addr string) (net.Listener, error) {
	ln, err := net.Listen(network, addr)
	if err != nil {
		s.Logger.Error("SERVER - Failed to create TCP listener", "err", err)
		return nil, err
	}

	s.Logger.Info("SERVER - Started TCP listener", "ln", ln.Addr().String())
	s.Logger.Info("SERVER - Setting up connection pool")
	return ln, nil
}

func (s *Server) setup(version uint8, network string, addr string) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	s.Logger = logger
	s.Version = version
	s.GS = s.setupGameState(version)
	ln, err := s.setupListener(network, addr)
	if err != nil {
		os.Exit(1)
	}
	s.Listener = ln
	gConnPool := make(map[*net.Conn]uint32)
	s.ConnPool = gConnPool
	s.Status = Running
}

// func (s *Server) closeAllConnections() {

// 	for c := range s.ConnPool {
// 		s.closeConn(*c)
// 	}

// }

func (s *Server) stop() {
	s.Status = Stopped
	s.teardownListener()
}

func (s *Server) teardownListener() {
	s.Logger.Info("SERVER - Tearing down TCP listener.")
	s.mu.Lock()
	err := s.Listener.Close()
	s.mu.Unlock()
	if err != nil {
		s.Logger.Error("SERVER - Failed to teardown listener", "err", err)
		os.Exit(1)
	}
	s.Logger.Info("SERVER - TCP listener is down")
}

func (s *Server) run(runnerKill, runnerDone chan bool) {
	s.Logger.Info("SERVER - Ready to accept incoming connections.")

	gameLoopDone := make(chan bool)
	actions := make(chan PlayerAction, 1024)

	connections := make([]net.Conn, 0)

	go s.gameLoop(actions, gameLoopDone)

	go func() {
		for {
			select {
			case <-runnerKill:
				s.Logger.Info("SERVER - Received runnerKill signal; cleaning up resources")
				for _, c := range connections {
					err := c.Close()
					if err != nil {
						s.Logger.Error("SERVER - Failed to close TCP connection", "err", err)
					}
				}
				runnerDone <- true
			}
		}
	}()

	for {
		s.mu.RLock()
		poolLen := len(s.ConnPool)
		s.mu.RUnlock()

		if poolLen >= CONN_LIMIT {
			s.Logger.Error("SERVER - Reached connection limit", "CONN_LIMIT", CONN_LIMIT)
			continue
		}

		s.Logger.Info("SERVER - waiting for connection")
		conn, err := s.Listener.Accept()
		if err != nil {
			if s.Status == Stopped {
				break
			} else {
				s.Logger.Error("SERVER - Failed to accept incoming connection", "err", err)
				break
			}
		}

		s.Logger.Info("SERVER - Connection received from: ", "remote", conn.RemoteAddr().String())

		go s.ConnHandler(conn, actions)
	}
}

func main() {
	runtime.GOMAXPROCS(4)

	done := make(chan bool)
	runnerDone := make(chan bool)
	runnerKill := make(chan bool)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	var version uint8
	version = 1
	s := Server{}

	s.setup(version, PRODNET, PRODADDR)
	go s.run(runnerKill, runnerDone)

	go func() {
		for {
			select {
			case sig := <-sigs:
				s.Logger.Info("SERVER - Received signal ", "SIGNAL", sig.String())
				runnerKill <- true
				<-runnerDone
				s.Logger.Info("SERVER - Runner is done")
				s.stop()
				done <- true
			}

		}
	}()

	<-done
}
