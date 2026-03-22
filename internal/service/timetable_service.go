package service

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"

	"arc-lms/internal/domain"
	"arc-lms/internal/repository"
	"arc-lms/internal/repository/postgres"

	"github.com/google/uuid"
)

// TimetableService handles timetable management operations
type TimetableService struct {
	db              *sql.DB
	timetableRepo   *postgres.TimetableRepository
	periodRepo      *postgres.PeriodRepository
	swapRequestRepo *postgres.SwapRequestRepository
	courseRepo      *postgres.CourseRepository
	classRepo       *postgres.ClassRepository
	termRepo        *postgres.TermRepository
	tenantRepo      *postgres.TenantRepository
	auditService    *AuditService
}

// NewTimetableService creates a new timetable service
func NewTimetableService(
	db *sql.DB,
	timetableRepo *postgres.TimetableRepository,
	periodRepo *postgres.PeriodRepository,
	swapRequestRepo *postgres.SwapRequestRepository,
	courseRepo *postgres.CourseRepository,
	classRepo *postgres.ClassRepository,
	termRepo *postgres.TermRepository,
	tenantRepo *postgres.TenantRepository,
	auditService *AuditService,
) *TimetableService {
	return &TimetableService{
		db:              db,
		timetableRepo:   timetableRepo,
		periodRepo:      periodRepo,
		swapRequestRepo: swapRequestRepo,
		courseRepo:      courseRepo,
		classRepo:       classRepo,
		termRepo:        termRepo,
		tenantRepo:      tenantRepo,
		auditService:    auditService,
	}
}

// TimetableConfig holds configuration for timetable generation
type TimetableConfig struct {
	PeriodDuration    int               // Duration in minutes (default: 40)
	DailyPeriodLimit  int               // Max periods per day (default: 8)
	MaxPeriodsPerWeek map[string]int    // Course name -> max weekly periods
	StartTime         time.Time         // Daily start time (e.g., 8:00 AM)
	BreakAfterPeriod  int               // Break after which period (e.g., 4)
	BreakDuration     int               // Break duration in minutes (e.g., 30)
	InstructionalDays []domain.DayOfWeek // Which days have classes
}

// DefaultTimetableConfig returns default timetable configuration
func DefaultTimetableConfig() *TimetableConfig {
	startTime, _ := time.Parse("15:04", "08:00")
	return &TimetableConfig{
		PeriodDuration:    40,
		DailyPeriodLimit:  8,
		MaxPeriodsPerWeek: make(map[string]int),
		StartTime:         startTime,
		BreakAfterPeriod:  4,
		BreakDuration:     30,
		InstructionalDays: []domain.DayOfWeek{
			domain.DayMonday,
			domain.DayTuesday,
			domain.DayWednesday,
			domain.DayThursday,
			domain.DayFriday,
		},
	}
}

// GenerateTimetableRequest represents a request to generate a timetable
type GenerateTimetableRequest struct {
	ClassID uuid.UUID `json:"class_id" validate:"required,uuid"`
	TermID  uuid.UUID `json:"term_id" validate:"required,uuid"`
	Notes   *string   `json:"notes,omitempty" validate:"omitempty,max=1000"`
}

// TimetableWithPeriods combines a timetable with its periods
type TimetableWithPeriods struct {
	Timetable *domain.Timetable `json:"timetable"`
	Periods   []*domain.Period  `json:"periods"`
}

