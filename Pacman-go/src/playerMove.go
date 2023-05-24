package src

import (
	"math/rand"
)

func MakeMove(oldRow, oldCol int, dir string, maze *[]string) (newRow, newCol int) {
	newRow, newCol = oldRow, oldCol

	switch dir {
	case "UP":
		newRow = newRow - 1
		if newRow < 0 {
			newRow = len(*maze) - 1
		}
	case "DOWN":
		newRow = newRow + 1
		if newRow == len(*maze)-1 {
			newRow = 0
		}
	case "RIGHT":
		newCol = newCol + 1
		if newCol == len((*maze)[0]) {
			newCol = 0
		}
	case "LEFT":
		newCol = newCol - 1
		if newCol < 0 {
			newCol = len((*maze)[0]) - 1
		}
	}

	if (*maze)[newRow][newCol] == '#' {
		newRow = oldRow
		newCol = oldCol
	}

	return
}

func DrawDirection() string {
	dir := rand.Intn(4)
	move := map[int]string{
		0: "UP",
		1: "DOWN",
		2: "RIGHT",
		3: "LEFT",
	}
	return move[dir]
}
