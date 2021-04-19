package handlers

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/romankhrapachev/go-redis-example/data"
)

// ShowAlbum handles GET requests
func (a *Albums) GetAlbum(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	// Retrieve the id from the request URL query string. If there is
	// no id key in the query string then Get() will return an empty
	// string. We check for this, returning a 400 Bad Request response
	// if it's missing.
	vars := mux.Vars(r)
	id, ok := vars["id"]
	if !ok {
		http.Error(w, http.StatusText(400), 400)
		return
	}
	a.l.Println("[DEBUG] get record id", id)

	// Call the GetAlbumByID() function passing in the user-provided id.
	// If there's no matching album found, return a 404 Not Found
	// response. In the event of any other errors, return a 500
	// Internal Server Error response.
	bk, err := a.d.GetAlbumByID(id)

	switch err {
	case nil:

	case data.ErrNoAlbum:
		a.l.Println("[ERROR] fetching album", err)
		http.NotFound(w, r)
		return
	default:
		a.l.Println("[ERROR] fetching album", err)
		http.Error(w, http.StatusText(500), 500)
		return
	}

	// Write the album details as plain text to the client.
	fmt.Fprintf(w, "%s by %s: £%.2f [%d likes] \n", bk.Title, bk.Artist, bk.Price, bk.Likes)
}

func (a *Albums) ListPopular(w http.ResponseWriter, r *http.Request) {

	// Call the FindTopThree() function, returning a return a 500 Internal
	// Server Error response if there's any error.
	albums, err := a.d.FindTopThree()
	if err != nil {
		http.Error(w, http.StatusText(500), 500)
		return
	}

	// Loop through the 3 albums, writing the details as a plain text list
	// to the client.
	for i, ab := range albums {
		fmt.Fprintf(w, "%d) %s by %s: £%.2f [%d likes] \n", i+1, ab.Title, ab.Artist, ab.Price, ab.Likes)
	}
}
