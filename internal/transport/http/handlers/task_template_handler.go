package handlers

import (
	"net/http"

	tasktemplateusecase "example.com/taskservice/internal/usecase/tasktemplate"
)

type TaskTemplateHandler struct {
	usecase tasktemplateusecase.Usecase
}

func NewTaskTemplateHandler(usecase tasktemplateusecase.Usecase) *TaskTemplateHandler {
	return &TaskTemplateHandler{usecase: usecase}
}

func (h *TaskTemplateHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req taskTemplateMutationDTO
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	created, err := h.usecase.Create(r.Context(), tasktemplateusecase.CreateInput{
		Title:          req.Title,
		Description:    req.Description,
		Timezone:       req.Timezone,
		StartDate:      req.StartDate,
		EndDate:        req.EndDate,
		RecurrenceType: req.RecurrenceType,
		RecurrenceSettings: tasktemplateusecase.RecurrenceSettingsInput{
			EveryNDays:         req.RecurrenceSettings.EveryNDays,
			DayOfMonth:         req.RecurrenceSettings.DayOfMonth,
			ShortMonthStrategy: req.RecurrenceSettings.ShortMonthStrategy,
			SpecificDates:      req.RecurrenceSettings.SpecificDates,
			EvenOddMode:        req.RecurrenceSettings.EvenOddMode,
			Weekdays:           req.RecurrenceSettings.Weekdays,
		},
	})
	if err != nil {
		writeTaskTemplateUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, newTaskTemplateDTO(created))
}

func (h *TaskTemplateHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	template, err := h.usecase.GetByID(r.Context(), id)
	if err != nil {
		writeTaskTemplateUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newTaskTemplateDTO(template))
}

func (h *TaskTemplateHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	var req taskTemplateMutationDTO
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	updated, err := h.usecase.Update(r.Context(), id, tasktemplateusecase.UpdateInput{
		Title:          req.Title,
		Description:    req.Description,
		Timezone:       req.Timezone,
		StartDate:      req.StartDate,
		EndDate:        req.EndDate,
		RecurrenceType: req.RecurrenceType,
		RecurrenceSettings: tasktemplateusecase.RecurrenceSettingsInput{
			EveryNDays:         req.RecurrenceSettings.EveryNDays,
			DayOfMonth:         req.RecurrenceSettings.DayOfMonth,
			ShortMonthStrategy: req.RecurrenceSettings.ShortMonthStrategy,
			SpecificDates:      req.RecurrenceSettings.SpecificDates,
			EvenOddMode:        req.RecurrenceSettings.EvenOddMode,
			Weekdays:           req.RecurrenceSettings.Weekdays,
		},
	})
	if err != nil {
		writeTaskTemplateUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newTaskTemplateDTO(updated))
}

func (h *TaskTemplateHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if err := h.usecase.Delete(r.Context(), id); err != nil {
		writeTaskTemplateUsecaseError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *TaskTemplateHandler) List(w http.ResponseWriter, r *http.Request) {
	templates, err := h.usecase.List(r.Context())
	if err != nil {
		writeTaskTemplateUsecaseError(w, err)
		return
	}

	response := make([]taskTemplateDTO, 0, len(templates))
	for i := range templates {
		response = append(response, newTaskTemplateDTO(&templates[i]))
	}

	writeJSON(w, http.StatusOK, response)
}
