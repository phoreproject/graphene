package sync

// Signal represents a signal that can be sent out on many channels at once.
type Signal struct {
	observers []chan interface{}
}

// NewSignal creates a new signal.
func NewSignal() *Signal {
	return &Signal{
		observers: make([]chan interface{}, 0),
	}
}

// Watch watches for a signal to be fired.
func (s *Signal) Watch() chan interface{} {
	watcherChannel := make(chan interface{})

	s.observers = append(s.observers, watcherChannel)

	return watcherChannel
}

// Signal sends a signal to the watchers.
func (s *Signal) Signal(data interface{}) {
	for _, watcher := range s.observers {
		select {
		case watcher <- data:
		default:
		}
	}
}
