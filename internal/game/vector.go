package game

import "math"

type Vector2 struct {
	X float64
	Y float64
}

var (
	Vector2Zero = Vector2{X: 0.0, Y: 0.0}
)

func (v Vector2) Add(a Vector2) Vector2 {
	return Vector2{
		X: v.X + a.X,
		Y: v.Y + a.Y,
	}
}

func (v Vector2) Multiply(scalar float64) Vector2 {
	return Vector2{
		X: v.X * scalar,
		Y: v.Y * scalar,
	}
}

func (v Vector2) Length() float64 {
	if v == Vector2Zero {
		return 0.0
	}

	return math.Sqrt(v.X*v.X + v.Y*v.Y)
}

func (v Vector2) Normalized() Vector2 {
	if v == Vector2Zero {
		return v
	}

	length := v.Length()
	return Vector2{
		X: v.X / length,
		Y: v.Y / length,
	}
}
