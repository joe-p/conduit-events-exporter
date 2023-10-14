package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/lesismal/nbio/nbhttp/websocket"
)

type Pubsub struct {
	subs map[string]map[uuid.UUID]chan string
}

func NewPubsub() *Pubsub {
	ps := &Pubsub{}
	ps.subs = make(map[string]map[uuid.UUID]chan string)
	return ps
}

func (ps *Pubsub) Subscribe(topic string) uuid.UUID {
	ch := make(chan string, 1)
	uid := uuid.New()

	if ps.subs[topic] == nil {
		ps.subs[topic] = map[uuid.UUID]chan string{}
	}

	ps.subs[topic][uid] = ch
	return uid
}

func (ps *Pubsub) Publish(topic string, msg string) {
	for _, ch := range ps.subs[topic] {
		go func(ch chan string) {
			ch <- msg
		}(ch)
	}
}

func (ps *Pubsub) Unsubscribe(topic string, uid uuid.UUID) {
	close(ps.subs[topic][uid])
	delete(ps.subs[topic], uid)
}

func getUpgrader(appIDString string, ps *Pubsub, uid uuid.UUID) *websocket.Upgrader {
	u := websocket.NewUpgrader()
	u.CheckOrigin = func(r *http.Request) bool { return true }

	u.OnOpen(func(c *websocket.Conn) {
		c.WriteMessage(websocket.TextMessage, []byte("WELCOME: "+appIDString))
	})

	u.OnClose(func(c *websocket.Conn, err error) {
		ps.Unsubscribe(appIDString, uid)
		fmt.Printf("closed %s %s for app %s\n", c.RemoteAddr().String(), uid.String(), appIDString)
	})

	return u
}

func pubLoop(ps *Pubsub) {
	i := 0
	for {
		appID := fmt.Sprintf("%d", i%3)
		ps.Publish(appID, fmt.Sprintf("msg %d", i))
		time.Sleep(time.Millisecond * 1000)
		fmt.Println("Published ", i)
		i++
	}
}

func main() {
	ps := NewPubsub()
	go pubLoop(ps)

	r := chi.NewRouter()

	r.Use(middleware.Logger)

	r.Route("/{appID}", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			appIDString := chi.URLParam(r, "appID")

			_, err := strconv.ParseUint(appIDString, 10, 64)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			uid := ps.Subscribe(appIDString)
			upgrader := getUpgrader(appIDString, ps, uid)

			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				ps.Unsubscribe(appIDString, uid)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			conn.SetReadDeadline(time.Time{})

			go func() {
				for channelMsg := range ps.subs[appIDString][uid] {
					conn.SetWriteDeadline(time.Now().Add(time.Second * 5))
					err = conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("got %s\n", channelMsg)))

					if err != nil {
						conn.Close()
					}
				}

				conn.Close()
			}()

		})
	})

	http.ListenAndServe("localhost:3333", r)
}
