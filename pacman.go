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

// Se obtiene la configuración del json y el nombre del archivo del mapa/laberinto
var (
	configFile = flag.String("config-file", "config.json", "path to custom configuration file")
	mazeFile   = flag.String("maze-file", "maze01.txt", "path to a custom maze file")
)

// Informacion del movimiento anterior y actual
type sprite struct {
	row      int
	col      int
	startRow int
	startCol int
}

type enemieStruct struct {
	position sprite
	status   EnemieStatus
}

type EnemieStatus string

const (
	EnemyStatusNormal EnemieStatus = "Normal"
	EnemyStatusBlue   EnemieStatus = "Blue"
)

var enemiesStatusMx sync.RWMutex
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

var enemies []*enemieStruct
var score int
var cfg config
var player sprite
var maze_level []string
var lives = 3
var numDots int

func loadGame(file string) error {
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

// Lee el archivo del mapa/laberinto
func loadLevelMap(file string, numenemy int) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		maze_level = append(maze_level, line)
	}

	var aux int = 0
	for row, line := range maze_level {
		for col, char := range line {
			switch char {
			case 'P':
				player = sprite{row, col, row, col}
			case 'G':
				if aux < numenemy {
					enemies = append(enemies, &enemieStruct{sprite{row, col, row, col}, EnemyStatusNormal})
					aux++
				}
			case '.':
				numDots++
			}
		}
	}

	return nil
}

// Se hace la configuracion para poder obtener el input de las teclas en la terminal
func initialiseGame() {
	cbTerm := exec.Command("stty", "cbreak", "-echo")
	cbTerm.Stdin = os.Stdin

	err := cbTerm.Run()
	if err != nil {
		log.Fatalln("unable to activate cbreak mode:", err)
	}
}

func moveCursor(row, col int) {
	if cfg.UseEmoji {
		simpleansi.MoveCursor(row, col*2)
	} else {
		simpleansi.MoveCursor(row, col)
	}
}

// Muesra las vidas del jugador
func showLives() string {
	buf := bytes.Buffer{}
	for i := lives; i > 0; i-- {
		buf.WriteString(cfg.Player)
	}
	return buf.String()
}

// Lee el input del teclado que se envia, en este caso las teclas de las flechas
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

func move(oldRow, oldCol int, dir string) (newRow, newCol int) {
	newRow, newCol = oldRow, oldCol

	switch dir {
	case "UP":
		newRow = newRow - 1
		if newRow < 0 {
			newRow = len(maze_level) - 1
		}
	case "DOWN":
		newRow = newRow + 1
		if newRow == len(maze_level)-1 {
			newRow = 0
		}
	case "RIGHT":
		newCol = newCol + 1
		if newCol == len(maze_level[0]) {
			newCol = 0
		}
	case "LEFT":
		newCol = newCol - 1
		if newCol < 0 {
			newCol = len(maze_level[0]) - 1
		}
	}

	if maze_level[newRow][newCol] == '#' {
		newRow = oldRow
		newCol = oldCol
	}

	return
}

// Mueve al jugador dentro del laberinto
func movePlayer(dir string) {
	player.row, player.col = move(player.row, player.col, dir)

	removeDot := func(row, col int) {
		maze_level[row] = maze_level[row][0:col] + " " + maze_level[row][col+1:]
	}

	switch maze_level[player.row][player.col] {
	case '.':
		numDots--
		score++
		removeDot(player.row, player.col)
	case 'X':
		score += 10
		removeDot(player.row, player.col)
		go power()
	}
}
func enemyMoveListDirection() string {
	dir := rand.Intn(8)
	move := map[int]string{
		0: "UP",
		1: "DOWN",
		2: "RIGHT",
		3: "LEFT",
	}
	return move[dir]
}

// Actualiza la posicion de los enemigos
func updateEnemies(enemies []*enemieStruct, enemieStatus EnemieStatus) {
	enemiesStatusMx.Lock()
	defer enemiesStatusMx.Unlock()
	for _, g := range enemies {
		g.status = enemieStatus
		//g.position.row, g.position.col = g.position.startRow, g.position.startCol
	}
}

var pillTimer *time.Timer

