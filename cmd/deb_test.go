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
		// Names may contain underscores; version and arch never do, so the
		// trailing two fields are authoritative.
		{"wafcloud_trigger_0.0.1_amd64.deb", "wafcloud_trigger", "0.0.1", "amd64"},
		{"./a_b_c_1.0-2_all.deb", "a_b_c", "1.0-2", "all"},
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
	if packagePresent(keys, debInfo{Name: "pp", Version: "1.2.3"}) {
		t.Error("should not match a partial package name")
	}
	if packagePresent(keys, debInfo{Name: "app", Version: "1.2.3", Arch: "arm64"}) {
		t.Error("should not match a different architecture")
	}
	underscore := []string{"Pamd64 wafcloud_trigger 0.0.1 abc123"}
	if !packagePresent(underscore, parseDeb("wafcloud_trigger_0.0.1_amd64.deb")) {
		t.Error("should find a package whose name contains underscores")
	}
}
