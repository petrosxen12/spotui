package app

import "testing"

func TestBestDeviceMatchPrefersFuzzyFallback(t *testing.T) {
	devices := []Device{
		{Name: "Kitchen Speaker", ID: "kitchen"},
		{Name: "Desk Headphones", ID: "desk"},
	}

	match, err := bestDeviceMatch(devices, "kithn")
	if err != nil {
		t.Fatalf("bestDeviceMatch returned error: %v", err)
	}
	if match.ID != "kitchen" {
		t.Fatalf("expected kitchen device, got %q", match.ID)
	}
}

func TestBestDeviceMatchRejectsAmbiguousSubstring(t *testing.T) {
	devices := []Device{
		{Name: "Living Room", ID: "one"},
		{Name: "Living Room TV", ID: "two"},
	}

	_, err := bestDeviceMatch(devices, "living")
	if err == nil {
		t.Fatal("expected ambiguous match error")
	}
}

func TestFindDeviceByID(t *testing.T) {
	devices := []Device{{ID: "abc", Name: "Office"}}
	match, ok := findDeviceByID(devices, "abc")
	if !ok {
		t.Fatal("expected device to be found")
	}
	if match.Name != "Office" {
		t.Fatalf("unexpected device %+v", match)
	}
}
