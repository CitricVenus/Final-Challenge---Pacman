package src

import (
	"log"
	"os"
	"os/exec"
	"sync"
	"time"
)

type Sprite struct { // A sprite is a position on the board.
	Row      int
	Col      int
	StartRow int
	StartCol int
}

func MovePlayer(dir string, player *Sprite, maze *[]string, numDots, score *int, pillMx *sync.Mutex, enemiesStatusMx *sync.RWMutex, enemies *[]*Enemy, pillTimer *time.Timer, cfg *Config) {
	player.Row, player.Col = MakeMove(player.Row, player.Col, dir, maze)

	removeDot := func(row, col int) {
		(*maze)[row] = (*maze)[row][0:col] + " " + (*maze)[row][col+1:]
	}

	switch (*maze)[player.Row][player.Col] {
	case '.':
		*numDots--
		*score++
		removeDot(player.Row, player.Col)
	case 'X':
		*score += 10
		removeDot(player.Row, player.Col)
		go ProcessPill(pillMx, enemiesStatusMx, enemies, pillTimer, cfg)
	}
}

func ReadInput() (string, error) {
	buffer := make([]byte, 100)

	cnt, err := os.Stdin.Read(buffer)
	if err != nil {
		return "", err
	}

	if cnt == 1 && buffer[0] == 0x1b {
		return "ESC", nil
	} else if cnt >= 3 {
		if buffer[0] == 0x1b && buffer[1] == '[' {
			switch buffer[2] {
			case 'A':
				return "UP", nil
			case 'B':
				return "DOWN", nil
			case 'C':
				return "RIGHT", nil
			case 'D':
				return "LEFT", nil
			}
		}
	}

	return "", nil
}

func Initialise() {
	// disable terminal echo so we don't polute the screen with the output of key presses
	cbTerm := exec.Command("stty", "cbreak", "-echo")
	cbTerm.Stdin = os.Stdin

	err := cbTerm.Run()
	if err != nil {
		log.Fatalln("unable to activate cbreak mode:", err)
	}
}

func Cleanup() {
	cookedTerm := exec.Command("stty", "-cbreak", "echo")
	// restore terminal mode, echo on
	cookedTerm.Stdin = os.Stdin

	err := cookedTerm.Run()
	if err != nil {
		log.Fatalln("unable to activate cooked mode:", err)
	}
}
