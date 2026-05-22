// setup:feature:demo

package demo

import (
	"strings"
	"sync"
	"time"
	"unicode"
)

// SharedDocument holds the collaborative document state. All methods are
// thread-safe via an embedded RWMutex.
type SharedDocument struct {
	content   string
	revisions []DocRevision
	nextID    int
	mu        sync.RWMutex
}

// DocRevision is a single entry in the rolling revision log; WordDelta is signed against the prior word count.
type DocRevision struct {
	Timestamp time.Time
	Summary   string
	ID        int
	WordDelta int
}

// NewSharedDocument pre-populates with a sample paragraph and an initial revision entry.
func NewSharedDocument() *SharedDocument {
	initial := "The quick brown fox jumps over the lazy dog. " +
		"This shared document demonstrates how a single mutation signal " +
		"cascades through reactive hooks to update derived panels in real time."
	return &SharedDocument{
		content: initial,
		revisions: []DocRevision{
			{ID: 1, Timestamp: time.Now(), Summary: "initial content", WordDelta: wordCount(initial)},
		},
		nextID: 2,
	}
}

// Update replaces the document content and appends a revision entry.
func (d *SharedDocument) Update(content string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	oldWC := wordCount(d.content)
	d.content = content
	newWC := wordCount(content)

	d.revisions = append(d.revisions, DocRevision{
		ID:        d.nextID,
		Timestamp: time.Now(),
		Summary:   "edited by user",
		WordDelta: newWC - oldWC,
	})
	d.nextID++
}

// Content is safe for concurrent reads.
func (d *SharedDocument) Content() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.content
}

// Revisions returns up to 20 entries, newest first.
func (d *SharedDocument) Revisions() []DocRevision {
	d.mu.RLock()
	defer d.mu.RUnlock()

	n := len(d.revisions)
	start := 0
	if n > 20 {
		start = n - 20
	}
	out := make([]DocRevision, n-start)
	// Reverse so newest is first.
	for i, j := n-1, 0; i >= start; i, j = i-1, j+1 {
		out[j] = d.revisions[i]
	}
	return out
}

// WordCount returns the number of whitespace-separated tokens.
func (d *SharedDocument) WordCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return wordCount(d.content)
}

// CharCount returns the number of runes (characters) in the document.
func (d *SharedDocument) CharCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len([]rune(d.content))
}

// Sentiment performs simple keyword-based sentiment analysis.
// Returns "positive", "negative", or "neutral".
func (d *SharedDocument) Sentiment() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return sentiment(d.content)
}

// BatchEdit applies a named preset transformation to the document content,
// appends a revision, and returns the new content.
func (d *SharedDocument) BatchEdit(name string) string {
	d.mu.Lock()
	defer d.mu.Unlock()

	oldWC := wordCount(d.content)
	switch name {
	case "Fix Typos":
		d.content = fixTypos(d.content)
	case "Add Header":
		d.content = "# Document\n\n" + d.content
	case "Clean Whitespace":
		d.content = cleanWhitespace(d.content)
	}
	newWC := wordCount(d.content)

	d.revisions = append(d.revisions, DocRevision{
		ID:        d.nextID,
		Timestamp: time.Now(),
		Summary:   "batch edit: " + name,
		WordDelta: newWC - oldWC,
	})
	d.nextID++

	return d.content
}

// --- helpers ---

func wordCount(s string) int {
	return len(strings.Fields(s))
}

var (
	positiveWords = []string{"great", "awesome", "success", "happy", "good", "excellent"}
	negativeWords = []string{"error", "fail", "broken", "bad", "terrible", "crash"}
)

func sentiment(text string) string {
	lower := strings.ToLower(text)
	var pos, neg int
	for _, w := range positiveWords {
		pos += strings.Count(lower, w)
	}
	for _, w := range negativeWords {
		neg += strings.Count(lower, w)
	}
	switch {
	case pos > neg:
		return "positive"
	case neg > pos:
		return "negative"
	default:
		return "neutral"
	}
}

// fixTypos uppercases the first letter of each sentence.
func fixTypos(s string) string {
	runes := []rune(s)
	capNext := true
	for i, r := range runes {
		if capNext && unicode.IsLetter(r) {
			runes[i] = unicode.ToUpper(r)
			capNext = false
		}
		if r == '.' || r == '!' || r == '?' {
			capNext = true
		}
	}
	return string(runes)
}

// cleanWhitespace normalises runs of whitespace to single spaces and trims.
func cleanWhitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
