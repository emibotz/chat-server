package game

import "github.com/google/uuid"

type Player struct {
	id     uuid.UUID
	userID uuid.UUID
	name   string

	position Vector2
}

func NewPlayer(userID uuid.UUID, name string) (*Player, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	return &Player{
		id:     id,
		userID: userID,
		name:   name,

		position: Vector2Zero,
	}, nil
}

func (p *Player) GetID() uuid.UUID {
	return p.id
}

func (p *Player) GetUserID() uuid.UUID {
	return p.userID
}

func (p *Player) GetName() string {
	return p.name
}

func (p *Player) GetPosition() Vector2 {
	return p.position
}

func (p *Player) SetPosition(position Vector2) {
	p.position = position
}

// 玩家移动意图
type playerMoveIntention struct {
	playerID  uuid.UUID
	Direction Vector2
}

func (i *playerMoveIntention) GetPlayerID() uuid.UUID {
	return i.playerID
}
