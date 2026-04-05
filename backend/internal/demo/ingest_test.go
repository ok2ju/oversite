package demo

import (
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ok2ju/oversite/backend/internal/store"
)

func TestSyntheticTime(t *testing.T) {
	base := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		base     time.Time
		tick     int
		tickRate float64
		want     time.Time
	}{
		{
			name:     "tick 0 returns base unchanged",
			base:     base,
			tick:     0,
			tickRate: 64,
			want:     base,
		},
		{
			name:     "tick 64 at 64 tick/s = +1 second",
			base:     base,
			tick:     64,
			tickRate: 64,
			want:     base.Add(1 * time.Second),
		},
		{
			name:     "tick 128 at 128 tick/s = +1 second",
			base:     base,
			tick:     128,
			tickRate: 128,
			want:     base.Add(1 * time.Second),
		},
		{
			name:     "tick 256000 at 128 tick/s = +2000 seconds",
			base:     base,
			tick:     256000,
			tickRate: 128,
			want:     base.Add(2000 * time.Second),
		},
		{
			name:     "tickRate 0 defaults to 64",
			base:     base,
			tick:     64,
			tickRate: 0,
			want:     base.Add(1 * time.Second),
		},
		{
			name:     "tickRate negative defaults to 64",
			base:     base,
			tick:     64,
			tickRate: -10,
			want:     base.Add(1 * time.Second),
		},
		{
			name:     "fractional: tick 32 at 64 tick/s = +500ms",
			base:     base,
			tick:     32,
			tickRate: 64,
			want:     base.Add(500 * time.Millisecond),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := syntheticTime(tt.base, tt.tick, tt.tickRate)
			if !got.Equal(tt.want) {
				t.Errorf("syntheticTime(%v, %d, %f) = %v, want %v", tt.base, tt.tick, tt.tickRate, got, tt.want)
			}
		})
	}
}

func TestChunkTickParams(t *testing.T) {
	makeRows := func(n int) []store.InsertTickDataParams {
		rows := make([]store.InsertTickDataParams, n)
		for i := range rows {
			rows[i].Tick = int32(i)
		}
		return rows
	}

	tests := []struct {
		name       string
		rows       []store.InsertTickDataParams
		batchSize  int
		wantChunks int
		wantLast   int // length of last chunk
	}{
		{
			name:       "empty slice",
			rows:       nil,
			batchSize:  100,
			wantChunks: 0,
			wantLast:   0,
		},
		{
			name:       "fewer than batch",
			rows:       makeRows(50),
			batchSize:  100,
			wantChunks: 1,
			wantLast:   50,
		},
		{
			name:       "exact batch size",
			rows:       makeRows(100),
			batchSize:  100,
			wantChunks: 1,
			wantLast:   100,
		},
		{
			name:       "one over",
			rows:       makeRows(101),
			batchSize:  100,
			wantChunks: 2,
			wantLast:   1,
		},
		{
			name:       "exact multiple",
			rows:       makeRows(300),
			batchSize:  100,
			wantChunks: 3,
			wantLast:   100,
		},
		{
			name:       "batch size 1",
			rows:       makeRows(5),
			batchSize:  1,
			wantChunks: 5,
			wantLast:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := chunkTickParams(tt.rows, tt.batchSize)
			if len(chunks) != tt.wantChunks {
				t.Fatalf("got %d chunks, want %d", len(chunks), tt.wantChunks)
			}
			if tt.wantChunks > 0 {
				lastLen := len(chunks[len(chunks)-1])
				if lastLen != tt.wantLast {
					t.Errorf("last chunk len = %d, want %d", lastLen, tt.wantLast)
				}
			}

			// Verify total row count across all chunks
			total := 0
			for _, c := range chunks {
				total += len(c)
			}
			if total != len(tt.rows) {
				t.Errorf("total rows in chunks = %d, want %d", total, len(tt.rows))
			}
		})
	}
}

