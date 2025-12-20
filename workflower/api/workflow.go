package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/sazonovItas/mini-ci/workflower/model"
)

type WorkflowRequest struct {
	Name   string               `json:"name"`
	Config model.WorkflowConfig `json:"config"`
}

func (a *API) listWorkflows(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	limit := 10
	offset := 0

	if l := query.Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil {
			limit = val
		}
	}

	if o := query.Get("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil {
			offset = val
		}
	}

	workflows, err := a.db.WorkflowRepository().Workflows(r.Context(), offset, limit)
	respond(w, workflows, err)
}

func (a *API) createWorkflow(w http.ResponseWriter, r *http.Request) {
	var req WorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Workflow name is required", http.StatusBadRequest)
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
		http.Error(w, "Workflow not found", http.StatusNotFound)
		return
	}

	var req WorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Workflow name is required", http.StatusBadRequest)
		return
	}

	updatedModel := model.Workflow{
		ID:     id,
		Name:   req.Name,
		Config: req.Config,
	}

	err = existing.Update(ctx, updatedModel)
	if err != nil {
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
