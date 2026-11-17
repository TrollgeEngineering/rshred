package main

import (
	"fmt"
	"io"

	"charm.land/lipgloss/v2"
)

type logLevel uint8

const (
	logFatal logLevel = iota
	logError
	logWarning
	logInfo
	logDebug
	barrier
)

func (l logLevel) String() string {
	switch l {
	case logFatal:
		style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#800000"))
		return style.Render("fatal:")
	case logError:
		style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#f00000"))
		return style.Render("error:")
	case logWarning:
		style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#fff300"))
		return style.Render("warning:")
	case logInfo:
		style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ffffff"))
		return style.Render("info:")
	case logDebug:
		style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#727272"))
		return style.Render("debug:")
	default:
		return fmt.Sprintf("invalid logLevel: %d", l)
	}
}

func log(writer io.Writer, level logLevel, crntLevel logLevel, message string, a ...any) error {
	if level >= barrier {
		panic(fmt.Sprintf("invalid logLevel %d", level))
	}
	if crntLevel >= barrier {
		panic(fmt.Sprintf("invalid logLevel %d", crntLevel))
	}
	if level <= crntLevel {
		if _, err := lipgloss.Fprintf(writer, fmt.Sprintf("%s %s", level, message), a...); err != nil {
			return fmt.Errorf("failed to log message %q", message)
		}
	}
	return nil
}
