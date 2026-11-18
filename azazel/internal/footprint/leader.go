package footprint

// Leader binds the launched process to a boot session and kernel birth time.
type Leader struct {
	BootID   string
	Identity Identity
}

type identityReader interface{ Identity(int) (Identity, error) }

// CaptureLeader requires the same boot and leader identity across the read.
// It returns unknown on any failure and never modifies a process.
func CaptureLeader(pid int, source identityReader, boot func() (string, error)) (Leader, bool) {
	if pid <= 1 || source == nil || boot == nil {
		return Leader{}, false
	}
	first, err := boot()
	if err != nil || !ValidBootID(first) {
		return Leader{}, false
	}
	identity, err := source.Identity(pid)
	if err != nil || !validIdentity(identity, pid, pid) {
		return Leader{}, false
	}
	again, err := source.Identity(pid)
	if err != nil || again != identity {
		return Leader{}, false
	}
	last, err := boot()
	if err != nil || first != last {
		return Leader{}, false
	}
	return Leader{BootID: first, Identity: identity}, true
}

// LeaderMatches refuses unknown metadata, a changed boot or a reused PID.
func LeaderMatches(leader Leader, boot string, sample Observation) bool {
	if !sample.Complete || !ValidBootID(leader.BootID) || leader.BootID != boot || !validIdentity(leader.Identity, leader.Identity.PID, leader.Identity.PID) {
		return false
	}
	for _, member := range sample.Members {
		if member.Identity == leader.Identity {
			return true
		}
	}
	return false
}

// ValidBootID accepts canonical lowercase UUIDs without inferring a boot.
func ValidBootID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for i, c := range value {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
			continue
		}
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
