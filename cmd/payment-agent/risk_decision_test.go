package main

import "testing"

func TestLocalRisk(t *testing.T) {
	tests := []struct {
		name    string
		payment Payment
		want    string
	}{
		{"ordinary card", Payment{AmountCents: 4200, Country: "SE", DeviceAge: 90}, "approve"},
		{"large transfer", Payment{AmountCents: 100000, Country: "US", DeviceAge: 90}, "review"},
		{"new device", Payment{AmountCents: 4200, Country: "SE", DeviceAge: 2}, "review"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := localRisk(tt.payment).Action; got != tt.want {
				t.Fatalf("action = %q, want %q", got, tt.want)
			}
		})
	}
}
