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
	limit := 10
	offset := 0

	query := r.URL.Query()

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

func (a *API) deleteWorkflow(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := a.db.WorkflowRepository().Delete(r.Context(), id); err != nil {
		respondError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
