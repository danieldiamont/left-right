package main

import (
	"encoding/gob"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

const TESTNET = "tcp4"
const TESTADDR = "127.0.0.1:8080"

func serverSetup(version uint8) Server {
    s := Server{}
    s.setup(version, TESTNET, TESTADDR)
    return s
}

func closeConnTest(t *testing.T, c net.Conn) {
    err := c.Close()
    if err != nil {
        t.Fatal("TEST - Failed to clean up TCP connection", "err", err)
    }
}

func generateTestPlayer(id uint16) Player {
    p := Player{}
    p.ID = id
    return p
}

type Client struct {
    ID      uint16
    conn    net.Conn
}

func ClientSetup(id uint16, network string, addr string) (*Client, error) {

    client := Client{}
    conn, err := net.Dial(network, addr)
    if err != nil {
        return nil, err
    }
    client.conn = conn
    return &client, nil
}

func TestServerVersions(t *testing.T) {
    var version uint8
    version = 2
    s := serverSetup(version)
    defer s.stop()
    
    assert.Equal(t, s.Status, Running, "Server should be running")
    assert.Equal(t, s.Version, version, "Versions should match")
    assert.Equal(t, s.GS.Version, version, "Versions should match")
}

func TestServerEcho(t *testing.T) {
    var ver uint8
    var id uint16
    ver = 1
    id = 3
    s := serverSetup(ver)
    defer s.stop()
    go s.run()
    c, err := ClientSetup(id, TESTNET, TESTADDR)
    if err != nil {
        t.Fatalf("TEST - Failed to create client with error: %v\n", err)
    }

    var magic uint16
    magic = 42

    defer closeConnTest(t, c.conn)
    enc := gob.NewEncoder(c.conn)
    dec := gob.NewDecoder(c.conn)

    msg := Msg{ Echo: true, Magic: magic, Player: generateTestPlayer(id)}

    s.Logger.Info("TEST - Sending message", "msg", msg)

    err = enc.Encode(&msg)
    if err != nil {
        t.Fatalf("TEST - Failed to encode msg with error: %v\n", err)
    }

    res := Msg{}
    err = dec.Decode(&res)
    if err != nil {
        t.Fatalf("TEST - Failed to decode server response with error :%v\n", err)
    }

    assert.Equal(t, msg.Echo, res.Echo, "Echo field should match")
    assert.Equal(t, msg.Magic, res.Magic, "Magic field should match")
    assert.Equal(t, msg.Player.ID, res.Player.ID, "Player ID field should match")
}
