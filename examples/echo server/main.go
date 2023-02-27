package main

/*
	trying to create a crud app with echo and actor frameworks
*/

import (
	"fmt"
	"net/http"

	"github.com/anthdm/hollywood/actor"
	"github.com/labstack/echo/v4"
)

const PORT string = "3000"

type App struct {
	Actors []string

	Echo        *echo.Echo
	ActorEngine *actor.Engine
}

func NewApp() *App {

	a := actor.NewEngine()
	e := echo.New()
	s := make([]string, 0)

	app := &App{
		Actors:      s,
		Echo:        e,
		ActorEngine: a,
	}
	r := actor.New(app.ActorEngine, Config{"127.0.0.1:" + PORT})
	app.ActorEngine.WithRemote(r)
	return app
}

type message struct {
	data string
}

type User struct {
}

func newUser() actor.Receiver {
	return &User{}
}

func (f *User) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Started:
		fmt.Println("User started", ctx.PID())
	case *message:
		fmt.Println("User has received", msg.data)
	}
}
func main() {
	app := NewApp()

	app.Echo.GET("/:actorname/:message", app.SendMessage)
	app.Echo.GET("/", app.AllActors)
	app.Echo.GET("/create/:actorname", app.CreateActor)

	app.Echo.Logger.Fatal(app.Echo.Start(":" + PORT))
}

func (a *App) SendMessage(c echo.Context) error {
	msg := c.Param("message")
	nme := c.Param("actorname")

	if !a.contains(nme) {
		return c.String(http.StatusOK, fmt.Sprintf("FAILED : this actor is not exist %+s", nme))
	}

	pid := actor.NewPID("127.0.0.1:"+PORT, nme)
	//here has a issue . message cant send because remoter and engine
	a.ActorEngine.Send(pid, msg)
	return c.String(http.StatusOK, fmt.Sprintf("message send : %s ", msg))
}

func (a *App) AllActors(c echo.Context) error {
	c.String(http.StatusOK, "You will see al the actors below.\n\nIn order to create Actor   ->-> /create/linaund1000 \n\n")
	return c.JSON(http.StatusOK, a.Actors)
}

func (a *App) CreateActor(c echo.Context) error {
	actorName := c.Param("actorname")
	if a.contains(actorName) {
		return c.String(http.StatusOK, "this actor is already exist you cant create with this name ")
	}

	actrPid := a.ActorEngine.Spawn(newUser, actorName)
	a.Actors = append(a.Actors, actorName)

	return c.String(http.StatusOK, fmt.Sprintf("Hello spawny you are created! your name:  %+v your adress:%s", actorName, actrPid.GetAddress()))
}

func (a *App) contains(actorName string) bool {
	for _, v := range a.Actors {
		if v == actorName {
			return true
		}
	}
	return false
}
