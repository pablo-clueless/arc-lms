package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"arc-lms/internal/domain"
	"arc-lms/internal/repository"
	"arc-lms/internal/repository/postgres"

	"github.com/google/uuid"
)

// MeetingService handles meeting operations
type MeetingService struct {
	meetingRepo         *postgres.MeetingRepository
	classRepo           *postgres.ClassRepository
	courseRepo          *postgres.CourseRepository
	enrollmentRepo      *postgres.EnrollmentRepository
	notificationService *NotificationService
	auditService        *AuditService
}

// NewMeetingService creates a new meeting service
func NewMeetingService(
	meetingRepo *postgres.MeetingRepository,
	classRepo *postgres.ClassRepository,
	courseRepo *postgres.CourseRepository,
	enrollmentRepo *postgres.EnrollmentRepository,
	notificationService *NotificationService,
	auditService *AuditService,
) *MeetingService {
	return &MeetingService{
		meetingRepo:         meetingRepo,
		classRepo:           classRepo,
		courseRepo:          courseRepo,
		enrollmentRepo:      enrollmentRepo,
		notificationService: notificationService,
		auditService:        auditService,
	}
}

// ScheduleMeetingRequest represents a request to schedule a meeting
type ScheduleMeetingRequest struct {
	ClassID           uuid.UUID            `json:"class_id" validate:"required,uuid"`
	CourseID          *uuid.UUID           `json:"course_id,omitempty" validate:"omitempty,uuid"`
	Title             string               `json:"title" validate:"required,min=3,max=200"`
	Description       *string              `json:"description,omitempty" validate:"omitempty,max=1000"`
	ScheduledStart    time.Time            `json:"scheduled_start" validate:"required"`
	EstimatedDuration int                  `json:"estimated_duration" validate:"required,min=15,max=240"`
	Provider          domain.MeetingProvider `json:"provider" validate:"required,oneof=DAILY ZOOM JITSI CUSTOM"`
	AccessCode        *string              `json:"access_code,omitempty"`
}

// ScheduleMeeting schedules a new meeting
func (s *MeetingService) ScheduleMeeting(
	ctx context.Context,
	tenantID uuid.UUID,
	tutorID uuid.UUID,
	req *ScheduleMeetingRequest,
	ipAddress string,
) (*domain.Meeting, error) {
	// Verify tutor can host meetings for this class
	canHost, err := s.meetingRepo.ValidateTutorCanHost(ctx, tutorID, req.ClassID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate tutor access: %w", err)
	}
	if !canHost {
		return nil, fmt.Errorf("you are not assigned to any course in this class")
	}

	// Validate scheduled start is in the future
	if req.ScheduledStart.Before(time.Now()) {
		return nil, fmt.Errorf("scheduled start time must be in the future")
	}

	// Generate meeting URL and provider meeting ID
	// In a real implementation, this would integrate with the video provider API
	meetingURL, providerMeetingID := s.generateMeetingCredentials(req.Provider)

	now := time.Now()
	meeting := &domain.Meeting{
		ID:                uuid.New(),
		TenantID:          tenantID,
		ClassID:           req.ClassID,
		CourseID:          req.CourseID,
		HostTutorID:       tutorID,
		Title:             req.Title,
		Description:       req.Description,
		ScheduledStart:    req.ScheduledStart,
		EstimatedDuration: req.EstimatedDuration,
		Status:            domain.MeetingStatusScheduled,
		Provider:          req.Provider,
		MeetingURL:        meetingURL,
		ProviderMeetingID: providerMeetingID,
		AccessCode:        req.AccessCode,
		ParticipantEvents: make([]domain.ParticipantEvent, 0),
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := s.meetingRepo.Create(ctx, meeting, nil); err != nil {
		return nil, fmt.Errorf("failed to create meeting: %w", err)
	}

	// Notify students about the meeting
	go s.notifyStudentsAboutMeeting(ctx, meeting, "scheduled")

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionMeetingScheduled,
		tutorID,
		domain.RoleTutor,
		&tenantID,
		domain.AuditResourceMeeting,
		meeting.ID,
		nil,
		meeting,
		ipAddress,
	)

	return meeting, nil
}

