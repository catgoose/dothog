// setup:feature:demo

package routes

import (
	"fmt"
	"io"
	"math/rand/v2"
	"strconv"
	"strings"
	"sync"

	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/web/views"

	"github.com/labstack/echo/v4"
)

// carouselSlide holds data for a single carousel slide.
type carouselSlide struct {
	Index int
	Title string
	Desc  string
	Color string // DaisyUI color class
}

// accordionPanel holds data for a lazy-loaded accordion panel.
type accordionPanel struct {
	ID      int
	Title   string
	Content string
}

// components2State holds mutable demo state for /hypermedia/components2.
type components2State struct {
	mu             sync.RWMutex
	carouselSlides []carouselSlide
	searchItems    []string
	categories     map[string][]string
	rangeValue     int
	accordionPanels []accordionPanel
	notifCount     int
	selectedTheme  string
}

func newComponents2State() *components2State {
	return &components2State{
		carouselSlides: []carouselSlide{
			{Index: 0, Title: "Welcome", Desc: "Getting started with HTMX and DaisyUI", Color: "bg-primary text-primary-content"},
			{Index: 1, Title: "Components", Desc: "Reusable UI building blocks", Color: "bg-secondary text-secondary-content"},
			{Index: 2, Title: "Hypermedia", Desc: "Server-driven interactions", Color: "bg-accent text-accent-content"},
			{Index: 3, Title: "Patterns", Desc: "Common HTMX patterns demonstrated", Color: "bg-info text-info-content"},
			{Index: 4, Title: "Go + templ", Desc: "Type-safe HTML templating in Go", Color: "bg-success text-success-content"},
		},
		searchItems: []string{
			"Go", "Python", "JavaScript", "TypeScript", "Rust", "Ruby",
			"Java", "C#", "C++", "Swift", "Kotlin", "Scala", "Elixir",
			"Haskell", "Clojure", "Dart", "PHP", "Perl", "Lua", "Zig",
		},
		categories: map[string][]string{
			"USA":    {"New York", "Los Angeles", "Chicago", "Houston", "Phoenix"},
			"UK":     {"London", "Manchester", "Birmingham", "Leeds", "Glasgow"},
			"Japan":  {"Tokyo", "Osaka", "Kyoto", "Yokohama", "Nagoya"},
			"Brazil": {"São Paulo", "Rio de Janeiro", "Brasília", "Salvador", "Fortaleza"},
		},
		rangeValue:   50,
		accordionPanels: []accordionPanel{
			{ID: 0, Title: "What is HTMX?", Content: "HTMX gives you access to AJAX, CSS Transitions, WebSockets and Server Sent Events directly in HTML, using attributes. It allows you to build modern user interfaces with the simplicity and power of hypertext."},
			{ID: 1, Title: "What is DaisyUI?", Content: "DaisyUI is a component library for Tailwind CSS. It provides semantic class names for common UI components like buttons, cards, modals, and more — reducing the need for long utility class chains."},
			{ID: 2, Title: "What is templ?", Content: "templ is a Go HTML templating language that provides type-safe templates with Go expressions. It compiles to Go code, giving you compile-time checks and excellent performance."},
		},
		notifCount:    3,
		selectedTheme: "default",
	}
}

const components2Base = hypermediaBase + "/components2"

func (ar *appRoutes) initComponents2Routes() {
	s := newComponents2State()

	ar.e.GET(components2Base, s.handleComponents2Page)
	ar.e.GET(components2Base+"/carousel/:index", s.handleCarouselSlide)
	ar.e.GET(components2Base+"/dropdown/search", s.handleDropdownSearch)
	ar.e.GET(components2Base+"/cascading/:category", s.handleCascadingSelect)
	ar.e.POST(components2Base+"/range", s.handleRange)
	ar.e.POST(components2Base+"/upload", s.handleUpload)
	ar.e.GET(components2Base+"/accordion/:panel", s.handleAccordionPanel)
	ar.e.GET(components2Base+"/indicator/count", s.handleIndicatorCount)
	ar.e.POST(components2Base+"/indicator/reset", s.handleIndicatorReset)
	ar.e.POST(components2Base+"/theme", s.handleTheme)
}

// ─── Page handler ───────────────────────────────────────────────────────────────

func (s *components2State) handleComponents2Page(c echo.Context) error {
	s.mu.Lock()
	fresh := newComponents2State()
	s.carouselSlides = fresh.carouselSlides
	s.searchItems = fresh.searchItems
	s.categories = fresh.categories
	s.rangeValue = fresh.rangeValue
	s.accordionPanels = fresh.accordionPanels
	s.notifCount = fresh.notifCount
	s.selectedTheme = fresh.selectedTheme

	// Build category keys
	categoryKeys := make([]string, 0, len(s.categories))
	for k := range s.categories {
		categoryKeys = append(categoryKeys, k)
	}

	data := views.Components2PageData{
		Slide:         carouselSlideToView(s.carouselSlides[0]),
		TotalSlides:   len(s.carouselSlides),
		Categories:    categoryKeys,
		RangeValue:    s.rangeValue,
		RangePrice:    fmt.Sprintf("$%.2f", float64(s.rangeValue)*2.50),
		Panels:        accordionPanelsToView(s.accordionPanels),
		NotifCount:    s.notifCount,
		SelectedTheme: s.selectedTheme,
	}
	s.mu.Unlock()

	return handler.RenderBaseLayout(c, views.Components2Page(data))
}

// ─── Carousel handler ───────────────────────────────────────────────────────────

