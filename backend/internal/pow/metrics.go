package pow

import (
	"net/http"
	"sync/atomic"

	"todo-app-backend/internal/utils"
)

type PoWMetrics struct {
	ChallengesIssued      uint64
	ChallengesValidated   uint64
	ChallengesRejected    uint64
	ReplayAttemptsBlocked uint64
	ExpiredChallenges     uint64
	RateLimitHits         uint64
}

var metrics PoWMetrics

func RecordChallengeIssued() {
	atomic.AddUint64(&metrics.ChallengesIssued, 1)
}

func RecordValidation(success bool) {
	if success {
		atomic.AddUint64(&metrics.ChallengesValidated, 1)
	} else {
		atomic.AddUint64(&metrics.ChallengesRejected, 1)
	}
}

func RecordReplayAttempt() {
	atomic.AddUint64(&metrics.ReplayAttemptsBlocked, 1)
}

func RecordExpired() {
	atomic.AddUint64(&metrics.ExpiredChallenges, 1)
}

func RecordRateLimitHit() {
	atomic.AddUint64(&metrics.RateLimitHits, 1)
}

func GetMetrics() PoWMetrics {
	return PoWMetrics{
		ChallengesIssued:      atomic.LoadUint64(&metrics.ChallengesIssued),
		ChallengesValidated:   atomic.LoadUint64(&metrics.ChallengesValidated),
		ChallengesRejected:    atomic.LoadUint64(&metrics.ChallengesRejected),
		ReplayAttemptsBlocked: atomic.LoadUint64(&metrics.ReplayAttemptsBlocked),
		ExpiredChallenges:     atomic.LoadUint64(&metrics.ExpiredChallenges),
		RateLimitHits:         atomic.LoadUint64(&metrics.RateLimitHits),
	}
}

// GetMetricsHandler returns current PoW metrics
func GetMetricsHandler(w http.ResponseWriter, r *http.Request) {
	// In a real implementation, you might want to add admin authentication here
	// For now, we'll just return the metrics
	
	m := GetMetrics()
	utils.EncodeJSONResponse(w, m, http.StatusOK)
}