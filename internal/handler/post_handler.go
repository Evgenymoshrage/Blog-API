package handler

import (
	"encoding/json"
	"final_project/internal/model"
	"final_project/internal/service"
	"net/http"
	"strconv"
	"strings"
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
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := getUserIDFromContext(r.Context())
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
		writeError(w, "Failed to create post", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(post)
}

// GetByID возвращает пост по ID
func (h *PostHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := extractIDFromPath(r.URL.Path, "/api/posts/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	post, err := h.postService.GetByID(r.Context(), id)
	if err != nil {
		if err == service.ErrPostNotFound {
			writeError(w, "Post not found", http.StatusNotFound)
		} else {
			writeError(w, "Failed to get post", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(post)
}

// GetAll возвращает список постов с пагинацией
func (h *PostHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// Update обновляет пост
func (h *PostHandler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := getUserIDFromContext(r.Context())
	if !ok {
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := extractIDFromPath(r.URL.Path, "/api/posts/")
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
		switch err {
		case service.ErrPostNotFound:
			writeError(w, "Post not found", http.StatusNotFound)
		case service.ErrForbidden:
			writeError(w, "You can only update your own posts", http.StatusForbidden)
		default:
			writeError(w, "Failed to update post", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(post)
}

// Delete удаляет пост
func (h *PostHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := getUserIDFromContext(r.Context())
	if !ok {
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := extractIDFromPath(r.URL.Path, "/api/posts/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	err = h.postService.Delete(r.Context(), id, userID)
	if err != nil {
		switch err {
		case service.ErrPostNotFound:
			writeError(w, "Post not found", http.StatusNotFound)
		case service.ErrForbidden:
			writeError(w, "You can only delete your own posts", http.StatusForbidden)
		default:
			writeError(w, "Failed to delete post", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetByAuthor возвращает посты конкретного автора
func (h *PostHandler) GetByAuthor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 5 {
		writeError(w, "Invalid path", http.StatusBadRequest)
		return
	}
	authorID, err := strconv.Atoi(pathParts[4])
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// extractIDFromPath извлекает ID из пути URL
func extractIDFromPath(path, prefix string) string {
	return strings.TrimPrefix(path, prefix)
}
