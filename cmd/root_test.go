package cmd

import (
	"slices"
	"testing"
)

func TestRedactArgs(t *testing.T) {
	cases := []struct {
		in   []string
		want []string
	}{
		{[]string{"--password", "secret", "ping"}, []string{"--password", "***", "ping"}},
		{[]string{"--password=secret", "ping"}, []string{"--password=***", "ping"}},
		{[]string{"--user", "deploy", "ping"}, []string{"--user", "deploy", "ping"}},
		{[]string{"deploy", "./a.deb"}, []string{"deploy", "./a.deb"}},
	}
	for _, c := range cases {
		if got := redactArgs(c.in); !slices.Equal(got, c.want) {
			t.Errorf("redactArgs(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}
