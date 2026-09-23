// Package awsconfig reads and classifies AWS CLI profiles from ~/.aws/config.
package awsconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/ini.v1"
)

// AuthKind is the authentication mechanism deduced from a profile.
type AuthKind int

const (
	AuthUnknown AuthKind = iota
	AuthSSO
	AuthLogin
	AuthCredentialProcess
	AuthStatic
)

// String returns a short human-readable label for the auth kind.
func (k AuthKind) String() string {
	switch k {
	case AuthSSO:
		return "sso"
	case AuthLogin:
		return "login"
	case AuthCredentialProcess:
		return "cred-process"
	case AuthStatic:
		return "static"
	default:
		return "unknown"
	}
}

// Profile is a single AWS profile with its deduced auth kind.
type Profile struct {
	Name       string
	AuthKind   AuthKind
	Region     string
	SSOSession string
	// SourceProfile is the profile this one derives from, resolved from either the
	// standard "source_profile" key or a profile name referenced inside the
	// "credential_process" command (e.g. "aws configure export-credentials
	// --profile <parent>", common in Terraform setups). Empty when standalone.
	SourceProfile string
	// CredentialProcess is the raw credential_process command, when present.
	CredentialProcess string
}

// ParentProfile returns the name of the profile this one depends on, if any.
func (p Profile) ParentProfile() string {
	return p.SourceProfile
}

// ConfigPath returns the path to the AWS config file, honouring AWS_CONFIG_FILE.
func ConfigPath() string {
	if p := os.Getenv("AWS_CONFIG_FILE"); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".aws", "config")
}

// LoadProfiles reads the AWS config file and returns the profiles sorted by name.
func LoadProfiles() ([]Profile, error) {
	path := ConfigPath()
	if path == "" {
		return nil, fmt.Errorf("unable to determine the path to ~/.aws/config")
	}
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("AWS configuration file not found (%s): %w", path, err)
	}

	cfg, err := ini.Load(path)
	if err != nil {
		return nil, fmt.Errorf("lecture de %s: %w", path, err)
	}

	var profiles []Profile
	for _, sec := range cfg.Sections() {
		name, ok := profileName(sec.Name())
		if !ok {
			continue
		}
		profiles = append(profiles, classify(name, sec))
	}

	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].Name < profiles[j].Name
	})
	return profiles, nil
}

// profileName extracts the profile name from an ini section name.
// Sections are named "default" or "profile <name>" in ~/.aws/config.
func profileName(section string) (string, bool) {
	if section == "default" {
		return "default", true
	}
	if strings.HasPrefix(section, "profile ") {
		return strings.TrimSpace(strings.TrimPrefix(section, "profile ")), true
	}
	return "", false
}

// classify deduces the AuthKind from the keys present in the section.
// Priority: sso_session > login_session > credential_process > aws_access_key_id.
func classify(name string, sec *ini.Section) Profile {
	p := Profile{
		Name:   name,
		Region: sec.Key("region").String(),
	}

	// A profile may depend on another via the standard source_profile key.
	if sec.HasKey("source_profile") {
		p.SourceProfile = strings.TrimSpace(sec.Key("source_profile").String())
	}

	switch {
	case sec.HasKey("sso_session") && sec.Key("sso_session").String() != "":
		p.AuthKind = AuthSSO
		p.SSOSession = sec.Key("sso_session").String()
	case sec.HasKey("login_session") && sec.Key("login_session").String() != "":
		p.AuthKind = AuthLogin
	case sec.HasKey("credential_process") && sec.Key("credential_process").String() != "":
		p.AuthKind = AuthCredentialProcess
		p.CredentialProcess = sec.Key("credential_process").String()
		// Terraform-style setups reference a parent profile inside the command,
		// e.g. "aws configure export-credentials --profile <parent>". Use it as
		// the source profile when no explicit source_profile key was given.
		if p.SourceProfile == "" {
			p.SourceProfile = parentFromCredentialProcess(p.CredentialProcess)
		}
	case sec.HasKey("aws_access_key_id") && sec.Key("aws_access_key_id").String() != "":
		p.AuthKind = AuthStatic
	default:
		p.AuthKind = AuthUnknown
	}
	return p
}

// parentFromCredentialProcess extracts a referenced profile name from a
// credential_process command line. It recognises the common conventions:
//   - "--profile <name>" / "--profile=<name>"
//   - "-p <name>" / "-p=<name>"
//   - "aws-vault exec <name> ..."
// It returns an empty string when no profile can be confidently determined.
func parentFromCredentialProcess(cmd string) string {
	fields := strings.Fields(cmd)
	for i, f := range fields {
		switch {
		case f == "--profile" || f == "-p":
			if i+1 < len(fields) {
				return unquote(fields[i+1])
			}
		case strings.HasPrefix(f, "--profile="):
			return unquote(strings.TrimPrefix(f, "--profile="))
		case strings.HasPrefix(f, "-p="):
			return unquote(strings.TrimPrefix(f, "-p="))
		}
	}
	// aws-vault exec <name>: the token following "exec".
	for i, f := range fields {
		if f == "exec" && strings.Contains(cmd, "aws-vault") && i+1 < len(fields) {
			cand := unquote(fields[i+1])
			if !strings.HasPrefix(cand, "-") {
				return cand
			}
		}
	}
	return ""
}

// unquote strips a single pair of surrounding single or double quotes.
func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}
