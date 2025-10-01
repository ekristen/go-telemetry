package zerolog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
)

const (
	colorBlack = iota + 30
	colorRed
	colorGreen
	colorYellow
	colorBlue
	colorMagenta
	colorCyan
	colorWhite

	colorBold     = 1
	colorDarkGray = 90
	colorOther    = 81
)

const (
	consoleDefaultTimeFormat = time.Kitchen
)

// colorize returns the string s wrapped in ANSI code c, unless disabled is true or c is 0.
func colorize(s interface{}, c int, disabled bool) string {
	e := os.Getenv("NO_COLOR")
	if e != "" || c == 0 {
		disabled = true
	}

	if disabled {
		return fmt.Sprintf("%s", s)
	}
	return fmt.Sprintf("\x1b[%dm%v\x1b[0m", c, s)
}

// consoleDefaultFormatCaller formats the caller field.
func consoleDefaultFormatCaller(noColor bool) zerolog.Formatter {
	return func(i interface{}) string {
		var c string
		if cc, ok := i.(string); ok {
			c = cc
		}
		if len(c) > 0 {
			if cwd, err := os.Getwd(); err == nil {
				if rel, err := filepath.Rel(cwd, c); err == nil {
					c = rel
				}
			}
			c = colorize(c, colorBold, noColor)
		}
		return fmt.Sprintf("%-42s", c) + colorize(" >", colorCyan, noColor)
	}
}

// consoleDefaultFormatTimestamp formats the timestamp field.
func consoleDefaultFormatTimestamp(timeFormat string, location *time.Location, noColor bool) zerolog.Formatter {
	if timeFormat == "" {
		timeFormat = consoleDefaultTimeFormat
	}
	if location == nil {
		location = time.Local
	}

	return func(i interface{}) string {
		t := "<nil>"
		switch tt := i.(type) {
		case string:
			ts, err := time.ParseInLocation(zerolog.TimeFieldFormat, tt, location)
			if err != nil {
				t = tt
			} else {
				t = ts.In(location).Format(timeFormat)
			}
		case json.Number:
			i, err := tt.Int64()
			if err != nil {
				t = tt.String()
			} else {
				var sec, nsec int64

				switch zerolog.TimeFieldFormat {
				case zerolog.TimeFormatUnixNano:
					sec, nsec = 0, i
				case zerolog.TimeFormatUnixMicro:
					sec, nsec = 0, int64(time.Duration(i)*time.Microsecond)
				case zerolog.TimeFormatUnixMs:
					sec, nsec = 0, int64(time.Duration(i)*time.Millisecond)
				default:
					sec, nsec = i, 0
				}

				ts := time.Unix(sec, nsec)
				t = ts.In(location).Format(timeFormat)
			}
		}
		return colorize(t, colorOther, noColor)
	}
}

// NewConsoleWriter creates a new console writer with sensible defaults.
func NewConsoleWriter(enableColor bool) zerolog.ConsoleWriter {
	return zerolog.ConsoleWriter{
		Out:             os.Stdout,
		TimeFormat:      time.RFC3339Nano,
		NoColor:         !enableColor,
		FormatLevel:     func(i interface{}) string { return fmt.Sprintf("| %-7s|", i) },
		FormatCaller:    consoleDefaultFormatCaller(!enableColor),
		FormatTimestamp: consoleDefaultFormatTimestamp(time.RFC3339Nano, nil, !enableColor),
	}
}
