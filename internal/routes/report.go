package routes

import "github.com/catgoose/promolog"

// IssueReporter handles "Report Issue" actions from the error banner.
// Implementations decide what to do with the report — send an email, post to
// a Teams channel, write to a ticket system, etc.
// The consumer defines the interface; the implementation is plugged in at startup.
type IssueReporter interface {
	// Report is called when a user submits the Report Issue modal.
	// requestID identifies the failing request. description is user-provided
	// context about what they were doing. trace contains the full error trace
	// including the error chain, request metadata, and captured log entries
	// (may be nil if the request aged out of the store).
	Report(requestID string, description string, trace *promolog.Trace) error
}

// defaultReporter is a no-op implementation used when no IssueReporter is configured.
// It always succeeds — the endpoint still triggers the browser alert.
type defaultReporter struct{}

func (defaultReporter) Report(string, string, *promolog.Trace) error { return nil }
