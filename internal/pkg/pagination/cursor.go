package pagination

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"time"
)

// CursorParams represents pagination parameters
type CursorParams struct {
	Limit  int
	Cursor string // Base64 encoded cursor
}

// CursorResponse represents paginated response metadata
type CursorResponse struct {
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}

// ParseCursorParams parses cursor pagination parameters from query
func ParseCursorParams(cursorStr string, limitStr string, defaultLimit int) CursorParams {
	limit := defaultLimit
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	return CursorParams{
		Limit:  limit,
		Cursor: cursorStr,
	}
}

// EncodeCursor encodes a timestamp and ID into a cursor
func EncodeCursor(timestamp time.Time, id string) string {
	cursor := fmt.Sprintf("%d:%s", timestamp.Unix(), id)
	return base64.StdEncoding.EncodeToString([]byte(cursor))
}

// DecodeCursor decodes a cursor into timestamp and ID
func DecodeCursor(cursor string) (time.Time, string, error) {
	if cursor == "" {
		return time.Time{}, "", nil
	}

	decoded, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("invalid cursor format")
	}

	var timestamp int64
	var id string
	_, err = fmt.Sscanf(string(decoded), "%d:%s", &timestamp, &id)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("invalid cursor format")
	}

	return time.Unix(timestamp, 0), id, nil
}

// BuildCursorResponse builds a cursor response with next cursor if there are more results
func BuildCursorResponse(hasMore bool, lastTimestamp time.Time, lastID string) CursorResponse {
	resp := CursorResponse{
		HasMore: hasMore,
	}

	if hasMore {
		resp.NextCursor = EncodeCursor(lastTimestamp, lastID)
	}

	return resp
}
