package api

import (
	"net/http"
	"strconv"

	"github.com/sazonovItas/mini-ci/workflower/model"
)

func (a *API) listJobs(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	b, found, err := a.db.BuildFactory().Build(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}
	if !found {
		respondErrorMessage(w, http.StatusNotFound, "Build not found")
		return
	}

	jobs, err := b.Jobs(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}
	var res []model.Job
	for _, j := range jobs {
		res = append(res, j.Model())
	}
	respondJSON(w, res)
}

func (a *API) getJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	j, found, err := a.db.JobFactory().Job(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}
	if !found {
		respondErrorMessage(w, http.StatusNotFound, "Job not found")
		return
	}
	respondJSON(w, j.Model())
}

func (a *API) listTasks(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	j, found, err := a.db.JobFactory().Job(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}
	if !found {
		respondErrorMessage(w, http.StatusNotFound, "Job not found")
		return
	}

	tasks, err := j.Tasks(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}
	var res []model.Task
	for _, t := range tasks {
		res = append(res, t.Model())
	}
	respondJSON(w, res)
}

func (a *API) getTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	t, found, err := a.db.TaskFactory().Task(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}
	if !found {
		respondErrorMessage(w, http.StatusNotFound, "Task not found")
		return
	}
	respondJSON(w, t.Model())
}

func (a *API) getTaskLogs(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	limit := 2000
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

	logs, err := a.db.TaskLogRepository().LastLogs(r.Context(), id, offset, limit)
	respond(w, logs, err)
}
