package main

import "errors"

type BulletStatus int
type BulletDirection int

const (
	Inactive BulletStatus = iota
	Active
)

const (
	Left BulletDirection = iota
	Right
)

var BULLET_STEP_SIZE = 1

type Bullet struct {
	ID        uint32
	pos       Position
	status    BulletStatus
	direction BulletDirection
}

func (b *Bullet) handleAction(action *int) error {
	if action != nil {
		return errors.New("Bullet expected nil action")
	}

	if b.direction == Left {
		b.pos.x -= uint8(BULLET_STEP_SIZE)
	} else {
		b.pos.x += uint8(BULLET_STEP_SIZE)
	}

	return nil
}
