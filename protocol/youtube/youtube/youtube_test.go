package youtube

import "testing"

func TestYoutube(t *testing.T) {
	y, err := New("wZNYDzNGB-Q")
	if err != nil {
		t.Fatal(err)
	}
	if y.Formats["140"] == "" {
		t.Error("expected format 140")
	}
}
