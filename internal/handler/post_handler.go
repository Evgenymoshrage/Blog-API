package handler

import (
	"encoding/json"
	"errors"
	"final_project/internal/middleware"
	"final_project/internal/model"
	"final_project/internal/service"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type PostHandler struct {
	postService *service.PostService
}

func NewPostHandler(postService *service.PostService) *PostHandler {
	return &PostHandler{
		postService: postService,
	}
}

// Create обрабатывает создание нового поста
func (h *PostHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req model.PostCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	post, err := h.postService.Create(r.Context(), userID, &req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrValidation):
			writeError(w, err.Error(), http.StatusBadRequest)
		default:
			writeError(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	writeJSON(w, post, http.StatusCreated)
}

// GetByID возвращает пост по ID
func (h *PostHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	post, err := h.postService.GetByID(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPostNotFound):
			writeError(w, "Post not found", http.StatusNotFound)
		default:
			writeError(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	writeJSON(w, post, http.StatusOK)
}

// GetAll возвращает список постов с пагинацией
func (h *PostHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	limit, _ := strconv.Atoi(query.Get("limit"))
	if limit <= 0 {
		limit = 10
	}
	offset, _ := strconv.Atoi(query.Get("offset"))
	if offset < 0 {
		offset = 0
	}

	posts, total, err := h.postService.GetAll(r.Context(), limit, offset)
	if err != nil {
		writeError(w, "Failed to get posts", http.StatusInternalServerError)
		return
	}

	resp := struct {
		Posts  []*model.Post `json:"posts"`
		Total  int           `json:"total"`
		Limit  int           `json:"limit"`
		Offset int           `json:"offset"`
	}{
		Posts:  posts,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}

	writeJSON(w, resp, http.StatusOK)

}

// Update обновляет пост
func (h *PostHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	var req model.PostUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	post, err := h.postService.Update(r.Context(), id, userID, &req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPostNotFound):
			writeError(w, err.Error(), http.StatusNotFound)
		case errors.Is(err, service.ErrForbidden):
			writeError(w, err.Error(), http.StatusForbidden)
		case errors.Is(err, service.ErrValidation):
			writeError(w, err.Error(), http.StatusBadRequest)
		default:
			writeError(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	writeJSON(w, post, http.StatusOK)

}

// Delete удаляет пост
func (h *PostHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	err = h.postService.Delete(r.Context(), id, userID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPostNotFound):
			writeError(w, "Post not found", http.StatusNotFound)
		case errors.Is(err, service.ErrForbidden):
			writeError(w, "Forbidden", http.StatusForbidden)
		default:
			writeError(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetByAuthor возвращает посты конкретного автора
func (h *PostHandler) GetByAuthor(w http.ResponseWriter, r *http.Request) {
	authorStr := chi.URLParam(r, "id")
	authorID, err := strconv.Atoi(authorStr)
	if err != nil {
		writeError(w, "Invalid author ID", http.StatusBadRequest)
		return
	}

	query := r.URL.Query()
	limit, _ := strconv.Atoi(query.Get("limit"))
	if limit <= 0 {
		limit = 10
	}
	offset, _ := strconv.Atoi(query.Get("offset"))
	if offset < 0 {
		offset = 0
	}

	posts, total, err := h.postService.GetByAuthor(r.Context(), authorID, limit, offset)
	if err != nil {
		writeError(w, "Failed to get posts", http.StatusInternalServerError)
		return
	}

	resp := struct {
		Posts  []*model.Post `json:"posts"`
		Total  int           `json:"total"`
		Limit  int           `json:"limit"`
		Offset int           `json:"offset"`
		Author int           `json:"author_id"`
	}{
		Posts:  posts,
		Total:  total,
		Limit:  limit,
		Offset: offset,
		Author: authorID,
	}

	writeJSON(w, resp, http.StatusOK)
}

func writeJSON(w http.ResponseWriter, data any, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(data)
}
