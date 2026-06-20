package cmd

import (
	"testing"

	"github.com/7c/aptbase/internal/client"
)

func pub(prefix, dist, kind, repo string) client.PublishedRepo {
	return client.PublishedRepo{
		Prefix:       prefix,
		Distribution: dist,
		SourceKind:   kind,
		Sources:      []client.PublishedSource{{Component: "main", Name: repo}},
	}
}

func TestFilterPublications(t *testing.T) {
	pubs := []client.PublishedRepo{
		pub("99835", "focal", "local", "99835"),
		pub("99835", "jammy", "local", "99835"),
		pub("99835", "noble", "local", "99835"),
		pub("ef03ab", "noble", "local", "ef03ab"),
		pub("snap", "noble", "snapshot", "99835"), // wrong kind
	}

	// All distributions for the repo (prefix auto-discovered, not from config).
	all := filterPublications(pubs, "99835", nil)
	if len(all) != 3 {
		t.Fatalf("got %d, want 3 local pubs for 99835", len(all))
	}
	for _, p := range all {
		if p.Prefix != "99835" {
			t.Errorf("prefix = %q, want 99835", p.Prefix)
		}
	}

	// Distribution filter narrows the set.
	only := filterPublications(pubs, "99835", []string{"noble"})
	if len(only) != 1 || only[0].Distribution != "noble" {
		t.Fatalf("dist filter: got %+v", only)
	}

	// Unpublished repo yields nothing.
	if got := filterPublications(pubs, "ghost", nil); len(got) != 0 {
		t.Errorf("ghost repo should have no publications, got %d", len(got))
	}

	// Snapshot-sourced publication is excluded even if it names the repo.
	for _, p := range all {
		if p.SourceKind != "local" {
			t.Errorf("non-local publication leaked: %+v", p)
		}
	}
}
