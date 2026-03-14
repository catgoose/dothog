// setup:feature:demo

package routes

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/web/views"

	"github.com/labstack/echo/v4"
)

// chatMsg is a single chat message.
type chatMsg struct {
	ID     int
	Author string
	Text   string
	Time   string
}

// timelineEvt is a single timeline event.
type timelineEvt struct {
	ID    int
	Title string
	Desc  string
	Time  string
	Kind  string // "primary", "secondary", "accent"
}

// componentsState holds mutable demo state for /hypermedia/components.
type componentsState struct {
	mu           sync.RWMutex
	wizardStep   int
	chatMsgs     []chatMsg
	nextMsgID    int
	bookmarked   bool
	liked        bool
	timelineEvts []timelineEvt
	nextEvtID    int
	rating       int
}

func newComponentsState() *componentsState {
	s := &componentsState{
		wizardStep: 1,
		nextMsgID:  1,
		nextEvtID:  1,
		rating:     0,
	}
	s.chatMsgs = []chatMsg{
		{ID: s.nextMsgID, Author: "bot", Text: "Hello! How can I help you today?", Time: "just now"},
	}
	s.nextMsgID = 2

	kinds := []string{"primary", "secondary", "accent"}
	for i := 0; i < 5; i++ {
		s.timelineEvts = append(s.timelineEvts, timelineEvt{
			ID:    s.nextEvtID,
			Title: fmt.Sprintf("Event %d", s.nextEvtID),
			Desc:  fmt.Sprintf("Description for event %d", s.nextEvtID),
			Time:  fmt.Sprintf("%d min ago", (5-i)*10),
			Kind:  kinds[i%len(kinds)],
		})
		s.nextEvtID++
	}
	return s
}

const componentsBase = hypermediaBase + "/components"

func (ar *appRoutes) initComponentsRoutes() {
	s := newComponentsState()

	ar.e.GET(componentsBase, s.handleComponentsPage)
	ar.e.GET(componentsBase+"/steps/:step", s.handleStep)
	ar.e.GET(componentsBase+"/tabs/:tab", s.handleTab)
	ar.e.POST(componentsBase+"/toast", s.handleToast)
	ar.e.POST(componentsBase+"/chat", s.handleChatSend)
	ar.e.POST(componentsBase+"/swap/like", s.handleSwapLike)
	ar.e.POST(componentsBase+"/swap/bookmark", s.handleSwapBookmark)
	ar.e.GET(componentsBase+"/skeleton/content", s.handleSkeletonContent)
	ar.e.GET(componentsBase+"/timeline", s.handleTimelineMore)
	ar.e.GET(componentsBase+"/drawer/content", s.handleDrawerContent)
	ar.e.POST(componentsBase+"/rating", s.handleRating)
}

// ─── Page handler ───────────────────────────────────────────────────────────────

func (s *componentsState) handleComponentsPage(c echo.Context) error {
	s.mu.Lock()
	// Reset state on full page load (copy fields, not the mutex)
	fresh := newComponentsState()
	s.wizardStep = fresh.wizardStep
	s.chatMsgs = fresh.chatMsgs
	s.nextMsgID = fresh.nextMsgID
	s.bookmarked = fresh.bookmarked
	s.liked = fresh.liked
	s.timelineEvts = fresh.timelineEvts
	s.nextEvtID = fresh.nextEvtID
	s.rating = fresh.rating

	msgs := make([]chatMsg, len(s.chatMsgs))
	copy(msgs, s.chatMsgs)
	evts := make([]timelineEvt, len(s.timelineEvts))
	copy(evts, s.timelineEvts)
	step := s.wizardStep
	liked := s.liked
	bookmarked := s.bookmarked
	rating := s.rating
	s.mu.Unlock()

	return handler.RenderBaseLayout(c, views.ComponentsPage(views.ComponentsPageData{
		WizardStep:    step,
		ChatMessages:  chatMsgsToView(msgs),
		Liked:         liked,
		Bookmarked:    bookmarked,
		TimelineEvts:  timelineEvtsToView(evts),
		TimelineNext:  len(evts) + 1,
		TimelineMore:  true,
		Rating:        rating,
	}))
}

// ─── Steps handler ──────────────────────────────────────────────────────────────

func (s *componentsState) handleStep(c echo.Context) error {
	step, err := strconv.Atoi(c.Param("step"))
	if err != nil || step < 1 || step > 4 {
		return handler.HandleHypermediaError(c, 400, "Invalid step", fmt.Errorf("step=%q", c.Param("step")))
	}
	s.mu.Lock()
	s.wizardStep = step
	s.mu.Unlock()
	return handler.RenderComponent(c, views.StepFragment(step))
}

// ─── Tabs handler ───────────────────────────────────────────────────────────────

func (s *componentsState) handleTab(c echo.Context) error {
	tab := c.Param("tab")
	switch tab {
	case "overview", "details", "settings":
	default:
		return handler.HandleHypermediaError(c, 400, "Invalid tab", fmt.Errorf("tab=%q", tab))
	}
	return handler.RenderComponent(c, views.TabContentFragment(tab))
}

