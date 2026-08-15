package notes

import (
	"errors"
	"fmt"
)

// Page returns a slice of notes for the requested page.
// perPage <= 0 disables pagination and returns all notes.
func Page(ns []Note, page, perPage int) (paged []Note, currentPage, totalPages int, err error) {
	if perPage <= 0 {
		return ns, 0, 0, nil
	}
	if page < 1 {
		return nil, 0, 0, errors.New("page must be greater than zero")
	}

	totalPages = (len(ns) + perPage - 1) / perPage
	if totalPages == 0 {
		totalPages = 1
	}
	if page > totalPages {
		return nil, 0, 0, fmt.Errorf("page %d does not exist; only %d page(s) available", page, totalPages)
	}

	start := (page - 1) * perPage
	if start >= len(ns) {
		return nil, page, totalPages, nil
	}
	end := start + perPage
	if end > len(ns) {
		end = len(ns)
	}
	return ns[start:end], page, totalPages, nil
}
