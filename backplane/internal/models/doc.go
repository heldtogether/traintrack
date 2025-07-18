/*
Package datasets provides dataset creation and listing via HTTP.

The main entrypoint is NewHandler, which returns an http.Handler for models.
To integrate this into your app:

	// construct dependencies
	db := ...
	store := NewStore(db)
	creator := NewCreator(store, ...)
	handler := NewHandler(creator, store)

	http.HandleFunc("/models", handler.Models)

Interfaces like Creator and Lister allow for easy mocking and dependency injection.
*/
package models
