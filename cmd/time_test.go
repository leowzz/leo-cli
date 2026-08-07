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

func TestTimeCommandFromFlagDefaultsToUTCPlus8(t *testing.T) {
	flag := timeCmd.Flags().Lookup("from")
	if flag == nil {
		t.Fatal("time command is missing --from")
	}
	if got, want := flag.DefValue, "+8"; got != want {
		t.Fatalf("--from default = %q, want %q", got, want)
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

func TestRunTimeInterpretsNaiveInputInSourceZone(t *testing.T) {
	var stdout bytes.Buffer
	err := runTime([]string{"2026-08-06 22:26:09"}, "-4", "+8", config.Config{}, &stdout, stdtime.Now)
	if err != nil {
		t.Fatalf("runTime() error = %v", err)
	}

	want := "时间: 2026-08-07 10:26:09 UTC+8\n时间戳: 1786069569\n毫秒: 1786069569000\n"
	if stdout.String() != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
}

func TestRunTimeAcceptsIANASourceZone(t *testing.T) {
	var stdout bytes.Buffer
	err := runTime([]string{"2026-08-06 22:26:09"}, "America/New_York", "+8", config.Config{}, &stdout, stdtime.Now)
	if err != nil {
		t.Fatalf("runTime() error = %v", err)
	}

	want := "时间: 2026-08-07 10:26:09 UTC+8\n时间戳: 1786069569\n毫秒: 1786069569000\n"
	if stdout.String() != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
}

func TestRunTimeExplicitOffsetOverridesSourceZone(t *testing.T) {
	var stdout bytes.Buffer
	err := runTime([]string{"2026-08-06 22:26:09 +02:00"}, "-4", "+8", config.Config{}, &stdout, stdtime.Now)
	if err != nil {
		t.Fatalf("runTime() error = %v", err)
	}

	want := "时间: 2026-08-07 04:26:09 UTC+8\n时间戳: 1786047969\n毫秒: 1786047969000\n"
	if stdout.String() != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
}

func TestRunTimeRejectsInvalidSourceZoneBeforeWritingOutput(t *testing.T) {
	var stdout bytes.Buffer
	err := runTime(nil, "invalid/source-zone", "+8", config.Config{}, &stdout, stdtime.Now)
	if err == nil {
		t.Fatal("runTime() error = nil, want invalid source timezone error")
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want no output", stdout.String())
	}
}

func TestRunTimeConvertsTimestampToTargetZone(t *testing.T) {
	var stdout bytes.Buffer
	err := runTime([]string{"1783512043"}, "-4", "+9", config.Config{}, &stdout, stdtime.Now)
	if err != nil {
		t.Fatalf("runTime() error = %v", err)
	}

	want := "时间: 2026-07-08 21:00:43 UTC+9\n时间戳: 1783512043\n毫秒: 1783512043000\n"
	if stdout.String() != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
}

func TestRunTimeConvertsTimestampToIANATargetZone(t *testing.T) {
	var stdout bytes.Buffer
	err := runTime([]string{"1783512043"}, "+8", "Asia/Tokyo", config.Config{}, &stdout, stdtime.Now)
	if err != nil {
		t.Fatalf("runTime() error = %v", err)
	}

	want := "时间: 2026-07-08 21:00:43 Asia/Tokyo\n时间戳: 1783512043\n毫秒: 1783512043000\n"
	if stdout.String() != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
}

func TestRunTimeUsesCurrentTimeWhenValueIsOmitted(t *testing.T) {
	var stdout bytes.Buffer
	now := stdtime.Date(2026, 7, 8, 20, 0, 43, 123000000, stdtime.UTC)

	err := runTime(nil, "+8", "+8", config.Config{}, &stdout, func() stdtime.Time { return now })
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

	err := runTime(nil, "+8", "+8", cfg, &stdout, func() stdtime.Time {
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

func TestRunTimePrintsConfiguredIANAZones(t *testing.T) {
	var stdout bytes.Buffer
	cfg := config.Config{Time: config.TimeConfig{Zones: []string{"Asia/Tokyo", "America/Los_Angeles", "+8"}}}

	err := runTime(nil, "+8", "+8", cfg, &stdout, func() stdtime.Time {
		return stdtime.Date(2026, 7, 8, 20, 0, 43, 0, stdtime.UTC)
	})
	if err != nil {
		t.Fatalf("runTime() error = %v", err)
	}

	want := "时间: 2026-07-09 04:00:43 UTC+8\n时间戳: 1783540843\n毫秒: 1783540843000\n常用时区:\n  Asia/Tokyo: 2026-07-09 05:00:43\n  America/Los_Angeles: 2026-07-08 13:00:43\n"
	if stdout.String() != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
}
