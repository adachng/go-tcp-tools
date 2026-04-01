package cli

import (
	"testing"
)

func Test1(t *testing.T) {
	app := New()

	var str string
	_, err := app.AddOptString("-a, --address", &str, "IPv4 address")

	if err != nil {
		t.Error(err)
	}
}
