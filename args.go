package main

import "strings"

type Args struct {
	Command string
	Name    string
	Profile string
	Flags   []string
	Line    string
}

func ParseArgs(line string) *Args {
	fields := strings.Fields(line)
	opts := Args{Line: line}
	if len(fields) == 0 {
		return &opts
	}
	opts.Command = fields[0]
	// Parse fields from user to detect special syntax
	aux := make([]string, 0)
	for _, f := range fields[1:] {
		if strings.HasPrefix(f, "@") {
			// profile
			opts.Profile = strings.Replace(f, "@", "", 1)
		} else if strings.HasPrefix(f, "+") {
			// flag
			opts.Flags = append(opts.Flags, f)
		} else {
			aux = append(aux, f)
		}
	}
	opts.Name = strings.Join(aux, " ")
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
