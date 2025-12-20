package api

import (
	"context"
	"net/http"

	"github.com/containerd/log"
	"github.com/sazonovItas/mini-ci/core/events"
	"github.com/sazonovItas/mini-ci/core/status"
	"github.com/sazonovItas/mini-ci/workflower/model"
)

func (a *API) startBuild(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	workflowID := r.PathValue("id")

	workflow, err := a.db.WorkflowFactory().Workflow(ctx, workflowID)
	if err != nil {
		http.Error(w, "Workflow not found", http.StatusNotFound)
		return
	}
	wfModel := workflow.Model()

	if wfModel.CurrBuild != nil && !wfModel.CurrBuild.Status.IsFinished() {
		http.Error(w, "Workflow is already running (Previous build active). Abort it first.", http.StatusConflict)
		return
	}

	if wfModel.CurrBuild == nil && workflow.CurrBuildID() != "" {
		prevBuild, found, _ := a.db.BuildFactory().Build(ctx, workflow.CurrBuildID())
		if found && !prevBuild.Status().IsFinished() {
			http.Error(w, "Workflow is already running (Previous build active).", http.StatusConflict)
			return
		}
	}

	planOutput, err := a.planner.Plan(wfModel)
	if err != nil {
		http.Error(w, "Planning failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	err = a.db.BuildFactory().WithTx(ctx, func(txCtx context.Context) error {
		if _, err := a.db.BuildFactory().New(txCtx, planOutput.Build); err != nil {
			return err
		}
		for _, job := range planOutput.Jobs {
			if _, err := a.db.JobFactory().New(txCtx, job); err != nil {
				return err
			}
		}
		for _, task := range planOutput.Tasks {
			if _, err := a.db.TaskFactory().New(txCtx, task); err != nil {
				return err
			}
		}
		return workflow.UpdateCurrentBuild(txCtx, planOutput.Build.ID)
	})
	if err != nil {
		respondError(w, err)
		return
	}

	err = a.publisher.Publish(ctx, events.BuildStatus{
		ChangeStatus: events.ChangeStatus{
			EventOrigin: events.NewEventOrigin(planOutput.Build.ID),
			Status:      status.StatusPending,
		},
		WorkflowID: workflowID,
	})
	if err != nil {
		log.G(ctx).WithError(err).Error("failed to publish build start event")
	}

	respondJSON(w, planOutput.Build)
}

func (a *API) listBuilds(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	wf, err := a.db.WorkflowFactory().Workflow(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}
	builds, err := wf.Builds(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}

	var res []model.Build
	for _, b := range builds {
		res = append(res, b.Model())
	}
	respondJSON(w, res)
}

func (a *API) getBuild(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	b, found, err := a.db.BuildFactory().Build(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}
	if !found {
		http.Error(w, "Build not found", http.StatusNotFound)
		return
	}
	respondJSON(w, b.Model())
}
