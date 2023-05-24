package src

import (
	"fmt"
	"log"
	"sync"
	"time"
)

func processCollisions(player *Sprite, maze *[]string, lives *int, enemiesStatusMx *sync.RWMutex, enemies *[]*Enemy, cfg *Config) {
	for _, e := range *enemies {
		if player.Row == e.Position.Row && player.Col == e.Position.Col {
			enemiesStatusMx.RLock()            // read lock
			if e.Status == EnemyStatusNormal { // normal ghost
				*lives = *lives - 1
				if *lives != 0 {
					MoveCursor(player.Row, player.Col, cfg)                       // move player to start position
					fmt.Print(cfg.Death)                                          // print death animation
					MoveCursor(len(*maze)+2, 0, cfg)                              // move cursor to bottom of screen
					enemiesStatusMx.RUnlock()                                     // unlock read lock
					go UpdateEnemies(enemies, EnemyStatusNormal, enemiesStatusMx) // update enemies
					time.Sleep(1000 * time.Millisecond)                           //dramatic pause before reseting player position
					player.Row, player.Col = player.StartRow, player.StartCol
				}
			} else if e.Status == EnemyStatusBlue { // blue ghost
				enemiesStatusMx.RUnlock()
				go UpdateEnemies(&[]*Enemy{e}, EnemyStatusNormal, enemiesStatusMx)
				e.Position.Row, e.Position.Col = e.Position.StartRow, e.Position.StartCol
			}
		}
	}
}

func Run(player *Sprite, maze *[]string, numDots, score, lives *int, pillMx *sync.Mutex, enemiesStatusMx *sync.RWMutex, enemies *[]*Enemy, pillTimer *time.Timer, cfg *Config) {
	// process input with a goroutine
	input := make(chan string)
	go func(ch chan<- string) {
		for {
			input, err := ReadInput()
			if err != nil {
				log.Print("error reading input:", err)
				ch <- "ESC"
			}
			ch <- input
		}
	}(input)

	// while true
	for {
		// process movement
		select {
		case inp := <-input:
			if inp == "ESC" {
				*lives = 0
			}
			MovePlayer(inp, player, maze, numDots, score, pillMx, enemiesStatusMx, enemies, pillTimer, cfg)
		default:
		}

		MoveEnemies(enemies, maze)

		// process collisions
		processCollisions(player, maze, lives, enemiesStatusMx, enemies, cfg)

		// update screen
		PrintScreen(cfg, maze, player, enemies, numDots, score, lives, pillMx, enemiesStatusMx)

		// check game over
		if *numDots == 0 || *lives <= 0 {
			if *lives == 0 {
				MoveCursor(player.Row, player.Col, cfg)
				fmt.Print(cfg.Death)
				MoveCursor(player.StartRow, player.StartCol-1, cfg)
				fmt.Print("GAME OVER\n")
				MoveCursor(len(*maze)+2, 0, cfg)
			}
			if *numDots == 0 {
				fmt.Print("YOU WIN!\n")
			}
			break
		}

		// repeat
		time.Sleep(100 * time.Millisecond)
	}
}
