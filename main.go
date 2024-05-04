package main

import (
	"encoding/gob"
	"io"
	"log/slog"
	"net"
	"os"
	"sync"

	"github.com/google/uuid"
)

type GameStateStatus uint8
type ServerStatus uint8

const (
    Idle        GameStateStatus = iota
    Started
    Testing
    Error
)

const (
    Running     ServerStatus = iota
    Stopped
)

const PRODNET = "tcp4"
const PRODADDR = "127.0.0.1:8080"
const CONN_LIMIT = 10001

type Msg struct {
    Echo    bool
    Magic   uint16
    Player  Player
}

type GameState struct {
    Version             uint8
    State               GameStateStatus
    Players             []Player
    Bullets             []Bullet
    IDtoPlayerMap       map[uuid.UUID]*Player
}

type Server struct {
    Version             uint8
    Status              ServerStatus
    Listener            net.Listener
    GS                  GameState
    ConnPool            map[*net.Conn]uint8
    Logger              *slog.Logger
    mu                  sync.Mutex
}

type Position interface {
    updatePosition(p int) bool
}

type Player struct {
    ID      uuid.UUID
    Score   uint16
    Y       uint8
    State   uint8   // bit packing (dead/alive), fired bullet, etc
}

type Bullet struct {
    X           uint8
    Y           uint8
    ID          uuid.UUID
    PlayerID    uuid.UUID
}

func (player *Player) updatePosition(p uint8) bool {
    if p >= 0 && p <= 255 {
        player.Y = p
        return true
    }
    return false
}

func (bullet *Bullet) updatePosition(p uint8) bool {
    if p >= 0 && p <= 255 {
        bullet.X = p
        return true
    }
    return false
}

func (bullet *Bullet) hasCollided(p *Player) bool {
    // TODO
    return false
}

func (s *Server) ConnHandler(c net.Conn) {
    defer s.closeConn(c)

    s.mu.Lock()
    s.ConnPool[&c] = 0
    s.mu.Unlock()

    dec := gob.NewDecoder(c)

    for {
        var msg Msg
        err := dec.Decode(&msg)
        if err != nil {
            if err == io.EOF {
                continue
            }
            s.Logger.Error("SERVER - Failed to decode", "err", err)
        }
        s.Logger.Info("SERVER - Received message", "msg", msg)

        if msg.Echo {
            enc := gob.NewEncoder(c)
            err = enc.Encode(&msg)
            if err != nil {
                s.Logger.Error("SERVER - Failed to encode", "err", err)
            }
            continue
        }

        // update game state
        s.mu.Lock()
        _, prs := s.GS.IDtoPlayerMap[msg.Player.ID]
        if !prs { // add player if DNE
            p := Player{}
            p.ID = uuid.New()
            s.GS.Players = append(s.GS.Players, p)
            s.GS.IDtoPlayerMap[p.ID] = &p
        }
        s.mu.Unlock()

        // update player position on server
        // check if they fired a bullet
        // check if they collided



        // TODO collision logic
    }
}

func (s *Server) closeConn(c net.Conn) {
    err := c.Close()
    if err != nil {
        s.Logger.Error("SERVER - Failed to clean up TCP connection", "err", err)
        os.Exit(1)
    }
    s.mu.Lock()
    _, prs := s.ConnPool[&c]
    if prs {
        delete(s.ConnPool, &c)
    }
    s.mu.Unlock()
}

func (s *Server) setupGameState(version uint8) GameState {
    gBullets := make([]Bullet, 0)
    gPlayers := make([]Player, 0)
    gIDtoPlayer := make(map[uuid.UUID]*Player)

    gameState := GameState{
        version,
        Idle,
        gPlayers,
        gBullets,
        gIDtoPlayer,
    }
    return gameState
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
    gConnPool := make(map[*net.Conn]uint8)
    s.ConnPool = gConnPool
    s.Status = Running
}

func (s *Server) stop() {
    s.Status = Stopped
    s.teardownListener()
}

func (s *Server) teardownListener() {
    s.Logger.Info("SERVER - Tearing down TCP listener.")
    err := s.Listener.Close()
    if err != nil {
        s.Logger.Error("SERVER - Failed to teardown listener", "err", err)
        os.Exit(1)
    }
}

func (s *Server) run() {
    s.Logger.Info("SERVER - Ready to accept incoming connections.")

    for {
        conn, err := s.Listener.Accept()
        if err != nil {
            if s.Status == Stopped {
                break
            } else {
                s.Logger.Error("SERVER - Failed to accept incoming connection", "err", err)
                return
            }
        } 

        s.Logger.Info("SERVER - Connection received from: ", "remote", conn.RemoteAddr().String())

        if len(s.ConnPool) + 1 >= CONN_LIMIT {
            s.Logger.Error("SERVER - Reached connection limit", "CONN_LIMIT", CONN_LIMIT)
            continue
        }
        go s.ConnHandler(conn)
    }
}

func main() {

    var version uint8
    version = 1
    s := Server{}

    defer s.stop()
    s.setup(version, PRODNET, PRODADDR)
    s.run()
}

