package domain

import (
	"time"

	"github.com/google/uuid"
)

// MeetingStatus represents the lifecycle status of a meeting
type MeetingStatus string

const (
	MeetingStatusScheduled MeetingStatus = "SCHEDULED"
	MeetingStatusLive      MeetingStatus = "LIVE"
	MeetingStatusEnded     MeetingStatus = "ENDED"
	MeetingStatusCancelled MeetingStatus = "CANCELLED"
)

// MeetingProvider represents the video conferencing provider
type MeetingProvider string

const (
	MeetingProviderDaily   MeetingProvider = "DAILY"
	MeetingProviderZoom    MeetingProvider = "ZOOM"
	MeetingProviderJitsi   MeetingProvider = "JITSI"
	MeetingProviderCustom  MeetingProvider = "CUSTOM"
)

// ParticipantEvent represents a join/leave event during a meeting
type ParticipantEvent struct {
	UserID    uuid.UUID `json:"user_id" validate:"required,uuid"`
	EventType string    `json:"event_type" validate:"required,oneof=JOIN LEAVE"`
	Timestamp time.Time `json:"timestamp" validate:"required"`
	Duration  *int      `json:"duration,omitempty"` // Duration in minutes (for LEAVE events)
}

// Meeting represents a virtual class session
type Meeting struct {
	ID                 uuid.UUID          `json:"id" validate:"required,uuid"`
	TenantID           uuid.UUID          `json:"tenant_id" validate:"required,uuid"`
	ClassID            uuid.UUID          `json:"class_id" validate:"required,uuid"`
	CourseID           *uuid.UUID         `json:"course_id,omitempty" validate:"omitempty,uuid"` // Optional course association
	HostTutorID        uuid.UUID          `json:"host_tutor_id" validate:"required,uuid"`
	Title              string             `json:"title" validate:"required,min=3,max=200"`
	Description        *string            `json:"description,omitempty" validate:"omitempty,max=1000"`
	ScheduledStart     time.Time          `json:"scheduled_start" validate:"required"`
	EstimatedDuration  int                `json:"estimated_duration" validate:"required,min=15,max=240"` // in minutes
	Status             MeetingStatus      `json:"status" validate:"required,oneof=SCHEDULED LIVE ENDED CANCELLED"`
	Provider           MeetingProvider    `json:"provider" validate:"required,oneof=DAILY ZOOM JITSI CUSTOM"`
	MeetingURL         string             `json:"meeting_url" validate:"required,url"`
	ProviderMeetingID  string             `json:"provider_meeting_id" validate:"required"`
	AccessCode         *string            `json:"access_code,omitempty"` // Optional password/PIN
	ActualStartTime    *time.Time         `json:"actual_start_time,omitempty"`
	ActualEndTime      *time.Time         `json:"actual_end_time,omitempty"`
	RecordingURL       *string            `json:"recording_url,omitempty" validate:"omitempty,url"`
	RecordingExpiresAt *time.Time         `json:"recording_expires_at,omitempty"`
	ParticipantEvents  []ParticipantEvent `json:"participant_events"`
	CancellationReason *string            `json:"cancellation_reason,omitempty" validate:"omitempty,max=500"`
	CancelledBy        *uuid.UUID         `json:"cancelled_by,omitempty" validate:"omitempty,uuid"`
	CancelledAt        *time.Time         `json:"cancelled_at,omitempty"`
	CreatedAt          time.Time          `json:"created_at" validate:"required"`
	UpdatedAt          time.Time          `json:"updated_at" validate:"required"`
}

// IsScheduled returns true if the meeting is scheduled
func (m *Meeting) IsScheduled() bool {
	return m.Status == MeetingStatusScheduled
}

// IsLive returns true if the meeting is currently live
func (m *Meeting) IsLive() bool {
	return m.Status == MeetingStatusLive
}

// IsEnded returns true if the meeting has ended
func (m *Meeting) IsEnded() bool {
	return m.Status == MeetingStatusEnded
}

// IsCancelled returns true if the meeting was cancelled
func (m *Meeting) IsCancelled() bool {
	return m.Status == MeetingStatusCancelled
}

// Start marks the meeting as live
func (m *Meeting) Start() {
	m.Status = MeetingStatusLive
	now := time.Now()
	m.ActualStartTime = &now
	m.UpdatedAt = now
}

// End marks the meeting as ended
func (m *Meeting) End() {
	m.Status = MeetingStatusEnded
	now := time.Now()
	m.ActualEndTime = &now
	m.UpdatedAt = now
}

// Cancel marks the meeting as cancelled with a reason
func (m *Meeting) Cancel(cancelledBy uuid.UUID, reason string) {
	m.Status = MeetingStatusCancelled
	m.CancelledBy = &cancelledBy
	m.CancellationReason = &reason
	now := time.Now()
	m.CancelledAt = &now
	m.UpdatedAt = now
}

// RecordParticipantJoin records when a participant joins the meeting
func (m *Meeting) RecordParticipantJoin(userID uuid.UUID) {
	event := ParticipantEvent{
		UserID:    userID,
		EventType: "JOIN",
		Timestamp: time.Now(),
	}
	m.ParticipantEvents = append(m.ParticipantEvents, event)
	m.UpdatedAt = time.Now()
}

// RecordParticipantLeave records when a participant leaves the meeting
func (m *Meeting) RecordParticipantLeave(userID uuid.UUID, durationMinutes int) {
	event := ParticipantEvent{
		UserID:    userID,
		EventType: "LEAVE",
		Timestamp: time.Now(),
		Duration:  &durationMinutes,
	}
	m.ParticipantEvents = append(m.ParticipantEvents, event)
	m.UpdatedAt = time.Now()
}

// GetActualDuration returns the actual meeting duration in minutes
func (m *Meeting) GetActualDuration() int {
	if m.ActualStartTime == nil || m.ActualEndTime == nil {
		return 0
	}
	return int(m.ActualEndTime.Sub(*m.ActualStartTime).Minutes())
}

// GetParticipantCount returns the number of unique participants
func (m *Meeting) GetParticipantCount() int {
	uniqueParticipants := make(map[uuid.UUID]bool)
	for _, event := range m.ParticipantEvents {
		uniqueParticipants[event.UserID] = true
	}
	return len(uniqueParticipants)
}

// HasRecording returns true if a recording is available
func (m *Meeting) HasRecording() bool {
	return m.RecordingURL != nil && *m.RecordingURL != ""
}

// IsRecordingExpired returns true if the recording has expired
func (m *Meeting) IsRecordingExpired(now time.Time) bool {
	if m.RecordingExpiresAt == nil {
		return false
	}
	return now.After(*m.RecordingExpiresAt)
}
