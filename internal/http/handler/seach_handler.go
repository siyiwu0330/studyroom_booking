package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"studyroom/internal/service"
)

type SearchHandler struct{ svc service.SearchService }

func NewSearchHandler(s service.SearchService) *SearchHandler { return &SearchHandler{svc: s} }

func (h *SearchHandler) SearchRooms(c *gin.Context) {
	start := c.Query("start")
	end := c.Query("end")
	minCap := 1
	if v := c.Query("min_capacity"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			minCap = n
		}
	}
	res, err := h.svc.FindAvailable(minCap, start, end)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}
