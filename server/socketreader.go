package server

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}
var savedsocketreader []*socketReader
var gameEngine GameEngine

func SocketReaderCreate(w http.ResponseWriter, r *http.Request) {
	log.Println("socket request")
	if savedsocketreader == nil {
		savedsocketreader = make([]*socketReader, 0)
	}

	defer func() {
		err := recover()
		if err != nil {
			log.Println(err)
		}
		r.Body.Close()

	}()
	con, _ := upgrader.Upgrade(w, r, nil)

	ptrSocketReader := &socketReader{
		con: con,
	}

	savedsocketreader = append(savedsocketreader, ptrSocketReader)

	ptrSocketReader.startThread()
}

//socketReader struct
type socketReader struct {
	con   *websocket.Conn
	mode  int
	name  string
	score int
	//Game can be null
	Game *Game
}

func (i *socketReader) broadcast(str string) {
	for _, g := range savedsocketreader {

		if g == i {
			// no send message to himself
			continue
		}

		if g.mode == 1 {
			// no send message to connected user before user write his name
			continue
		}
		g.writeMsg(i.name, str)
	}
}

func (i *socketReader) read() {
	_, b, er := i.con.ReadMessage()
	if er != nil {
		panic(er)
	}
	log.Println(i.name + " " + string(b))
	log.Println(i.mode)
	n := string(b)
	arr := strings.SplitN(n, ":", 2)
	command := arr[0]
	if i.mode == 1 {
		i.name = string(b)
		i.writeMsg("System", "Welcome "+i.name+", please write a message and we will broadcast it to other users.")
		i.mode = 2 // real msg mode

		return
	}
	if command == "draw" {
		i.broadcastToAll("draw:" + arr[1])
	}
	if command == "clear" && i.Game.Players[i.Game.CurrPlayer].Name == i.name {
		i.broadcastToAll("clear")
	}
	if command == "guess" {
		if i.Game != nil {

			text, toAll, toWinner, win := i.Game.NewGuess(i.name, arr[1])

			if toAll {
				i.broadcastToAll("guess:" + i.name + "," + text)
			}
			if toWinner {
				i.broadcastToWinner("guess:" + i.name + "," + arr[1])
			}
			if win {
				i.broadcastToAll("win:" + i.name)
			}
		}
	}
	if command == "create" {
		t := strings.Split(arr[1], ",")
		time, _ := strconv.Atoi(t[0])
		//i.writeMsg("", .GameState())
		i.Game = gameEngine.NewGame(time, t[1], t[2])
		i.Game.Savedsocketreader = append(i.Game.Savedsocketreader, i)
		i.Game.Join(i.name)
		i.broadcastToAll("created")
	}
	if command == "join" {
		g, _ := gameEngine.Join(arr[1], i.name)
		i.Game = g
		i.Game.Savedsocketreader = append(i.Game.Savedsocketreader, i)

		log.Println("joined:" + i.name)
		i.writeMsg("", "gameinfo:"+i.Game.GameState())
		i.broadcastToAll("joined:" + i.name)
	}
	if command == "start" {
		if i.Game != nil {
			i.Game.Start()
		}
	}
	if command == "select" {
		if i.Game != nil {
			time, _ := strconv.Atoi(arr[1])
			i.Game.SelectAndStart(i.name, time)
		}
	}
	log.Println(i.name + " " + string(b))
}
func (i *socketReader) broadcastToAll(text string) {
	if i.Game != nil {
		i.Game.BroadcastToAll(text)
	}
}
func (i *socketReader) broadcastToWinner(text string) {
	if i.Game != nil {
		for _, g := range i.Game.Savedsocketreader {
			for _, ir := range i.Game.Rounds[len(i.Game.Rounds)-1].Winners {
				if ir.Player == g.name {
					g.writeMsg("", text)
				}
			}
		}
	}
}
func (i *socketReader) writeMsg(name string, str string) {
	i.con.WriteMessage(websocket.TextMessage, []byte(str))
}

func (i *socketReader) startThread() {
	i.writeMsg("System", "Please write your name")
	i.mode = 1 //mode 1 get user name

	go func() {
		defer func() {
			err := recover()
			if err != nil {
				log.Println(err)
			}
			log.Println("thread socketreader finish")
		}()

		for {
			i.read()
		}

	}()
}
