package main

import (
	"regexp"
	"strings"
)

// Args is the parsed bot command arguments.
// Commands are parsed and each portion of the command
// is extracted into the appropriate fields.
//
// Given a command like this:
//
//	/stats tie fighter pilot [ronoaldo] +ships +nocache
//
// We can expect the resulting struct to contain
//
//	Command: "/stats"
//	Name:    "tie fighter pilot"
//	Profile: "ronoaldo"
//	Flags:   ["+ships", "+nocache"]
//	Line:    "/stats tie fighter pilot [ronoaldo] +shipts +nocache"
//
// The parsed struct can then be used by command implementations
// to provide rich iteractions via flags.
type Args struct {
	Command string
	Name    string
	Profile string
	Flags   []string
	Line    string
}

var (
	profileArgRe = regexp.MustCompile("\\[.*\\]")
	flagsRe      = regexp.MustCompile("\\+[a-zA-Z0-9]+")
	mentionRe    = regexp.MustCompile("\\<@!?-?[0-9]+\\>")
)

// ParseArgs parses the user command into a structured Args object.
func ParseArgs(line string) *Args {
	// Extract metadata first
	profile := profileArgRe.FindAllString(line, -1)
	flags := flagsRe.FindAllString(line, -1)
	// Clean up original args
	line = profileArgRe.ReplaceAllString(line, "")
	line = flagsRe.ReplaceAllString(line, "")
	line = mentionRe.ReplaceAllString(line, "")

	opts := Args{Line: line}
	if len(profile) > 0 {
		opts.Profile = strings.Trim(profile[0], "[]")
	}
	for _, f := range flags {
		opts.Flags = append(opts.Flags, strings.ToLower(f))
	}

	// Parse remaining as cmd, name
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return &opts
	}
	opts.Command = strings.ToLower(fields[0])
	opts.Name = strings.Join(fields[1:], " ")
	return &opts
}

// ContainsFlag returns true if any of the flags are present.
func (o *Args) ContainsFlag(flags ...string) bool {
	for _, f := range flags {
		for i := range o.Flags {
			if o.Flags[i] == f {
				return true
			}
		}
	}
	return false
}
