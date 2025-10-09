package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"studyroom/internal/models"
	"studyroom/internal/service"
)

type BookingHandler struct{ svc service.BookingService }

func NewBookingHandler(s service.BookingService) *BookingHandler { return &BookingHandler{svc: s} }

type bookingIn struct {
	RoomID string `json:"room_id" binding:"required"` // hex
	Start  string `json:"start" binding:"required"`   // RFC3339
	End    string `json:"end" binding:"required"`
}

type waitIn struct {
	RoomID string `json:"room_id" binding:"required"`
	Start  string `json:"start" binding:"required"`
	End    string `json:"end" binding:"required"`
}

func (h *BookingHandler) Create(c *gin.Context) {
	u := c.MustGet("user").(*models.User)
	var in bookingIn
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"}); return
	}
	id, err := h.svc.CreateBooking(in.RoomID, u.ID, in.Start, in.End)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusCreated, gin.H{"booking_id": id})
}

func (h *BookingHandler) Cancel(c *gin.Context) {
	u := c.MustGet("user").(*models.User)
	bid := c.Param("id") // hex booking id
	if bid == "" { c.JSON(http.StatusBadRequest, gin.H{"error":"bad id"}); return }
	if err := h.svc.CancelBooking(bid, u.ID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}
	c.JSON(http.StatusOK, gin.H{"status": "cancelled"})
}

func (h *BookingHandler) JoinWaitlist(c *gin.Context) {
	u := c.MustGet("user").(*models.User)
	var in waitIn
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"}); return
	}
	if err := h.svc.JoinWaitlist(in.RoomID, u.ID, in.Start, in.End); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}
	c.JSON(http.StatusOK, gin.H{"status": "queued"})
}
