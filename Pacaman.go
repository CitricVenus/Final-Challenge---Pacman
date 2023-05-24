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
)

type sprite struct {
	row      int
	col      int
	startRow int
	startCol int
}
type config struct {
	Player    string        `json:"player"`
	Ghost     string        `json:"ghost"`
	Wall      string        `json:"wall"`
	Dot       string        `json:"dot"`
	Pill      string        `json:"pill"`
	Death     string        `json:"death"`
	Space     string        `json:"space"`
	UseEmoji  bool          `json:"use_emoji"`
	GhostBlue string        `json:"ghost_blue"`
	PillTime  time.Duration `json:"pillTime"`
}

type enemyStatus string

type enemy struct {
	position sprite
	status   enemyStatus
}

const (
	enemyNormalMode enemyStatus = "Normal"
	enemyBlueMode   enemyStatus = "Blue"
)

var score int
var numDots int
var lives = 3
var enemies []*enemy
var player sprite
var level_map []string
var cfg config
var pillTime *time.Timer
var enemyStatusMx sync.RWMutex
var pillMx sync.Mutex

var (
	configFile = flag.String("config-file", "config.json", "path to custom configuration file")
	map_file   = flag.String("maze-file", "map01.txt", "path to a custom maze file")
)

// se lee el archivo que es el mapa
func loadLevelMap(filename string, enemynum int) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		level_map = append(level_map, line)
	}
	var counter int = 0

	for row, line := range level_map {
		for col, char := range line {
			switch char {
			case 'P':
				player = sprite{row, col, row, col}
			case 'G':
				if counter < enemynum {
					enemies = append(enemies, &enemy{sprite{row, col, row, col}, enemyNormalMode})
					counter++
				}

			case '.':
				numDots++
			}

		}
	}
	return nil
}

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
func MoveCursor(row, col int) {
	fmt.Printf("\x1b[%d;%df", row+1, col+1)
}
func moveCursorEmoji(row, col int) {
	if cfg.UseEmoji {
		MoveCursor(row, col*2)
	} else {
		MoveCursor(row, col)
	}
}

func pillProcess() {
	pillMx.Lock()
	updateEnemy(enemies, enemyBlueMode)
	if pillTime != nil {
		pillTime.Stop()
	}
	pillTime = time.NewTimer(time.Second * cfg.PillTime)
	pillMx.Unlock()
	<-pillTime.C
	pillMx.Lock()
	pillTime.Stop()
	updateEnemy(enemies, enemyNormalMode)
	pillMx.Unlock()
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
			newRow = len(level_map) - 1
		}
	case "DOWN":
		newRow = newRow + 1
		if newRow == len(level_map) {
			newRow = 0
		}
	case "RIGHT":
		newCol = newCol + 1
		if newCol == len(level_map[0]) {
			newCol = 0
		}
	case "LEFT":
		newCol = newCol - 1
		if newCol < 0 {
			newCol = len(level_map[0]) - 1
		}
	}
	if level_map[newRow][newCol] == '#' {
		newRow = oldRow
		newCol = oldCol
	}
	return
}

func movePlayer(dir string) {
	player.row, player.col = makeMove(player.row, player.col, dir)
	removeDot := func(row, col int) {
		level_map[row] = level_map[row][0:col] + " " + level_map[row][col+1:]
	}

	switch level_map[player.row][player.col] {
	case '.':
		numDots--
		score++
		removeDot(player.row, player.col)
	case 'X':
		score += 10
		removeDot(player.row, player.col)
		go pillProcess()
	}
}

func clearScreen() {
	fmt.Print("\x1b[2J")
	MoveCursor(0, 0)
}

