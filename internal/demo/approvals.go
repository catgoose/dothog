// setup:feature:demo

package demo

import (
	"sync"
	"time"
)

// ApprovalStatuses lists every status an approval request can be in.
var ApprovalStatuses = []string{"pending", "approved", "rejected", "escalated"}

// ApprovalCategories lists the valid spend categories.
var ApprovalCategories = []string{"Travel", "Equipment", "Software", "Training", "Contractor", "Other"}

// ApprovalRequest represents a single approval workflow item.
type ApprovalRequest struct {
	ID          int
	Title       string
	Requester   string
	Amount      float64
	Category    string
	Status      string
	SubmittedAt time.Time
	ReviewedAt  *time.Time
	ReviewedBy  string
	Notes       string
}

// ApprovalQueue is a thread-safe in-memory store for approval requests.
type ApprovalQueue struct {
	mu       sync.RWMutex
	requests []ApprovalRequest
	nextID   int
}

// AllowedActions returns the valid actions for a given status based on the
// HATEOAS state machine.
func AllowedActions(status string) []string {
	switch status {
	case "pending":
		return []string{"approve", "reject", "escalate"}
	case "escalated":
		return []string{"approve", "reject"}
	case "rejected":
		return []string{"resubmit"}
	case "approved":
		return nil // terminal
	default:
		return nil
	}
}

// NewApprovalQueue creates an ApprovalQueue seeded with sample requests.
func NewApprovalQueue() *ApprovalQueue {
	now := time.Now()
	reviewed := now.Add(-2 * time.Hour)

	q := &ApprovalQueue{
		nextID: 9,
		requests: []ApprovalRequest{
			{
				ID:          1,
				Title:       "Conference travel to GopherCon",
				Requester:   "Alice Johnson",
				Amount:      2400.00,
				Category:    "Travel",
				Status:      "pending",
				SubmittedAt: now.Add(-48 * time.Hour),
				Notes:       "Includes flight and hotel for 3 nights",
			},
			{
				ID:          2,
				Title:       "New ergonomic keyboard",
				Requester:   "Bob Smith",
				Amount:      350.00,
				Category:    "Equipment",
				Status:      "approved",
				SubmittedAt: now.Add(-72 * time.Hour),
				ReviewedAt:  &reviewed,
				ReviewedBy:  "Carol Davis",
				Notes:       "Split keyboard recommended by physio",
			},
			{
				ID:          3,
				Title:       "JetBrains IDE license",
				Requester:   "Diana Prince",
				Amount:      649.00,
				Category:    "Software",
				Status:      "pending",
				SubmittedAt: now.Add(-24 * time.Hour),
				Notes:       "Annual subscription for GoLand",
			},
			{
				ID:          4,
				Title:       "AWS certification training",
				Requester:   "Evan Torres",
				Amount:      1200.00,
				Category:    "Training",
				Status:      "escalated",
				SubmittedAt: now.Add(-96 * time.Hour),
				Notes:       "Solutions Architect Professional prep course",
			},
			{
				ID:          5,
				Title:       "Freelance design contractor",
				Requester:   "Fiona Green",
				Amount:      5000.00,
				Category:    "Contractor",
				Status:      "rejected",
				SubmittedAt: now.Add(-120 * time.Hour),
				ReviewedAt:  &reviewed,
				ReviewedBy:  "George Harris",
				Notes:       "UI redesign project - budget exceeded",
			},
			{
				ID:          6,
				Title:       "Team lunch for sprint review",
				Requester:   "Hank Miller",
				Amount:      180.00,
				Category:    "Other",
				Status:      "approved",
				SubmittedAt: now.Add(-36 * time.Hour),
				ReviewedAt:  &reviewed,
				ReviewedBy:  "Carol Davis",
				Notes:       "12 people at nearby restaurant",
			},
			{
				ID:          7,
				Title:       "Standing desk converter",
				Requester:   "Ivy Chen",
				Amount:      475.00,
				Category:    "Equipment",
				Status:      "pending",
				SubmittedAt: now.Add(-12 * time.Hour),
				Notes:       "VariDesk Pro Plus 36",
			},
			{
				ID:          8,
				Title:       "Security audit contractor",
				Requester:   "Jack Wilson",
				Amount:      8500.00,
				Category:    "Contractor",
				Status:      "escalated",
				SubmittedAt: now.Add(-168 * time.Hour),
				Notes:       "Penetration testing and vulnerability assessment",
			},
			{
				ID:          9,
				Title:       "Datadog monitoring license",
				Requester:   "Karen Lee",
				Amount:      950.00,
				Category:    "Software",
				Status:      "pending",
				SubmittedAt: now.Add(-6 * time.Hour),
				Notes:       "APM and infrastructure monitoring",
			},
		},
	}
	return q
}

// AllRequests returns a copy of all approval requests.
func (q *ApprovalQueue) AllRequests() []ApprovalRequest {
	q.mu.RLock()
	defer q.mu.RUnlock()
	result := make([]ApprovalRequest, len(q.requests))
	copy(result, q.requests)
	return result
}

// GetRequest returns the request with the given ID, if it exists.
func (q *ApprovalQueue) GetRequest(id int) (ApprovalRequest, bool) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	for _, r := range q.requests {
		if r.ID == id {
			return r, true
		}
	}
	return ApprovalRequest{}, false
}

// SubmitRequest creates a new approval request in "pending" status.
func (q *ApprovalQueue) SubmitRequest(title, requester string, amount float64, category, notes string) ApprovalRequest {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.nextID++
	r := ApprovalRequest{
		ID:          q.nextID,
		Title:       title,
		Requester:   requester,
		Amount:      amount,
		Category:    category,
		Status:      "pending",
		SubmittedAt: time.Now(),
		Notes:       notes,
	}
	q.requests = append(q.requests, r)
	return r
}

// TransitionRequest applies a HATEOAS state-machine action to the request
// identified by id. It returns the updated request and true if the transition
// was valid, or a zero-value request and false otherwise.
func (q *ApprovalQueue) TransitionRequest(id int, action, reviewer string) (ApprovalRequest, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for i, r := range q.requests {
		if r.ID != id {
			continue
		}

		allowed := AllowedActions(r.Status)
		valid := false
		for _, a := range allowed {
			if a == action {
				valid = true
				break
			}
		}
		if !valid {
			return ApprovalRequest{}, false
		}

		now := time.Now()

		switch action {
		case "approve":
			q.requests[i].Status = "approved"
			q.requests[i].ReviewedAt = &now
			q.requests[i].ReviewedBy = reviewer
		case "reject":
			q.requests[i].Status = "rejected"
			q.requests[i].ReviewedAt = &now
			q.requests[i].ReviewedBy = reviewer
		case "escalate":
			q.requests[i].Status = "escalated"
		case "resubmit":
			q.requests[i].Status = "pending"
			q.requests[i].ReviewedAt = nil
			q.requests[i].ReviewedBy = ""
		}

		return q.requests[i], true
	}

	return ApprovalRequest{}, false
}
