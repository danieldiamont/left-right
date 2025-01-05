package main

import (
	"errors"
	"sync/atomic"
)

type PlayerAction int

const (
	Down PlayerAction = iota
	Up
	Fire
)

var NEXT_USER_ID atomic.Uint32
var PLAYER_STEP_SIZE uint8 = 10
var PLAYER_DIMENSION uint8 = 10

func getNextUserId() uint32 {
	return NEXT_USER_ID.Add(1)
}

type Player struct {
	id     uint32
	pos    Position
	health int
}

func (p *Player) getHealth() int {
	return p.health
}

func (p *Player) getPosition() Position {
	return p.pos
}
func (p *Player) getId() uint32 {
	return p.id
}

func (p *Player) decreaseHealth(interval int) int {
	p.health -= interval
	return p.health
}

func (p *Player) isAlive() bool {
	return p.health > 0
}

func (p *Player) fireBullet() {

}

func (p *Player) updatePosition(action *int) error {
	var err error
	err = nil

	a := PlayerAction(*action)
	switch a {
	case Down:
		p.pos.y += uint8(PLAYER_STEP_SIZE)
	case Up:
		p.pos.y -= uint8(PLAYER_STEP_SIZE)
	case Fire:
		p.fireBullet()
	default:
		err = errors.New("Invalid action")
	}

	return err
}

func (p *Player) init() uint32 {
	p.id = getNextUserId()
	p.health = 100
	if p.id%2 == 0 {
		p.pos.x = 0
	} else {
		p.pos.x = 100 - PLAYER_DIMENSION
	}
	return p.id
}
