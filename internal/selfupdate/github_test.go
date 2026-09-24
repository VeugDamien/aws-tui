package selfupdate

import "testing"

func TestHostAllowed(t *testing.T) {
	allowed := []string{
		"github.com",
		"api.github.com",
		"codeload.github.com",
		"objects.githubusercontent.com",
		"release-assets.githubusercontent.com",
		"a-bucket.s3.amazonaws.com",
		"s3.eu-west-3.amazonaws.com",
		"GitHub.com",     // case-insensitive
		"github.com:443", // port stripped
		"foo.githubusercontent.com",
	}
	for _, h := range allowed {
		if !hostAllowed(h) {
			t.Errorf("expected %q to be allowed", h)
		}
	}

	denied := []string{
		"",
		"evil.com",
		"github.com.evil.com", // suffix trick
		"notgithub.com",
		"githubusercontent.com.evil", // parent must be a real suffix
		"amazonaws.com.attacker.net",
	}
	for _, h := range denied {
		if hostAllowed(h) {
			t.Errorf("expected %q to be denied", h)
		}
	}
}
