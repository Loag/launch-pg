package password

import (
	"bytes"
	"strings"
	"testing"
)

// Known-answer test: RFC 7677 section 3 uses user "user", password
// "pencil", salt W22ZaJ0SNY7soEsUEjb6gQ==, 4096 iterations.
func TestSCRAMVerifierRFC7677(t *testing.T) {
	salt := []byte{0x5b, 0x6d, 0x99, 0x68, 0x9d, 0x12, 0x35, 0x8e,
		0xec, 0xa0, 0x4b, 0x14, 0x12, 0x36, 0xfa, 0x81}

	got, err := scramVerifier("pencil", salt, 4096)
	if err != nil {
		t.Fatal(err)
	}
	// Expected value cross-checked with Python's hashlib/hmac.
	want := "SCRAM-SHA-256$4096:W22ZaJ0SNY7soEsUEjb6gQ==$WG5d8oPm3OtcPnkdi4Uo7BkeZkBFzpcXkuLmtbsT4qY=:wfPLwcE6nTWhTAmQ7tl2KeoiWGPlZqQxSrmfPwDl2dU="
	if got != want {
		t.Errorf("verifier:\n got: %s\nwant: %s", got, want)
	}
}

func TestIssuer(t *testing.T) {
	issuer := NewIssuer(Random{}, NewSCRAM(bytes.NewReader(make([]byte, 16))))
	s, err := issuer.Issue()
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Plain) < 20 {
		t.Errorf("password too short: %q", s.Plain)
	}
	if !strings.HasPrefix(s.Verifier, "SCRAM-SHA-256$4096:") {
		t.Errorf("verifier = %q", s.Verifier)
	}
	if strings.Contains(s.Verifier, s.Plain) {
		t.Error("verifier contains plaintext")
	}
}
