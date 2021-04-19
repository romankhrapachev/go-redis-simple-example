package handlers

import (
	"net/http"

	"github.com/romankhrapachev/go-redis-example/data"
)

// Update handles PUT requests to update album
func (a *Albums) UpdateLikes(w http.ResponseWriter, r *http.Request) {

	id := r.URL.Query().Get("id")
	a.l.Println("[DEBUG] updating record id", id)

	// Call the IncrementLikesInAlbumByID() function passing in the user-provided
	// id. If there's no album found with that id, return a 404 Not
	// Found response. In the event of any other errors, return a 500
	// Internal Server Error response.
	err := a.d.IncrementLikesInAlbumByID(id)
	if err == data.ErrNoAlbum {
		http.NotFound(w, r)
		return
	} else if err != nil {
		http.Error(w, http.StatusText(500), 500)
		return
	}

	// Redirect the client to the GET /album route, so they can see the
	// impact their like has had.
	http.Redirect(w, r, "/albums/"+id, 303)
}