// generateMeetingCredentials generates a meeting URL and provider meeting ID
func (s *MeetingService) generateMeetingCredentials(provider domain.MeetingProvider) (string, string) {
	// Generate a unique meeting ID
	bytes := make([]byte, 8)
	rand.Read(bytes)
	providerMeetingID := hex.EncodeToString(bytes)

	// In a real implementation, this would call the provider's API
	// For now, generate placeholder URLs
	var meetingURL string
	switch provider {
	case domain.MeetingProviderDaily:
		meetingURL = fmt.Sprintf("https://daily.co/arc-lms/%s", providerMeetingID)
	case domain.MeetingProviderZoom:
		meetingURL = fmt.Sprintf("https://zoom.us/j/%s", providerMeetingID)
	case domain.MeetingProviderJitsi:
		meetingURL = fmt.Sprintf("https://meet.jit.si/arc-lms-%s", providerMeetingID)
	default:
		meetingURL = fmt.Sprintf("https://meet.arc-lms.com/%s", providerMeetingID)
	}

	return meetingURL, providerMeetingID
}

// notifyStudentsAboutMeeting sends notifications to students about a meeting
func (s *MeetingService) notifyStudentsAboutMeeting(ctx context.Context, meeting *domain.Meeting, action string) {
	if s.notificationService == nil {
		return
	}

	// Get all enrolled students in the class
	enrollments, _, err := s.enrollmentRepo.ListByClass(ctx, meeting.ClassID, repository.PaginationParams{Limit: 1000})
	if err != nil {
		return
	}

	var title, body string
	var eventType domain.NotificationEventType
	switch action {
	case "scheduled":
		title = "New Meeting Scheduled"
		body = fmt.Sprintf("A new meeting '%s' has been scheduled for %s", meeting.Title, meeting.ScheduledStart.Format("Jan 2, 2006 at 3:04 PM"))
		eventType = domain.NotificationEventMeetingScheduled
	case "started":
		title = "Meeting Started"
		body = fmt.Sprintf("The meeting '%s' has started. Join now!", meeting.Title)
		eventType = domain.NotificationEventMeetingStarting
	case "cancelled":
		title = "Meeting Cancelled"
		body = fmt.Sprintf("The meeting '%s' has been cancelled", meeting.Title)
		eventType = domain.NotificationEventMeetingCancelled
	}

	// Collect student IDs
	studentIDs := make([]uuid.UUID, len(enrollments))
	for i, enrollment := range enrollments {
		studentIDs[i] = enrollment.StudentID
	}

	actionURL := fmt.Sprintf("/meetings/%s", meeting.ID.String())
	resourceType := "MEETING"

	_ = s.notificationService.SendNotificationToUsers(
		ctx,
		meeting.TenantID,
		studentIDs,
		eventType,
		title,
		body,
		[]domain.NotificationChannel{domain.NotificationChannelInApp},
		domain.NotificationPriorityHigh,
		&actionURL,
		&resourceType,
		&meeting.ID,
	)
}

// GetMeeting retrieves a meeting by ID
func (s *MeetingService) GetMeeting(ctx context.Context, id uuid.UUID) (*domain.Meeting, error) {
	return s.meetingRepo.Get(ctx, id)
}

// UpdateMeetingRequest represents a request to update a meeting
type UpdateMeetingRequest struct {
	Title             *string    `json:"title,omitempty" validate:"omitempty,min=3,max=200"`
	Description       *string    `json:"description,omitempty" validate:"omitempty,max=1000"`
	ScheduledStart    *time.Time `json:"scheduled_start,omitempty"`
	EstimatedDuration *int       `json:"estimated_duration,omitempty" validate:"omitempty,min=15,max=240"`
}

