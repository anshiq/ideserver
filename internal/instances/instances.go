// Package instances holds the VPS domain: lifecycle state machine,
// quota/capacity admission checks and plan/image catalog types.
// It is dependency-free so the API, jobs and tests share one definition.
package instances

import "fmt"

// State is the recorded lifecycle state of an instance row.
type State string

const (
	StatePending      State = "pending"
	StateProvisioning State = "provisioning"
	StateStarting     State = "starting"
	StateRunning      State = "running"
	StateStopping     State = "stopping"
	StateStopped      State = "stopped"
	StateRebooting    State = "rebooting"
	StateResizing     State = "resizing"
	StateDeleting     State = "deleting"
	StateDeleted      State = "deleted"
	StateError        State = "error"
)

// Action is a user-requested lifecycle transition.
type Action string

const (
	ActionStart  Action = "start"
	ActionStop   Action = "stop"
	ActionReboot Action = "reboot"
	ActionResize Action = "resize"
)

// allowed maps (from-state, action) → to-state.
var allowed = map[State]map[Action]State{
	StateRunning: {
		ActionStop:   StateStopping,
		ActionReboot: StateRebooting,
	},
	StateStopped: {
		ActionStart:  StateStarting,
		ActionResize: StateResizing,
	},
}

// Transition validates a user action against the current state.
func Transition(from State, act Action) (State, error) {
	next, ok := allowed[from][act]
	if !ok {
		return "", fmt.Errorf("instances: action %q not allowed from state %q", act, from)
	}
	return next, nil
}

// WorkerStep is the next internal step a River worker should run.
func WorkerStep(from State) (State, bool) {
	switch from {
	case StatePending:
		return StateProvisioning, true
	case StateStarting:
		return StateRunning, true
	case StateStopping:
		return StateStopped, true
	case StateRebooting:
		return StateRunning, true
	case StateResizing:
		return StateStopped, true
	case StateDeleting:
		return StateDeleted, true
	default:
		return "", false
	}
}

// Plan is a sizing template; Images/Plans are seeded by migrations.
type Plan struct {
	Slug     string
	VCPU     int
	MemoryMB int
	DiskGB   int
	Active   bool
}

// Quota is the per-user cap enforced in the create transaction.
type Quota struct {
	MaxInstances   int
	MaxVCPU        int
	MaxMemoryMB    int
	MaxDiskGB      int
	MaxMappedPorts int
}

// DefaultQuota is assigned at signup.
var DefaultQuota = Quota{
	MaxInstances:   2,
	MaxVCPU:        2,
	MaxMemoryMB:    4096,
	MaxDiskGB:      40,
	MaxMappedPorts: 5,
}

// Usage is the user's currently consumed resources.
type Usage struct {
	Instances  int
	VCPU       int
	MemoryMB   int
	DiskGB     int
	MappedPort int
}

// CheckQuota verifies a new instance of plan fits the quota.
func CheckQuota(q Quota, u Usage, p Plan, extraPorts int) error {
	if u.Instances+1 > q.MaxInstances {
		return fmt.Errorf("instances: quota exceeded: max %d instances", q.MaxInstances)
	}
	if u.VCPU+p.VCPU > q.MaxVCPU {
		return fmt.Errorf("instances: quota exceeded: max %d vCPU", q.MaxVCPU)
	}
	if u.MemoryMB+p.MemoryMB > q.MaxMemoryMB {
		return fmt.Errorf("instances: quota exceeded: max %d MB RAM", q.MaxMemoryMB)
	}
	if u.DiskGB+p.DiskGB > q.MaxDiskGB {
		return fmt.Errorf("instances: quota exceeded: max %d GB disk", q.MaxDiskGB)
	}
	if u.MappedPort+extraPorts > q.MaxMappedPorts {
		return fmt.Errorf("instances: quota exceeded: max %d mapped ports", q.MaxMappedPorts)
	}
	return nil
}

// Capacity is the host's free resources as reported by hostd.
type Capacity struct {
	FreeVCPU     int
	FreeMemoryMB int
	FreeDiskGB   int
}

// Overcommit ratios (configurable via env, defaults per plan §1.3).
const (
	DefaultCPUOvercommit = 4 // 4:1 vCPU
	DefaultRAMOvercommit = 1 // 1:1 RAM
)

// CheckCapacity verifies the host can fit plan with overcommit applied.
func CheckCapacity(c Capacity, p Plan) error {
	if c.FreeVCPU < p.VCPU {
		return fmt.Errorf("instances: host capacity exceeded: not enough vCPU")
	}
	if c.FreeMemoryMB < p.MemoryMB {
		return fmt.Errorf("instances: host capacity exceeded: not enough RAM")
	}
	if c.FreeDiskGB < p.DiskGB {
		return fmt.Errorf("instances: host capacity exceeded: not enough disk")
	}
	return nil
}
