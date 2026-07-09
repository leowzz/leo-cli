package cmd

import (
	"bytes"
	"testing"
	stdtime "time"

	"github.com/leo/leo-cli/internal/config"
)

func TestTimeCommandUse(t *testing.T) {
	if got, want := timeCmd.Use, "time [VALUE]"; got != want {
		t.Fatalf("timeCmd.Use = %q, want %q", got, want)
	}
}

func TestParseTimeValueDefaultsNaiveInputToUTCPlus8(t *testing.T) {
	loc := fixedZone(8)
	got, err := parseTimeValue("(2026-07-08 20:00:43)", loc)
	if err != nil {
		t.Fatalf("parseTimeValue() error = %v", err)
	}

	want := stdtime.Date(2026, 7, 8, 20, 0, 43, 0, loc)
	if !got.Equal(want) || got.Location().String() != want.Location().String() {
		t.Fatalf("parseTimeValue() = %v, want %v", got, want)
	}
}

func TestParseTimeValueUnixMilliseconds(t *testing.T) {
	got, err := parseTimeValue("1783512043000", fixedZone(8))
	if err != nil {
		t.Fatalf("parseTimeValue() error = %v", err)
	}

	if want := stdtime.UnixMilli(1783512043000); !got.Equal(want) {
		t.Fatalf("parseTimeValue() = %v, want %v", got, want)
	}
}

func TestRunTimeConvertsTimestampToTargetZone(t *testing.T) {
	var stdout bytes.Buffer
	err := runTime([]string{"1783512043"}, "+9", config.Config{}, &stdout, stdtime.Now)
	if err != nil {
		t.Fatalf("runTime() error = %v", err)
	}

	want := "时间: 2026-07-08 21:00:43 UTC+9\n时间戳: 1783512043\n毫秒: 1783512043000\n"
	if stdout.String() != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
}

func TestRunTimeUsesCurrentTimeWhenValueIsOmitted(t *testing.T) {
	var stdout bytes.Buffer
	now := stdtime.Date(2026, 7, 8, 20, 0, 43, 123000000, stdtime.UTC)

	err := runTime(nil, "+8", config.Config{}, &stdout, func() stdtime.Time { return now })
	if err != nil {
		t.Fatalf("runTime() error = %v", err)
	}

	want := "时间: 2026-07-09 04:00:43 UTC+8\n时间戳: 1783540843\n毫秒: 1783540843123\n"
	if stdout.String() != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
}

func TestRunTimePrintsConfiguredCommonZones(t *testing.T) {
	var stdout bytes.Buffer
	cfg := config.Config{Time: config.TimeConfig{Zones: []string{"+9", "+0", "+8"}}}

	err := runTime(nil, "+8", cfg, &stdout, func() stdtime.Time {
		return stdtime.Date(2026, 7, 8, 20, 0, 43, 0, stdtime.UTC)
	})
	if err != nil {
		t.Fatalf("runTime() error = %v", err)
	}

	want := "时间: 2026-07-09 04:00:43 UTC+8\n时间戳: 1783540843\n毫秒: 1783540843000\n常用时区:\n  UTC+9: 2026-07-09 05:00:43\n  UTC+0: 2026-07-08 20:00:43\n"
	if stdout.String() != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
}