// UpdateMeeting updates a scheduled meeting
func (s *MeetingService) UpdateMeeting(
	ctx context.Context,
	meetingID uuid.UUID,
	tutorID uuid.UUID,
	req *UpdateMeetingRequest,
	ipAddress string,
) (*domain.Meeting, error) {
	meeting, err := s.meetingRepo.Get(ctx, meetingID)
	if err != nil {
		return nil, fmt.Errorf("failed to get meeting: %w", err)
	}

	// Only the host can update
	if meeting.HostTutorID != tutorID {
		return nil, fmt.Errorf("only the meeting host can update the meeting")
	}

	// Can only update scheduled meetings
	if meeting.Status != domain.MeetingStatusScheduled {
		return nil, fmt.Errorf("can only update scheduled meetings")
	}

	// Store before state for audit
	beforeState := *meeting

	if req.Title != nil {
		meeting.Title = *req.Title
	}
	if req.Description != nil {
		meeting.Description = req.Description
	}
	if req.ScheduledStart != nil {
		if req.ScheduledStart.Before(time.Now()) {
			return nil, fmt.Errorf("scheduled start time must be in the future")
		}
		meeting.ScheduledStart = *req.ScheduledStart
	}
	if req.EstimatedDuration != nil {
		meeting.EstimatedDuration = *req.EstimatedDuration
	}

	meeting.UpdatedAt = time.Now()

	if err := s.meetingRepo.Update(ctx, meeting, nil); err != nil {
		return nil, fmt.Errorf("failed to update meeting: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionMeetingScheduled, // Using same action for update
		tutorID,
		domain.RoleTutor,
		&meeting.TenantID,
		domain.AuditResourceMeeting,
		meeting.ID,
		&beforeState,
		meeting,
		ipAddress,
	)

	return meeting, nil
}

// StartMeeting starts a scheduled meeting
func (s *MeetingService) StartMeeting(
	ctx context.Context,
	meetingID uuid.UUID,
	tutorID uuid.UUID,
	ipAddress string,
) (*domain.Meeting, error) {
	meeting, err := s.meetingRepo.Get(ctx, meetingID)
	if err != nil {
		return nil, fmt.Errorf("failed to get meeting: %w", err)
	}

	// Only the host can start
	if meeting.HostTutorID != tutorID {
		return nil, fmt.Errorf("only the meeting host can start the meeting")
	}

	// Can only start scheduled meetings
	if meeting.Status != domain.MeetingStatusScheduled {
		return nil, fmt.Errorf("can only start scheduled meetings")
	}

	meeting.Start()

	if err := s.meetingRepo.Update(ctx, meeting, nil); err != nil {
		return nil, fmt.Errorf("failed to start meeting: %w", err)
	}

	// Notify students that meeting has started
	go s.notifyStudentsAboutMeeting(ctx, meeting, "started")

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionMeetingStarted,
		tutorID,
		domain.RoleTutor,
		&meeting.TenantID,
		domain.AuditResourceMeeting,
		meeting.ID,
		nil,
		meeting,
		ipAddress,
	)

	return meeting, nil
}

