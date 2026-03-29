// setup:feature:demo

package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/web/views"

	"github.com/labstack/echo/v4"
)

// ─── HAL types ───────────────────────────────────────────────────────────────

// HALLink represents a single link in a HAL _links object.
type HALLink struct {
	Href       string `json:"href"`
	Title      string `json:"title,omitempty"`
	Templated  bool   `json:"templated,omitempty"`
	Type       string `json:"type,omitempty"`
	Deprecation string `json:"deprecation,omitempty"`
}

// HALResource is a generic HAL+JSON resource with _links and _embedded.
type HALResource struct {
	Links    map[string]any `json:"_links"`
	Embedded map[string]any `json:"_embedded,omitempty"`
	Props    map[string]any `json:"-"`
}

// MarshalJSON flattens Props into the top level alongside _links and _embedded.
func (r HALResource) MarshalJSON() ([]byte, error) {
	m := make(map[string]any)
	for k, v := range r.Props {
		m[k] = v
	}
	m["_links"] = r.Links
	if len(r.Embedded) > 0 {
		m["_embedded"] = r.Embedded
	}
	return json.Marshal(m)
}

// ─── demo data ───────────────────────────────────────────────────────────────

const halBase = hypermediaBase + "/hal"

func buildHALCatalog() HALResource {
	return HALResource{
		Links: map[string]any{
			"self":   HALLink{Href: halBase + "/api/catalog", Title: "Book Catalog"},
			"books":  HALLink{Href: halBase + "/api/books", Title: "All Books"},
			"search": HALLink{Href: halBase + "/api/books?q={query}", Title: "Search Books", Templated: true},
			"curies": []HALLink{{Href: halBase + "/docs/rels/{rel}", Title: "Catalog Relations", Templated: true}},
		},
		Props: map[string]any{
			"name":        "Hypermedia Bookshop",
			"description": "A HAL-powered catalog of books about hypermedia and the web.",
			"totalBooks":  4,
		},
	}
}

func buildHALBooks() HALResource {
	return HALResource{
		Links: map[string]any{
			"self": HALLink{Href: halBase + "/api/books", Title: "All Books"},
			"up":   HALLink{Href: halBase + "/api/catalog", Title: "Catalog"},
		},
		Embedded: map[string]any{
			"books": []HALResource{
				bookResource(1, "Hypermedia Systems", "Carson Gross, Adam Stepinski, Deniz Aksimsek", 2023,
					"A practical guide to building hypermedia-driven applications with htmx."),
				bookResource(2, "RESTful Web APIs", "Leonard Richardson, Mike Amundsen", 2013,
					"Covers hypermedia API design including HAL, Collection+JSON, and Siren."),
				bookResource(3, "REST in Practice", "Jim Webber, Savas Parastatidis, Ian Robinson", 2010,
					"Hypermedia and systems architecture using HTTP and REST."),
				bookResource(4, "Building Hypermedia APIs with HTML5 and Node", "Mike Amundsen", 2011,
					"Explores hypermedia types including HAL for building evolvable APIs."),
			},
		},
		Props: map[string]any{
			"count": 4,
		},
	}
}

func bookResource(id int, title, author string, year int, summary string) HALResource {
	return HALResource{
		Links: map[string]any{
			"self":   HALLink{Href: fmt.Sprintf("%s/api/books/%d", halBase, id), Title: title},
			"author": HALLink{Href: fmt.Sprintf("%s/api/authors/%d", halBase, id), Title: author},
			"up":     HALLink{Href: halBase + "/api/books", Title: "All Books"},
		},
		Props: map[string]any{
			"id":      id,
			"title":   title,
			"author":  author,
			"year":    year,
			"summary": summary,
		},
	}
}

func buildHALBook(id int) (HALResource, bool) {
	books := []struct {
		title, author, summary string
		year                   int
		tags                   []string
	}{
		{"Hypermedia Systems", "Carson Gross, Adam Stepinski, Deniz Aksimsek",
			"A practical guide to building hypermedia-driven applications with htmx. " +
				"Covers the philosophy of hypermedia, REST constraints, and hands-on patterns.", 2023,
			[]string{"htmx", "hypermedia", "web"}},
		{"RESTful Web APIs", "Leonard Richardson, Mike Amundsen",
			"Covers hypermedia API design including HAL, Collection+JSON, and Siren. " +
				"A comprehensive reference for designing APIs that leverage the web.", 2013,
			[]string{"REST", "HAL", "API design"}},
		{"REST in Practice", "Jim Webber, Savas Parastatidis, Ian Robinson",
			"Hypermedia and systems architecture using HTTP and REST. " +
				"Shows how to apply REST constraints to real distributed systems.", 2010,
			[]string{"REST", "HTTP", "architecture"}},
		{"Building Hypermedia APIs with HTML5 and Node", "Mike Amundsen",
			"Explores hypermedia types including HAL for building evolvable APIs. " +
				"Practical examples of multiple hypermedia formats.", 2011,
			[]string{"HAL", "Node.js", "hypermedia"}},
	}
	if id < 1 || id > len(books) {
		return HALResource{}, false
	}
	b := books[id-1]
	return HALResource{
		Links: map[string]any{
			"self":    HALLink{Href: fmt.Sprintf("%s/api/books/%d", halBase, id), Title: b.title},
			"author":  HALLink{Href: fmt.Sprintf("%s/api/authors/%d", halBase, id), Title: b.author},
			"up":      HALLink{Href: halBase + "/api/books", Title: "All Books"},
			"catalog": HALLink{Href: halBase + "/api/catalog", Title: "Catalog"},
		},
		Props: map[string]any{
			"id":      id,
			"title":   b.title,
			"author":  b.author,
			"year":    b.year,
			"summary": b.summary,
			"tags":    b.tags,
		},
	}, true
}

