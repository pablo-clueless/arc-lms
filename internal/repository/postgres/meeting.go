package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"arc-lms/internal/domain"
	"arc-lms/internal/repository"
)

// MeetingRepository handles database operations for meetings
type MeetingRepository struct {
	*repository.BaseRepository
}

// NewMeetingRepository creates a new meeting repository
func NewMeetingRepository(db *sql.DB) *MeetingRepository {
	return &MeetingRepository{
		BaseRepository: repository.NewBaseRepository(db),
	}
}

// Create creates a new meeting
func (r *MeetingRepository) Create(ctx context.Context, meeting *domain.Meeting, tx *sql.Tx) error {
	participantEventsJSON, err := json.Marshal(meeting.ParticipantEvents)
	if err != nil {
		return fmt.Errorf("failed to marshal participant events: %w", err)
	}

	query := `
		INSERT INTO meetings (
			id, tenant_id, class_id, course_id, host_tutor_id,
			title, description, scheduled_start, estimated_duration,
			status, provider, meeting_url, provider_meeting_id,
			access_code, actual_start_time, actual_end_time,
			recording_url, recording_expires_at, participant_events,
			cancellation_reason, cancelled_by, cancelled_at,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24)
	`

	execer := repository.GetExecer(r.GetDB(), tx)
	_, err = execer.ExecContext(ctx, query,
		meeting.ID,
		meeting.TenantID,
		meeting.ClassID,
		meeting.CourseID,
		meeting.HostTutorID,
		meeting.Title,
		meeting.Description,
		meeting.ScheduledStart,
		meeting.EstimatedDuration,
		meeting.Status,
		meeting.Provider,
		meeting.MeetingURL,
		meeting.ProviderMeetingID,
		meeting.AccessCode,
		meeting.ActualStartTime,
		meeting.ActualEndTime,
		meeting.RecordingURL,
		meeting.RecordingExpiresAt,
		participantEventsJSON,
		meeting.CancellationReason,
		meeting.CancelledBy,
		meeting.CancelledAt,
		meeting.CreatedAt,
		meeting.UpdatedAt,
	)

	if err != nil {
		return repository.ParseError(err)
	}

	return nil
}

// Get retrieves a meeting by ID
func (r *MeetingRepository) Get(ctx context.Context, id uuid.UUID) (*domain.Meeting, error) {
	query := `
		SELECT
			id, tenant_id, class_id, course_id, host_tutor_id,
			title, description, scheduled_start, estimated_duration,
			status, provider, meeting_url, provider_meeting_id,
			access_code, actual_start_time, actual_end_time,
			recording_url, recording_expires_at, participant_events,
			cancellation_reason, cancelled_by, cancelled_at,
			created_at, updated_at
		FROM meetings
		WHERE id = $1
	`

	return r.scanMeeting(r.GetDB().QueryRowContext(ctx, query, id))
}

// scanMeeting scans a meeting from a database row
func (r *MeetingRepository) scanMeeting(row *sql.Row) (*domain.Meeting, error) {
	var meeting domain.Meeting
	var courseID, cancelledBy sql.NullString
	var description, accessCode, recordingURL, cancellationReason sql.NullString
	var actualStartTime, actualEndTime, recordingExpiresAt, cancelledAt sql.NullTime
	var participantEventsJSON []byte

	err := row.Scan(
		&meeting.ID,
		&meeting.TenantID,
		&meeting.ClassID,
		&courseID,
		&meeting.HostTutorID,
		&meeting.Title,
		&description,
		&meeting.ScheduledStart,
		&meeting.EstimatedDuration,
		&meeting.Status,
		&meeting.Provider,
		&meeting.MeetingURL,
		&meeting.ProviderMeetingID,
		&accessCode,
		&actualStartTime,
		&actualEndTime,
		&recordingURL,
		&recordingExpiresAt,
		&participantEventsJSON,
		&cancellationReason,
		&cancelledBy,
		&cancelledAt,
		&meeting.CreatedAt,
		&meeting.UpdatedAt,
	)

	if err != nil {
		return nil, repository.ParseError(err)
	}

	if courseID.Valid {
		id, _ := uuid.Parse(courseID.String)
		meeting.CourseID = &id
	}
	if description.Valid {
		meeting.Description = &description.String
	}
	if accessCode.Valid {
		meeting.AccessCode = &accessCode.String
	}
	if actualStartTime.Valid {
		meeting.ActualStartTime = &actualStartTime.Time
	}
	if actualEndTime.Valid {
		meeting.ActualEndTime = &actualEndTime.Time
	}
	if recordingURL.Valid {
		meeting.RecordingURL = &recordingURL.String
	}
	if recordingExpiresAt.Valid {
		meeting.RecordingExpiresAt = &recordingExpiresAt.Time
	}
	if cancellationReason.Valid {
		meeting.CancellationReason = &cancellationReason.String
	}
	if cancelledBy.Valid {
		id, _ := uuid.Parse(cancelledBy.String)
		meeting.CancelledBy = &id
	}
	if cancelledAt.Valid {
		meeting.CancelledAt = &cancelledAt.Time
	}

	if len(participantEventsJSON) > 0 {
		if err := json.Unmarshal(participantEventsJSON, &meeting.ParticipantEvents); err != nil {
			return nil, fmt.Errorf("failed to unmarshal participant events: %w", err)
		}
	} else {
		meeting.ParticipantEvents = make([]domain.ParticipantEvent, 0)
	}

	return &meeting, nil
}

