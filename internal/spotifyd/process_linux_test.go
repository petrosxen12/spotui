package spotifyd

import (
	"reflect"
	"testing"
)

func TestParseCmdlineDropsEmptyEntries(t *testing.T) {
	args := parseCmdline([]byte("spotifyd\x00--no-daemon\x00\x00--config-path=/tmp/test.conf\x00"))
	want := []string{"spotifyd", "--no-daemon", "--config-path=/tmp/test.conf"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("parseCmdline() = %#v, want %#v", args, want)
	}
}

func TestHasExpectedBinaryArgMatchesBaseNameAndFullPath(t *testing.T) {
	if !hasExpectedBinaryArg([]string{"/usr/local/bin/spotifyd", "--no-daemon"}, "spotifyd", "/opt/bin/spotifyd") {
		t.Fatal("hasExpectedBinaryArg() = false, want true for basename match")
	}
	if !hasExpectedBinaryArg([]string{"/opt/bin/spotifyd", "--no-daemon"}, "spotifyd", "/opt/bin/spotifyd") {
		t.Fatal("hasExpectedBinaryArg() = false, want true for full path match")
	}
	if hasExpectedBinaryArg([]string{"/usr/bin/other"}, "spotifyd", "/opt/bin/spotifyd") {
		t.Fatal("hasExpectedBinaryArg() = true, want false")
	}
}
