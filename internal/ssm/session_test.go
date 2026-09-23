package ssm

import (
	"reflect"
	"testing"
)

func TestShellArgs(t *testing.T) {
	got := ShellArgs("egf", "eu-west-3", "i-123")
	want := []string{"ssm", "start-session", "--target", "i-123", "--profile", "egf", "--region", "eu-west-3"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ShellArgs = %v, want %v", got, want)
	}
}

func TestPortForwardArgs(t *testing.T) {
	got := PortForwardArgs("egf", "eu-west-3", "i-123", "5432", "15432")
	want := []string{
		"ssm", "start-session", "--target", "i-123", "--profile", "egf", "--region", "eu-west-3",
		"--document-name", "AWS-StartPortForwardingSession",
		"--parameters", "portNumber=5432,localPortNumber=15432",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("PortForwardArgs = %v, want %v", got, want)
	}
}

func TestPortForwardRemoteHostArgs(t *testing.T) {
	got := PortForwardRemoteHostArgs("egf", "eu-west-3", "i-123", "db.internal", "5432", "15432")
	want := []string{
		"ssm", "start-session", "--target", "i-123", "--profile", "egf", "--region", "eu-west-3",
		"--document-name", "AWS-StartPortForwardingSessionToRemoteHost",
		"--parameters", "host=db.internal,portNumber=5432,localPortNumber=15432",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("PortForwardRemoteHostArgs = %v, want %v", got, want)
	}
}