// EndMeeting ends a live meeting
func (s *MeetingService) EndMeeting(
	ctx context.Context,
	meetingID uuid.UUID,
	tutorID uuid.UUID,
	ipAddress string,
) (*domain.Meeting, error) {
	meeting, err := s.meetingRepo.Get(ctx, meetingID)
	if err != nil {
		return nil, fmt.Errorf("failed to get meeting: %w", err)
	}

	// Only the host can end
	if meeting.HostTutorID != tutorID {
		return nil, fmt.Errorf("only the meeting host can end the meeting")
	}

	// Can only end live meetings
	if meeting.Status != domain.MeetingStatusLive {
		return nil, fmt.Errorf("can only end live meetings")
	}

	meeting.End()

	if err := s.meetingRepo.Update(ctx, meeting, nil); err != nil {
		return nil, fmt.Errorf("failed to end meeting: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionMeetingEnded,
		tutorID,
		domain.RoleTutor,
		&meeting.TenantID,
		domain.AuditResourceMeeting,
		meeting.ID,
		nil,
		meeting,
		ipAddress,
	)

	return meeting, nil
}

// CancelMeetingRequest represents a request to cancel a meeting
type CancelMeetingRequest struct {
	Reason string `json:"reason" validate:"required,min=5,max=500"`
}

// CancelMeeting cancels a scheduled meeting
func (s *MeetingService) CancelMeeting(
	ctx context.Context,
	meetingID uuid.UUID,
	cancelledBy uuid.UUID,
	role domain.Role,
	req *CancelMeetingRequest,
	ipAddress string,
) (*domain.Meeting, error) {
	meeting, err := s.meetingRepo.Get(ctx, meetingID)
	if err != nil {
		return nil, fmt.Errorf("failed to get meeting: %w", err)
	}

	// Tutors can only cancel their own meetings, admins can cancel any
	if role == domain.RoleTutor && meeting.HostTutorID != cancelledBy {
		return nil, fmt.Errorf("you can only cancel your own meetings")
	}

	// Can only cancel scheduled meetings
	if meeting.Status != domain.MeetingStatusScheduled {
		return nil, fmt.Errorf("can only cancel scheduled meetings")
	}

	meeting.Cancel(cancelledBy, req.Reason)

	if err := s.meetingRepo.Update(ctx, meeting, nil); err != nil {
		return nil, fmt.Errorf("failed to cancel meeting: %w", err)
	}

	// Notify students about cancellation
	go s.notifyStudentsAboutMeeting(ctx, meeting, "cancelled")

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionMeetingCancelled,
		cancelledBy,
		role,
		&meeting.TenantID,
		domain.AuditResourceMeeting,
		meeting.ID,
		nil,
		meeting,
		ipAddress,
	)

	return meeting, nil
}

// RecordParticipantJoin records when a participant joins a meeting
func (s *MeetingService) RecordParticipantJoin(
	ctx context.Context,
	meetingID uuid.UUID,
	userID uuid.UUID,
) error {
	meeting, err := s.meetingRepo.Get(ctx, meetingID)
	if err != nil {
		return fmt.Errorf("failed to get meeting: %w", err)
	}

	// Can only record for live meetings
	if meeting.Status != domain.MeetingStatusLive {
		return fmt.Errorf("meeting is not live")
	}

	meeting.RecordParticipantJoin(userID)

	if err := s.meetingRepo.Update(ctx, meeting, nil); err != nil {
		return fmt.Errorf("failed to record participant join: %w", err)
	}

	return nil
}

// RecordParticipantLeave records when a participant leaves a meeting
func (s *MeetingService) RecordParticipantLeave(
	ctx context.Context,
	meetingID uuid.UUID,
	userID uuid.UUID,
	durationMinutes int,
) error {
	meeting, err := s.meetingRepo.Get(ctx, meetingID)
	if err != nil {
		return fmt.Errorf("failed to get meeting: %w", err)
	}

	meeting.RecordParticipantLeave(userID, durationMinutes)

	if err := s.meetingRepo.Update(ctx, meeting, nil); err != nil {
		return fmt.Errorf("failed to record participant leave: %w", err)
	}

	return nil
}

// AddRecordingRequest represents a request to add a recording
type AddRecordingRequest struct {
	RecordingURL       string     `json:"recording_url" validate:"required,url"`
	RecordingExpiresAt *time.Time `json:"recording_expires_at,omitempty"`
}

// AddRecording adds a recording URL to an ended meeting
func (s *MeetingService) AddRecording(
	ctx context.Context,
	meetingID uuid.UUID,
	req *AddRecordingRequest,
) (*domain.Meeting, error) {
	meeting, err := s.meetingRepo.Get(ctx, meetingID)
	if err != nil {
		return nil, fmt.Errorf("failed to get meeting: %w", err)
	}

	// Can only add recording to ended meetings
	if meeting.Status != domain.MeetingStatusEnded {
		return nil, fmt.Errorf("can only add recording to ended meetings")
	}

	meeting.RecordingURL = &req.RecordingURL
	meeting.RecordingExpiresAt = req.RecordingExpiresAt
	meeting.UpdatedAt = time.Now()

	if err := s.meetingRepo.Update(ctx, meeting, nil); err != nil {
		return nil, fmt.Errorf("failed to add recording: %w", err)
	}

	return meeting, nil
}

