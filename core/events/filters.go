package events

import "slices"

type FilterFunc func(e Event) bool

func WithEventTypes(types ...EventType) FilterFunc {
	return func(e Event) bool {
		return slices.Contains(types, e.Type())
	}
}

func ExcludeEventTypes(types ...EventType) FilterFunc {
	return func(e Event) bool {
		return !slices.Contains(types, e.Type())
	}
}

func WithEventOriginID(id string) FilterFunc {
	return func(e Event) bool {
		return e.Origin().ID == id
	}
}
