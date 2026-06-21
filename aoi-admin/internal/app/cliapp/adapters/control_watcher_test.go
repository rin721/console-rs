package adapters

import "testing"

func TestControlRequestMatchingRequiresServicePIDAndCreateTime(t *testing.T) {
	self := ProcessInfo{PID: 10, ProcessStartTime: 20}
	valid := ControlRequest{Service: "server", Action: ControlActionStop, PID: 10, ProcessStartTime: 20}
	if !matchesCurrentProcess(valid, "server", self) {
		t.Fatal("expected matching control request")
	}

	cases := []ControlRequest{
		{Service: "db", Action: ControlActionStop, PID: 10, ProcessStartTime: 20},
		{Service: "server", Action: "restart", PID: 10, ProcessStartTime: 20},
		{Service: "server", Action: ControlActionStop, PID: 11, ProcessStartTime: 20},
		{Service: "server", Action: ControlActionStop, PID: 10, ProcessStartTime: 21},
	}
	for _, tc := range cases {
		if matchesCurrentProcess(tc, "server", self) {
			t.Fatalf("unexpected match for %#v", tc)
		}
	}
}
