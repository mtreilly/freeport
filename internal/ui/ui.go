package ui

import (
	"os"

	"github.com/muesli/termenv"
)

var (
	stdout  *termenv.Output
	stderr  *termenv.Output
	profile termenv.Profile
)

func Configure(noColor bool) {
	profile = termenv.EnvColorProfile()
	if noColor {
		profile = termenv.Ascii
	}
	stdout = termenv.NewOutput(os.Stdout, termenv.WithProfile(profile), termenv.WithColorCache(true))
	stderr = termenv.NewOutput(os.Stderr, termenv.WithProfile(profile), termenv.WithColorCache(true))
}

func Stdout() *termenv.Output {
	if stdout == nil {
		Configure(false)
	}
	return stdout
}

func Stderr() *termenv.Output {
	if stderr == nil {
		Configure(false)
	}
	return stderr
}

func Header(out *termenv.Output, text string) string {
	return style(out, text, "6", true)
}

func Brand(out *termenv.Output, text string) string {
	return style(out, text, "6", true)
}

func Success(out *termenv.Output, text string) string {
	return style(out, text, "2", true)
}

func Warning(out *termenv.Output, text string) string {
	return style(out, text, "3", true)
}

func Error(out *termenv.Output, text string) string {
	return style(out, text, "1", true)
}

func Info(out *termenv.Output, text string) string {
	return style(out, text, "4", true)
}

func Emphasis(out *termenv.Output, text string) string {
	return style(out, text, "6", true)
}

func Muted(out *termenv.Output, text string) string {
	s := out.Profile.String(text)
	return s.Faint().String()
}

func LabelOK(out *termenv.Output) string {
	return style(out, "OK", "2", true)
}

func LabelWarn(out *termenv.Output) string {
	return style(out, "WARN", "3", true)
}

func LabelErr(out *termenv.Output) string {
	return style(out, "ERR", "1", true)
}

func LabelInfo(out *termenv.Output) string {
	return style(out, "INFO", "4", true)
}

func style(out *termenv.Output, text, color string, bold bool) string {
	s := out.Profile.String(text)
	if color != "" {
		s = s.Foreground(out.Profile.Color(color))
	}
	if bold {
		s = s.Bold()
	}
	return s.String()
}
