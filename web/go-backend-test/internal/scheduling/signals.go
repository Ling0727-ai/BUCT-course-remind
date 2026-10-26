package scheduling

// Buffered notifications coalesce bursts without allocating a goroutine per write.
var RefreshChanged = make(chan struct{}, 1)
var RemindersChanged = make(chan struct{}, 1)

func NotifyRefresh()   { notify(RefreshChanged) }
func NotifyReminders() { notify(RemindersChanged) }

func notify(ch chan struct{}) {
	select {
	case ch <- struct{}{}:
	default:
	}
}
