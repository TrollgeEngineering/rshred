package main

import "testing"

func TestParseSize(t *testing.T) {
	info := &Flag{}
	if err := ParseSize("32B", info); err != nil || info.Value != 32 {
		t.Errorf("error: input was valid, but parser did not assign 32. assigned: %v returned error: %v", info.Value, err)
	}

	if err := ParseSize("32G", info); err != nil || info.Value != 32000000000 {
		t.Errorf("error: input was valid, but parser did not assign 32000000000. assigned: %v returned error: %v", info.Value, err)
	}

	if err := ParseSize("243443243", info); err != nil || info.Value != 243443243 {
		t.Errorf("error: input was valid, but parser did not assign 243443243. assigned: %v returned error: %v", info.Value, err)
	}

	if err := ParseSize("224h2h4j34h2j4k2", info); err == nil {
		t.Errorf("error: input was invalid, but parser succeeded. assigned: %v", info.Value)
	}

	if err := ParseSize("EWGRH$%;h$G:F#Q>", info); err == nil {
		t.Errorf("error: input was invalid, but parser succeeded. assigned: %v", info.Value)
	}

	if err := ParseSize("G45", info); err == nil {
		t.Errorf("error: input was invalid, but parser succeeded. assigned: %v", info.Value)
	}

	if err := ParseSize("-45", info); err == nil {
		t.Errorf("error: input was invalid, but parser succeeded. assigned: %v", info.Value)
	}
}
