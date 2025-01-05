package main

type Position struct {
	x uint8
	y uint8
}

type Movable interface {
	getPosition() Position
}

type Actionable interface {
	handleAction(action *int) error
}
