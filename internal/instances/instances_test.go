package instances

import "testing"

func TestTransition(t *testing.T) {
	ok := []struct {
		from State
		act  Action
		next State
	}{
		{StateRunning, ActionStop, StateStopping},
		{StateRunning, ActionReboot, StateRebooting},
		{StateStopped, ActionStart, StateStarting},
		{StateStopped, ActionResize, StateResizing},
	}
	for _, tc := range ok {
		next, err := Transition(tc.from, tc.act)
		if err != nil || next != tc.next {
			t.Errorf("Transition(%q,%q) = %q,%v; want %q,nil", tc.from, tc.act, next, err, tc.next)
		}
	}
	bad := []struct {
		from State
		act  Action
	}{
		{StateRunning, ActionStart},
		{StateRunning, ActionResize}, // resize only from stopped
		{StateStopped, ActionStop},
		{StateStopped, ActionReboot},
		{StatePending, ActionStart},
		{StateDeleting, ActionStop},
		{StateDeleted, ActionStart},
	}
	for _, tc := range bad {
		if _, err := Transition(tc.from, tc.act); err == nil {
			t.Errorf("Transition(%q,%q) succeeded, want error", tc.from, tc.act)
		}
	}
}

func TestCheckQuota(t *testing.T) {
	p := Plan{VCPU: 2, MemoryMB: 4096, DiskGB: 20}
	if err := CheckQuota(DefaultQuota, Usage{}, p, 1); err != nil {
		t.Fatalf("fresh user should fit: %v", err)
	}
	if err := CheckQuota(DefaultQuota, Usage{Instances: 2}, p, 0); err == nil {
		t.Fatal("expected instance-count rejection")
	}
	if err := CheckQuota(DefaultQuota, Usage{MappedPort: 5}, p, 1); err == nil {
		t.Fatal("expected mapped-port rejection")
	}
	if err := CheckCapacity(Capacity{FreeVCPU: 1}, p); err == nil {
		t.Fatal("expected capacity rejection")
	}
}
