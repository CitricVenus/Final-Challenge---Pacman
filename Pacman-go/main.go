package main

import (
	"flag"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	pacman "PacmanGo/src"
)

func main() {
	var (
		configFile = flag.String("config-file", "config.json", "path to custom configuration file")
		mazeFile   = flag.String("maze-file", "maze01.txt", "path to a custom maze file")
	)

	var enemiesStatusMx sync.RWMutex // guards enemies and enemiesStatus
	var pillMx sync.Mutex            // guards pillTimer

	var cfg pacman.Config       // global configuration
	var player pacman.Sprite    // global player
	var enemies []*pacman.Enemy // global enemies
	var maze []string           // global maze
	var score int               // global score
	var numDots int             // global number of dots
	var lives int               // global number of lives

	var pillTimer *time.Timer

	flag.Parse() // parse the command line arguments

	var ghostNum int
	if len(os.Args) != 2 {
		log.Println("Incorrect Input. Correct usage: go run main.go [number of enemies]")
		return
	}

	ghostNum, _ = strconv.Atoi(os.Args[1])

	if ghostNum < 1 || ghostNum > 12 {
		log.Println("Invalid number of enemies. It must be between 1 and 12")
		return
	}

	pacman.Initialise()
	defer pacman.Cleanup()

	err := pacman.LoadResources(*mazeFile, *configFile, &maze, &enemies, &player, &numDots, &cfg, ghostNum)
	if err != nil {
		log.Fatal(err)
		return
	}

	lives = cfg.MaxLifestate
	// run the game
	pacman.Run(&player, &maze, &numDots, &score, &lives, &pillMx, &enemiesStatusMx, &enemies, pillTimer, &cfg)
}
