package handlers

import (
	"log"

	"github.com/romankhrapachev/go-redis-example/data"
)

// Albums handler for getting and updating albums
type Albums struct {
	l *log.Logger
	d *data.AlbumsDB
}

// Returns a new albums handler with the given logger
func NewAlbums(l *log.Logger, d *data.AlbumsDB) *Albums {
	return &Albums{l, d}
}
