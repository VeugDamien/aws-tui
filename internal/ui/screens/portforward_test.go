package screens

import "testing"

func TestValidHost(t *testing.T) {
	valid := []string{
		"db.internal",
		"example.com",
		"my-host",
		"a.b.c.d.example.org",
		"host1",
		"10.0.0.5",
		"192.168.1.1",
		"::1",
		"fe80::1",
		"2001:db8::1",
		"example.com.", // trailing dot (FQDN) tolerated
	}
	for _, h := range valid {
		if !validHost(h) {
			t.Errorf("expected %q to be valid", h)
		}
	}

	invalid := []string{
		"",
		"host,other",             // comma would break the --parameters list
		"host=value",             // equals would break a key/value pair
		"host with space",        // whitespace
		"-leadinghyphen.example", // label starts with hyphen
		"trailinghyphen-.example",
		"under_score.example",  // underscore not allowed in hostnames
		"portNumber=80,host=x", // injection attempt
		"a..b",                 // empty label
	}
	for _, h := range invalid {
		if validHost(h) {
			t.Errorf("expected %q to be invalid", h)
		}
	}
}

func TestValidHostRejectsOverlongName(t *testing.T) {
	long := make([]byte, 254)
	for i := range long {
		long[i] = 'a'
	}
	if validHost(string(long)) {
		t.Error("expected an over-253-character host to be invalid")
	}
}

// setPFValues is a test helper to populate the private input fields.
func setPFValues(f *PortForwardForm, remotePort, localPort, host string) {
	f.inputs[fieldRemotePort].SetValue(remotePort)
	f.inputs[fieldLocalPort].SetValue(localPort)
	f.inputs[fieldHost].SetValue(host)
}

func TestValuesRejectsMaliciousHost(t *testing.T) {
	f := NewPortForwardForm("i-123", "web")
	f.RemoteHost = true
	setPFValues(&f, "5432", "15432", "evil,portNumber=22")

	if _, _, _, err := f.Values(); err == nil {
		t.Fatal("expected Values to reject a host containing a comma")
	}
}

func TestValuesAcceptsValidRemoteHost(t *testing.T) {
	f := NewPortForwardForm("i-123", "web")
	f.RemoteHost = true
	setPFValues(&f, "5432", "15432", "db.internal")

	host, remote, local, err := f.Values()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if host != "db.internal" || remote != "5432" || local != "15432" {
		t.Fatalf("unexpected values: host=%q remote=%q local=%q", host, remote, local)
	}
}

func TestValuesInstanceModeIgnoresHost(t *testing.T) {
	f := NewPortForwardForm("i-123", "web")
	f.RemoteHost = false
	// Even a bogus host must be ignored when not in remote-host mode.
	setPFValues(&f, "5432", "15432", "evil,host=x")

	host, _, _, err := f.Values()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if host != "" {
		t.Fatalf("expected empty host in instance mode, got %q", host)
	}
}
