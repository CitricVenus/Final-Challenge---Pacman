package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"
	//tm "github.com/buger/goterm"
	"github.com/danicat/simpleansi"
)

var (
	configFile = flag.String("config-file", "config.json", "path to custom configuration file")
	mazeFile   = flag.String("maze-file", "maze01.txt", "path to a custom maze file")
)

type sprite struct {
	row      int
	col      int
	startRow int
	startCol int
}

type ghost struct {
	position sprite
	status   GhostStatus
}

type GhostStatus string

const (
	GhostStatusNormal GhostStatus = "Normal"
	GhostStatusBlue   GhostStatus = "Blue"
)

var ghostsStatusMx sync.RWMutex
var pillMx sync.Mutex

type config struct {
	Player           string        `json:"player"`
	Ghost            string        `json:"ghost"`
	GhostBlue        string        `json:"ghost_blue"`
	Wall             string        `json:"wall"`
	Dot              string        `json:"dot"`
	Pill             string        `json:"pill"`
	Death            string        `json:"death"`
	Space            string        `json:"space"`
	UseEmoji         bool          `json:"use_emoji"`
	PillDurationSecs time.Duration `json:"pill_duration_secs"`
}

var cfg config
var player sprite
var ghosts []*ghost
var maze []string
var score int
var numDots int
var lives = 3

func loadConfig(file string) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	decoder := json.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		return err
	}

	return nil
}

func loadMaze(file string,numenemy int) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		maze = append(maze, line)
	}

	var aux int = 0
	for row, line := range maze {
		for col, char := range line {
			switch char {
			case 'P':
				player = sprite{row, col, row, col}
			case 'G':
				if aux < numenemy {
					ghosts = append(ghosts, &ghost{sprite{row, col, row, col}, GhostStatusNormal})
					aux++
				}
			case '.':
				numDots++
			}
		}
	}

	return nil
}

func moveCursor(row, col int) {
	if cfg.UseEmoji {
		simpleansi.MoveCursor(row, col*2)
	} else {
		simpleansi.MoveCursor(row, col)
	}
}

func printScreen() {
	simpleansi.ClearScreen()
	for _, line := range maze {
		for _, chr := range line {
			switch chr {
			case '#':
				fmt.Print(simpleansi.WithBlueBackground(cfg.Wall))
			case '.':
				fmt.Print(cfg.Dot)
			case 'X':
				fmt.Print(cfg.Pill)
			default:
				fmt.Print(cfg.Space)
			}
		}
		fmt.Println()
	}

	moveCursor(player.row, player.col)
	fmt.Print(cfg.Player)

	ghostsStatusMx.RLock()
	for _, g := range ghosts {
		moveCursor(g.position.row, g.position.col)
		if g.status == GhostStatusNormal {
			fmt.Printf(cfg.Ghost)
		} else if g.status == GhostStatusBlue {
			fmt.Printf(cfg.GhostBlue)
		}
	}
	ghostsStatusMx.RUnlock()

	moveCursor(len(maze)+1, 0)

	livesRemaining := strconv.Itoa(lives) //converts lives int to a string
	if cfg.UseEmoji {
		livesRemaining = getlivesAsEmoji()
	}

	fmt.Println("Score:", score, "\tlives:", livesRemaining)
}

// concatenate the correct number of player emojis based on lives
func getlivesAsEmoji() string {
	buf := bytes.Buffer{}
	for i := lives; i > 0; i-- {
		buf.WriteString(cfg.Player)
	}
	return buf.String()
}

