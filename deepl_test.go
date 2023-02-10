package main

import "testing"

func TestTranslate(t *testing.T) {
	loadConfig()
	text := "Guten Morgen."
	want := "Good morning."
	got, err := translate(text)
	if err != nil {
		t.Errorf("Got an error: %v", err)
		return
	}
	if got != want {
		t.Errorf("Got %v, wanted %v", got, want)
	}
}
