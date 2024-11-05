package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"regexp"
	"sync"
	"time"

	"github.com/fancom/hollywood/actor"
)

type Storer interface {
	Store(key string, data []byte) error
	Load(key string) ([]byte, error)
}

func WithPersistence(store Storer) func(actor.ReceiveFunc) actor.ReceiveFunc {
	return func(next actor.ReceiveFunc) actor.ReceiveFunc {
		return func(c *actor.Context) {
			switch c.Message().(type) {
			case actor.Initialized:
				p, ok := c.Receiver().(Persister)
				if !ok {
					next(c)
					return
				}
				b, err := store.Load(c.PID().String())
				if err != nil {
					fmt.Println(err)
					next(c)
					return
				}
				var data map[string]any
				if err := json.Unmarshal(b, &data); err != nil {
					log.Fatal(err)
				}
				if err := p.LoadState(data); err != nil {
					fmt.Println("load state error:", err)
					next(c)
				}
			case actor.Stopped:
				if p, ok := c.Receiver().(Persister); ok {
					s, err := p.State()
					if err != nil {
						fmt.Println("could not get state", err)
						next(c)
						return
					}
					if err := store.Store(c.PID().String(), s); err != nil {
						fmt.Println("failed to store the state", err)
						next(c)
						return
					}
				}
			}
			next(c)
		}
	}
}

type Persister interface {
	State() ([]byte, error)
	LoadState(map[string]any) error
}

type State struct {
	Health   int    `json:"health"`
	Username string `json:"username"`
}

type PlayerState struct {
	Health   int
	Username string
}

type TakeDamage struct {
	Amount int
}

func (p *PlayerState) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:
	case actor.Stopped:
	case TakeDamage:
		p.Health -= msg.Amount
		fmt.Println("took damage, health ", p.Health)
	}
}

func newPlayerState(health int, username string) actor.Producer {
	return func() actor.Receiver {
		return &PlayerState{
			Health:   health,
			Username: username,
		}
	}
}

func (p *PlayerState) LoadState(data map[string]any) error {
	fmt.Println("player loading state", data)
	p.Health = int(data["health"].(float64))
	p.Username = data["username"].(string)
	return nil
}

func (p *PlayerState) State() ([]byte, error) {
	state := State{
		Username: p.Username,
		Health:   p.Health,
	}
	return json.Marshal(state)
}

type fileStore struct {
	path string
}

func newFileStore() *fileStore {
	// make a tmp dir:
	tmpdir := "/tmp/persistenceexample"
	err := os.Mkdir(tmpdir, 0755)
	if err != nil && !os.IsExist(err) {
		log.Fatal(err)
	}
	return &fileStore{
		path: tmpdir,
	}
}

// Store the state in a file name key
func (r *fileStore) Store(key string, state []byte) error {
	key = safeFileName(key)
	return os.WriteFile(path.Join(r.path, key), state, 0755)
}

func (r *fileStore) Load(key string) ([]byte, error) {
	key = safeFileName(key)
	return os.ReadFile(path.Join(r.path, key))
}

func main() {
	e, err := actor.NewEngine(actor.NewEngineConfig())
	if err != nil {
		log.Fatal(err)
	}
	var (
		store = newFileStore()
		pid   = e.Spawn(
			newPlayerState(100, "James"),
			"playerState",
			actor.WithMiddleware(WithPersistence(store)),
			actor.WithID("james"))
	)
	time.Sleep(time.Second * 1)
	e.Send(pid, TakeDamage{Amount: 9})
	time.Sleep(time.Second * 1)
	wg := &sync.WaitGroup{}
	e.Poison(pid, wg)
	wg.Wait()
}

var safeRx = regexp.MustCompile(`[^a-zA-Z0-9]`)

// safeFileName replaces all characters azAZ09 with _
func safeFileName(s string) string {
	res := safeRx.ReplaceAllString(s, "_")
	return res
}
