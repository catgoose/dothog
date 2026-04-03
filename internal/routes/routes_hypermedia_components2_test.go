// setup:feature:demo
package routes

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func comp2Context(method, path string, body string) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

func comp2MultipartContext(t *testing.T, path, fieldName, fileName string, fileContent []byte) (echo.Context, *httptest.ResponseRecorder) {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile(fieldName, fileName)
	require.NoError(t, err)
	_, err = part.Write(fileContent)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, path, &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

// ─── Carousel ───────────────────────────────────────────────────────────────────

func TestHandleCarouselSlide_ValidIndex(t *testing.T) {
	s := newComponents2State()
	c, rec := comp2Context(http.MethodGet, "/hypermedia/components2/carousel/0", "")
	c.SetParamNames("index")
	c.SetParamValues("0")

	err := s.handleCarouselSlide(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Welcome")
}

func TestHandleCarouselSlide_LastSlide(t *testing.T) {
	s := newComponents2State()
	c, rec := comp2Context(http.MethodGet, "/hypermedia/components2/carousel/4", "")
	c.SetParamNames("index")
	c.SetParamValues("4")

	err := s.handleCarouselSlide(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Go + templ")
}

func TestHandleCarouselSlide_ClampsNegative(t *testing.T) {
	s := newComponents2State()
	c, rec := comp2Context(http.MethodGet, "/hypermedia/components2/carousel/-1", "")
	c.SetParamNames("index")
	c.SetParamValues("-1")

	err := s.handleCarouselSlide(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	// Clamped to first slide
	assert.Contains(t, rec.Body.String(), "Welcome")
}

func TestHandleCarouselSlide_ClampsOverflow(t *testing.T) {
	s := newComponents2State()
	c, rec := comp2Context(http.MethodGet, "/hypermedia/components2/carousel/99", "")
	c.SetParamNames("index")
	c.SetParamValues("99")

	err := s.handleCarouselSlide(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	// Clamped to last slide
	assert.Contains(t, rec.Body.String(), "Go + templ")
}

func TestHandleCarouselSlide_InvalidIndex(t *testing.T) {
	s := newComponents2State()
	c, _ := comp2Context(http.MethodGet, "/hypermedia/components2/carousel/abc", "")
	c.SetParamNames("index")
	c.SetParamValues("abc")

	err := s.handleCarouselSlide(c)
	require.Error(t, err)
}

// ─── Dropdown/Typeahead ─────────────────────────────────────────────────────────

func TestHandleDropdownSearch_EmptyQuery(t *testing.T) {
	s := newComponents2State()
	c, rec := comp2Context(http.MethodGet, "/hypermedia/components2/dropdown/search", "")

	err := s.handleDropdownSearch(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	// Should return all items
	body := rec.Body.String()
	assert.Contains(t, body, "Go")
	assert.Contains(t, body, "Python")
	assert.Contains(t, body, "Rust")
}

func TestHandleDropdownSearch_FilterMatch(t *testing.T) {
	s := newComponents2State()
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/hypermedia/components2/dropdown/search?q=rust", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := s.handleDropdownSearch(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "Rust")
	assert.NotContains(t, body, "Python")
}

func TestHandleDropdownSearch_CaseInsensitive(t *testing.T) {
	s := newComponents2State()
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/hypermedia/components2/dropdown/search?q=JAVA", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := s.handleDropdownSearch(c)
	require.NoError(t, err)
	body := rec.Body.String()
	assert.Contains(t, body, "Java")
	assert.Contains(t, body, "JavaScript")
}

func TestHandleDropdownSearch_NoMatch(t *testing.T) {
	s := newComponents2State()
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/hypermedia/components2/dropdown/search?q=zzzzz", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := s.handleDropdownSearch(c)
	require.NoError(t, err)
	assert.Contains(t, rec.Body.String(), "No results found")
}

// ─── Cascading Select ───────────────────────────────────────────────────────────

func TestHandleCascadingSelect_ValidCategory(t *testing.T) {
	s := newComponents2State()
	c, rec := comp2Context(http.MethodGet, "/hypermedia/components2/cascading/USA", "")
	c.SetParamNames("category")
	c.SetParamValues("USA")

	err := s.handleCascadingSelect(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "New York")
	assert.Contains(t, body, "Chicago")
}

func TestHandleCascadingSelect_UnknownCategory(t *testing.T) {
	s := newComponents2State()
	c, _ := comp2Context(http.MethodGet, "/hypermedia/components2/cascading/Narnia", "")
	c.SetParamNames("category")
	c.SetParamValues("Narnia")

	err := s.handleCascadingSelect(c)
	require.Error(t, err)
}

func TestHandleCascadingSelect_AllCategories(t *testing.T) {
	s := newComponents2State()
	for _, cat := range []string{"USA", "UK", "Japan", "Brazil"} {
		c, rec := comp2Context(http.MethodGet, "/hypermedia/components2/cascading/"+cat, "")
		c.SetParamNames("category")
		c.SetParamValues(cat)

		err := s.handleCascadingSelect(c)
		require.NoError(t, err, "category=%s", cat)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "<option")
	}
}

// ─── Range Slider ───────────────────────────────────────────────────────────────

func TestHandleRange_ValidValue(t *testing.T) {
	s := newComponents2State()
	c, rec := comp2Context(http.MethodPost, "/hypermedia/components2/range", "range=40")

	err := s.handleRange(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "40")
	assert.Contains(t, body, "$100.00")

	s.mu.RLock()
	assert.Equal(t, 40, s.rangeValue)
	s.mu.RUnlock()
}

func TestHandleRange_ZeroValue(t *testing.T) {
	s := newComponents2State()
	c, rec := comp2Context(http.MethodPost, "/hypermedia/components2/range", "range=0")

	err := s.handleRange(c)
	require.NoError(t, err)
	assert.Contains(t, rec.Body.String(), "$0.00")
}

func TestHandleRange_MaxValue(t *testing.T) {
	s := newComponents2State()
	c, rec := comp2Context(http.MethodPost, "/hypermedia/components2/range", "range=100")

	err := s.handleRange(c)
	require.NoError(t, err)
	assert.Contains(t, rec.Body.String(), "$250.00")
}

func TestHandleRange_InvalidDefaultsTo50(t *testing.T) {
	s := newComponents2State()
	c, rec := comp2Context(http.MethodPost, "/hypermedia/components2/range", "range=abc")

	err := s.handleRange(c)
	require.NoError(t, err)
	assert.Contains(t, rec.Body.String(), "$125.00")

	s.mu.RLock()
	assert.Equal(t, 50, s.rangeValue)
	s.mu.RUnlock()
}

func TestHandleRange_OutOfRangeDefaultsTo50(t *testing.T) {
	s := newComponents2State()
	c, _ := comp2Context(http.MethodPost, "/hypermedia/components2/range", "range=200")

	err := s.handleRange(c)
	require.NoError(t, err)

	s.mu.RLock()
	assert.Equal(t, 50, s.rangeValue)
	s.mu.RUnlock()
}

// ─── File Upload ────────────────────────────────────────────────────────────────

func TestHandleUpload_ValidFile(t *testing.T) {
	s := newComponents2State()
	content := []byte("hello world test content")
	c, rec := comp2MultipartContext(t, "/hypermedia/components2/upload", "file", "test.txt", content)

	err := s.handleUpload(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "test.txt")
	assert.Contains(t, body, " B") // size in bytes
}

func TestHandleUpload_NoFile(t *testing.T) {
	s := newComponents2State()
	c, _ := comp2Context(http.MethodPost, "/hypermedia/components2/upload", "")

	err := s.handleUpload(c)
	require.Error(t, err)
}

func TestHandleUpload_LargerFile(t *testing.T) {
	s := newComponents2State()
	content := make([]byte, 2048)
	for i := range content {
		content[i] = 'x'
	}
	c, rec := comp2MultipartContext(t, "/hypermedia/components2/upload", "file", "big.bin", content)

	err := s.handleUpload(c)
	require.NoError(t, err)
	assert.Contains(t, rec.Body.String(), "2.00 KB")
}

// ─── Accordion ──────────────────────────────────────────────────────────────────

func TestHandleAccordionPanel_Valid(t *testing.T) {
	s := newComponents2State()
	for i, expected := range []string{"HTMX", "DaisyUI", "templ"} {
		c, rec := comp2Context(http.MethodGet, fmt.Sprintf("/hypermedia/components2/accordion/%d", i), "")
		c.SetParamNames("panel")
		c.SetParamValues(fmt.Sprintf("%d", i))

		err := s.handleAccordionPanel(c)
		require.NoError(t, err, "panel=%d", i)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), expected)
	}
}

func TestHandleAccordionPanel_OutOfRange(t *testing.T) {
	s := newComponents2State()
	c, _ := comp2Context(http.MethodGet, "/hypermedia/components2/accordion/99", "")
	c.SetParamNames("panel")
	c.SetParamValues("99")

	err := s.handleAccordionPanel(c)
	require.Error(t, err)
}

func TestHandleAccordionPanel_InvalidParam(t *testing.T) {
	s := newComponents2State()
	c, _ := comp2Context(http.MethodGet, "/hypermedia/components2/accordion/abc", "")
	c.SetParamNames("panel")
	c.SetParamValues("abc")

	err := s.handleAccordionPanel(c)
	require.Error(t, err)
}

// ─── Indicator ──────────────────────────────────────────────────────────────────

func TestHandleIndicatorCount_Increments(t *testing.T) {
	s := newComponents2State()
	initial := s.notifCount
	c, rec := comp2Context(http.MethodGet, "/hypermedia/components2/indicator/count", "")

	err := s.handleIndicatorCount(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	s.mu.RLock()
	assert.GreaterOrEqual(t, s.notifCount, initial) // increases by 0-2
	assert.LessOrEqual(t, s.notifCount, initial+2)
	s.mu.RUnlock()
}

func TestHandleIndicatorReset(t *testing.T) {
	s := newComponents2State()
	s.notifCount = 10
	c, rec := comp2Context(http.MethodPost, "/hypermedia/components2/indicator/reset", "")

	err := s.handleIndicatorReset(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	s.mu.RLock()
	assert.Equal(t, 0, s.notifCount)
	s.mu.RUnlock()

	assert.Contains(t, rec.Body.String(), "0")
}

// ─── Theme ──────────────────────────────────────────────────────────────────────

func TestHandleTheme_SetDark(t *testing.T) {
	s := newComponents2State()
	c, rec := comp2Context(http.MethodPost, "/hypermedia/components2/theme", "theme=dark")

	err := s.handleTheme(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "dark")

	s.mu.RLock()
	assert.Equal(t, "dark", s.selectedTheme)
	s.mu.RUnlock()
}

func TestHandleTheme_EmptyDefaultsToDefault(t *testing.T) {
	s := newComponents2State()
	c, rec := comp2Context(http.MethodPost, "/hypermedia/components2/theme", "theme=")

	err := s.handleTheme(c)
	require.NoError(t, err)
	assert.Contains(t, rec.Body.String(), "default")

	s.mu.RLock()
	assert.Equal(t, "default", s.selectedTheme)
	s.mu.RUnlock()
}

func TestHandleTheme_CustomValue(t *testing.T) {
	s := newComponents2State()
	c, rec := comp2Context(http.MethodPost, "/hypermedia/components2/theme", "theme=cyberpunk")

	err := s.handleTheme(c)
	require.NoError(t, err)
	assert.Contains(t, rec.Body.String(), "cyberpunk")
}

// ─── State reset ────────────────────────────────────────────────────────────────

func TestNewComponents2State_Defaults(t *testing.T) {
	s := newComponents2State()
	assert.Len(t, s.carouselSlides, 5)
	assert.Len(t, s.searchItems, 20)
	assert.Len(t, s.categories, 4)
	assert.Equal(t, 50, s.rangeValue)
	assert.Len(t, s.accordionPanels, 3)
	assert.Equal(t, 3, s.notifCount)
	assert.Equal(t, "default", s.selectedTheme)
}

// ─── Helpers ────────────────────────────────────────────────────────────────────

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		expected string
		bytes    int64
	}{
		{bytes: 0, expected: "0 B"},
		{bytes: 512, expected: "512 B"},
		{bytes: 1024, expected: "1.00 KB"},
		{bytes: 1536, expected: "1.50 KB"},
		{bytes: 1048576, expected: "1.00 MB"},
		{bytes: 1073741824, expected: "1.00 GB"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, formatFileSize(tt.bytes), "bytes=%d", tt.bytes)
	}
}

// ─── Concurrency ────────────────────────────────────────────────────────────────

func TestConcurrentComponents2Access(t *testing.T) {
	s := newComponents2State()
	var wg sync.WaitGroup

	for range 20 {
		wg.Add(4)
		go func() {
			defer wg.Done()
			c, _ := comp2Context(http.MethodGet, "/hypermedia/components2/carousel/2", "")
			c.SetParamNames("index")
			c.SetParamValues("2")
			_ = s.handleCarouselSlide(c)
		}()
		go func() {
			defer wg.Done()
			c, _ := comp2Context(http.MethodGet, "/hypermedia/components2/indicator/count", "")
			_ = s.handleIndicatorCount(c)
		}()
		go func() {
			defer wg.Done()
			c, _ := comp2Context(http.MethodPost, "/hypermedia/components2/indicator/reset", "")
			_ = s.handleIndicatorReset(c)
		}()
		go func() {
			defer wg.Done()
			c, _ := comp2Context(http.MethodPost, "/hypermedia/components2/range", "range=75")
			_ = s.handleRange(c)
		}()
	}
	wg.Wait()
}