func (s *components2State) handleCarouselSlide(c echo.Context) error {
	idx, err := strconv.Atoi(c.Param("index"))
	if err != nil {
		return handler.HandleHypermediaError(c, 400, "Invalid index", err)
	}
	s.mu.RLock()
	total := len(s.carouselSlides)
	if idx < 0 {
		idx = 0
	}
	if idx >= total {
		idx = total - 1
	}
	slide := s.carouselSlides[idx]
	s.mu.RUnlock()

	return handler.RenderComponent(c, views.CarouselSlideFragment(views.CarouselSlideData{
		Index:      slide.Index,
		Title:      slide.Title,
		Desc:       slide.Desc,
		Color:      slide.Color,
		TotalSlides: total,
	}))
}

// ─── Dropdown/Typeahead handler ─────────────────────────────────────────────────

func (s *components2State) handleDropdownSearch(c echo.Context) error {
	q := strings.TrimSpace(strings.ToLower(c.QueryParam("q")))

	s.mu.RLock()
	var results []string
	for _, item := range s.searchItems {
		if q == "" || strings.Contains(strings.ToLower(item), q) {
			results = append(results, item)
		}
	}
	s.mu.RUnlock()

	return handler.RenderComponent(c, views.DropdownResultsFragment(results))
}

// ─── Cascading Select handler ───────────────────────────────────────────────────

func (s *components2State) handleCascadingSelect(c echo.Context) error {
	category := c.Param("category")

	s.mu.RLock()
	subItems, ok := s.categories[category]
	if !ok {
		s.mu.RUnlock()
		return handler.HandleHypermediaError(c, 400, "Unknown category", fmt.Errorf("category=%q", category))
	}
	items := make([]string, len(subItems))
	copy(items, subItems)
	s.mu.RUnlock()

	return handler.RenderComponent(c, views.CascadingOptionsFragment(items))
}

// ─── Range handler ──────────────────────────────────────────────────────────────

func (s *components2State) handleRange(c echo.Context) error {
	val, err := strconv.Atoi(c.FormValue("range"))
	if err != nil || val < 0 || val > 100 {
		val = 50
	}
	s.mu.Lock()
	s.rangeValue = val
	s.mu.Unlock()

	price := fmt.Sprintf("$%.2f", float64(val)*2.50)
	return handler.RenderComponent(c, views.RangeResultFragment(views.RangeResultData{
		Value: val,
		Price: price,
	}))
}

// ─── Upload handler ─────────────────────────────────────────────────────────────

func (s *components2State) handleUpload(c echo.Context) error {
	file, err := c.FormFile("file")
	if err != nil {
		return handler.HandleHypermediaError(c, 400, "No file uploaded", err)
	}

	// Open and immediately close to validate readability
	src, err := file.Open()
	if err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to read file", err)
	}
	defer func() { _ = src.Close() }()
	// Count actual bytes read
	n, err := io.Copy(io.Discard, src)
	if err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to read file", err)
	}

	return handler.RenderComponent(c, views.UploadResultFragment(views.UploadResultData{
		Name: file.Filename,
		Size: formatFileSize(n),
		Type: file.Header.Get("Content-Type"),
	}))
}

// ─── Accordion handler ──────────────────────────────────────────────────────────

func (s *components2State) handleAccordionPanel(c echo.Context) error {
	idx, err := strconv.Atoi(c.Param("panel"))
	if err != nil {
		return handler.HandleHypermediaError(c, 400, "Invalid panel", err)
	}
	s.mu.RLock()
	if idx < 0 || idx >= len(s.accordionPanels) {
		s.mu.RUnlock()
		return handler.HandleHypermediaError(c, 400, "Panel out of range", fmt.Errorf("panel=%d", idx))
	}
	panel := s.accordionPanels[idx]
	s.mu.RUnlock()

	return handler.RenderComponent(c, views.AccordionContentFragment(views.AccordionPanelData{
		ID:      panel.ID,
		Title:   panel.Title,
		Content: panel.Content,
	}))
}

// ─── Indicator handlers ─────────────────────────────────────────────────────────

func (s *components2State) handleIndicatorCount(c echo.Context) error {
	s.mu.Lock()
	s.notifCount += rand.IntN(3) // 0-2
	count := s.notifCount
	s.mu.Unlock()

	return handler.RenderComponent(c, views.IndicatorBadgeFragment(count))
}

func (s *components2State) handleIndicatorReset(c echo.Context) error {
	s.mu.Lock()
	s.notifCount = 0
	s.mu.Unlock()

	return handler.RenderComponent(c, views.IndicatorBadgeFragment(0))
}

// ─── Theme handler ──────────────────────────────────────────────────────────────

func (s *components2State) handleTheme(c echo.Context) error {
	theme := c.FormValue("theme")
	if theme == "" {
		theme = "default"
	}
	s.mu.Lock()
	s.selectedTheme = theme
	s.mu.Unlock()

	return handler.RenderComponent(c, views.ThemeConfirmFragment(theme))
}

// ─── helpers ────────────────────────────────────────────────────────────────────

func carouselSlideToView(s carouselSlide) views.CarouselSlideData {
	return views.CarouselSlideData{
		Index: s.Index,
		Title: s.Title,
		Desc:  s.Desc,
		Color: s.Color,
	}
}

func accordionPanelsToView(panels []accordionPanel) []views.AccordionPanelData {
	out := make([]views.AccordionPanelData, len(panels))
	for i, p := range panels {
		out[i] = views.AccordionPanelData{ID: p.ID, Title: p.Title, Content: p.Content}
	}
	return out
}

func formatFileSize(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
