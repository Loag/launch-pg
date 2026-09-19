package tui

// cursor tracks the selected row of a list.
type cursor struct {
	pos int
}

// move handles navigation keys; it reports whether the key was used.
func (c *cursor) move(key string, n int) bool {
	switch key {
	case "up", "k":
		c.pos--
	case "down", "j":
		c.pos++
	case "home":
		c.pos = 0
	case "end":
		c.pos = n - 1
	case "pgup":
		c.pos -= 10
	case "pgdown":
		c.pos += 10
	default:
		return false
	}
	c.clamp(n)
	return true
}

func (c *cursor) clamp(n int) {
	if c.pos >= n {
		c.pos = n - 1
	}
	if c.pos < 0 {
		c.pos = 0
	}
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