func readInput() (string, error) {
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

func makeMove(oldRow, oldCol int, dir string) (newRow, newCol int) {
	newRow, newCol = oldRow, oldCol

	switch dir {
	case "UP":
		newRow = newRow - 1
		if newRow < 0 {
			newRow = len(maze) - 1
		}
	case "DOWN":
		newRow = newRow + 1
		if newRow == len(maze)-1 {
			newRow = 0
		}
	case "RIGHT":
		newCol = newCol + 1
		if newCol == len(maze[0]) {
			newCol = 0
		}
	case "LEFT":
		newCol = newCol - 1
		if newCol < 0 {
			newCol = len(maze[0]) - 1
		}
	}

	if maze[newRow][newCol] == '#' {
		newRow = oldRow
		newCol = oldCol
	}

	return
}

func movePlayer(dir string) {
	player.row, player.col = makeMove(player.row, player.col, dir)

	removeDot := func(row, col int) {
		maze[row] = maze[row][0:col] + " " + maze[row][col+1:]
	}

	switch maze[player.row][player.col] {
	case '.':
		numDots--
		score++
		removeDot(player.row, player.col)
	case 'X':
		score += 10
		removeDot(player.row, player.col)
		go processPill()
	}
}

func updateGhosts(ghosts []*ghost, ghostStatus GhostStatus) {
	ghostsStatusMx.Lock()
	defer ghostsStatusMx.Unlock()
	for _, g := range ghosts {
		g.status = ghostStatus
		g.position.row, g.position.col = g.position.startRow, g.position.startCol
	}
}

var pillTimer *time.Timer

func processPill() {
	pillMx.Lock()
	updateGhosts(ghosts, GhostStatusBlue)
	if pillTimer != nil {
		pillTimer.Stop()
	}
	pillTimer = time.NewTimer(time.Second * cfg.PillDurationSecs)
	pillMx.Unlock()
	<-pillTimer.C
	pillMx.Lock()
	pillTimer.Stop()
	updateGhosts(ghosts, GhostStatusNormal)
	pillMx.Unlock()
}

func drawDirection() string {
	dir := rand.Intn(4)
	move := map[int]string{
		0: "UP",
		1: "DOWN",
		2: "RIGHT",
		3: "LEFT",
	}
	return move[dir]
}

func moveGhosts() {
	for _, g := range ghosts {
		dir := drawDirection()
		g.position.row, g.position.col = makeMove(g.position.row, g.position.col, dir)
	}
}

func initialise() {
	cbTerm := exec.Command("stty", "cbreak", "-echo")
	cbTerm.Stdin = os.Stdin

	err := cbTerm.Run()
	if err != nil {
		log.Fatalln("unable to activate cbreak mode:", err)
	}
}

func cleanup() {
	cookedTerm := exec.Command("stty", "-cbreak", "echo")
	cookedTerm.Stdin = os.Stdin

	err := cookedTerm.Run()
	if err != nil {
		log.Fatalln("unable to activate cooked mode:", err)
	}
}

func printCounter(wait int){
	for i:= 0; i< wait; i++{
		time.Sleep(750 * time.Millisecond)
		fmt.Println("RESTARTING in " + strconv.Itoa(3-i) + " seconds")
	}
}


func main() {
	flag.Parse()

	fmt.Println("Escribe el numero de enemigos a crear (1-12) :")
	var num string
	fmt.Scanln(&num)
	numenemy, err := strconv.Atoi(num)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if numenemy > 0 && numenemy < 13 {
	// initialize game
	initialise()
	defer cleanup()

	// load resources
	err := loadMaze(*mazeFile,numenemy)
	if err != nil {
		log.Println("failed to load maze:", err)
		return
	}

	err = loadConfig(*configFile)
	if err != nil {
		log.Println("failed to load configuration:", err)
		return
	}

	// process input (async)
	input := make(chan string)
	go func(ch chan<- string) {
		for {
			input, err := readInput()
			if err != nil {
				log.Print("error reading input:", err)
				ch <- "ESC"
			}
			ch <- input
		}
	}(input)

	// game loop
	for {
		// process movement
		select {
		case inp := <-input:
			if inp == "ESC" {
				lives = 0
			}
			movePlayer(inp)
		default:
		}

		moveGhosts()

		// process collisions
		for _, g := range ghosts {
			if player.row == g.position.row && player.col == g.position.col {
				ghostsStatusMx.RLock()
				if g.status == GhostStatusNormal {
					lives = lives - 1
					if lives != 0 {
						moveCursor(player.row, player.col)
						fmt.Print(cfg.Death)
						moveCursor(len(maze)+2, 0)
						ghostsStatusMx.RUnlock()
						updateGhosts(ghosts, GhostStatusNormal)
						go printCounter(3)
						time.Sleep(3000 * time.Millisecond) //dramatic pause before reseting player position
						player.row, player.col = player.startRow, player.startCol
					}
				} else if g.status == GhostStatusBlue {
					ghostsStatusMx.RUnlock()
					updateGhosts([]*ghost{g}, GhostStatusNormal)
					g.position.row, g.position.col = g.position.startRow, g.position.startCol
				}
			}
		}
		//clear terminal
		fmt.Print("\033[H\033[2J")
		// update screen
		printScreen()

		// check game over
		if numDots == 0 || lives <= 0 {
			if lives == 0 {
				moveCursor(player.row, player.col)
				fmt.Print(cfg.Death)
				moveCursor(player.startRow, player.startCol-1)
				fmt.Print("GAME OVER")
				moveCursor(len(maze)+2, 0)
			}
			break
		}

		// repeat
		time.Sleep(200 * time.Millisecond)
	}
	}else{
		fmt.Println("El numero de fantaasmas debe de ser entre 1 a 12")
		os.Exit(1)
	}

}
