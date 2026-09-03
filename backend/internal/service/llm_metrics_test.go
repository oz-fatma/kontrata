package service

import (
	"math"
	"testing"
	"time"
)

func TestMetricsSince_BaslangicOverridesHours(t *testing.T) {
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	start := now.Add(-2 * time.Minute)
	saat := int32(24)
	got := metricsSince(now, &saat, &start)
	if !got.Equal(start) {
		t.Fatalf("baslangic ezilmedi: %v", got)
	}
}

func TestMetricsSince_DefaultLast24h(t *testing.T) {
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	got := metricsSince(now, nil, nil)
	want := now.Add(-24 * time.Hour)
	if !got.Equal(want) {
		t.Fatalf("varsayılan = %v, %v beklenirdi", got, want)
	}
}

func TestP95_EmptyAndSingle(t *testing.T) {
	if got := p95(nil); got != 0 {
		t.Fatalf("boş = %v", got)
	}
	if got := p95([]int64{42}); got != 42 {
		t.Fatalf("n=1 = %v", got)
	}
}

func TestP95_SmallSampleIsMax(t *testing.T) {
	// n<20 iken ceil(0.95*n)=n, p95 son (en büyük) değerdir.
	cases := [][]int64{
		{10, 20},
		{9302, 9302, 9302, 15000},
		{1, 2, 3, 4, 5, 6, 7, 8, 9, 100},
		{9000, 9100, 9200, 9300, 9400, 20000},
	}
	for _, vals := range cases {
		got := p95(vals)
		max := vals[0]
		var sum int64
		for _, v := range vals {
			sum += v
			if v > max {
				max = v
			}
		}
		if got != float64(max) {
			t.Fatalf("n=%d p95=%v max=%d vals=%v", len(vals), got, max, vals)
		}
		avg := float64(sum) / float64(len(vals))
		if got < avg {
			t.Fatalf("n=%d p95=%v < ortalama=%v", len(vals), got, avg)
		}
	}
}

func TestP95_NUnder20AtLeastMean(t *testing.T) {
	for n := 1; n < 20; n++ {
		vals := make([]int64, n)
		var sum int64
		for i := 0; i < n; i++ {
			// Sağ kuyruk: son eleman ortalamayı yukarı çeker.
			vals[i] = int64(1000 + i)
			if i == n-1 {
				vals[i] = 50000
			}
			sum += vals[i]
		}
		got := p95(vals)
		avg := float64(sum) / float64(n)
		if got < avg {
			t.Fatalf("n=%d p95=%v < ortalama=%v (eski (n-1)*0.95 kırpımı düşük seçer)", n, got, avg)
		}
		if got != 50000 {
			t.Fatalf("n=%d p95=%v, max 50000 beklenirdi", n, got)
		}
		if p95Index(n) != n-1 {
			t.Fatalf("n=%d indis=%d, son eleman değil", n, p95Index(n))
		}
	}
}

func TestP95_N20NearestRank(t *testing.T) {
	vals := make([]int64, 20)
	for i := 0; i < 20; i++ {
		vals[i] = int64(i + 1)
	}
	// ceil(0.95*20)=19 → 19. değer
	if got := p95(vals); got != 19 {
		t.Fatalf("n=20 p95=%v, 19 beklenirdi", got)
	}
}

func TestP95Index_OldFormulaWouldUndershoot(t *testing.T) {
	// n=10: (n-1)*0.95 kırpımı 8 (9. değer); ceil 9 (10. değer).
	n := 10
	old := int(float64(n-1) * 0.95)
	got := p95Index(n)
	if old >= n-1 {
		t.Fatal("eski formül bu n'de zaten sondaydı")
	}
	if got != n-1 {
		t.Fatalf("indis=%d, %d beklenirdi", got, n-1)
	}
	if math.Ceil(0.95*float64(n)) != float64(n) {
		t.Fatalf("n=%d için ceil(0.95n) n olmalı", n)
	}
}