// GenerateTimetable automatically generates a timetable for a class
func (s *TimetableService) GenerateTimetable(
	ctx context.Context,
	tenantID uuid.UUID,
	req *GenerateTimetableRequest,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*TimetableWithPeriods, error) {
	// Verify class belongs to tenant
	class, err := s.classRepo.Get(ctx, req.ClassID)
	if err != nil {
		return nil, fmt.Errorf("failed to get class: %w", err)
	}
	if class.TenantID != tenantID {
		return nil, fmt.Errorf("class does not belong to tenant")
	}

	// Verify term belongs to tenant
	term, err := s.termRepo.Get(ctx, req.TermID)
	if err != nil {
		return nil, fmt.Errorf("failed to get term: %w", err)
	}
	if term.TenantID != tenantID {
		return nil, fmt.Errorf("term does not belong to tenant")
	}

	// Get tenant configuration
	tenant, err := s.tenantRepo.Get(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	// Get all active courses for this class and term
	courses, _, err := s.courseRepo.ListByClass(ctx, req.ClassID, repository.PaginationParams{Limit: 100})
	if err != nil {
		return nil, fmt.Errorf("failed to list courses: %w", err)
	}

	// Filter courses for the specific term
	var termCourses []*domain.Course
	for _, course := range courses {
		if course.TermID == req.TermID && course.Status == domain.CourseStatusActive {
			termCourses = append(termCourses, course)
		}
	}

	if len(termCourses) == 0 {
		return nil, fmt.Errorf("no active courses found for this class and term")
	}

	// Build configuration from tenant settings
	config := DefaultTimetableConfig()
	if tenant.Configuration.PeriodDuration > 0 {
		config.PeriodDuration = tenant.Configuration.PeriodDuration
	}
	if tenant.Configuration.DailyPeriodLimit > 0 {
		config.DailyPeriodLimit = tenant.Configuration.DailyPeriodLimit
	}
	if len(tenant.Configuration.MaxPeriodsPerWeek) > 0 {
		config.MaxPeriodsPerWeek = tenant.Configuration.MaxPeriodsPerWeek
	}

	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Check for existing timetable and archive it if needed
	existingTimetable, err := s.timetableRepo.GetByClassAndTerm(ctx, req.ClassID, req.TermID)
	if err == nil && existingTimetable != nil {
		// Archive existing timetable
		existingTimetable.Archive()
		if err := s.timetableRepo.Archive(ctx, existingTimetable.ID, tx); err != nil {
			return nil, fmt.Errorf("failed to archive existing timetable: %w", err)
		}

		// Cancel pending swap requests for the old timetable
		if err := s.swapRequestRepo.CancelPendingByTimetable(ctx, existingTimetable.ID, tx); err != nil {
			return nil, fmt.Errorf("failed to cancel pending swap requests: %w", err)
		}
	}

	// Determine generation version
	generationVersion := 1
	if existingTimetable != nil {
		generationVersion = existingTimetable.GenerationVersion + 1
	}

	// Create new timetable
	now := time.Now()
	timetable := &domain.Timetable{
		ID:                uuid.New(),
		TenantID:          tenantID,
		ClassID:           req.ClassID,
		TermID:            req.TermID,
		Status:            domain.TimetableStatusDraft,
		GeneratedAt:       now,
		GeneratedBy:       actorID,
		GenerationVersion: generationVersion,
		Notes:             req.Notes,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := s.timetableRepo.Create(ctx, timetable, tx); err != nil {
		return nil, fmt.Errorf("failed to create timetable: %w", err)
	}

	// Generate periods using the scheduling algorithm
	periods, err := s.generatePeriods(ctx, timetable, termCourses, config, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate periods: %w", err)
	}

	// Create all periods
	if err := s.periodRepo.CreateBatch(ctx, periods, tx); err != nil {
		return nil, fmt.Errorf("failed to create periods: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionTimetableGenerated,
		actorID,
		actorRole,
		&tenantID,
		domain.AuditResourceTimetable,
		timetable.ID,
		nil,
		timetable,
		ipAddress,
	)

	return &TimetableWithPeriods{
		Timetable: timetable,
		Periods:   periods,
	}, nil
}

// generatePeriods creates a balanced schedule using a constraint-based algorithm
func (s *TimetableService) generatePeriods(
	ctx context.Context,
	timetable *domain.Timetable,
	courses []*domain.Course,
	config *TimetableConfig,
	tx *sql.Tx,
) ([]*domain.Period, error) {
	var periods []*domain.Period
	now := time.Now()

	// Track tutor schedules to avoid conflicts
	tutorSchedule := make(map[uuid.UUID]map[domain.DayOfWeek][]int) // tutor -> day -> period numbers

	// Track course periods per week
	coursePeriodCount := make(map[uuid.UUID]int)

	// Calculate periods per course
	totalPeriods := config.DailyPeriodLimit * len(config.InstructionalDays)
	periodsPerCourse := totalPeriods / len(courses)
	if periodsPerCourse < 1 {
		periodsPerCourse = 1
	}

	// Sort courses by their assigned tutor to group them
	sort.Slice(courses, func(i, j int) bool {
		return courses[i].AssignedTutorID.String() < courses[j].AssignedTutorID.String()
	})

	periodNumber := 1
	for _, day := range config.InstructionalDays {
		periodNumber = 1
		for periodNumber <= config.DailyPeriodLimit {
			for _, course := range courses {
				// Check if we've reached the max periods for this course
				maxPeriods := periodsPerCourse
				if course.MaxPeriodsPerWeek != nil && *course.MaxPeriodsPerWeek > 0 {
					maxPeriods = *course.MaxPeriodsPerWeek
				}
				if coursePeriodCount[course.ID] >= maxPeriods {
					continue
				}

				// Check if tutor is available
				if tutorSchedule[course.AssignedTutorID] == nil {
					tutorSchedule[course.AssignedTutorID] = make(map[domain.DayOfWeek][]int)
				}

				// Check for tutor conflict on this day/period
				tutorPeriods := tutorSchedule[course.AssignedTutorID][day]
				hasConflict := false
				for _, p := range tutorPeriods {
					if p == periodNumber {
						hasConflict = true
						break
					}
				}

				if hasConflict {
					continue
				}

				// Calculate start and end times
				startTime := s.calculatePeriodTime(config.StartTime, periodNumber, config.PeriodDuration, config.BreakAfterPeriod, config.BreakDuration)
				endTime := startTime.Add(time.Duration(config.PeriodDuration) * time.Minute)

				period := &domain.Period{
					ID:           uuid.New(),
					TenantID:     timetable.TenantID,
					TimetableID:  timetable.ID,
					CourseID:     course.ID,
					TutorID:      course.AssignedTutorID,
					ClassID:      timetable.ClassID,
					DayOfWeek:    day,
					StartTime:    startTime,
					EndTime:      endTime,
					PeriodNumber: periodNumber,
					CreatedAt:    now,
					UpdatedAt:    now,
				}

				periods = append(periods, period)
				coursePeriodCount[course.ID]++
				tutorSchedule[course.AssignedTutorID][day] = append(tutorSchedule[course.AssignedTutorID][day], periodNumber)

				periodNumber++
				if periodNumber > config.DailyPeriodLimit {
					break
				}
			}

			if periodNumber <= config.DailyPeriodLimit {
				periodNumber++
			}
		}
	}

	return periods, nil
}

// calculatePeriodTime calculates the start time for a period
func (s *TimetableService) calculatePeriodTime(startOfDay time.Time, periodNumber int, periodDuration int, breakAfterPeriod int, breakDuration int) time.Time {
	offset := (periodNumber - 1) * periodDuration
	if periodNumber > breakAfterPeriod {
		offset += breakDuration
	}
	return startOfDay.Add(time.Duration(offset) * time.Minute)
}

// GetTimetable retrieves a timetable with its periods
func (s *TimetableService) GetTimetable(ctx context.Context, id uuid.UUID) (*TimetableWithPeriods, error) {
	timetable, err := s.timetableRepo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get timetable: %w", err)
	}

	periods, err := s.periodRepo.ListByTimetable(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get periods: %w", err)
	}

	return &TimetableWithPeriods{
		Timetable: timetable,
		Periods:   periods,
	}, nil
}

// GetTimetableByClassAndTerm retrieves the published timetable for a class and term
func (s *TimetableService) GetTimetableByClassAndTerm(ctx context.Context, classID, termID uuid.UUID) (*TimetableWithPeriods, error) {
	timetable, err := s.timetableRepo.GetPublishedTimetable(ctx, classID, termID)
	if err != nil {
		return nil, fmt.Errorf("failed to get timetable: %w", err)
	}

	periods, err := s.periodRepo.ListByTimetable(ctx, timetable.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get periods: %w", err)
	}

	return &TimetableWithPeriods{
		Timetable: timetable,
		Periods:   periods,
	}, nil
}

// ListTimetables lists timetables for a class or term
func (s *TimetableService) ListTimetables(
	ctx context.Context,
	tenantID uuid.UUID,
	classID *uuid.UUID,
	termID *uuid.UUID,
	params repository.PaginationParams,
) ([]*domain.Timetable, *repository.PaginatedResult, error) {
	var timetables []*domain.Timetable
	var total int
	var err error

	if classID != nil {
		timetables, total, err = s.timetableRepo.ListByClass(ctx, *classID, params)
	} else if termID != nil {
		timetables, total, err = s.timetableRepo.ListByTerm(ctx, *termID, params)
	} else {
		timetables, total, err = s.timetableRepo.ListByTenant(ctx, tenantID, params)
	}

	if err != nil {
		return nil, nil, fmt.Errorf("failed to list timetables: %w", err)
	}

	pagination := repository.BuildPaginatedResult(total, params)

	return timetables, &pagination, nil
}

// PublishTimetable publishes a draft timetable
func (s *TimetableService) PublishTimetable(
	ctx context.Context,
	id uuid.UUID,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.Timetable, error) {
	timetable, err := s.timetableRepo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get timetable: %w", err)
	}

	if !timetable.IsDraft() {
		return nil, fmt.Errorf("timetable is not in draft status")
	}

	// Archive any existing published timetable for the same class/term
	existingPublished, err := s.timetableRepo.GetPublishedTimetable(ctx, timetable.ClassID, timetable.TermID)
	if err == nil && existingPublished != nil && existingPublished.ID != id {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer tx.Rollback()

		if err := s.timetableRepo.Archive(ctx, existingPublished.ID, tx); err != nil {
			return nil, fmt.Errorf("failed to archive existing timetable: %w", err)
		}

		if err := s.timetableRepo.Publish(ctx, id, actorID, tx); err != nil {
			return nil, fmt.Errorf("failed to publish timetable: %w", err)
		}

		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("failed to commit transaction: %w", err)
		}
	} else {
		if err := s.timetableRepo.Publish(ctx, id, actorID, nil); err != nil {
			return nil, fmt.Errorf("failed to publish timetable: %w", err)
		}
	}

	// Refresh timetable
	timetable, err = s.timetableRepo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh timetable: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionTimetablePublished,
		actorID,
		actorRole,
		&timetable.TenantID,
		domain.AuditResourceTimetable,
		timetable.ID,
		nil,
		timetable,
		ipAddress,
	)

	return timetable, nil
}

