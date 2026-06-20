package cmd

import "errors"

var (
	errNoRepo = errors.New("no repo given and no default 'repos' configured (set --api/repos or pass a repo argument)")
	errNoDist = errors.New("no distribution given and no default 'distributions' configured (use -d or set 'distributions' in config)")
)
