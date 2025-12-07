package events

import "slices"

type FilterFunc func(e Event) bool

func WithEventType(types ...EventType) FilterFunc {
	return func(e Event) bool {
		return slices.Contains(types, e.Type())
	}
}