// Update updates a meeting
func (r *MeetingRepository) Update(ctx context.Context, meeting *domain.Meeting, tx *sql.Tx) error {
	participantEventsJSON, err := json.Marshal(meeting.ParticipantEvents)
	if err != nil {
		return fmt.Errorf("failed to marshal participant events: %w", err)
	}

	query := `
		UPDATE meetings
		SET
			title = $2,
			description = $3,
			scheduled_start = $4,
			estimated_duration = $5,
			status = $6,
			meeting_url = $7,
			access_code = $8,
			actual_start_time = $9,
			actual_end_time = $10,
			recording_url = $11,
			recording_expires_at = $12,
			participant_events = $13,
			cancellation_reason = $14,
			cancelled_by = $15,
			cancelled_at = $16,
			updated_at = $17
		WHERE id = $1
	`

	execer := repository.GetExecer(r.GetDB(), tx)
	result, err := execer.ExecContext(ctx, query,
		meeting.ID,
		meeting.Title,
		meeting.Description,
		meeting.ScheduledStart,
		meeting.EstimatedDuration,
		meeting.Status,
		meeting.MeetingURL,
		meeting.AccessCode,
		meeting.ActualStartTime,
		meeting.ActualEndTime,
		meeting.RecordingURL,
		meeting.RecordingExpiresAt,
		participantEventsJSON,
		meeting.CancellationReason,
		meeting.CancelledBy,
		meeting.CancelledAt,
		meeting.UpdatedAt,
	)

	if err != nil {
		return repository.ParseError(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return repository.ErrNotFound
	}

	return nil
}

// Delete deletes a meeting
func (r *MeetingRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM meetings WHERE id = $1`

	result, err := r.GetDB().ExecContext(ctx, query, id)
	if err != nil {
		return repository.ParseError(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return repository.ErrNotFound
	}

	return nil
}

// ListByTenant retrieves meetings for a tenant
func (r *MeetingRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, status *domain.MeetingStatus, params repository.PaginationParams) ([]*domain.Meeting, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, err
	}

	query := `
		SELECT
			id, tenant_id, class_id, course_id, host_tutor_id,
			title, description, scheduled_start, estimated_duration,
			status, provider, meeting_url, provider_meeting_id,
			access_code, actual_start_time, actual_end_time,
			recording_url, recording_expires_at, participant_events,
			cancellation_reason, cancelled_by, cancelled_at,
			created_at, updated_at
		FROM meetings
		WHERE tenant_id = $1
	`

	args := []interface{}{tenantID}
	argIndex := 2

	if status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, *status)
		argIndex++
	}

	if params.Cursor != nil {
		if params.SortOrder == "DESC" {
			query += fmt.Sprintf(" AND id < $%d", argIndex)
		} else {
			query += fmt.Sprintf(" AND id > $%d", argIndex)
		}
		args = append(args, *params.Cursor)
		argIndex++
	}

	query += fmt.Sprintf(" ORDER BY scheduled_start %s LIMIT $%d", params.SortOrder, argIndex)
	args = append(args, params.Limit+1)

	return r.queryMeetings(ctx, query, args...)
}

// ListByClass retrieves meetings for a class
func (r *MeetingRepository) ListByClass(ctx context.Context, classID uuid.UUID, params repository.PaginationParams) ([]*domain.Meeting, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, err
	}

	query := `
		SELECT
			id, tenant_id, class_id, course_id, host_tutor_id,
			title, description, scheduled_start, estimated_duration,
			status, provider, meeting_url, provider_meeting_id,
			access_code, actual_start_time, actual_end_time,
			recording_url, recording_expires_at, participant_events,
			cancellation_reason, cancelled_by, cancelled_at,
			created_at, updated_at
		FROM meetings
		WHERE class_id = $1
	`

	args := []interface{}{classID}
	argIndex := 2

	if params.Cursor != nil {
		if params.SortOrder == "DESC" {
			query += fmt.Sprintf(" AND id < $%d", argIndex)
		} else {
			query += fmt.Sprintf(" AND id > $%d", argIndex)
		}
		args = append(args, *params.Cursor)
		argIndex++
	}

	query += fmt.Sprintf(" ORDER BY scheduled_start %s LIMIT $%d", params.SortOrder, argIndex)
	args = append(args, params.Limit+1)

	return r.queryMeetings(ctx, query, args...)
}

// ListByTutor retrieves meetings hosted by a tutor
func (r *MeetingRepository) ListByTutor(ctx context.Context, tutorID uuid.UUID, params repository.PaginationParams) ([]*domain.Meeting, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, err
	}

	query := `
		SELECT
			id, tenant_id, class_id, course_id, host_tutor_id,
			title, description, scheduled_start, estimated_duration,
			status, provider, meeting_url, provider_meeting_id,
			access_code, actual_start_time, actual_end_time,
			recording_url, recording_expires_at, participant_events,
			cancellation_reason, cancelled_by, cancelled_at,
			created_at, updated_at
		FROM meetings
		WHERE host_tutor_id = $1
	`

	args := []interface{}{tutorID}
	argIndex := 2

	if params.Cursor != nil {
		if params.SortOrder == "DESC" {
			query += fmt.Sprintf(" AND id < $%d", argIndex)
		} else {
			query += fmt.Sprintf(" AND id > $%d", argIndex)
		}
		args = append(args, *params.Cursor)
		argIndex++
	}

	query += fmt.Sprintf(" ORDER BY scheduled_start %s LIMIT $%d", params.SortOrder, argIndex)
	args = append(args, params.Limit+1)

	return r.queryMeetings(ctx, query, args...)
}

// ListUpcoming retrieves upcoming meetings for a tenant
func (r *MeetingRepository) ListUpcoming(ctx context.Context, tenantID uuid.UUID, limit int) ([]*domain.Meeting, error) {
	query := `
		SELECT
			id, tenant_id, class_id, course_id, host_tutor_id,
			title, description, scheduled_start, estimated_duration,
			status, provider, meeting_url, provider_meeting_id,
			access_code, actual_start_time, actual_end_time,
			recording_url, recording_expires_at, participant_events,
			cancellation_reason, cancelled_by, cancelled_at,
			created_at, updated_at
		FROM meetings
		WHERE tenant_id = $1 AND status = $2 AND scheduled_start > $3
		ORDER BY scheduled_start ASC
		LIMIT $4
	`

	return r.queryMeetings(ctx, query, tenantID, domain.MeetingStatusScheduled, time.Now(), limit)
}

// ListLive retrieves currently live meetings for a tenant
func (r *MeetingRepository) ListLive(ctx context.Context, tenantID uuid.UUID) ([]*domain.Meeting, error) {
	query := `
		SELECT
			id, tenant_id, class_id, course_id, host_tutor_id,
			title, description, scheduled_start, estimated_duration,
			status, provider, meeting_url, provider_meeting_id,
			access_code, actual_start_time, actual_end_time,
			recording_url, recording_expires_at, participant_events,
			cancellation_reason, cancelled_by, cancelled_at,
			created_at, updated_at
		FROM meetings
		WHERE tenant_id = $1 AND status = $2
		ORDER BY actual_start_time ASC
	`

	return r.queryMeetings(ctx, query, tenantID, domain.MeetingStatusLive)
}

// ListByDateRange retrieves meetings within a date range
// If tenantID is uuid.Nil, it returns meetings across all tenants
func (r *MeetingRepository) ListByDateRange(ctx context.Context, tenantID uuid.UUID, startDate, endDate time.Time, params repository.PaginationParams) ([]*domain.Meeting, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, err
	}

	var query string
	var args []interface{}

	if tenantID == uuid.Nil {
		// Cross-tenant query (for background jobs)
		query = `
			SELECT
				id, tenant_id, class_id, course_id, host_tutor_id,
				title, description, scheduled_start, estimated_duration,
				status, provider, meeting_url, provider_meeting_id,
				access_code, actual_start_time, actual_end_time,
				recording_url, recording_expires_at, participant_events,
				cancellation_reason, cancelled_by, cancelled_at,
				created_at, updated_at
			FROM meetings
			WHERE scheduled_start >= $1 AND scheduled_start < $2
			ORDER BY scheduled_start ASC
			LIMIT $3
		`
		args = []interface{}{startDate, endDate, params.Limit}
	} else {
		query = `
			SELECT
				id, tenant_id, class_id, course_id, host_tutor_id,
				title, description, scheduled_start, estimated_duration,
				status, provider, meeting_url, provider_meeting_id,
				access_code, actual_start_time, actual_end_time,
				recording_url, recording_expires_at, participant_events,
				cancellation_reason, cancelled_by, cancelled_at,
				created_at, updated_at
			FROM meetings
			WHERE tenant_id = $1 AND scheduled_start >= $2 AND scheduled_start < $3
			ORDER BY scheduled_start ASC
			LIMIT $4
		`
		args = []interface{}{tenantID, startDate, endDate, params.Limit}
	}

	return r.queryMeetings(ctx, query, args...)
}

// queryMeetings executes a query and returns a list of meetings
func (r *MeetingRepository) queryMeetings(ctx context.Context, query string, args ...interface{}) ([]*domain.Meeting, error) {
	rows, err := r.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, repository.ParseError(err)
	}
	defer rows.Close()

	meetings := make([]*domain.Meeting, 0)
	for rows.Next() {
		var meeting domain.Meeting
		var courseID, cancelledBy sql.NullString
		var description, accessCode, recordingURL, cancellationReason sql.NullString
		var actualStartTime, actualEndTime, recordingExpiresAt, cancelledAt sql.NullTime
		var participantEventsJSON []byte

		err := rows.Scan(
			&meeting.ID,
			&meeting.TenantID,
			&meeting.ClassID,
			&courseID,
			&meeting.HostTutorID,
			&meeting.Title,
			&description,
			&meeting.ScheduledStart,
			&meeting.EstimatedDuration,
			&meeting.Status,
			&meeting.Provider,
			&meeting.MeetingURL,
			&meeting.ProviderMeetingID,
			&accessCode,
			&actualStartTime,
			&actualEndTime,
			&recordingURL,
			&recordingExpiresAt,
			&participantEventsJSON,
			&cancellationReason,
			&cancelledBy,
			&cancelledAt,
			&meeting.CreatedAt,
			&meeting.UpdatedAt,
		)

		if err != nil {
			return nil, repository.ParseError(err)
		}

		if courseID.Valid {
			id, _ := uuid.Parse(courseID.String)
			meeting.CourseID = &id
		}
		if description.Valid {
			meeting.Description = &description.String
		}
		if accessCode.Valid {
			meeting.AccessCode = &accessCode.String
		}
		if actualStartTime.Valid {
			meeting.ActualStartTime = &actualStartTime.Time
		}
		if actualEndTime.Valid {
			meeting.ActualEndTime = &actualEndTime.Time
		}
		if recordingURL.Valid {
			meeting.RecordingURL = &recordingURL.String
		}
		if recordingExpiresAt.Valid {
			meeting.RecordingExpiresAt = &recordingExpiresAt.Time
		}
		if cancellationReason.Valid {
			meeting.CancellationReason = &cancellationReason.String
		}
		if cancelledBy.Valid {
			id, _ := uuid.Parse(cancelledBy.String)
			meeting.CancelledBy = &id
		}
		if cancelledAt.Valid {
			meeting.CancelledAt = &cancelledAt.Time
		}

		if len(participantEventsJSON) > 0 {
			if err := json.Unmarshal(participantEventsJSON, &meeting.ParticipantEvents); err != nil {
				return nil, fmt.Errorf("failed to unmarshal participant events: %w", err)
			}
		} else {
			meeting.ParticipantEvents = make([]domain.ParticipantEvent, 0)
		}

		meetings = append(meetings, &meeting)
	}

	if err := rows.Err(); err != nil {
		return nil, repository.ParseError(err)
	}

	return meetings, nil
}

// GetMeetingStatistics retrieves statistics for meetings in a tenant
func (r *MeetingRepository) GetMeetingStatistics(ctx context.Context, tenantID uuid.UUID, startDate, endDate time.Time) (*MeetingStatistics, error) {
	query := `
		SELECT
			COUNT(*) as total_meetings,
			COUNT(*) FILTER (WHERE status = 'SCHEDULED') as scheduled_count,
			COUNT(*) FILTER (WHERE status = 'LIVE') as live_count,
			COUNT(*) FILTER (WHERE status = 'ENDED') as ended_count,
			COUNT(*) FILTER (WHERE status = 'CANCELLED') as cancelled_count,
			COALESCE(AVG(EXTRACT(EPOCH FROM (actual_end_time - actual_start_time)) / 60), 0) as avg_duration_minutes,
			COALESCE(SUM(jsonb_array_length(participant_events) / 2), 0) as total_participants
		FROM meetings
		WHERE tenant_id = $1 AND scheduled_start >= $2 AND scheduled_start <= $3
	`

	var stats MeetingStatistics
	stats.TenantID = tenantID
	stats.StartDate = startDate
	stats.EndDate = endDate

	err := r.GetDB().QueryRowContext(ctx, query, tenantID, startDate, endDate).Scan(
		&stats.TotalMeetings,
		&stats.ScheduledCount,
		&stats.LiveCount,
		&stats.EndedCount,
		&stats.CancelledCount,
		&stats.AvgDurationMinutes,
		&stats.TotalParticipants,
	)

	if err != nil {
		return nil, repository.ParseError(err)
	}

	return &stats, nil
}

// MeetingStatistics represents aggregated meeting statistics
type MeetingStatistics struct {
	TenantID           uuid.UUID `json:"tenant_id"`
	StartDate          time.Time `json:"start_date"`
	EndDate            time.Time `json:"end_date"`
	TotalMeetings      int       `json:"total_meetings"`
	ScheduledCount     int       `json:"scheduled_count"`
	LiveCount          int       `json:"live_count"`
	EndedCount         int       `json:"ended_count"`
	CancelledCount     int       `json:"cancelled_count"`
	AvgDurationMinutes float64   `json:"avg_duration_minutes"`
	TotalParticipants  int       `json:"total_participants"`
}

// ValidateTutorCanHost checks if a tutor can host meetings for a class
func (r *MeetingRepository) ValidateTutorCanHost(ctx context.Context, tutorID, classID uuid.UUID) (bool, error) {
	// Check if the tutor is assigned to any course in this class
	query := `
		SELECT EXISTS(
			SELECT 1 FROM courses
			WHERE class_id = $1 AND assigned_tutor_id = $2
		)
	`

	var canHost bool
	err := r.GetDB().QueryRowContext(ctx, query, classID, tutorID).Scan(&canHost)
	if err != nil {
		return false, repository.ParseError(err)
	}

	return canHost, nil
}

// GetUpcomingMeetingsForStudent retrieves upcoming meetings for a student's class
func (r *MeetingRepository) GetUpcomingMeetingsForStudent(ctx context.Context, studentID uuid.UUID, limit int) ([]*domain.Meeting, error) {
	query := `
		SELECT
			m.id, m.tenant_id, m.class_id, m.course_id, m.host_tutor_id,
			m.title, m.description, m.scheduled_start, m.estimated_duration,
			m.status, m.provider, m.meeting_url, m.provider_meeting_id,
			m.access_code, m.actual_start_time, m.actual_end_time,
			m.recording_url, m.recording_expires_at, m.participant_events,
			m.cancellation_reason, m.cancelled_by, m.cancelled_at,
			m.created_at, m.updated_at
		FROM meetings m
		INNER JOIN enrollments e ON m.class_id = e.class_id
		WHERE e.student_id = $1 AND e.status = 'ACTIVE' AND m.status = 'SCHEDULED' AND m.scheduled_start > $2
		ORDER BY m.scheduled_start ASC
		LIMIT $3
	`

	return r.queryMeetings(ctx, query, studentID, time.Now(), limit)
}
