package main

import (
	"log"
	"log/slog"
	"net"
    "reflect"
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
}

func (g *GameState) ConnHandler(c net.Conn) {
    defer c.Close()

    t := reflect.TypeOf((*GameState)(nil)).Elem()
    bufsize := t.Size()
    
    buf := make([]byte, bufsize)

    for {
        bytes, err := c.Read(buf)
        if err != nil {
            slog.Error("Failed to read from buffer", "err", err)
        }
        // TODO deserialize

        // TODO collision logic
        _ = bytes
    }
}


func main() {

    gBullets := make([]Bullet, 0)
    gPlayers := make([]Player, 0)
    gConnPool := make([]*net.Conn, 0)

    gameState := GameState{
        0,
        Idle,
        0,
        0,
        0,
        gPlayers,
        gBullets,
        gConnPool,
    }

    ln, err := net.Listen("tcp4", "127.0.0.1:1773")
    if err != nil {
        slog.Error("Failed to create TCP listener", "err", err)
        log.Fatal("Terminating")
    }
    slog.Info("Started TCP listener", "ln", ln)

    slog.Info("Setting up connection pool")

    for {
        conn, err := ln.Accept()
        if err != nil {
            slog.Error("Failed to accept incoming connection", "err", err)
        }
        go gameState.ConnHandler(conn)
    }

    // TODO ticker and connection pool
}

