package engine

type Engine struct {
	planner *Planner
}

func New() *Engine {
	return &Engine{
		planner: NewPlanner(),
	}
}
