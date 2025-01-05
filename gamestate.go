package main

type GameState struct {
	Version       uint8
	Players       []*Player
	Bullets       []*Bullet
	IDtoPlayerMap map[uint32]*Player
}

func (gs *GameState) makePlayer() uint32 {
	p := &Player{}
	id := p.init()

	gs.IDtoPlayerMap[id] = p
	gs.Players = append(gs.Players, p)
	return id
}
