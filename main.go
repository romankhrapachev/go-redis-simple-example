package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gomodule/redigo/redis"
	gohandlres "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/romankhrapachev/go-redis-example/data"
	"github.com/romankhrapachev/go-redis-example/handlers"
)

func main() {

	l := log.New(os.Stdout, "redis-example", log.LstdFlags)
	// Create database instance
	// Initialize a connection pool and assign it to the pool variable.
	pool := &redis.Pool{
		MaxIdle:     10,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", "localhost:6379")
		},
	}
	db := data.NewAlbumsDB(l, pool)

	// create the handlers
	ah := handlers.NewAlbums(l, db)

	// create a new serve mux and register the handlers
	sm := mux.NewRouter()

	// handlers for API
	getR := sm.Methods(http.MethodGet).Subrouter()
	getR.HandleFunc("/albums/{id:[0-9]+}", ah.GetAlbum)
	getR.HandleFunc("/popular", ah.ListPopular)

	putR := sm.Methods(http.MethodPut).Subrouter()
	putR.HandleFunc("/like", ah.UpdateLikes).Queries("id", "{[0-9]+}")

	// CORS
	ch := gohandlres.CORS(gohandlres.AllowedOrigins([]string{"*"}))

	// create a new server
	s := &http.Server{
		Addr:         ":9090",           // configure the bind address
		Handler:      ch(sm),            // set the defoult handler
		ErrorLog:     l,                 // set the logger for the server
		IdleTimeout:  120 * time.Second, // max time for connections using TCP Keep-Alive
		ReadTimeout:  5 * time.Second,   // max time to read request from the client
		WriteTimeout: 10 * time.Second,  // max time to write response  to yhe client
	}

	// start the server
	go func() {
		l.Println("Starting server on port 9090")
		err := s.ListenAndServe()
		if err != nil {
			l.Fatal(err.Error())
		}
	}()

	// trap sigterm or interupt and gracefully shutdown the server
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, os.Kill)

	// Block until a signal is received.
	sig := <-c
	log.Println("Got signal:", sig)

	// gracefully shutdown the server, waiting max 30 seconds for current operations to complete
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	s.Shutdown(ctx)
}
