package pagination

import (
	"math"
	"net/http"
	"strconv"
)

// Pagination holds all pagination metadata for a paginated result set.
type Pagination struct {
	Page       int   // current page (1-based)
	PerPage    int   // items per page
	Total      int64 // total item count
	TotalPages int   // total pages
	HasPrev    bool
	HasNext    bool
	Offset     int // SQL offset for queries
	Limit      int // SQL limit for queries
}

// New constructs a Pagination from page number, items per page, and total count.
// If page < 1 it defaults to 1. If perPage < 1 it defaults to 20.
func New(page, perPage int, total int64) Pagination {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}

	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(perPage)))
	}

	// Clamp page to totalPages (at least 1)
	if totalPages > 0 && page > totalPages {
		page = totalPages
	}

	offset := (page - 1) * perPage

	return Pagination{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
		Offset:     offset,
		Limit:      perPage,
	}
}

// FromRequest reads "page" and "per_page" query parameters from the request
// and constructs a Pagination. Defaults to page=1 and per_page=20.
func FromRequest(r *http.Request, total int64) Pagination {
	page := 1
	perPage := 20

	if v := r.URL.Query().Get("page"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			page = parsed
		}
	}

	if v := r.URL.Query().Get("per_page"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			perPage = parsed
		}
	}

	return New(page, perPage, total)
}

// Pages returns a slice of page numbers for rendering pagination links.
// For 7 or fewer pages, all pages are returned. For more pages, a windowed
// approach is used: first page, last page, and 2 pages around the current
// page, with -1 representing gaps (ellipsis).
func (p Pagination) Pages() []int {
	if p.TotalPages <= 0 {
		return nil
	}

	if p.TotalPages <= 7 {
		pages := make([]int, p.TotalPages)
		for i := range pages {
			pages[i] = i + 1
		}
		return pages
	}

	// Windowed pagination for many pages
	seen := make(map[int]bool)
	var pages []int

	// Collect the page numbers we want to show (deduplicated)
	candidates := []int{1} // always show first
	for i := p.Page - 2; i <= p.Page+2; i++ {
		if i >= 1 && i <= p.TotalPages {
			candidates = append(candidates, i)
		}
	}
	candidates = append(candidates, p.TotalPages) // always show last

	for _, c := range candidates {
		if !seen[c] {
			seen[c] = true
			pages = append(pages, c)
		}
	}

	// Insert gaps (-1) where page numbers are not consecutive
	var result []int
	for i, pg := range pages {
		if i > 0 && pg-pages[i-1] > 1 {
			result = append(result, -1) // gap / ellipsis
		}
		result = append(result, pg)
	}

	return result
}
