// Package humanize formats values for display.
package humanize

import "fmt"

// Bytes formats a size like "12.3 MB"; nil (unknown) is "?".
func Bytes(b *int64) string {
	if b == nil {
		return "?"
	}
	if *b < 1024 {
		return fmt.Sprintf("%d B", *b)
	}
	size := float64(*b)
	for _, suffix := range []string{"kB", "MB", "GB", "TB"} {
		size /= 1024
		if size < 1024 || suffix == "TB" {
			return fmt.Sprintf("%.1f %s", size, suffix)
		}
	}
	return "?"
}
