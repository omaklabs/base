package pagination

import (
	"net/http"
	"net/url"
	"reflect"
	"testing"
)

func TestNewDefaults(t *testing.T) {
	p := New(0, 0, 100)

	if p.Page != 1 {
		t.Errorf("Page = %d, want 1", p.Page)
	}
	if p.PerPage != 20 {
		t.Errorf("PerPage = %d, want 20", p.PerPage)
	}
}

func TestNewCalculation(t *testing.T) {
	p := New(2, 10, 55)

	if p.TotalPages != 6 {
		t.Errorf("TotalPages = %d, want 6", p.TotalPages)
	}
	if !p.HasPrev {
		t.Error("HasPrev should be true on page 2")
	}
	if !p.HasNext {
		t.Error("HasNext should be true on page 2 of 6")
	}
	if p.Offset != 10 {
		t.Errorf("Offset = %d, want 10", p.Offset)
	}
	if p.Limit != 10 {
		t.Errorf("Limit = %d, want 10", p.Limit)
	}
}

func TestNewSinglePage(t *testing.T) {
	p := New(1, 20, 5)

	if p.TotalPages != 1 {
		t.Errorf("TotalPages = %d, want 1", p.TotalPages)
	}
	if p.HasPrev {
		t.Error("HasPrev should be false on single page")
	}
	if p.HasNext {
		t.Error("HasNext should be false on single page")
	}
	if p.Offset != 0 {
		t.Errorf("Offset = %d, want 0", p.Offset)
	}
}

func TestNewLastPage(t *testing.T) {
	p := New(5, 10, 50)

	if p.TotalPages != 5 {
		t.Errorf("TotalPages = %d, want 5", p.TotalPages)
	}
	if !p.HasPrev {
		t.Error("HasPrev should be true on last page")
	}
	if p.HasNext {
		t.Error("HasNext should be false on last page")
	}
}

func TestFromRequest(t *testing.T) {
	u, _ := url.Parse("http://example.com/items?page=3&per_page=15")
	r := &http.Request{URL: u}

	p := FromRequest(r, 100)

	if p.Page != 3 {
		t.Errorf("Page = %d, want 3", p.Page)
	}
	if p.PerPage != 15 {
		t.Errorf("PerPage = %d, want 15", p.PerPage)
	}
	if p.TotalPages != 7 {
		t.Errorf("TotalPages = %d, want 7", p.TotalPages)
	}
}

func TestFromRequestDefaults(t *testing.T) {
	u, _ := url.Parse("http://example.com/items")
	r := &http.Request{URL: u}

	p := FromRequest(r, 50)

	if p.Page != 1 {
		t.Errorf("Page = %d, want 1", p.Page)
	}
	if p.PerPage != 20 {
		t.Errorf("PerPage = %d, want 20", p.PerPage)
	}
}

func TestPages(t *testing.T) {
	// Small number of pages: show all
	p := New(1, 10, 50)
	got := p.Pages()
	want := []int{1, 2, 3, 4, 5}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Pages() = %v, want %v", got, want)
	}

	// Exactly 7 pages: show all
	p = New(4, 10, 70)
	got = p.Pages()
	want = []int{1, 2, 3, 4, 5, 6, 7}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Pages() = %v, want %v", got, want)
	}
}

func TestPagesWindow(t *testing.T) {
	tests := []struct {
		name string
		page int
		want []int
	}{
		{
			name: "beginning",
			page: 1,
			want: []int{1, 2, 3, -1, 20},
		},
		{
			name: "near beginning",
			page: 3,
			want: []int{1, 2, 3, 4, 5, -1, 20},
		},
		{
			name: "middle",
			page: 10,
			want: []int{1, -1, 8, 9, 10, 11, 12, -1, 20},
		},
		{
			name: "near end",
			page: 18,
			want: []int{1, -1, 16, 17, 18, 19, 20},
		},
		{
			name: "end",
			page: 20,
			want: []int{1, -1, 18, 19, 20},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(tt.page, 5, 100)
			got := p.Pages()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Pages() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewZeroTotal(t *testing.T) {
	p := New(1, 20, 0)

	if p.TotalPages != 0 {
		t.Errorf("TotalPages = %d, want 0", p.TotalPages)
	}
	if p.HasPrev {
		t.Error("HasPrev should be false with zero total")
	}
	if p.HasNext {
		t.Error("HasNext should be false with zero total")
	}

	pages := p.Pages()
	if pages != nil {
		t.Errorf("Pages() = %v, want nil", pages)
	}
}
