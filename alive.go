package main

type Alive interface {
	decreaseHealth(interval int) int
	isAlive() bool
}
