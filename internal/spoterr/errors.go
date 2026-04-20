package spoterr

import (
	"errors"
	"fmt"
	"time"
)

type Kind string

const (
	KindUnknown         Kind = "unknown"
	KindNoActiveDevice  Kind = "no_active_device"
	KindAuthExpired     Kind = "auth_expired"
	KindPremiumRequired Kind = "premium_required"
	KindRateLimited     Kind = "rate_limited"
	KindNetworkFailure  Kind = "network_failure"
)

type Error struct {
	Kind       Kind
	Message    string
	RetryAfter time.Duration
	Err        error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return string(e.Kind)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func New(kind Kind, message string) error {
	return &Error{Kind: kind, Message: message}
}

func Wrap(kind Kind, message string, err error) error {
	return &Error{Kind: kind, Message: message, Err: err}
}

func Wrapf(kind Kind, err error, format string, args ...any) error {
	return &Error{
		Kind:    kind,
		Message: fmt.Sprintf(format, args...),
		Err:     err,
	}
}

func WithRetryAfter(err error, retryAfter time.Duration) error {
	var typed *Error
	if errors.As(err, &typed) {
		copy := *typed
		copy.RetryAfter = retryAfter
		return &copy
	}
	return &Error{
		Kind:       KindRateLimited,
		Message:    err.Error(),
		RetryAfter: retryAfter,
		Err:        err,
	}
}

func KindOf(err error) Kind {
	var typed *Error
	if errors.As(err, &typed) && typed.Kind != "" {
		return typed.Kind
	}
	return KindUnknown
}

func RetryAfter(err error) time.Duration {
	var typed *Error
	if errors.As(err, &typed) {
		return typed.RetryAfter
	}
	return 0
}

func BannerMessage(err error) string {
	switch KindOf(err) {
	case KindNoActiveDevice:
		return "No active device. Use `/local start` in the TUI or `spotui local use` from the CLI."
	case KindAuthExpired:
		return "Spotify login expired. Run `spotui login` again."
	case KindPremiumRequired:
		return "Spotify Premium is required for playback control."
	case KindRateLimited:
		return "Spotify is rate limiting requests. spotui is backing off."
	case KindNetworkFailure:
		return "Network request failed. spotui will retry with backoff."
	default:
		if err == nil {
			return ""
		}
		return err.Error()
	}
}
