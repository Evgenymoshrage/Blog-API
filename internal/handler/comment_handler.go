package handler

import (
	"encoding/json"
	"final_project/internal/model"
	"final_project/internal/service"
	"net/http"
	"strconv"
	"strings"
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
	if r.Method != http.MethodPost { // Проверка HTTP-метода
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Получение ID пользователя из контекста
	userID, ok := getUserIDFromContext(r.Context())
	if !ok {
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Декодирование тела запроса
	var req model.CommentCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Извлекаем postID из URL
	postIDStr := extractPostIDFromCommentsPath(r.URL.Path)
	postID, err := strconv.Atoi(postIDStr)
	if err != nil {
		writeError(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	// Вызов сервиса для создания комментария
	comment, err := h.commentService.Create(r.Context(), userID, postID, &req)
	if err != nil {
		switch err {
		case service.ErrPostNotExists:
			writeError(w, "Post not found", http.StatusNotFound)
		default:
			writeError(w, "Failed to create comment", http.StatusInternalServerError)
		}
		return
	}

	// Отправка успешного ответа
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(comment)
}

// GetByID возвращает комментарий по ID
func (h *CommentHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet { // Проверяем метод
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Извлекаем ID комментария из URL
	idStr := extractIDFromPath(r.URL.Path, "/api/comments/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, "Invalid comment ID", http.StatusBadRequest)
		return
	}

	// Вызываем соответствующий метод сервиса
	comment, err := h.commentService.GetByID(r.Context(), id)
	if err != nil { // Обрабатываем ошибки
		if err == service.ErrCommentNotFound {
			writeError(w, "Comment not found", http.StatusNotFound)
		} else {
			writeError(w, "Failed to get comment", http.StatusInternalServerError)
		}
		return
	}

	// Возвращаем JSON с данными комментариев
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(comment)
}

// GetByPost возвращает комментарии к посту с пагинацией
func (h *CommentHandler) GetByPost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := extractPostIDFromCommentsPath(r.URL.Path)
	postID, err := strconv.Atoi(idStr)
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
		if err == service.ErrPostNotExists {
			writeError(w, "Post not found", http.StatusNotFound)
		} else {
			writeError(w, "Failed to get comments", http.StatusInternalServerError)
		}
		return
	}

	type CommentsResponse struct {
		Comments []*model.Comment `json:"comments"`
		Total    int              `json:"total"`
		Limit    int              `json:"limit"`
		Offset   int              `json:"offset"`
		PostID   int              `json:"post_id"`
	}

	resp := CommentsResponse{
		Comments: comments,
		Total:    total,
		Limit:    limit,
		Offset:   offset,
		PostID:   postID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// Update обновляет комментарий
func (h *CommentHandler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut { // Проверяем метод
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := getUserIDFromContext(r.Context()) // Получаем userID из контекста
	if !ok {
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Извлекаем commentID из URL
	idStr := extractIDFromPath(r.URL.Path, "/api/comments/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, "Invalid comment ID", http.StatusBadRequest)
		return
	}

	// Декодируем JSON тела запроса
	var req model.CommentUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Вызываем commentService.Update с userID и commentID
	comment, err := h.commentService.Update(r.Context(), id, userID, &req)
	if err != nil { // Обрабатываем ошибки
		switch err {
		case service.ErrCommentNotFound:
			writeError(w, "Comment not found", http.StatusNotFound)
		case service.ErrForbidden:
			writeError(w, "You can only update your own comments", http.StatusForbidden)
		default:
			writeError(w, "Failed to update comment", http.StatusInternalServerError)
		}
		return
	}

	// Возвращаем обновлённый комментарий в JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(comment)
}

// Извлекает ID поста из пути /api/posts/{id}/comments
func extractPostIDFromCommentsPath(path string) string {
	path = strings.TrimPrefix(path, "/api/posts/")
	return strings.Split(path, "/")[0]
}
