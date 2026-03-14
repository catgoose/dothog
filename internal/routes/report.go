package routes

import "catgoose/dothog/internal/requestlog"

// IssueReporter handles "Report Issue" actions from the error banner.
// Implementations decide what to do with the report — send an email, post to
// a Teams channel, write to a ticket system, etc.
// The consumer defines the interface; the implementation is plugged in at startup.
type IssueReporter interface {
	// Report is called when a user submits the Report Issue modal.
	// requestID identifies the failing request. description is user-provided
	// context about what they were doing. entries contains the captured log
	// trail for that request (may be empty if the request aged out of the
	// ring buffer). Implementations should not modify entries.
	Report(requestID string, description string, entries []requestlog.Entry) error
}

// defaultReporter is a no-op implementation used when no IssueReporter is configured.
// It always succeeds — the endpoint still triggers the browser alert.
type defaultReporter struct{}

func (defaultReporter) Report(string, string, []requestlog.Entry) error { return nil }
