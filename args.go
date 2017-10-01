package main

import (
	"regexp"
	"strings"
)

type Args struct {
	Command string
	Name    string
	Profile string
	Flags   []string
	Line    string
}

var (
	profileArgRe = regexp.MustCompile("\\[.*\\]")
	flagsRe      = regexp.MustCompile("\\+[a-z0-9]+")
)

func ParseArgs(line string) *Args {
	// Extract metadata first
	profile := profileArgRe.FindAllString(line, -1)
	flags := flagsRe.FindAllString(line, -1)
	// Clean up original args
	line = profileArgRe.ReplaceAllString(line, "")
	line = flagsRe.ReplaceAllString(line, "")

	opts := Args{Line: line}
	if len(profile) > 0 {
		opts.Profile = strings.Trim(profile[0], "[]")
	}
	if len(flags) > 0 {
		opts.Flags = flags
	}

	// Parse remaining as cmd, name
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return &opts
	}
	opts.Command = fields[0]
	opts.Name = strings.Join(fields[1:], " ")
	return &opts
}

func (o *Args) ContainsFlag(f string) bool {
	for i := range o.Flags {
		if o.Flags[i] == f {
			return true
		}
	}
	return false
}