func buildHALAuthor(id int) (HALResource, bool) {
	authors := []struct {
		name, bio string
		bookIDs   []int
	}{
		{"Carson Gross, Adam Stepinski, Deniz Aksimsek",
			"The creators of htmx and authors of Hypermedia Systems. Advocates for returning " +
				"to the original architecture of the web.", []int{1}},
		{"Leonard Richardson, Mike Amundsen",
			"API design experts who have written extensively about REST, hypermedia, and the " +
				"semantic web.", []int{2}},
		{"Jim Webber, Savas Parastatidis, Ian Robinson",
			"Distributed systems practitioners who bridge the gap between REST theory and " +
				"real-world architecture.", []int{3}},
		{"Mike Amundsen",
			"A prolific author and speaker on hypermedia, API design, and the programmable web. " +
				"Creator of multiple hypermedia format specifications.", []int{4}},
	}
	if id < 1 || id > len(authors) {
		return HALResource{}, false
	}
	a := authors[id-1]
	bookLinks := make([]HALLink, len(a.bookIDs))
	for i, bid := range a.bookIDs {
		bookLinks[i] = HALLink{Href: fmt.Sprintf("%s/api/books/%d", halBase, bid)}
	}
	return HALResource{
		Links: map[string]any{
			"self":  HALLink{Href: fmt.Sprintf("%s/api/authors/%d", halBase, id), Title: a.name},
			"books": bookLinks,
			"up":    HALLink{Href: halBase + "/api/catalog", Title: "Catalog"},
		},
		Props: map[string]any{
			"id":   id,
			"name": a.name,
			"bio":  a.bio,
		},
	}, true
}

// ─── routes ──────────────────────────────────────────────────────────────────

func (ar *appRoutes) initHALRoutes() {
	// Page
	ar.e.GET(halBase, ar.handleHALPage)

	// HAL+JSON API endpoints
	ar.e.GET(halBase+"/api/catalog", handleHALCatalog)
	ar.e.GET(halBase+"/api/books", handleHALBookList)
	ar.e.GET(halBase+"/api/books/:id", handleHALBookDetail)
	ar.e.GET(halBase+"/api/authors/:id", handleHALAuthorDetail)

	// Fragment endpoint: renders a HAL resource as an HTML card (for HTMX)
	ar.e.GET(halBase+"/explore", handleHALExplore)
}

// ─── page handler ────────────────────────────────────────────────────────────

func (ar *appRoutes) handleHALPage(c echo.Context) error {
	catalog := buildHALCatalog()
	raw, _ := json.MarshalIndent(catalog, "", "  ")
	return handler.RenderBaseLayout(c, views.HALPage(string(raw), halResourceToView(catalog)))
}

// ─── API handlers (return application/hal+json) ──────────────────────────────

func handleHALCatalog(c echo.Context) error {
	return halJSON(c, http.StatusOK, buildHALCatalog())
}

func handleHALBookList(c echo.Context) error {
	res := buildHALBooks()
	q := strings.TrimSpace(c.QueryParam("q"))
	if q != "" {
		res = filterBooks(res, q)
	}
	return halJSON(c, http.StatusOK, res)
}

func handleHALBookDetail(c echo.Context) error {
	id, err := parseIntParam(c, "id")
	if err != nil {
		return halError(c, http.StatusBadRequest, "Invalid book ID")
	}
	book, ok := buildHALBook(id)
	if !ok {
		return halError(c, http.StatusNotFound, "Book not found")
	}
	return halJSON(c, http.StatusOK, book)
}

func handleHALAuthorDetail(c echo.Context) error {
	id, err := parseIntParam(c, "id")
	if err != nil {
		return halError(c, http.StatusBadRequest, "Invalid author ID")
	}
	author, ok := buildHALAuthor(id)
	if !ok {
		return halError(c, http.StatusNotFound, "Author not found")
	}
	return halJSON(c, http.StatusOK, author)
}

