package data

import (
	"errors"
	"log"

	"github.com/gomodule/redigo/redis"
)

var ErrNoAlbum = errors.New("no album found")

// Define a custom struct to hold Album data. The struct tags
// indicate to redigo how to assign the data from the reply into
// the struct.
type Album struct {
	Title  string  `redis:"title"`
	Artist string  `redis:"artist"`
	Price  float64 `redis:"price"`
	Likes  int     `redis:"likes"`
}

type AlbumsDB struct {
	log  *log.Logger // Base logger
	pool *redis.Pool // Declare a pool variable to hold the pool of Redis connections.
}

func NewAlbumsDB(l *log.Logger, p *redis.Pool) *AlbumsDB {
	return &AlbumsDB{l, p}
}

func (a *AlbumsDB) GetAlbumByID(id string) (*Album, error) {
	// Use the connection pool's Get() method to fetch a single Redis
	// connection from the pool.
	conn := a.pool.Get()

	// Importantly, use defer and the connection's Close() method to
	// ensure that the connection is always returned to the pool before
	// FindAlbum() exits.
	defer conn.Close()
	a.log.Println("[DEBUG] HGETALL album:", id)

	// Fetch the details of a specific album. If no album is found
	// the given id, the []interface{} slice returned by redis.Values
	// will have a length of zero. So check for this and return an
	// ErrNoAlbum error as necessary.
	values, err := redis.Values(conn.Do("HGETALL", "album:"+id))
	if err != nil {
		a.log.Println("Unable to get album, error", err)
		return nil, err
	} else if len(values) == 0 {
		a.log.Println("No values, error", err)
		return nil, ErrNoAlbum
	}

	var album Album
	err = redis.ScanStruct(values, &album)
	if err != nil {
		return nil, err
	}

	return &album, nil
}

func (a *AlbumsDB) IncrementLikesInAlbumByID(id string) error {
	conn := a.pool.Get()
	defer conn.Close()

	// Before we do anything else, check that an album with the given
	// id exists. The EXISTS command returns 1 if a specific key exists
	// in the database, and 0 if it doesn't.
	exists, err := redis.Int(conn.Do("EXISTS", "album:"+id))
	if err != nil {
		return err
	} else if exists == 0 {
		return ErrNoAlbum
	}

	// Use the MULTI command to inform Redis that we are starting a new
	// transaction. The conn.Send() method writes the command to the
	// connection's output buffer -- it doesn't actually send it to the
	// Redis server... despite it's name!
	err = conn.Send("MULTI")
	if err != nil {
		return err
	}

	// Increment the number of likes in the album hash by 1. Because it
	// follows a MULTI command, this HINCRBY command is NOT executed but
	// it is QUEUED as part of the transaction. We still need to check
	// the reply's Err field at this point in case there was a problem
	// queueing the command.
	err = conn.Send("HINCRBY", "album:"+id, "likes", 1)
	if err != nil {
		return err
	}
	// And we do the same with the increment on our sorted set.
	err = conn.Send("ZINCRBY", "likes", 1, id)
	if err != nil {
		return err
	}

	// Execute both commands in our transaction together as an atomic
	// group. EXEC returns the replies from both commands but, because
	// we're not interested in either reply in this example, it
	// suffices to simply check for any errors. Note that calling the
	// conn.Do() method flushes the previous commands from the
	// connection output buffer and sends them to the Redis server.
	_, err = conn.Do("EXEC")
	if err != nil {
		return err
	}

	return nil

}

func (a *AlbumsDB) FindTopThree() ([]*Album, error) {
	conn := a.pool.Get()
	defer conn.Close()

	// Begin an infinite loop. In a real application limit this to a set number of attempts,
	// and return an error if the transaction doesn't successfully complete within those attempts.
	for {
		// Instruct Redis to watch the likes sorted set for any changes.
		_, err := conn.Do("WATCH", "likes")
		if err != nil {
			return nil, err
		}

		// Use the ZREVRANGE command to fetch the album ids with the
		// highest score (i.e. most likes) from our 'likes' sorted set.
		// The ZREVRANGE start and stop values are zero-based indexes,
		// so we use 0 and 2 respectively to limit the reply to the top
		// three. Because ZREVRANGE returns an array response, we use
		// the Strings() helper function to convert the reply into a
		// []string.
		ids, err := redis.Strings(conn.Do("ZREVRANGE", "likes", 0, 2))
		if err != nil {
			return nil, err
		}

		// Use the MULTI command to inform Redis that we are starting
		// a new transaction.
		err = conn.Send("MULTI")
		if err != nil {
			return nil, err
		}

		// Loop through the ids returned by ZREVRANGE, queuing HGETALL
		// commands to fetch the individual album details.
		for _, id := range ids {
			err := conn.Send("HGETALL", "album:"+id)
			if err != nil {
				return nil, err
			}
		}

		// Execute the transaction. Importantly, use the redis.ErrNil
		// type to check whether the reply from EXEC was nil or not. If
		// it is nil it means that another client changed the WATCHed
		// likes sorted set, so we use the continue command to re-run
		// the loop.
		replies, err := redis.Values(conn.Do("EXEC"))
		if err == redis.ErrNil {
			log.Println("trying again")
			continue
		} else if err != nil {
			return nil, err
		}

		// Create a new slice to store the album details.
		albums := make([]*Album, 3)

		// Iterate through the array of response objects, using the
		// ScanStruct() function to assign the data to Album structs.
		for i, reply := range replies {
			var album Album
			err = redis.ScanStruct(reply.([]interface{}), &album)
			if err != nil {
				return nil, err
			}

			albums[i] = &album
		}

		return albums, nil
	}
}
