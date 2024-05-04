package main

import (
	"log"
	"log/slog"
	"net"
    "encoding/gob"
    "os"
)

type GameStateStatus uint8

const (
    Idle        GameStateStatus = iota
    Started
    Testing
    Error
)

type Position interface {
    updatePosition(p int) bool
}

type Player struct {
    ID      uint16
    Score   uint16
    Y       uint8
    State   uint8   // bit packing (dead/alive), fired bullet, etc
}

type Bullet struct {
    X           uint8
    Y           uint8
    PlayerID    uint16
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

type GameState struct {
    Version             uint8
    State               GameStateStatus
    activeConnections   uint16
    NumPlayers          uint16
    NumBullets          uint16
    Players             []Player
    Bullets             []Bullet
    ConnPool            []*net.Conn
    IDtoPlayerMap       map[uint16]*Player
}

func (g *GameState) ConnHandler(c net.Conn) {
    defer closeConn(c)

    dec := gob.NewDecoder(c)

    for {
        var p Player
        err := dec.Decode(&p)
        if err != nil {
            slog.Error("Failed to decode", "err", err)
        }
        log.Printf("Received %+v\n", p)

        // update game state

        _, prs := g.IDtoPlayerMap[p.ID]
        if !prs { // add player if DNE
            g.Players = append(g.Players, p)
            g.IDtoPlayerMap[p.ID] = &p
        }

        // update player position on server
        // check if they fired a bullet
        // check if they collided


        // TODO collision logic
    }
}

func closeConn(c net.Conn) {
    err := c.Close()
    if err != nil {
        slog.Error("Failed to clean up TCP connection", "err", err)
        os.Exit(1)
    }
}


func main() {

    gBullets := make([]Bullet, 0)
    gPlayers := make([]Player, 0)
    gConnPool := make([]*net.Conn, 0)
    gIDtoPlayer := make(map[uint16]*Player)

    gameState := GameState{
        0,
        Idle,
        0,
        0,
        0,
        gPlayers,
        gBullets,
        gConnPool,
        gIDtoPlayer,
    }

    ln, err := net.Listen("tcp4", "127.0.0.1:1773")
    if err != nil {
        slog.Error("Failed to create TCP listener", "err", err)
        os.Exit(1)
    }
    slog.Info("Started TCP listener", "ln", ln)

    slog.Info("Setting up connection pool")

    for {
        conn, err := ln.Accept()
        if err != nil {
            slog.Error("Failed to accept incoming connection", "err", err)
            os.Exit(1)
        }
        go gameState.ConnHandler(conn)
    }

    // TODO ticker and connection pool
}

