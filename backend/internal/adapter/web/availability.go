package web

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strconv"

	"github.com/RinZ5/converge/backend/internal/core/models"
	"github.com/RinZ5/converge/backend/internal/core/ports"

	"github.com/gin-gonic/gin"
)

type AvailabilityHandler struct {
	repo ports.AvailabilityRepository
}

func NewAvailabilityHandler(repo ports.AvailabilityRepository) *AvailabilityHandler {
	return &AvailabilityHandler{repo: repo}
}

func (h *AvailabilityHandler) GetTeachers(c *gin.Context) {
	teachers, err := h.repo.GetActiveTeachers(c.Request.Context())
	if err != nil {
		log.Printf("GetTeachers error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve teachers"})

		return
	}
	c.JSON(http.StatusOK, teachers)
}

func (h *AvailabilityHandler) SubmitWeeklyAvailability(c *gin.Context) {
	var payload models.AvailabilityPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := validateWeeklySlots(payload.Weekly); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rawJSON, _ := json.Marshal(payload)
	ctx := c.Request.Context()

	if err := h.repo.ReplaceWeeklyAvailability(ctx, payload.TeacherID, payload.Weekly); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save availability"})
		return
	}

	if err := h.repo.SaveRawSubmission(ctx, payload.TeacherID, rawJSON); err != nil {
		log.Printf("Warning: failed to save submission audit log: %v", err)
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Availability saved successfully"})
}

func validateWeeklySlots(slots []models.WeeklySlot) error {
	byDay := make(map[int][]models.WeeklySlot)
	for _, s := range slots {
		byDay[s.DayOfWeek] = append(byDay[s.DayOfWeek], s)
	}
	for day, daySlots := range byDay {
		sort.Slice(daySlots, func(i, j int) bool { return daySlots[i].Start < daySlots[j].Start })
		for i := 1; i < len(daySlots); i++ {
			if daySlots[i].Start < daySlots[i-1].End {
				return &overlapError{day: day, a: daySlots[i-1], b: daySlots[i]}
			}
		}
	}
	return nil
}

type overlapError struct {
	day  int
	a, b models.WeeklySlot
}

func (e *overlapError) Error() string {
	return "overlapping slots on day " + strconv.Itoa(e.day) + ": " + e.a.Start + "-" + e.a.End + " and " + e.b.Start + "-" + e.b.End
}
