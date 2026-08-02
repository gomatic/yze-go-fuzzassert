package errs

import "testing"

// spec carries the very shapes the analyzer bans elsewhere. Inside the sentinel
// mechanism itself they are correct: the rendered message IS the contract under
// test here, and no amount of errors.Is matching can verify what Error() says.
// No diagnostic is expected anywhere in this file.
type spec struct {
	wantErr bool
	errMsg  string
}

func TestMessageRendering(t *testing.T) {
	s := spec{wantErr: true, errMsg: "boom"}
	if got := Const("boom").Error(); got != s.errMsg && s.wantErr {
		t.Fatalf("Error() = %q, want %q", got, s.errMsg)
	}
}
