package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"studyroom/internal/service"
)

type AdminHandler struct{ svc service.BookingService }

func NewAdminHandler(s service.BookingService) *AdminHandler { return &AdminHandler{svc: s} }

type roomIn struct {
	Name     string `json:"name" binding:"required"`
	Capacity int    `json:"capacity" binding:"required"`
}
type scheduleIn struct {
	Start  string `json:"start" binding:"required"`
	End    string `json:"end" binding:"required"`
	IsOpen bool   `json:"is_open"`
}

func (h *AdminHandler) CreateRoom(c *gin.Context) {
	var in roomIn
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error":"invalid JSON"}); return
	}
	id, err := h.svc.CreateRoom(in.Name, in.Capacity)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *AdminHandler) ListRooms(c *gin.Context) {
	rs, err := h.svc.ListRooms()
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error":"db error"}); return }
	c.JSON(http.StatusOK, rs)
}

func (h *AdminHandler) SetRoomSchedule(c *gin.Context) {
	var in scheduleIn
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error":"invalid JSON"}); return
	}
	roomID := c.Param("id") // hex
	if roomID == "" { c.JSON(http.StatusBadRequest, gin.H{"error":"bad id"}); return }
	if err := h.svc.SetRoomSchedule(roomID, in.Start, in.End, in.IsOpen); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}
	c.JSON(http.StatusOK, gin.H{"status":"ok"})
}
