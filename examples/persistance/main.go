package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/redis/go-redis/v9"
)

type Storer interface {
	Store(key string, data []byte) error
	Load(key string) ([]byte, error)
}

func WithPersistance(store Storer) func(actor.ReceiveFunc) actor.ReceiveFunc {
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

type RedisStore struct {
	client *redis.Client
}

func newRedisStore(c *redis.Client) *RedisStore {
	return &RedisStore{
		client: c,
	}
}

func (r *RedisStore) Store(key string, state []byte) error {
	return r.client.Set(context.TODO(), key, state, 0).Err()
}

func (r *RedisStore) Load(key string) ([]byte, error) {
	val, err := r.client.Get(context.TODO(), key).Result()
	return []byte(val), err
}

func main() {
	var (
		e           = actor.NewEngine()
		redisClient = redis.NewClient(&redis.Options{
			Addr:     "localhost:6379",
			Password: "", // no password set
			DB:       0,  // use default DB
		})
		store = newRedisStore(redisClient)
		pid   = e.Spawn(newPlayerState(100, "James"), "playerState", actor.WithMiddleware(WithPersistance(store)))
	)
	time.Sleep(time.Second * 1)
	e.Send(pid, TakeDamage{Amount: 9})
	time.Sleep(time.Second * 1)
	wg := &sync.WaitGroup{}
	e.Poison(pid, wg)
	wg.Wait()
}
