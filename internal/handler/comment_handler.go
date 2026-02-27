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

type CommentHandler struct {
	commentService *service.CommentService
}

func NewCommentHandler(commentService *service.CommentService) *CommentHandler {
	return &CommentHandler{
		commentService: commentService,
	}
}

// Create обрабатывает создание нового комментария
func (h *CommentHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req model.CommentCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	postIDStr := chi.URLParam(r, "id")
	postID, err := strconv.Atoi(postIDStr)
	if err != nil {
		writeError(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	comment, err := h.commentService.Create(r.Context(), postID, userID, &req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrValidation):
			writeError(w, err.Error(), http.StatusBadRequest)

		case errors.Is(err, service.ErrNotFound):
			writeError(w, "Post not found", http.StatusNotFound)

		case errors.Is(err, service.ErrForbidden):
			writeError(w, "Forbidden", http.StatusForbidden)

		default:
			writeError(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	writeJSON(w, comment, http.StatusCreated)
}

// GetByID возвращает комментарий по ID
func (h *CommentHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, "Invalid comment ID", http.StatusBadRequest)
		return
	}

	comment, err := h.commentService.GetByID(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNotFound):
			writeError(w, "Comment not found", http.StatusNotFound)
		default:
			writeError(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	writeJSON(w, comment, http.StatusOK)
}

// GetByPost возвращает комментарии к посту с пагинацией
func (h *CommentHandler) GetByPost(w http.ResponseWriter, r *http.Request) {
	postIDStr := chi.URLParam(r, "id")
	postID, err := strconv.Atoi(postIDStr)
	if err != nil {
		writeError(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	query := r.URL.Query()
	limit, _ := strconv.Atoi(query.Get("limit"))
	if limit <= 0 {
		limit = 20
	}
	offset, _ := strconv.Atoi(query.Get("offset"))
	if offset < 0 {
		offset = 0
	}

	comments, total, err := h.commentService.GetByPost(r.Context(), postID, limit, offset)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNotFound):
			writeError(w, "Post not found", http.StatusNotFound)
		default:
			writeError(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	resp := struct {
		Comments []*model.Comment `json:"comments"`
		Total    int              `json:"total"`
		Limit    int              `json:"limit"`
		Offset   int              `json:"offset"`
		PostID   int              `json:"post_id"`
	}{
		Comments: comments,
		Total:    total,
		Limit:    limit,
		Offset:   offset,
		PostID:   postID,
	}

	writeJSON(w, resp, http.StatusOK)
}

// Update обновляет комментарий
func (h *CommentHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, "Invalid comment ID", http.StatusBadRequest)
		return
	}

	var req model.CommentUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	comment, err := h.commentService.Update(r.Context(), id, userID, &req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrValidation):
			writeError(w, err.Error(), http.StatusBadRequest)

		case errors.Is(err, service.ErrNotFound):
			writeError(w, "Comment not found", http.StatusNotFound)

		case errors.Is(err, service.ErrForbidden):
			writeError(w, "Forbidden", http.StatusForbidden)

		default:
			writeError(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	writeJSON(w, comment, http.StatusOK)
}
