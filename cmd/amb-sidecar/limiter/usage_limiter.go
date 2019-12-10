package limiter

// UsageLimiter is a limiter that reports on current usage
type UsageLimiter interface {
	// GetUsageAtPointInTime retrieves the limiter's usage at this point in time.
	GetUsageAtPointInTime() (int, error)
}
