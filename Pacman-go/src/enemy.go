package src

import (
	"sync"
)

type Enemy struct {
	Position Sprite
	Status   EnemyStatus
}

type EnemyStatus string

const (
	EnemyNormalStatus EnemyStatus = "Normal"
	EnemyBlueStatus   EnemyStatus = "Blue"
)

func UpdateEnemies(enemies *[]*Enemy, enemyStatus EnemyStatus, enemiesStatusMx *sync.RWMutex) {
	enemiesStatusMx.Lock()
	defer enemiesStatusMx.Unlock()
	for _, e := range *enemies {
		e.Status = enemyStatus
	}
}

func MoveEnemies(enemies *[]*Enemy, maze *[]string) {
	for _, e := range *enemies {
		dir := DrawDirection()
		e.Position.Row, e.Position.Col = MakeMove(e.Position.Row, e.Position.Col, dir, maze)
	}
}
