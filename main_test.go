package main

import (
	"encoding/gob"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
    "github.com/google/uuid"
)

const TESTNET = "tcp4"
const TESTADDR = "127.0.0.1:8080"

func serverSetup(version uint8) *Server {
    s := Server{}
    s.setup(version, TESTNET, TESTADDR)
    return &s
}

func closeConnTest(t *testing.T, c net.Conn) {
    err := c.Close()
    if err != nil {
        t.Fatal("TEST - Failed to clean up TCP connection", "err", err)
    }
}

func generateTestPlayer() Player {
    p := Player{}
    p.ID, _ = uuid.Parse("0")
    return p
}

type Client struct {
    ID      uint16
    conn    net.Conn
    enc     *gob.Encoder
    dec     *gob.Decoder
    t       *testing.T
}

func (c *Client) Write(msg *Msg) {
    err := c.enc.Encode(&msg)
    if err != nil {
        c.t.Fatalf("TEST - Failed to encode msg with error: %v\n", err)
    }
}

func (c *Client) Read() Msg {
    res := Msg{}
    err := c.dec.Decode(&res)
    if err != nil {
        c.t.Fatalf("TEST - Failed to decode server response with error :%v\n", err)
    }
    return res
}

func ClientSetup(t *testing.T, network string, addr string) (*Client, error) {

    client := Client{}
    conn, err := net.Dial(network, addr)
    if err != nil {
        return nil, err
    }
    client.conn = conn
    client.enc = gob.NewEncoder(client.conn)
    client.dec = gob.NewDecoder(client.conn)

    return &client, nil
}

func TestServerVersions(t *testing.T) {
    var version uint8
    version = 2
    s := serverSetup(version)
    defer s.stop()
    
    assert.Equal(t, Running, s.Status, "Server should be running")
    assert.Equal(t, version, s.Version, "Versions should match")
    assert.Equal(t, 0, len(s.ConnPool), "Connection pool size should match")
    assert.Equal(t, version, s.GS.Version, "Versions should match")
    assert.Equal(t, 0, len(s.GS.Players), "Num Players should match")
    assert.Equal(t, Idle, s.GS.State, "GS State should match")
}

func TestServerEcho(t *testing.T) {
    var ver uint8
    ver = 1
    s := serverSetup(ver)
    defer s.stop()
    go s.run()
    c, err := ClientSetup(t, TESTNET, TESTADDR)
    if err != nil {
        t.Fatalf("TEST - Failed to create client with error: %v\n", err)
    }

    var magic uint16
    magic = 42

    defer closeConnTest(t, c.conn)

    msg := Msg{ Echo: true, Magic: magic, Player: generateTestPlayer()}
    c.Write(&msg)
    
    res := c.Read()

    assert.Equal(t, msg.Echo, res.Echo, "Echo field should match")
    assert.Equal(t, msg.Magic, res.Magic, "Magic field should match")
    assert.Equal(t, msg.Player.ID, res.Player.ID, "Player ID field should match")
    assert.Equal(t, 1, len(s.ConnPool), "Connection pool size should match")
}


func TestAddPlayers(t *testing.T) {
    numPlayers := 1000

    var ver uint8
    ver = 1
    s := serverSetup(ver)
    defer s.stop()
    go s.run()

    clients := make([]*Client, 0)
    for i := 0; i < numPlayers; i++ {
        c, err := ClientSetup(t, TESTNET, TESTADDR)
        if err != nil {
            t.Fatalf("TEST - Failed to create client with error: %v\n", err)
        }
        defer closeConnTest(t, c.conn)
        clients = append(clients, c)
    }

    for _, c := range clients {
        msg := Msg{ Echo: false, Magic: uint16(3), Player: generateTestPlayer()}
        c.Write(&msg)
    }

    for {
        if len(clients) == len(s.GS.Players) {
            break
        }
    }

    assert.Equal(t, len(clients), len(s.ConnPool), "Number of clients should match size of connection pool")
    assert.Equal(t, len(clients), len(s.GS.Players), "Number of clients should num players on server")

    for _, p := range s.GS.Players {
        assert.NotNil(t, p.ID, "Player ID should not be nil")
        assert.NotEqual(t, uint32(0), p.ID.ID(), "Player ID should be non-zero")
    }
}

