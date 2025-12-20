package api

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/sazonovItas/mini-ci/workflower/model"
)

type WorkflowRequest struct {
	Name   string               `json:"name"`
	Config model.WorkflowConfig `json:"config"`
}

func (a *API) listWorkflows(w http.ResponseWriter, r *http.Request) {
	workflows, err := a.db.WorkflowRepository().Workflows(r.Context(), 0, 100)
	respond(w, workflows, err)
}

func (a *API) createWorkflow(w http.ResponseWriter, r *http.Request) {
	var req WorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondErrorMessage(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	if req.Name == "" {
		respondErrorMessage(w, http.StatusBadRequest, "Workflow name is required")
		return
	}

	id := uuid.New().String()
	workflow := model.Workflow{
		ID:     id,
		Name:   req.Name,
		Config: req.Config,
	}

	created, err := a.db.WorkflowFactory().New(r.Context(), workflow)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, created.Model())
}

func (a *API) updateWorkflow(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ctx := r.Context()

	existing, err := a.db.WorkflowFactory().Workflow(ctx, id)
	if err != nil {
		return
	}

	var req WorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondErrorMessage(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	if req.Name == "" {
		respondErrorMessage(w, http.StatusBadRequest, "Workflow name is required")
		return
	}

	updatedModel := model.Workflow{
		ID:     id,
		Name:   req.Name,
		Config: req.Config,
	}

	if err := existing.Update(ctx, updatedModel); err != nil {
		respondError(w, err)
		return
	}

	respondJSON(w, existing.Model())
}

func (a *API) getWorkflow(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	wf, err := a.db.WorkflowFactory().Workflow(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, wf.Model())
}
