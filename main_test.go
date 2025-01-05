package main

import (
	"encoding/json"
	"net"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
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
	p.id = 1
	return p
}

type Client struct {
	ID   uint32
	conn net.Conn
	enc  *json.Encoder
	dec  *json.Decoder
	t    *testing.T
}

func (c *Client) Write(msg UserMsg) {
	err := c.enc.Encode(&msg)
	if err != nil {
		c.t.Fatalf("TEST - Failed to encode msg with error: %v\n", err)
	}
}

func (c *Client) Read() ServerMsg {
	res := ServerMsg{}
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
	client.enc = json.NewEncoder(client.conn)
	client.dec = json.NewDecoder(client.conn)

	return &client, nil
}

func cleanup(s *Server, runnerKill, runnerDone chan bool) {
	s.Logger.Info("TEST - cleaning up server")
	runnerKill <- true
	<-runnerDone
	s.stop()
}

func TestAddPlayers(t *testing.T) {
	lastArg := os.Args[len(os.Args)-1]
	numPlayers, _ := strconv.ParseInt(lastArg, 10, 0)

	var ver uint8
	ver = 1
	s := serverSetup(ver)
	runnerKill := make(chan bool)
	runnerDone := make(chan bool)
	go s.run(runnerKill, runnerDone)

	clients := make([]*Client, 0)
	for i := 0; i < int(numPlayers); i++ {
		c, err := ClientSetup(t, TESTNET, TESTADDR)
		if err != nil {
			t.Fatalf("TEST - Failed to create client with error: %v\n", err)
		}
		defer closeConnTest(t, c.conn)
		clients = append(clients, c)
	}

	for _, c := range clients {
		msg := UserMsg{Action: Down}
		c.Write(msg)
	}

	for {
		s.mu.RLock()
		numPlayers := len(s.GS.Players)
		s.mu.RUnlock()
		if len(clients) == numPlayers {
			break
		}
	}

	assert.Equal(t, len(clients), len(s.ConnPool), "Number of clients should match size of connection pool")
	assert.Equal(t, len(clients), len(s.GS.Players), "Number of clients should num players on server")

	for _, p := range s.GS.Players {
		assert.NotNil(t, p.id, "Player ID should not be nil")
		assert.NotEqual(t, uint32(0), p.id, "Player ID should be non-zero")
	}

	cleanup(s, runnerKill, runnerDone)

}
