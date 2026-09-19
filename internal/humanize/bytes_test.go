package humanize

import "testing"

func TestBytes(t *testing.T) {
	n := func(v int64) *int64 { return &v }
	cases := []struct {
		in   *int64
		want string
	}{
		{nil, "?"},
		{n(512), "512 B"},
		{n(1536), "1.5 kB"},
		{n(8 * 1024 * 1024), "8.0 MB"},
	}
	for _, c := range cases {
		if got := Bytes(c.in); got != c.want {
			t.Errorf("Bytes(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}
