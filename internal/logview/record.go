package logview

import (
	"strings"
	"time"
)

type Record struct {
	FileID    string     `json:"fileId"`
	FileName  string     `json:"fileName"`
	Offset    int64      `json:"offset"`
	Timestamp *time.Time `json:"timestamp,omitempty"`
	TimeText  string     `json:"timeText,omitempty"`
	Level     string     `json:"level,omitempty"`
	SearchID  string     `json:"searchId,omitempty"`
	UserID    string     `json:"userId,omitempty"`
	Source    string     `json:"source,omitempty"`
	Message   string     `json:"message"`
	Raw       string     `json:"raw"`
	Parsed    bool       `json:"parsed"`
	Truncated bool       `json:"truncated,omitempty"`
}

func ParseLine(fileID, fileName string, offset int64, line []byte) Record {
	raw := strings.ToValidUTF8(string(line), "\ufffd")
	record := Record{
		FileID:   fileID,
		FileName: fileName,
		Offset:   offset,
		Message:  raw,
		Raw:      raw,
	}

	parts := strings.SplitN(raw, "|", 5)
	if len(parts) != 5 {
		return record
	}
	sourceAndMessage := strings.SplitN(parts[4], " - ", 2)
	if len(sourceAndMessage) != 2 {
		return record
	}

	record.Parsed = true
	record.TimeText = strings.TrimSpace(parts[0])
	record.Level = strings.TrimSpace(parts[1])
	record.SearchID = strings.TrimSpace(parts[2])
	record.UserID = strings.TrimSpace(parts[3])
	record.Source = strings.TrimSpace(sourceAndMessage[0])
	record.Message = strings.TrimSpace(sourceAndMessage[1])
	if parsed, ok := parseRecordTime(record.TimeText); ok {
		record.Timestamp = &parsed
	}
	return record
}

func parseRecordTime(value string) (time.Time, bool) {
	for _, layout := range []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05.999999999Z07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05,999999999",
		"2006-01-02 15:04:05",
	} {
		parsed, err := time.ParseInLocation(layout, value, time.Local)
		if err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}
