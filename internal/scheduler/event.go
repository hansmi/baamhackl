package scheduler

type event struct {
	C chan struct{}
}

func newEvent() event {
	return event{
		C: make(chan struct{}, 1),
	}
}

func (c *event) Chan() <-chan struct{} {
	return c.C
}

func (c *event) Set() {
	select {
	case c.C <- struct{}{}:
	default:
	}
}
