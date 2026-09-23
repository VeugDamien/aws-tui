package awsconfig

import (
	"os"
	"path/filepath"
	"testing"
)

const sampleConfig = `[default]
region = eu-west-3
output = json

[profile sso-mgmt]
sso_session = eGF
sso_account_id = 111111111111
sso_role_name = AWSAdministratorAccess
region = eu-west-3

[profile login-egf]
region = eu-west-3
login_session = arn:aws:sts::222222222222:assumed-role/Role/user

[profile credproc]
credential_process = /usr/local/bin/cred-helper
region = us-east-1

[profile staticp]
aws_access_key_id = AKIAEXAMPLE
aws_secret_access_key = secret
region = eu-central-1

[profile barep]
region = ap-south-1

[profile tf-egf]
credential_process = aws configure export-credentials --profile staticp

[profile assumed]
source_profile = staticp
role_arn = arn:aws:iam::333333333333:role/App

[sso-session eGF]
sso_start_url = https://example.awsapps.com/start
sso_region = eu-west-3
`

func writeTempConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	if err := os.WriteFile(path, []byte(sampleConfig), 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return path
}

func TestLoadProfilesClassification(t *testing.T) {
	path := writeTempConfig(t)
	t.Setenv("AWS_CONFIG_FILE", path)

	profiles, err := LoadProfiles()
	if err != nil {
		t.Fatalf("LoadProfiles: %v", err)
	}

	want := map[string]struct {
		kind    AuthKind
		region  string
		session string
		source  string
	}{
		"default":   {AuthUnknown, "eu-west-3", "", ""},
		"sso-mgmt":  {AuthSSO, "eu-west-3", "eGF", ""},
		"login-egf": {AuthLogin, "eu-west-3", "", ""},
		"credproc":  {AuthCredentialProcess, "us-east-1", "", ""},
		"staticp":   {AuthStatic, "eu-central-1", "", ""},
		"barep":     {AuthUnknown, "ap-south-1", "", ""},
		"tf-egf":    {AuthCredentialProcess, "", "", "staticp"},
		"assumed":   {AuthUnknown, "", "", "staticp"},
	}

	if len(profiles) != len(want) {
		t.Fatalf("got %d profiles, want %d: %+v", len(profiles), len(want), profiles)
	}

	for _, p := range profiles {
		w, ok := want[p.Name]
		if !ok {
			t.Errorf("unexpected profile %q", p.Name)
			continue
		}
		if p.AuthKind != w.kind {
			t.Errorf("profile %q: AuthKind = %v, want %v", p.Name, p.AuthKind, w.kind)
		}
		if p.Region != w.region {
			t.Errorf("profile %q: Region = %q, want %q", p.Name, p.Region, w.region)
		}
		if p.SSOSession != w.session {
			t.Errorf("profile %q: SSOSession = %q, want %q", p.Name, p.SSOSession, w.session)
		}
		if p.SourceProfile != w.source {
			t.Errorf("profile %q: SourceProfile = %q, want %q", p.Name, p.SourceProfile, w.source)
		}
	}
}

func TestLoadProfilesSorted(t *testing.T) {
	path := writeTempConfig(t)
	t.Setenv("AWS_CONFIG_FILE", path)

	profiles, err := LoadProfiles()
	if err != nil {
		t.Fatalf("LoadProfiles: %v", err)
	}
	for i := 1; i < len(profiles); i++ {
		if profiles[i-1].Name > profiles[i].Name {
			t.Errorf("profiles not sorted: %q before %q", profiles[i-1].Name, profiles[i].Name)
		}
	}
}

func TestLoadProfilesMissingFile(t *testing.T) {
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(t.TempDir(), "does-not-exist"))
	if _, err := LoadProfiles(); err == nil {
		t.Fatal("expected error for missing config file, got nil")
	}
}
