package awsclient

import "testing"

func TestProtoName(t *testing.T) {
	cases := map[string]string{
		"-1": "all", "": "all", "6": "tcp", "17": "udp", "1": "icmp", "47": "47",
	}
	for in, want := range cases {
		if got := protoName(in); got != want {
			t.Errorf("protoName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPortRange(t *testing.T) {
	i := func(v int32) *int32 { return &v }
	cases := []struct {
		from, to *int32
		proto    string
		want     string
	}{
		{i(22), i(22), "tcp", "22"},
		{i(80), i(443), "tcp", "80-443"},
		{nil, nil, "tcp", "all"},
		{i(0), i(0), "all", "all"},
		{i(-1), i(-1), "tcp", "all"},
	}
	for _, c := range cases {
		if got := portRange(c.from, c.to, c.proto); got != c.want {
			t.Errorf("portRange(%v,%v,%q) = %q, want %q", c.from, c.to, c.proto, got, c.want)
		}
	}
}

func TestComputeStats(t *testing.T) {
	s := MetricSeries{Values: []float64{10, 20, 30, 40}}
	s.computeStats()
	if s.Latest != 40 {
		t.Errorf("Latest = %v, want 40", s.Latest)
	}
	if s.Max != 40 {
		t.Errorf("Max = %v, want 40", s.Max)
	}
	if s.Avg != 25 {
		t.Errorf("Avg = %v, want 25", s.Avg)
	}

	empty := MetricSeries{}
	empty.computeStats() // must not panic
	if empty.Latest != 0 || empty.Max != 0 || empty.Avg != 0 {
		t.Errorf("empty series should have zero stats")
	}
}

func TestPeriodForWindow(t *testing.T) {
	cases := map[int]int32{
		60:   300,
		180:  300,
		720:  900,
		1440: 3600,
	}
	for win, want := range cases {
		if got := periodForWindow(win); got != want {
			t.Errorf("periodForWindow(%d) = %d, want %d", win, got, want)
		}
	}
}
