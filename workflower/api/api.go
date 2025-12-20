package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/containerd/log"
	"github.com/sazonovItas/mini-ci/core/events"
	"github.com/sazonovItas/mini-ci/workflower/db"
	"github.com/sazonovItas/mini-ci/workflower/planner"
)

type API struct {
	db        *db.DB
	publisher events.Publisher
	planner   planner.Planner
}

func New(db *db.DB, publisher events.Publisher) *API {
	return &API{
		db:        db,
		publisher: publisher,
		planner:   planner.NewPlanner(),
	}
}

func (a *API) RegisterRoutes(mux *http.ServeMux) {
	r := http.NewServeMux()

	// Workflows
	r.HandleFunc("GET /api/workflows", a.listWorkflows)
	r.HandleFunc("POST /api/workflows", a.createWorkflow)
	r.HandleFunc("GET /api/workflows/{id}", a.getWorkflow)
	r.HandleFunc("PUT /api/workflows/{id}", a.updateWorkflow)

	// Builds
	r.HandleFunc("GET /api/workflows/{id}/builds", a.listBuilds)
	r.HandleFunc("POST /api/workflows/{id}/builds", a.startBuild)
	r.HandleFunc("GET /api/builds/{id}", a.getBuild)

	// Jobs
	r.HandleFunc("GET /api/builds/{id}/jobs", a.listJobs)
	r.HandleFunc("GET /api/jobs/{id}", a.getJob)

	// Tasks
	r.HandleFunc("GET /api/jobs/{id}/tasks", a.listTasks)
	r.HandleFunc("GET /api/tasks/{id}", a.getTask)
	r.HandleFunc("GET /api/tasks/{id}/logs", a.getTaskLogs)

	mux.Handle("/", a.withMiddleware(r))
}

func respond(w http.ResponseWriter, data any, err error) {
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, data)
}

func respondError(w http.ResponseWriter, err error) {
	log.G(context.TODO()).WithError(err).Error("API Error")
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func respondJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.G(context.TODO()).WithError(err).Error("JSON encode error")
	}
}