// SwapRequestInput represents input for creating a swap request
type SwapRequestInput struct {
	RequestingPeriodID uuid.UUID `json:"requesting_period_id" validate:"required,uuid"`
	TargetPeriodID     uuid.UUID `json:"target_period_id" validate:"required,uuid"`
	Reason             *string   `json:"reason,omitempty" validate:"omitempty,max=500"`
}

// CreateSwapRequest creates a new period swap request
func (s *TimetableService) CreateSwapRequest(
	ctx context.Context,
	tenantID uuid.UUID,
	req *SwapRequestInput,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.SwapRequest, error) {
	// Get both periods
	requestingPeriod, err := s.periodRepo.Get(ctx, req.RequestingPeriodID)
	if err != nil {
		return nil, fmt.Errorf("failed to get requesting period: %w", err)
	}

	targetPeriod, err := s.periodRepo.Get(ctx, req.TargetPeriodID)
	if err != nil {
		return nil, fmt.Errorf("failed to get target period: %w", err)
	}

	// Verify both periods belong to the same tenant
	if requestingPeriod.TenantID != tenantID || targetPeriod.TenantID != tenantID {
		return nil, fmt.Errorf("periods do not belong to tenant")
	}

	// Verify the requester is the tutor of the requesting period
	if requestingPeriod.TutorID != actorID {
		return nil, fmt.Errorf("you can only request swaps for your own periods")
	}

	// Verify both periods are on the same day (as per BRD)
	if requestingPeriod.DayOfWeek != targetPeriod.DayOfWeek {
		return nil, fmt.Errorf("periods must be on the same day for swapping")
	}

	// Verify both periods belong to the same class
	if requestingPeriod.ClassID != targetPeriod.ClassID {
		return nil, fmt.Errorf("periods must be in the same class for swapping")
	}

	// Check for existing pending request
	hasPending, err := s.swapRequestRepo.HasPendingRequest(ctx, req.RequestingPeriodID, req.TargetPeriodID)
	if err != nil {
		return nil, fmt.Errorf("failed to check pending requests: %w", err)
	}
	if hasPending {
		return nil, fmt.Errorf("a pending swap request already exists for these periods")
	}

	now := time.Now()
	swapRequest := &domain.SwapRequest{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		RequestingPeriodID: req.RequestingPeriodID,
		TargetPeriodID:     req.TargetPeriodID,
		RequestingTutorID:  requestingPeriod.TutorID,
		TargetTutorID:      targetPeriod.TutorID,
		Status:             domain.SwapRequestStatusPending,
		Reason:             req.Reason,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if err := s.swapRequestRepo.Create(ctx, swapRequest, nil); err != nil {
		return nil, fmt.Errorf("failed to create swap request: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionPeriodSwapRequested,
		actorID,
		actorRole,
		&tenantID,
		domain.AuditResourcePeriod,
		swapRequest.ID,
		nil,
		swapRequest,
		ipAddress,
	)

	return swapRequest, nil
}

// ApproveSwapRequest approves a swap request (by target tutor)
func (s *TimetableService) ApproveSwapRequest(
	ctx context.Context,
	requestID uuid.UUID,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.SwapRequest, error) {
	swapRequest, err := s.swapRequestRepo.Get(ctx, requestID)
	if err != nil {
		return nil, fmt.Errorf("failed to get swap request: %w", err)
	}

	// Verify the approver is the target tutor
	if swapRequest.TargetTutorID != actorID {
		return nil, fmt.Errorf("only the target tutor can approve this request")
	}

	if !swapRequest.IsPending() {
		return nil, fmt.Errorf("swap request is not pending")
	}

	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Perform the swap
	if err := s.periodRepo.SwapPeriods(ctx, swapRequest.RequestingPeriodID, swapRequest.TargetPeriodID, tx); err != nil {
		return nil, fmt.Errorf("failed to swap periods: %w", err)
	}

	// Update swap request status
	swapRequest.Approve()
	if err := s.swapRequestRepo.Update(ctx, swapRequest, tx); err != nil {
		return nil, fmt.Errorf("failed to update swap request: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionPeriodSwapApproved,
		actorID,
		actorRole,
		&swapRequest.TenantID,
		domain.AuditResourcePeriod,
		swapRequest.ID,
		nil,
		swapRequest,
		ipAddress,
	)

	return swapRequest, nil
}

// RejectSwapRequest rejects a swap request (by target tutor)
func (s *TimetableService) RejectSwapRequest(
	ctx context.Context,
	requestID uuid.UUID,
	reason string,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.SwapRequest, error) {
	swapRequest, err := s.swapRequestRepo.Get(ctx, requestID)
	if err != nil {
		return nil, fmt.Errorf("failed to get swap request: %w", err)
	}

	// Verify the rejector is the target tutor
	if swapRequest.TargetTutorID != actorID {
		return nil, fmt.Errorf("only the target tutor can reject this request")
	}

	if !swapRequest.IsPending() {
		return nil, fmt.Errorf("swap request is not pending")
	}

	swapRequest.Reject(reason)
	if err := s.swapRequestRepo.Update(ctx, swapRequest, nil); err != nil {
		return nil, fmt.Errorf("failed to update swap request: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionPeriodSwapRejected,
		actorID,
		actorRole,
		&swapRequest.TenantID,
		domain.AuditResourcePeriod,
		swapRequest.ID,
		nil,
		swapRequest,
		ipAddress,
	)

	return swapRequest, nil
}

// EscalateSwapRequest escalates a rejected swap request to admin
func (s *TimetableService) EscalateSwapRequest(
	ctx context.Context,
	requestID uuid.UUID,
	reason string,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.SwapRequest, error) {
	swapRequest, err := s.swapRequestRepo.Get(ctx, requestID)
	if err != nil {
		return nil, fmt.Errorf("failed to get swap request: %w", err)
	}

	// Verify the escalator is the requesting tutor
	if swapRequest.RequestingTutorID != actorID {
		return nil, fmt.Errorf("only the requesting tutor can escalate this request")
	}

	if !swapRequest.IsRejected() {
		return nil, fmt.Errorf("only rejected requests can be escalated")
	}

	swapRequest.Escalate(reason)
	if err := s.swapRequestRepo.Update(ctx, swapRequest, nil); err != nil {
		return nil, fmt.Errorf("failed to update swap request: %w", err)
	}

	return swapRequest, nil
}

// AdminOverrideSwapRequest allows admin to force approve an escalated swap request
func (s *TimetableService) AdminOverrideSwapRequest(
	ctx context.Context,
	requestID uuid.UUID,
	reason string,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.SwapRequest, error) {
	if actorRole != domain.RoleAdmin && actorRole != domain.RoleSuperAdmin {
		return nil, fmt.Errorf("only admins can override swap requests")
	}

	swapRequest, err := s.swapRequestRepo.Get(ctx, requestID)
	if err != nil {
		return nil, fmt.Errorf("failed to get swap request: %w", err)
	}

	if !swapRequest.IsEscalated() {
		return nil, fmt.Errorf("only escalated requests can be overridden")
	}

	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Perform the swap
	if err := s.periodRepo.SwapPeriods(ctx, swapRequest.RequestingPeriodID, swapRequest.TargetPeriodID, tx); err != nil {
		return nil, fmt.Errorf("failed to swap periods: %w", err)
	}

	// Update swap request status with admin override
	swapRequest.AdminOverride(actorID, reason)
	if err := s.swapRequestRepo.Update(ctx, swapRequest, tx); err != nil {
		return nil, fmt.Errorf("failed to update swap request: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionPeriodSwapApproved,
		actorID,
		actorRole,
		&swapRequest.TenantID,
		domain.AuditResourcePeriod,
		swapRequest.ID,
		nil,
		swapRequest,
		ipAddress,
	)

	return swapRequest, nil
}

// CancelSwapRequest cancels a pending swap request (by requesting tutor)
func (s *TimetableService) CancelSwapRequest(
	ctx context.Context,
	requestID uuid.UUID,
	actorID uuid.UUID,
) (*domain.SwapRequest, error) {
	swapRequest, err := s.swapRequestRepo.Get(ctx, requestID)
	if err != nil {
		return nil, fmt.Errorf("failed to get swap request: %w", err)
	}

	// Verify the canceler is the requesting tutor
	if swapRequest.RequestingTutorID != actorID {
		return nil, fmt.Errorf("only the requesting tutor can cancel this request")
	}

	if !swapRequest.IsPending() && !swapRequest.IsEscalated() {
		return nil, fmt.Errorf("swap request cannot be cancelled")
	}

	swapRequest.Cancel()
	if err := s.swapRequestRepo.Update(ctx, swapRequest, nil); err != nil {
		return nil, fmt.Errorf("failed to update swap request: %w", err)
	}

	return swapRequest, nil
}

// GetSwapRequest retrieves a swap request by ID
func (s *TimetableService) GetSwapRequest(ctx context.Context, id uuid.UUID) (*domain.SwapRequest, error) {
	return s.swapRequestRepo.Get(ctx, id)
}

// ListSwapRequests lists swap requests with filters
func (s *TimetableService) ListSwapRequests(
	ctx context.Context,
	tenantID uuid.UUID,
	tutorID *uuid.UUID,
	status *domain.SwapRequestStatus,
	params repository.PaginationParams,
) ([]*domain.SwapRequest, *repository.PaginatedResult, error) {
	var requests []*domain.SwapRequest
	var total int
	var err error

	if tutorID != nil {
		requests, total, err = s.swapRequestRepo.ListByTutor(ctx, *tutorID, status, params)
	} else {
		requests, total, err = s.swapRequestRepo.ListByTenant(ctx, tenantID, status, params)
	}

	if err != nil {
		return nil, nil, fmt.Errorf("failed to list swap requests: %w", err)
	}

	pagination := repository.BuildPaginatedResult(total, params)

	return requests, &pagination, nil
}

// ListPendingSwapRequestsForTutor lists pending swap requests where the tutor is the target
func (s *TimetableService) ListPendingSwapRequestsForTutor(
	ctx context.Context,
	tutorID uuid.UUID,
	params repository.PaginationParams,
) ([]*domain.SwapRequest, *repository.PaginatedResult, error) {
	requests, total, err := s.swapRequestRepo.ListPendingForTutor(ctx, tutorID, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list pending swap requests: %w", err)
	}

	pagination := repository.BuildPaginatedResult(total, params)

	return requests, &pagination, nil
}

// ListEscalatedSwapRequests lists escalated swap requests for admin review
func (s *TimetableService) ListEscalatedSwapRequests(
	ctx context.Context,
	tenantID uuid.UUID,
	params repository.PaginationParams,
) ([]*domain.SwapRequest, *repository.PaginatedResult, error) {
	requests, total, err := s.swapRequestRepo.ListEscalated(ctx, tenantID, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list escalated swap requests: %w", err)
	}

	pagination := repository.BuildPaginatedResult(total, params)

	return requests, &pagination, nil
}

// GetTutorTimetable retrieves the timetable filtered for a specific tutor
func (s *TimetableService) GetTutorTimetable(
	ctx context.Context,
	tutorID uuid.UUID,
	termID uuid.UUID,
) ([]*domain.Period, error) {
	return s.periodRepo.ListByTutor(ctx, tutorID, termID)
}

// RegenerateTimetable regenerates a timetable for a class (ADMIN only)
func (s *TimetableService) RegenerateTimetable(
	ctx context.Context,
	tenantID uuid.UUID,
	classID uuid.UUID,
	termID uuid.UUID,
	notes *string,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*TimetableWithPeriods, error) {
	// Simply call GenerateTimetable which handles archiving the old one
	return s.GenerateTimetable(ctx, tenantID, &GenerateTimetableRequest{
		ClassID: classID,
		TermID:  termID,
		Notes:   notes,
	}, actorID, actorRole, ipAddress)
}