// ListMeetings lists meetings with filters
func (s *MeetingService) ListMeetings(
	ctx context.Context,
	tenantID uuid.UUID,
	status *domain.MeetingStatus,
	params repository.PaginationParams,
) ([]*domain.Meeting, *repository.PaginatedResult, error) {
	meetings, total, err := s.meetingRepo.ListByTenant(ctx, tenantID, status, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list meetings: %w", err)
	}

	pagination := repository.BuildPaginatedResult(total, params)

	return meetings, &pagination, nil
}

// ListMeetingsByClass lists meetings for a class
func (s *MeetingService) ListMeetingsByClass(
	ctx context.Context,
	classID uuid.UUID,
	params repository.PaginationParams,
) ([]*domain.Meeting, *repository.PaginatedResult, error) {
	meetings, total, err := s.meetingRepo.ListByClass(ctx, classID, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list meetings: %w", err)
	}

	pagination := repository.BuildPaginatedResult(total, params)

	return meetings, &pagination, nil
}

// ListMeetingsByTutor lists meetings hosted by a tutor
func (s *MeetingService) ListMeetingsByTutor(
	ctx context.Context,
	tutorID uuid.UUID,
	params repository.PaginationParams,
) ([]*domain.Meeting, *repository.PaginatedResult, error) {
	meetings, total, err := s.meetingRepo.ListByTutor(ctx, tutorID, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list meetings: %w", err)
	}

	pagination := repository.BuildPaginatedResult(total, params)

	return meetings, &pagination, nil
}

// ListUpcomingMeetings lists upcoming meetings for a tenant
func (s *MeetingService) ListUpcomingMeetings(ctx context.Context, tenantID uuid.UUID, limit int) ([]*domain.Meeting, error) {
	return s.meetingRepo.ListUpcoming(ctx, tenantID, limit)
}

// ListLiveMeetings lists currently live meetings
func (s *MeetingService) ListLiveMeetings(ctx context.Context, tenantID uuid.UUID) ([]*domain.Meeting, error) {
	return s.meetingRepo.ListLive(ctx, tenantID)
}

// ListUpcomingMeetingsForStudent lists upcoming meetings for a student
func (s *MeetingService) ListUpcomingMeetingsForStudent(ctx context.Context, studentID uuid.UUID, limit int) ([]*domain.Meeting, error) {
	return s.meetingRepo.GetUpcomingMeetingsForStudent(ctx, studentID, limit)
}

// GetMeetingStatistics retrieves meeting statistics
func (s *MeetingService) GetMeetingStatistics(ctx context.Context, tenantID uuid.UUID, startDate, endDate time.Time) (*postgres.MeetingStatistics, error) {
	return s.meetingRepo.GetMeetingStatistics(ctx, tenantID, startDate, endDate)
}

// GetMeetingJoinURL returns the meeting join URL for a user
func (s *MeetingService) GetMeetingJoinURL(
	ctx context.Context,
	meetingID uuid.UUID,
	userID uuid.UUID,
) (string, error) {
	meeting, err := s.meetingRepo.Get(ctx, meetingID)
	if err != nil {
		return "", fmt.Errorf("failed to get meeting: %w", err)
	}

	// Meeting must be scheduled or live
	if meeting.Status != domain.MeetingStatusScheduled && meeting.Status != domain.MeetingStatusLive {
		return "", fmt.Errorf("meeting is not accessible")
	}

	// In a real implementation, this might generate a unique join URL with the user's identity
	return meeting.MeetingURL, nil
}