// Funcion que activa el poder del poder
func power() {
	pillMx.Lock()
	updateEnemies(enemies, EnemyStatusBlue)
	if pillTimer != nil {
		pillTimer.Stop()
	}
	pillTimer = time.NewTimer(time.Second * cfg.PillDurationSecs)
	pillMx.Unlock()
	//canal para el timer
	<-pillTimer.C
	pillMx.Lock()
	pillTimer.Stop()
	updateEnemies(enemies, EnemyStatusNormal)
	pillMx.Unlock()
}

func moveEnemies() {
	for _, g := range enemies {
		dir := enemyMoveListDirection()
		g.position.row, g.position.col = move(g.position.row, g.position.col, dir)
	}
}

func showCounter(wait int) {
	for i := 0; i < wait; i++ {
		time.Sleep(750 * time.Millisecond)
		fmt.Println("RESTARTING in " + strconv.Itoa(3-i) + " seconds")
	}
}

func cleanTerminal() {
	cookedTerm := exec.Command("stty", "-cbreak", "echo")
	cookedTerm.Stdin = os.Stdin

	err := cookedTerm.Run()
	if err != nil {
		log.Fatalln("unable to activate cooked mode:", err)
	}
}
func printScreen() {
	simpleansi.ClearScreen()
	for _, line := range maze_level {
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

	enemiesStatusMx.RLock()
	for _, g := range enemies {
		moveCursor(g.position.row, g.position.col)
		if g.status == EnemyStatusNormal {
			fmt.Printf(cfg.Ghost)
		} else if g.status == EnemyStatusBlue {
			fmt.Printf(cfg.GhostBlue)
		}
	}
	enemiesStatusMx.RUnlock()

	moveCursor(len(maze_level)+1, 0)

	livesRemaining := strconv.Itoa(lives) //converts lives int to a string
	if cfg.UseEmoji {
		livesRemaining = showLives()
	}

	fmt.Println("Score:", score, "\tlives:", livesRemaining)
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
		// se inicia el juego
		initialiseGame()
		defer cleanTerminal()

		//Se lee el mapa y el juego
		err := loadLevelMap(*mazeFile, numenemy)
		if err != nil {
			log.Println("failed to load maze:", err)
			return
		}

		err = loadGame(*configFile)
		if err != nil {
			log.Println("failed to load configuration:", err)
			return
		}

		//se crea el canal para el input, el movimiento del jugador
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
		for {
			//Se usa un canal para el movimiento del jugador
			select {
			case inp := <-input:
				if inp == "ESC" {
					lives = 0
				}
				movePlayer(inp)
			default:
			}
			moveEnemies()
			// SE Checa las colisiones entre enemigo y jugador
			for _, g := range enemies {
				if player.row == g.position.row && player.col == g.position.col {
					enemiesStatusMx.RLock()
					if g.status == EnemyStatusNormal {
						lives = lives - 1
						if lives != 0 {
							moveCursor(player.row, player.col)
							fmt.Print(cfg.Death)
							moveCursor(len(maze_level)+2, 0)
							enemiesStatusMx.RUnlock()
							updateEnemies(enemies, EnemyStatusNormal)
							go showCounter(3)
							time.Sleep(3000 * time.Millisecond)
							player.row, player.col = player.startRow, player.startCol
						}
					} else if g.status == EnemyStatusBlue {
						enemiesStatusMx.RUnlock()
						updateEnemies([]*enemieStruct{g}, EnemyStatusNormal)
						g.position.row, g.position.col = g.position.startRow, g.position.startCol
					}
				}
			}
			//Se limpia la terminal
			fmt.Print("\033[H\033[2J")
			// Se actualiza la terminal con todo movido
			printScreen()

			//Checa si quedan vidas
			if numDots == 0 || lives <= 0 {
				if lives == 0 {
					moveCursor(player.row, player.col)
					fmt.Print(cfg.Death)
					moveCursor(player.startRow, player.startCol-1)
					fmt.Print("GAME OVER")
					moveCursor(len(maze_level)+2, 0)
				}
				//Si ya no hay puntos se gana el juego
				if numDots == 0 {
					fmt.Println("You Win")
				}
				break
			}

			// repeat
			time.Sleep(200 * time.Millisecond)
		}
	} else {
		fmt.Println("El numero de fantasmas debe de ser entre 1 a 12")
		os.Exit(1)
	}

}
