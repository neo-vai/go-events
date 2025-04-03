package pagination

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Pagination holds the parsed react-admin compatible pagination and sorting parameters.
type Pagination struct {
	Offset    int
	Limit     int
	SortField string
	SortOrder string
	Filters   map[string]interface{}
}

const (
	defaultLimit = 20
	maxLimit     = 100
)

// ParseReactAdminParams extracts _start, _end, _sort, _order, and filter from the query string.
// It returns an error response via the gin.Context if parameters are invalid.
func ParseReactAdminParams(c *gin.Context) (*Pagination, error) {
	startStr := c.Query("_start")
	endStr := c.Query("_end")

	var offset, limit int
	if startStr == "" || endStr == "" {
		offset = 0
		limit = defaultLimit
	} else {
		start, err := strconv.Atoi(startStr)
		if err != nil || start < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid _start parameter"})
			return nil, fmt.Errorf("invalid _start")
		}
		end, err := strconv.Atoi(endStr)
		if err != nil || end < start {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid _end parameter"})
			return nil, fmt.Errorf("invalid _end")
		}
		offset = start
		limit = end - start
		if limit <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "_end must be greater than _start"})
			return nil, fmt.Errorf("_end <= _start")
		}
		if limit > maxLimit {
			limit = maxLimit
		}
	}

	sort := c.DefaultQuery("_sort", "id")
	order := c.DefaultQuery("_order", "ASC")
	if order != "ASC" && order != "DESC" {
		order = "ASC"
	}

	filters := make(map[string]interface{})
	filterJSON := c.Query("filter")
	if filterJSON != "" {
		if err := json.Unmarshal([]byte(filterJSON), &filters); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid filter JSON"})
			return nil, fmt.Errorf("invalid filter JSON")
		}
	}

	return &Pagination{
		Offset:    offset,
		Limit:     limit,
		SortField: sort,
		SortOrder: order,
		Filters:   filters,
	}, nil
}
