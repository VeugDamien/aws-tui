package awsconfig

import "testing"

func TestParentFromCredentialProcess(t *testing.T) {
	cases := map[string]string{
		"aws configure export-credentials --profile egf": "egf",
		"aws configure export-credentials --profile=egf": "egf",
		"aws-vault exec prod --json":                     "prod",
		"some-helper -p staging":                         "staging",
		"some-helper -p=staging":                         "staging",
		`helper --profile 'quoted'`:                      "quoted",
		"/usr/local/bin/cred-helper":                     "",
		"aws sso-oidc do-thing":                          "",
	}
	for cmd, want := range cases {
		if got := parentFromCredentialProcess(cmd); got != want {
			t.Errorf("parentFromCredentialProcess(%q) = %q, want %q", cmd, got, want)
		}
	}
}

func TestGroupProfilesSSO(t *testing.T) {
	profiles := []Profile{
		{Name: "sso-b", AuthKind: AuthSSO, SSOSession: "eGF"},
		{Name: "sso-a", AuthKind: AuthSSO, SSOSession: "eGF"},
		{Name: "sso-x", AuthKind: AuthSSO, SSOSession: "Other"},
		{Name: "lonely", AuthKind: AuthStatic},
	}

	groups := GroupProfiles(profiles)

	// Expect: SSO "Other", SSO "eGF" (alphabetical), then standalone.
	if len(groups) != 3 {
		t.Fatalf("got %d groups, want 3: %+v", len(groups), groups)
	}
	if groups[0].Kind != GroupSSO || groups[0].Label != "Other" {
		t.Errorf("group 0 = %v %q, want SSO Other", groups[0].Kind, groups[0].Label)
	}
	if groups[1].Kind != GroupSSO || groups[1].Label != "eGF" {
		t.Errorf("group 1 = %v %q, want SSO eGF", groups[1].Kind, groups[1].Label)
	}
	// Members of eGF must be alphabetical.
	if got := names(groups[1].Profiles); got[0] != "sso-a" || got[1] != "sso-b" {
		t.Errorf("eGF members = %v, want [sso-a sso-b]", got)
	}
	if groups[2].Kind != GroupStandalone {
		t.Errorf("group 2 = %v, want standalone", groups[2].Kind)
	}
}

func TestGroupProfilesDependency(t *testing.T) {
	profiles := []Profile{
		{Name: "egf", AuthKind: AuthUnknown},
		{Name: "tf-egf", AuthKind: AuthCredentialProcess, SourceProfile: "egf"},
		{Name: "tf-egf2", AuthKind: AuthCredentialProcess, SourceProfile: "egf"},
	}

	groups := GroupProfiles(profiles)

	if len(groups) != 1 {
		t.Fatalf("got %d groups, want 1: %+v", len(groups), groups)
	}
	g := groups[0]
	if g.Kind != GroupDependency || g.Label != "egf" {
		t.Fatalf("group = %v %q, want dependency egf", g.Kind, g.Label)
	}
	if len(g.Profiles) != 3 {
		t.Fatalf("got %d members, want 3", len(g.Profiles))
	}
	// Parent first at depth 0, children indented at depth 1.
	if g.Profiles[0].Profile.Name != "egf" || g.Profiles[0].Depth != 0 {
		t.Errorf("member 0 = %q depth %d, want egf depth 0", g.Profiles[0].Profile.Name, g.Profiles[0].Depth)
	}
	for _, m := range g.Profiles[1:] {
		if m.Depth != 1 {
			t.Errorf("child %q depth = %d, want 1", m.Profile.Name, m.Depth)
		}
	}
}

func TestGroupProfilesNoDuplicates(t *testing.T) {
	// A profile referenced as a source but also carrying an sso_session must not
	// appear twice; SSO group takes it and its child attaches beneath.
	profiles := []Profile{
		{Name: "root", AuthKind: AuthSSO, SSOSession: "S"},
		{Name: "child", AuthKind: AuthCredentialProcess, SourceProfile: "root"},
	}

	groups := GroupProfiles(profiles)

	seen := map[string]int{}
	for _, g := range groups {
		for _, m := range g.Profiles {
			seen[m.Profile.Name]++
		}
	}
	for name, n := range seen {
		if n != 1 {
			t.Errorf("profile %q appears %d times, want 1", name, n)
		}
	}
	// child should be nested under the SSO group, at depth 1.
	if len(groups) != 1 || groups[0].Kind != GroupSSO {
		t.Fatalf("want single SSO group, got %+v", groups)
	}
	if len(groups[0].Profiles) != 2 || groups[0].Profiles[1].Depth != 1 {
		t.Errorf("child not nested under SSO parent: %+v", groups[0].Profiles)
	}
}

func names(gp []GroupedProfile) []string {
	out := make([]string, len(gp))
	for i, p := range gp {
		out[i] = p.Profile.Name
	}
	return out
}