func dirEnemy() string {
	dir := rand.Intn(14)
	move := map[int]string{
		0:  "UP",
		1:  "DOWN",
		2:  "RIGHT",
		3:  "LEFT",
		4:  "UP",
		5:  "DOWN",
		6:  "RIGHT",
		7:  "LEFT",
		8:  "UP",
		9:  "UP",
		10: "RIGHT",
		11: "LEFT",
		12: "UP",
		13: "DOWN",
	}
	return move[dir]
}
func moveEnemy() {
	for _, g := range enemies {
		dir := dirEnemy()
		g.position.row, g.position.col = makeMove(g.position.row, g.position.col, dir)
		time.Sleep(800 * time.Millisecond)
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

func clean() {
	cookedTerm := exec.Command("stty", "-cbreak", "echo")
	// restore terminal mode, echo on
	cookedTerm.Stdin = os.Stdin

	err := cookedTerm.Run()
	if err != nil {
		log.Fatalln("unable to activate cooked mode:", err)
	}
}

func printToScreen() {
	clearScreen()
	for _, line := range level_map {
		for _, chr := range line {
			switch chr {
			case '#':
				fmt.Print(WithBackground(cfg.Wall, BLUE))
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
	moveCursorEmoji(player.row, player.col)
	fmt.Print(cfg.Player)

	enemyStatusMx.RLock()

	for _, g := range enemies {
		moveCursorEmoji(g.position.row, g.position.col)
		if g.status == enemyNormalMode {
			fmt.Printf(cfg.Ghost)
		} else if g.status == enemyBlueMode {
			fmt.Printf(cfg.GhostBlue)
		}
	}
	enemyStatusMx.RUnlock()
	moveCursorEmoji(len(level_map)+1, 0)
	livesremaining := strconv.Itoa(lives)
	if cfg.UseEmoji {
		livesremaining = getLives()
	}
	fmt.Println("SCORE: ", score, "\tLives: ", livesremaining)
}

func getLives() string {
	buf := bytes.Buffer{}
	for i := lives; i > 0; i-- {
		buf.WriteString(cfg.Player)
	}
	return buf.String()
}
func updateEnemy(enemies []*enemy, enemystatus enemyStatus) {
	enemyStatusMx.Lock()
	defer enemyStatusMx.Unlock()
	for _, g := range enemies {
		g.status = enemystatus
	}
}

const reset = "\x1b[0m"

type Colour int

const (
	BLACK Colour = iota
	RED
	GREEN
	BROWN
	BLUE
)

var colours = map[Colour]string{
	BLACK: "\x1b[1;30;40m",
	GREEN: "\x1b[1;32;42m",
	BROWN: "\x1b[1;33;43m",
	BLUE:  "\x1b[1;34;44m",
}

func WithBlueBackground(text string) string {
	return "\x1b[44m" + text + reset
}
func WithBackground(text string, colour Colour) string {
	if c, ok := colours[colour]; ok {
		return c + text + reset
	}
	//Default to blue if none resolved
	return WithBlueBackground(text)
}

func main() {

	flag.Parse()

	fmt.Println("Enter the numbers of ghosts 1-12: ")
	var num string
	fmt.Scanln(&num)

	enemynum, err := strconv.Atoi(num)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if enemynum > 0 && enemynum < 13 {
		initialise()
		defer clean()

		//load resources
		err := loadLevelMap(*map_file, enemynum)
		if err != nil {
			log.Println("No se pudo leer el mapa", err)
			return
		}

		err = loadConfig(*configFile)
		if err != nil {
			log.Println("failed to load configuration:", err)
			return
		}
		//se crea el canal
		input := make(chan string)
		go func(ch chan<- string) {
			for {
				input, err := readInput()
				if err != nil {
					log.Println("error reading input:", err)
					ch <- "ESC"
				}
				ch <- input
			}
		}(input)

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

			moveEnemy()

			// process collisions
			for _, g := range enemies {
				if player.row == g.position.row && player.col == g.position.col {
					enemyStatusMx.RLock()
					if g.status == enemyNormalMode {
						lives = lives - 1
						if lives != 0 {
							moveCursorEmoji(player.row, player.col)
							fmt.Print(cfg.Death)
							moveCursorEmoji(len(level_map)+2, 0)
							enemyStatusMx.RUnlock()
							updateEnemy(enemies, enemyNormalMode)
							time.Sleep(1000 * time.Millisecond)
							player.row, player.col = player.startRow, player.startCol
						}
					} else if g.status == enemyBlueMode {
						enemyStatusMx.RUnlock()
						updateEnemy([]*enemy{g}, enemyNormalMode)
						g.position.row, g.position.col = g.position.startRow, g.position.startCol
					}
				}
			}

			// update screen
			printToScreen()

			// check game over
			if numDots == 0 || lives <= 0 {
				if lives == 0 {
					moveCursorEmoji(player.row, player.col)
					fmt.Print(cfg.Death)
					MoveCursor(player.startRow, player.startCol-1)
					fmt.Print("GAME OVER")
					moveCursorEmoji(len(level_map)+2, 0)
				} else if numDots == 0 {
					moveCursorEmoji(player.row, player.col)
					fmt.Print(cfg.Death)
					MoveCursor(player.startRow, player.startCol-1)
					fmt.Print("YOU WIN!!")
					moveCursorEmoji(len(level_map)+2, 0)
				}
				break
			}
			// repeat
			time.Sleep(200 * time.Millisecond)
		}
	} else {
		fmt.Println("Inroduce un numero valido entre 1 a 12")
		os.Exit(1)
	}
}