// ─── explore handler (HTMX fragment) ────────────────────────────────────────

func handleHALExplore(c echo.Context) error {
	url := c.QueryParam("url")
	if url == "" {
		url = halBase + "/api/catalog"
	}
	res, err := resolveHALResource(url)
	if err != nil {
		return handler.RenderComponent(c, views.HALErrorFragment(url, err.Error()))
	}
	raw, _ := json.MarshalIndent(res, "", "  ")
	return handler.RenderComponent(c, views.HALExploreFragment(string(raw), halResourceToView(res), url))
}

// resolveHALResource maps an internal URL path to a HAL resource.
func resolveHALResource(url string) (HALResource, error) {
	// Strip query params for routing
	path := url
	if idx := strings.Index(url, "?"); idx >= 0 {
		path = url[:idx]
	}

	switch {
	case path == halBase+"/api/catalog":
		return buildHALCatalog(), nil
	case path == halBase+"/api/books":
		res := buildHALBooks()
		if idx := strings.Index(url, "q="); idx >= 0 {
			q := url[idx+2:]
			if amp := strings.Index(q, "&"); amp >= 0 {
				q = q[:amp]
			}
			res = filterBooks(res, q)
		}
		return res, nil
	case strings.HasPrefix(path, halBase+"/api/books/"):
		id, err := parsePathID(path, halBase+"/api/books/")
		if err != nil {
			return HALResource{}, err
		}
		book, ok := buildHALBook(id)
		if !ok {
			return HALResource{}, fmt.Errorf("book %d not found", id)
		}
		return book, nil
	case strings.HasPrefix(path, halBase+"/api/authors/"):
		id, err := parsePathID(path, halBase+"/api/authors/")
		if err != nil {
			return HALResource{}, err
		}
		author, ok := buildHALAuthor(id)
		if !ok {
			return HALResource{}, fmt.Errorf("author %d not found", id)
		}
		return author, nil
	default:
		return HALResource{}, fmt.Errorf("unknown resource: %s", url)
	}
}

func filterBooks(res HALResource, q string) HALResource {
	q = strings.ToLower(q)
	embedded, ok := res.Embedded["books"].([]HALResource)
	if !ok {
		return res
	}
	var filtered []HALResource
	for _, b := range embedded {
		title, _ := b.Props["title"].(string)
		author, _ := b.Props["author"].(string)
		if strings.Contains(strings.ToLower(title), q) || strings.Contains(strings.ToLower(author), q) {
			filtered = append(filtered, b)
		}
	}
	res.Embedded["books"] = filtered
	if res.Props == nil {
		res.Props = make(map[string]any)
	}
	res.Props["count"] = len(filtered)
	return res
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func halJSON(c echo.Context, status int, r HALResource) error {
	c.Response().Header().Set("Content-Type", "application/hal+json")
	return c.JSON(status, r)
}

func halError(c echo.Context, status int, msg string) error {
	c.Response().Header().Set("Content-Type", "application/hal+json")
	return c.JSON(status, map[string]any{
		"_links":  map[string]any{"self": HALLink{Href: c.Request().URL.String()}},
		"error":   msg,
		"status":  status,
	})
}

func parseIntParam(c echo.Context, name string) (int, error) {
	var id int
	_, err := fmt.Sscanf(c.Param(name), "%d", &id)
	return id, err
}

func parsePathID(path, prefix string) (int, error) {
	suffix := strings.TrimPrefix(path, prefix)
	var id int
	_, err := fmt.Sscanf(suffix, "%d", &id)
	return id, err
}

// halResourceToView converts a HALResource into the template view model.
func halResourceToView(r HALResource) views.HALResourceView {
	v := views.HALResourceView{
		Props: make([]views.HALPropView, 0),
		Links: make([]views.HALLinkView, 0),
	}

	// Props
	for k, val := range r.Props {
		v.Props = append(v.Props, views.HALPropView{
			Key:   k,
			Value: fmt.Sprintf("%v", val),
		})
	}

	// Links
	for rel, link := range r.Links {
		switch l := link.(type) {
		case HALLink:
			v.Links = append(v.Links, views.HALLinkView{
				Rel:       rel,
				Href:      l.Href,
				Title:     l.Title,
				Templated: l.Templated,
			})
		case []HALLink:
			for _, ll := range l {
				v.Links = append(v.Links, views.HALLinkView{
					Rel:       rel,
					Href:      ll.Href,
					Title:     ll.Title,
					Templated: ll.Templated,
				})
			}
		}
	}

	// Embedded
	if embBooks, ok := r.Embedded["books"]; ok {
		if books, ok := embBooks.([]HALResource); ok {
			for _, b := range books {
				v.EmbeddedBooks = append(v.EmbeddedBooks, halResourceToView(b))
			}
		}
	}

	return v
}
