package server

import (
	"bufio"
	"encoding/json"
	"log"
	"math/rand"
	"os"
	"time"
)

var Words map[string][]string

type Player struct {
	Name  string
	Score int
	//the current round
	Round int
}
type GameEngine struct {
	Games []*Game
	Words []string
}
type Guess struct {
	Player string
	Text   string
	Time   time.Time
}
type Round struct {
	Drawer    string
	GuessWord string
	Guess     []Guess
	EndTime   time.Time
	Winners   []Guess
}

const STARTED = 1
const CHOOSING = 2
const GUESSING = 3
const SHOW_SCORES = 4
const OVER = 5

type Game struct {
	TimeInSeconds int
	Language      string
	Players       []Player
	Rounds        []Round
	Code          string
	CurrPlayer    int
	//Choosing, Drawing time, over
	State             int
	TimesDrawing      int
	Savedsocketreader []*socketReader
	Words             []string
}

func (g Guess) Score(correct string, end time.Time) int {
	if correct == g.Text {
		return int(end.Sub(g.Time).Seconds())
	}
	return 0
}

func loadWords(filename string) []string {
	file, err := os.Open(filename)

	if err != nil {
		log.Fatalf("failed opening file: %s", err)
	}

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	var txtlines []string

	for scanner.Scan() {
		txtlines = append(txtlines, scanner.Text())
	}
	return txtlines
}
func init() {
	rand.Seed(time.Now().UTC().UnixNano())
	Words = make(map[string][]string)
	Words["en"] = loadWords("en.txt")
}

func (g *Game) Start() {
	if g.State == 0 {
		g.State = STARTED
		g.NewWords()
	}
}
func (g *Game) NewWords() {
	w1 := Words[g.Language][rand.Intn(len(Words[g.Language]))]
	w2 := Words[g.Language][rand.Intn(len(Words[g.Language]))]
	w3 := Words[g.Language][rand.Intn(len(Words[g.Language]))]
	g.Savedsocketreader[g.CurrPlayer].writeMsg("", "words:"+w1+","+w2+","+w3)
	g.Words = []string{w1, w2, w3}
	g.State = CHOOSING
}

func (g *Game) SelectAndStart(player string, i int) {
	if g.State == CHOOSING && g.Players[g.CurrPlayer].Name == player {
		g.NewRound(g.Words[i], player)
	}
}

func (g *Game) Join(playerName string) string {
	for _, p := range g.Players {
		if p.Name == playerName {
			return g.GameState()
		}
	}
	g.Players = append(g.Players, Player{Name: playerName, Score: 0, Round: 0})
	return g.GameState()
}
func (g *Game) NewRound(word, drawer string) {
	g.Rounds = append(g.Rounds, Round{Drawer: drawer, GuessWord: word, Guess: make([]Guess, 0), EndTime: time.Now().Add(time.Second * time.Duration(g.TimeInSeconds))})
	g.State = GUESSING
	g.BroadcastToAll("start:" + drawer)
	time.AfterFunc(time.Duration(g.TimeInSeconds)*time.Second, func() {
		g.ScoreUpdate()
		g.BroadcastToAll("ended!" + g.GameState())
		g.CurrPlayer = (g.CurrPlayer + 1) % len(g.Players)
		g.NewWords()
	})
}
func (g *Game) ScoreUpdate() {
	if len(g.Rounds) > 0 {
		r := g.Rounds[len(g.Rounds)-1]
		for id, w := range r.Winners {
			for i := range g.Players {
				if g.Players[i].Name == w.Player {
					g.Players[i].Score += (len(g.Players) - id) * w.Score(r.GuessWord, r.EndTime)
				}
			}
		}
	}
}
func (r *Round) IsWinner(user string) bool {
	for _, g := range r.Winners {
		if g.Player == user {
			return true
		}
	}
	return false
}
func (g *Game) BroadcastToAll(text string) {
	for _, i := range g.Savedsocketreader {
		i.writeMsg("", text)
	}
}
func (g *Game) NewGuess(name string, word string) (string, bool, bool, bool) {
	if len(g.Rounds) > 0 {
		r := g.Rounds[len(g.Rounds)-1]
		if time.Now().Before(r.EndTime) {
			return r.NewGuess(name, word)
		} else {
		}
	}
	return word, true, false, false
}

//1. bool means broadcast
//bool means broadcast to all
func (r *Round) NewGuess(user, word string) (string, bool, bool, bool) {
	if r.IsWinner(user) || user == r.Drawer {
		return word, false, true, true
	}
	if word == r.GuessWord {
		r.Winners = append(r.Winners, Guess{Text: word, Time: time.Now(), Player: user})
		return word, false, true, true
	}
	r.Guess = append(r.Guess, Guess{Text: word, Time: time.Now(), Player: user})
	return word, true, true, false
}

//New Game generates a new game
func (ge *GameEngine) NewGame(timeSeconds int, Language string, code string) *Game {
	game := Game{TimeInSeconds: timeSeconds, Language: Language, Players: make([]Player, 0), Code: code, Rounds: make([]Round, 0)}
	ge.Games = append(ge.Games, &game)
	return &game
}

func (ge *GameEngine) Join(game string, user string) (*Game, string) {
	for i := range ge.Games {
		if ge.Games[i].Code == game {
			return ge.Games[i], ge.Games[i].Join(user)
		}
	}
	return nil, ""
}

func (g *Game) GameState() string {
	bytes, e := json.Marshal(g.Players)
	if e != nil {
		log.Println(e)
	}
	return string(bytes)
}