func TestConvertTicks(t *testing.T) {
	demoID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	base := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)

	t.Run("single tick all fields", func(t *testing.T) {
		ticks := []TickSnapshot{
			{
				Tick:    64,
				SteamID: "76561198012345",
				X:       1234.5678,
				Y:       -987.6543,
				Z:       128.256,
				Yaw:     45.5,
				Health:  100,
				Armor:   50,
				IsAlive: true,
				Weapon:  "ak47",
			},
		}

		got := convertTicks(ticks, demoID, base, 64)
		if len(got) != 1 {
			t.Fatalf("got %d rows, want 1", len(got))
		}

		r := got[0]
		if r.DemoID != demoID {
			t.Errorf("DemoID = %v, want %v", r.DemoID, demoID)
		}
		if r.Tick != 64 {
			t.Errorf("Tick = %d, want 64", r.Tick)
		}
		if r.SteamID != "76561198012345" {
			t.Errorf("SteamID = %q, want %q", r.SteamID, "76561198012345")
		}
		if r.X != float32(1234.5678) {
			t.Errorf("X = %f, want %f", r.X, float32(1234.5678))
		}
		if r.Y != float32(-987.6543) {
			t.Errorf("Y = %f, want %f", r.Y, float32(-987.6543))
		}
		if r.Z != float32(128.256) {
			t.Errorf("Z = %f, want %f", r.Z, float32(128.256))
		}
		if r.Yaw != float32(45.5) {
			t.Errorf("Yaw = %f, want %f", r.Yaw, float32(45.5))
		}
		if r.Health != 100 {
			t.Errorf("Health = %d, want 100", r.Health)
		}
		if r.Armor != 50 {
			t.Errorf("Armor = %d, want 50", r.Armor)
		}
		if !r.IsAlive {
			t.Error("IsAlive = false, want true")
		}
		if !r.Weapon.Valid || r.Weapon.String != "ak47" {
			t.Errorf("Weapon = %v, want {ak47 true}", r.Weapon)
		}
		wantTime := base.Add(1 * time.Second)
		if !r.Time.Equal(wantTime) {
			t.Errorf("Time = %v, want %v", r.Time, wantTime)
		}
	})

	t.Run("empty weapon -> NullString invalid", func(t *testing.T) {
		ticks := []TickSnapshot{
			{Tick: 0, SteamID: "123", Weapon: ""},
		}
		got := convertTicks(ticks, demoID, base, 64)
		if got[0].Weapon.Valid {
			t.Error("expected Weapon.Valid = false for empty weapon string")
		}
	})

	t.Run("non-empty weapon -> NullString valid", func(t *testing.T) {
		ticks := []TickSnapshot{
			{Tick: 0, SteamID: "123", Weapon: "awp"},
		}
		got := convertTicks(ticks, demoID, base, 64)
		if !got[0].Weapon.Valid || got[0].Weapon.String != "awp" {
			t.Errorf("Weapon = %v, want {awp true}", got[0].Weapon)
		}
	})

	t.Run("multiple ticks correct times and demoID", func(t *testing.T) {
		ticks := []TickSnapshot{
			{Tick: 0, SteamID: "A"},
			{Tick: 64, SteamID: "B"},
			{Tick: 128, SteamID: "C"},
		}
		got := convertTicks(ticks, demoID, base, 64)
		if len(got) != 3 {
			t.Fatalf("got %d rows, want 3", len(got))
		}
		for i, r := range got {
			if r.DemoID != demoID {
				t.Errorf("row[%d].DemoID = %v, want %v", i, r.DemoID, demoID)
			}
		}
		if !got[0].Time.Equal(base) {
			t.Errorf("row[0].Time = %v, want %v", got[0].Time, base)
		}
		if !got[1].Time.Equal(base.Add(1 * time.Second)) {
			t.Errorf("row[1].Time = %v, want %v", got[1].Time, base.Add(1*time.Second))
		}
		if !got[2].Time.Equal(base.Add(2 * time.Second)) {
			t.Errorf("row[2].Time = %v, want %v", got[2].Time, base.Add(2*time.Second))
		}
	})

	t.Run("empty input", func(t *testing.T) {
		got := convertTicks(nil, demoID, base, 64)
		if len(got) != 0 {
			t.Errorf("got %d rows for nil input, want 0", len(got))
		}
	})
}

// Ensure NullString is used in test (avoid unused import).
var _ = sql.NullString{}
