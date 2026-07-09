package cmd

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	stdtime "time"

	"github.com/leo/leo-cli/internal/config"
	"github.com/spf13/cobra"
)

var timeToZone string

var timeCmd = &cobra.Command{
	Use:   "time [VALUE]",
	Short: "Convert timestamps and common time strings",
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		return runTime(args, timeToZone, cfg, cmd.OutOrStdout(), stdtime.Now)
	},
}

func init() {
	timeCmd.Flags().StringVar(&timeToZone, "to", "+8", "Output timezone, for example +8, +9, or Asia/Tokyo")
	rootCmd.AddCommand(timeCmd)
}

func runTime(args []string, toZone string, cfg config.Config, stdout io.Writer, now func() stdtime.Time) error {
	loc, label, err := parseTimeZone(toZone)
	if err != nil {
		return err
	}
	parsed := now()
	if len(args) > 0 {
		value := strings.Join(args, " ")
		parsed, err = parseTimeValue(value, fixedZone(8))
		if err != nil {
			return err
		}
	}

	output := parsed.In(loc)
	if _, err := fmt.Fprintf(stdout, "时间: %s %s\n时间戳: %d\n毫秒: %d\n", output.Format("2006-01-02 15:04:05"), label, output.Unix(), output.UnixMilli()); err != nil {
		return err
	}
	return printConfiguredTimeZones(stdout, parsed, label, cfg.Time.Zones)
}

func printConfiguredTimeZones(stdout io.Writer, value stdtime.Time, primaryZone string, zones []string) error {
	wroteHeader := false
	for _, zone := range zones {
		loc, label, err := parseTimeZone(zone)
		if err != nil {
			return fmt.Errorf("invalid configured time zone %q: %w", zone, err)
		}
		converted := value.In(loc)
		if label == primaryZone {
			continue
		}
		if !wroteHeader {
			if _, err := fmt.Fprintln(stdout, "常用时区:"); err != nil {
				return err
			}
			wroteHeader = true
		}
		if _, err := fmt.Fprintf(stdout, "  %s: %s\n", label, converted.Format("2006-01-02 15:04:05")); err != nil {
			return err
		}
	}
	return nil
}

func parseTimeValue(value string, defaultLoc *stdtime.Location) (stdtime.Time, error) {
	cleaned := strings.Trim(strings.TrimSpace(value), "()[]{}\"'")
	if cleaned == "" {
		return stdtime.Time{}, fmt.Errorf("time value is required")
	}

	if isDigits(cleaned) {
		n, err := strconv.ParseInt(cleaned, 10, 64)
		if err != nil {
			return stdtime.Time{}, err
		}
		if len(cleaned) >= 13 {
			return stdtime.UnixMilli(n), nil
		}
		return stdtime.Unix(n, 0), nil
	}

	for _, layout := range []string{
		stdtime.RFC3339Nano,
		"2006-01-02 15:04:05 -07:00",
		"2006-01-02 15:04:05 -0700",
		"2006/01/02 15:04:05 -07:00",
		"2006/01/02 15:04:05 -0700",
	} {
		if parsed, err := stdtime.Parse(layout, cleaned); err == nil {
			return parsed, nil
		}
	}

	for _, layout := range []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
		"2006/01/02 15:04:05",
		"2006/01/02 15:04",
		"2006/01/02",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
	} {
		if parsed, err := stdtime.ParseInLocation(layout, cleaned, defaultLoc); err == nil {
			return parsed, nil
		}
	}

	return stdtime.Time{}, fmt.Errorf("unsupported time value %q", value)
}

func parseTimeZone(value string) (*stdtime.Location, string, error) {
	if loc, label, ok, err := parseZoneOffset(value); ok || err != nil {
		return loc, label, err
	}
	cleaned := strings.TrimSpace(value)
	loc, err := stdtime.LoadLocation(cleaned)
	if err != nil {
		return nil, "", err
	}
	return loc, cleaned, nil
}

func parseZoneOffset(value string) (*stdtime.Location, string, bool, error) {
	cleaned := strings.TrimSpace(value)
	if cleaned == "" {
		cleaned = "+8"
	}
	cleaned = strings.ToUpper(cleaned)
	cleaned = strings.TrimPrefix(strings.TrimPrefix(cleaned, "UTC"), "GMT")
	if cleaned == "" {
		cleaned = "+0"
	}
	if cleaned[0] != '+' && cleaned[0] != '-' {
		return nil, "", false, nil
	}

	sign := 1
	if cleaned[0] == '-' {
		sign = -1
	}
	parts := strings.Split(strings.TrimPrefix(strings.TrimPrefix(cleaned, "+"), "-"), ":")
	if len(parts) > 2 || parts[0] == "" {
		return nil, "", true, fmt.Errorf("timezone must be like +8, +09:00, or Asia/Tokyo, got %q", value)
	}
	hours, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, "", true, fmt.Errorf("timezone must be like +8, +09:00, or Asia/Tokyo, got %q", value)
	}
	minutes := 0
	if len(parts) == 2 {
		minutes, err = strconv.Atoi(parts[1])
		if err != nil {
			return nil, "", true, fmt.Errorf("timezone must be like +8, +09:00, or Asia/Tokyo, got %q", value)
		}
	}
	if hours > 14 || minutes >= 60 {
		return nil, "", true, fmt.Errorf("invalid timezone offset %q", value)
	}
	loc := fixedZoneMinutes(sign * (hours*60 + minutes))
	return loc, loc.String(), true, nil
}

func fixedZone(hours int) *stdtime.Location {
	return fixedZoneMinutes(hours * 60)
}

func fixedZoneMinutes(minutes int) *stdtime.Location {
	sign := "+"
	displayMinutes := minutes
	if displayMinutes < 0 {
		sign = "-"
		displayMinutes = -displayMinutes
	}
	hours := displayMinutes / 60
	remainder := displayMinutes % 60
	name := fmt.Sprintf("UTC%s%d", sign, hours)
	if remainder != 0 {
		name = fmt.Sprintf("UTC%s%d:%02d", sign, hours, remainder)
	}
	return stdtime.FixedZone(name, minutes*60)
}

func isDigits(value string) bool {
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return value != ""
}