// ─── Toast handler ──────────────────────────────────────────────────────────────

func (s *componentsState) handleToast(c echo.Context) error {
	return handler.RenderComponent(c, views.ToastResultFragment("Action completed successfully!"))
}

// ─── Chat handler ───────────────────────────────────────────────────────────────

func (s *componentsState) handleChatSend(c echo.Context) error {
	text := c.FormValue("chat-msg")
	if text == "" {
		return c.NoContent(200)
	}

	s.mu.Lock()
	userMsg := chatMsg{ID: s.nextMsgID, Author: "user", Text: text, Time: "now"}
	s.nextMsgID++
	botMsg := chatMsg{ID: s.nextMsgID, Author: "bot", Text: fmt.Sprintf("You said: %q — interesting!", text), Time: "now"}
	s.nextMsgID++
	s.chatMsgs = append(s.chatMsgs, userMsg, botMsg)
	s.mu.Unlock()

	return handler.RenderComponent(c, views.ChatMessagesFragment([]views.ChatMessage{
		chatMsgToView(userMsg),
		chatMsgToView(botMsg),
	}))
}

// ─── Swap handlers ──────────────────────────────────────────────────────────────

func (s *componentsState) handleSwapLike(c echo.Context) error {
	s.mu.Lock()
	s.liked = !s.liked
	liked := s.liked
	s.mu.Unlock()
	return handler.RenderComponent(c, views.SwapLikeFragment(liked))
}

func (s *componentsState) handleSwapBookmark(c echo.Context) error {
	s.mu.Lock()
	s.bookmarked = !s.bookmarked
	bookmarked := s.bookmarked
	s.mu.Unlock()
	return handler.RenderComponent(c, views.SwapBookmarkFragment(bookmarked))
}

// ─── Skeleton handler ───────────────────────────────────────────────────────────

func (s *componentsState) handleSkeletonContent(c echo.Context) error {
	time.Sleep(1500 * time.Millisecond)
	return handler.RenderComponent(c, views.SkeletonContentFragment())
}

// ─── Timeline handler ───────────────────────────────────────────────────────────

const timelineBatchSize = 3
const timelineMaxEvents = 15

func (s *componentsState) handleTimelineMore(c echo.Context) error {
	after, _ := strconv.Atoi(c.QueryParam("after"))
	if after < 1 {
		after = 1
	}

	s.mu.Lock()
	kinds := []string{"primary", "secondary", "accent"}
	var batch []timelineEvt
	for i := 0; i < timelineBatchSize; i++ {
		id := after + i
		if id > timelineMaxEvents {
			break
		}
		evt := timelineEvt{
			ID:    id,
			Title: fmt.Sprintf("Event %d", id),
			Desc:  fmt.Sprintf("Description for event %d", id),
			Time:  fmt.Sprintf("%d min ago", id*5),
			Kind:  kinds[id%len(kinds)],
		}
		batch = append(batch, evt)
		// Store if not already present
		if id >= s.nextEvtID {
			s.timelineEvts = append(s.timelineEvts, evt)
			s.nextEvtID = id + 1
		}
	}
	s.mu.Unlock()

	nextAfter := after + len(batch)
	hasMore := nextAfter <= timelineMaxEvents

	return handler.RenderComponent(c, views.TimelineBatchFragment(
		timelineEvtsToView(batch),
		nextAfter,
		hasMore,
	))
}

// ─── Drawer handler ─────────────────────────────────────────────────────────────

func (s *componentsState) handleDrawerContent(c echo.Context) error {
	return handler.RenderComponent(c, views.DrawerContentFragment())
}

// ─── Rating handler ─────────────────────────────────────────────────────────────

func (s *componentsState) handleRating(c echo.Context) error {
	r, err := strconv.Atoi(c.FormValue("rating"))
	if err != nil || r < 0 || r > 5 {
		r = 0
	}
	s.mu.Lock()
	s.rating = r
	rating := s.rating
	s.mu.Unlock()
	return handler.RenderComponent(c, views.RatingFragment(rating))
}

// ─── helpers ────────────────────────────────────────────────────────────────────

func chatMsgToView(m chatMsg) views.ChatMessage {
	return views.ChatMessage{ID: m.ID, Author: m.Author, Text: m.Text, Time: m.Time}
}

func chatMsgsToView(msgs []chatMsg) []views.ChatMessage {
	out := make([]views.ChatMessage, len(msgs))
	for i, m := range msgs {
		out[i] = chatMsgToView(m)
	}
	return out
}

func timelineEvtToView(e timelineEvt) views.TimelineEvent {
	return views.TimelineEvent{ID: e.ID, Title: e.Title, Desc: e.Desc, Time: e.Time, Kind: e.Kind}
}

func timelineEvtsToView(evts []timelineEvt) []views.TimelineEvent {
	out := make([]views.TimelineEvent, len(evts))
	for i, e := range evts {
		out[i] = timelineEvtToView(e)
	}
	return out
}
