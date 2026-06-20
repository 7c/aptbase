package cmd

import "testing"

func TestParseDeb(t *testing.T) {
	cases := []struct {
		path                string
		name, version, arch string
	}{
		{"app_1.2.3_amd64.deb", "app", "1.2.3", "amd64"},
		{"/tmp/my-pkg_0.1.0-1_all.deb", "my-pkg", "0.1.0-1", "all"},
		{"weird.deb", "weird", "", ""},
		{"a_b.deb", "a", "b", ""},
	}
	for _, c := range cases {
		got := parseDeb(c.path)
		if got.Name != c.name || got.Version != c.version || got.Arch != c.arch {
			t.Errorf("parseDeb(%q) = %+v, want name=%q version=%q arch=%q",
				c.path, got, c.name, c.version, c.arch)
		}
	}
}

func TestPackagePresent(t *testing.T) {
	keys := []string{"Pamd64 app 1.2.3 abc123", "Pall other 0.1 def456"}
	if !packagePresent(keys, debInfo{Name: "app", Version: "1.2.3"}) {
		t.Error("should find app 1.2.3")
	}
	if packagePresent(keys, debInfo{Name: "app", Version: "9.9.9"}) {
		t.Error("should not find wrong version")
	}
	if !packagePresent(keys, debInfo{Name: "other"}) {
		t.Error("should find by name when version empty")
	}
}
