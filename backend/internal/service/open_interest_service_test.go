package service

import (
	"errors"
	"testing"
	"time"

	"github.com/kudaompq/ai_trending/backend/internal/model"
)

type openInterestRepoStub struct {
	calls          int
	symbol, period string
	limit          int
	start, end     *int64
	samples        []model.OpenInterestSample
	err            error
}

func (r *openInterestRepoStub) GetOpenInterestHistory(symbol, period string, limit int, startTime, endTime *int64) ([]model.OpenInterestSample, error) {
	r.calls++
	r.symbol, r.period, r.limit, r.start, r.end = symbol, period, limit, startTime, endTime
	return r.samples, r.err
}

func TestOpenInterestServiceMapsOnlyBinancePeriods(t *testing.T) {
	supported := map[string]string{
		"5m": "5m", "15m": "15m", "30m": "30m", "1h": "1h", "2h": "2h",
		"4h": "4h", "6h": "6h", "12h": "12h", "1d": "1d",
	}
	for interval, wantPeriod := range supported {
		t.Run(interval, func(t *testing.T) {
			repo := &openInterestRepoStub{samples: []model.OpenInterestSample{{Timestamp: 10, Value: 20}}}
			got, err := NewOpenInterestServiceWithRepository(repo).GetOpenInterestData("ETHUSDT", interval, 20, nil, nil)
			if err != nil || got.Interval != interval || len(got.Data) != 1 || repo.period != wantPeriod {
				t.Fatalf("data=%+v period=%q err=%v, want period %q", got, repo.period, err, wantPeriod)
			}
		})
	}
}

func TestOpenInterestServiceLeavesUnsupportedOrExpiredPeriodsEmpty(t *testing.T) {
	for _, interval := range []string{"1m", "3m", "8h", "3d", "1w", "1M"} {
		t.Run(interval, func(t *testing.T) {
			repo := &openInterestRepoStub{}
			got, err := NewOpenInterestServiceWithRepository(repo).GetOpenInterestData("ETHUSDT", interval, 20, nil, nil)
			if err != nil || len(got.Data) != 0 || repo.calls != 0 {
				t.Fatalf("unsupported interval should be empty without upstream call: data=%+v calls=%d err=%v", got, repo.calls, err)
			}
		})
	}

	repo := &openInterestRepoStub{}
	service := NewOpenInterestServiceWithRepository(repo)
	service.now = func() time.Time { return time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC) }
	old := service.now().AddDate(0, -1, -1).UnixMilli()
	end := service.now().AddDate(0, -1, 0).Add(-time.Minute).UnixMilli()
	got, err := service.GetOpenInterestData("ETHUSDT", "5m", 20, &old, &end)
	if err != nil || len(got.Data) != 0 || repo.calls != 0 {
		t.Fatalf("expired range should return empty without upstream call: data=%+v calls=%d err=%v", got, repo.calls, err)
	}
}

func TestOpenInterestServiceClampsRetentionAndLimitAndPropagatesSourceErrors(t *testing.T) {
	repo := &openInterestRepoStub{}
	service := NewOpenInterestServiceWithRepository(repo)
	service.now = func() time.Time { return time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC) }
	old := service.now().AddDate(0, -2, 0).UnixMilli()
	end := service.now().UnixMilli()
	_, err := service.GetOpenInterestData("ETHUSDT", "5m", 900, &old, &end)
	if err != nil || repo.limit != 500 || repo.start == nil || *repo.start != service.now().Add(-30*24*time.Hour).UnixMilli() {
		t.Fatalf("retention/limit not normalized: repo=%+v err=%v", repo, err)
	}

	repo.err = errors.New("upstream unavailable")
	if _, err := service.GetOpenInterestData("ETHUSDT", "5m", 10, nil, nil); err == nil {
		t.Fatal("expected source failure to propagate")
	}
}
