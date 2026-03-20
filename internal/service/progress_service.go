package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"arc-lms/internal/domain"
	"arc-lms/internal/repository"
	"arc-lms/internal/repository/postgres"

	"github.com/google/uuid"
)

// ProgressService handles progress and reporting operations
type ProgressService struct {
	progressRepo   *postgres.ProgressRepository
	enrollmentRepo *postgres.EnrollmentRepository
	courseRepo     *postgres.CourseRepository
	classRepo      *postgres.ClassRepository
	termRepo       *postgres.TermRepository
	quizRepo       *postgres.QuizRepository
	assignmentRepo *postgres.AssignmentRepository
	examRepo       *postgres.ExaminationRepository
	auditService   *AuditService
}

// NewProgressService creates a new progress service
func NewProgressService(
	progressRepo *postgres.ProgressRepository,
	enrollmentRepo *postgres.EnrollmentRepository,
	courseRepo *postgres.CourseRepository,
	classRepo *postgres.ClassRepository,
	termRepo *postgres.TermRepository,
	quizRepo *postgres.QuizRepository,
	assignmentRepo *postgres.AssignmentRepository,
	examRepo *postgres.ExaminationRepository,
	auditService *AuditService,
) *ProgressService {
	return &ProgressService{
		progressRepo:   progressRepo,
		enrollmentRepo: enrollmentRepo,
		courseRepo:     courseRepo,
		classRepo:      classRepo,
		termRepo:       termRepo,
		quizRepo:       quizRepo,
		assignmentRepo: assignmentRepo,
		examRepo:       examRepo,
		auditService:   auditService,
	}
}

// GradeWeighting represents the weighting configuration for grades
type GradeWeighting struct {
	ContinuousAssessmentWeight int `json:"continuous_assessment_weight"` // e.g., 40
	ExaminationWeight          int `json:"examination_weight"`           // e.g., 60
}

// DefaultGradeWeighting returns the default grade weighting (40% CA, 60% Exam)
func DefaultGradeWeighting() GradeWeighting {
	return GradeWeighting{
		ContinuousAssessmentWeight: 40,
		ExaminationWeight:          60,
	}
}

// GetProgress retrieves a progress record by ID
func (s *ProgressService) GetProgress(ctx context.Context, id uuid.UUID) (*domain.Progress, error) {
	return s.progressRepo.Get(ctx, id)
}

// GetStudentProgress retrieves a student's progress for a specific course and term
func (s *ProgressService) GetStudentProgress(ctx context.Context, studentID, courseID, termID uuid.UUID) (*domain.Progress, error) {
	return s.progressRepo.GetByStudentCourseAndTerm(ctx, studentID, courseID, termID)
}

// GetOrCreateProgress retrieves or creates a progress record for a student in a course
func (s *ProgressService) GetOrCreateProgress(
	ctx context.Context,
	tenantID, studentID, courseID, termID, classID uuid.UUID,
) (*domain.Progress, error) {
	// Try to get existing progress
	progress, err := s.progressRepo.GetByStudentCourseAndTerm(ctx, studentID, courseID, termID)
	if err == nil {
		return progress, nil
	}

	// Create new progress record
	now := time.Now()
	progress = &domain.Progress{
		ID:               uuid.New(),
		TenantID:         tenantID,
		StudentID:        studentID,
		CourseID:         courseID,
		TermID:           termID,
		ClassID:          classID,
		Status:           domain.ProgressStatusInProgress,
		QuizScores:       make([]int, 0),
		AssignmentScores: make([]int, 0),
		Attendance: domain.AttendanceRecord{
			TotalPeriods:    0,
			PeriodsAttended: 0,
			PeriodsAbsent:   0,
			Percentage:      0,
		},
		IsFlagged: false,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.progressRepo.Create(ctx, progress, nil); err != nil {
		return nil, fmt.Errorf("failed to create progress record: %w", err)
	}

	return progress, nil
}

// RecordQuizScoreRequest represents a request to record a quiz score
type RecordQuizScoreRequest struct {
	StudentID uuid.UUID `json:"student_id" validate:"required,uuid"`
	CourseID  uuid.UUID `json:"course_id" validate:"required,uuid"`
	TermID    uuid.UUID `json:"term_id" validate:"required,uuid"`
	Score     int       `json:"score" validate:"required,min=0"`
}

// RecordQuizScore records a quiz score for a student
func (s *ProgressService) RecordQuizScore(
	ctx context.Context,
	tenantID uuid.UUID,
	req *RecordQuizScoreRequest,
	classID uuid.UUID,
) (*domain.Progress, error) {
	progress, err := s.GetOrCreateProgress(ctx, tenantID, req.StudentID, req.CourseID, req.TermID, classID)
	if err != nil {
		return nil, err
	}

	progress.QuizScores = append(progress.QuizScores, req.Score)
	progress.UpdatedAt = time.Now()

	if err := s.progressRepo.Update(ctx, progress, nil); err != nil {
		return nil, fmt.Errorf("failed to update progress: %w", err)
	}

	return progress, nil
}

// RecordAssignmentScoreRequest represents a request to record an assignment score
type RecordAssignmentScoreRequest struct {
	StudentID uuid.UUID `json:"student_id" validate:"required,uuid"`
	CourseID  uuid.UUID `json:"course_id" validate:"required,uuid"`
	TermID    uuid.UUID `json:"term_id" validate:"required,uuid"`
	Score     int       `json:"score" validate:"required,min=0"`
}

// RecordAssignmentScore records an assignment score for a student
func (s *ProgressService) RecordAssignmentScore(
	ctx context.Context,
	tenantID uuid.UUID,
	req *RecordAssignmentScoreRequest,
	classID uuid.UUID,
) (*domain.Progress, error) {
	progress, err := s.GetOrCreateProgress(ctx, tenantID, req.StudentID, req.CourseID, req.TermID, classID)
	if err != nil {
		return nil, err
	}

	progress.AssignmentScores = append(progress.AssignmentScores, req.Score)
	progress.UpdatedAt = time.Now()

	if err := s.progressRepo.Update(ctx, progress, nil); err != nil {
		return nil, fmt.Errorf("failed to update progress: %w", err)
	}

	return progress, nil
}

// RecordExaminationScoreRequest represents a request to record an examination score
type RecordExaminationScoreRequest struct {
	StudentID uuid.UUID `json:"student_id" validate:"required,uuid"`
	CourseID  uuid.UUID `json:"course_id" validate:"required,uuid"`
	TermID    uuid.UUID `json:"term_id" validate:"required,uuid"`
	Score     int       `json:"score" validate:"required,min=0"`
}

// RecordExaminationScore records an examination score for a student
func (s *ProgressService) RecordExaminationScore(
	ctx context.Context,
	tenantID uuid.UUID,
	req *RecordExaminationScoreRequest,
	classID uuid.UUID,
) (*domain.Progress, error) {
	progress, err := s.GetOrCreateProgress(ctx, tenantID, req.StudentID, req.CourseID, req.TermID, classID)
	if err != nil {
		return nil, err
	}

	progress.ExaminationScore = &req.Score
	progress.UpdatedAt = time.Now()

	if err := s.progressRepo.Update(ctx, progress, nil); err != nil {
		return nil, fmt.Errorf("failed to update progress: %w", err)
	}

	return progress, nil
}

// ComputeGrade computes the grade for a progress record
func (s *ProgressService) ComputeGrade(
	ctx context.Context,
	progressID uuid.UUID,
	weighting GradeWeighting,
) (*domain.Progress, error) {
	progress, err := s.progressRepo.Get(ctx, progressID)
	if err != nil {
		return nil, fmt.Errorf("failed to get progress: %w", err)
	}

	progress.ComputeGrade(weighting.ContinuousAssessmentWeight, weighting.ExaminationWeight)
	progress.UpdatedAt = time.Now()

	// Check if grade is below threshold and flag if needed
	if progress.Grade != nil && progress.Grade.Percentage < 40 {
		progress.FlagForLowPerformance("Overall grade below 40%")
	}

	if err := s.progressRepo.Update(ctx, progress, nil); err != nil {
		return nil, fmt.Errorf("failed to update progress: %w", err)
	}

	return progress, nil
}

// ComputeAllGradesForCourse computes grades for all students in a course
func (s *ProgressService) ComputeAllGradesForCourse(
	ctx context.Context,
	courseID, termID uuid.UUID,
	weighting GradeWeighting,
) ([]*domain.Progress, error) {
	progresses, err := s.progressRepo.ListByCourseAndTerm(ctx, courseID, termID)
	if err != nil {
		return nil, fmt.Errorf("failed to list progress records: %w", err)
	}

	for _, progress := range progresses {
		progress.ComputeGrade(weighting.ContinuousAssessmentWeight, weighting.ExaminationWeight)
		progress.UpdatedAt = time.Now()

		if progress.Grade != nil && progress.Grade.Percentage < 40 {
			progress.FlagForLowPerformance("Overall grade below 40%")
		}

		if err := s.progressRepo.Update(ctx, progress, nil); err != nil {
			return nil, fmt.Errorf("failed to update progress for student %s: %w", progress.StudentID, err)
		}
	}

	return progresses, nil
}

// ComputeClassPositions computes class positions for all students in a class for a term
func (s *ProgressService) ComputeClassPositions(
	ctx context.Context,
	classID, termID uuid.UUID,
) error {
	// Get all progress records for the class
	progresses, err := s.progressRepo.ListByClassAndTerm(ctx, classID, termID)
	if err != nil {
		return fmt.Errorf("failed to list progress records: %w", err)
	}

	// Group by student and compute average
	studentAverages := make(map[uuid.UUID]float64)
	studentProgresses := make(map[uuid.UUID][]*domain.Progress)

	for _, p := range progresses {
		studentProgresses[p.StudentID] = append(studentProgresses[p.StudentID], p)
		if p.Grade != nil {
			studentAverages[p.StudentID] += p.Grade.Percentage
		}
	}

	// Compute average for each student
	for studentID, progs := range studentProgresses {
		if len(progs) > 0 {
			studentAverages[studentID] /= float64(len(progs))
		}
	}

	// Sort students by average (descending)
	type studentRank struct {
		StudentID uuid.UUID
		Average   float64
	}
	rankings := make([]studentRank, 0, len(studentAverages))
	for studentID, avg := range studentAverages {
		rankings = append(rankings, studentRank{StudentID: studentID, Average: avg})
	}
	sort.Slice(rankings, func(i, j int) bool {
		return rankings[i].Average > rankings[j].Average
	})

	// Assign positions
	studentPositions := make(map[uuid.UUID]int)
	for i, rank := range rankings {
		studentPositions[rank.StudentID] = i + 1
	}

	// Update progress records with positions
	for _, p := range progresses {
		if pos, ok := studentPositions[p.StudentID]; ok {
			p.ClassPosition = &pos
			p.UpdatedAt = time.Now()
			if err := s.progressRepo.Update(ctx, p, nil); err != nil {
				return fmt.Errorf("failed to update progress position: %w", err)
			}
		}
	}

	return nil
}

// MarkAttendanceRequest represents a request to mark attendance
type MarkAttendanceRequest struct {
	PeriodID  uuid.UUID   `json:"period_id" validate:"required,uuid"`
	Date      time.Time   `json:"date" validate:"required"`
	Present   []uuid.UUID `json:"present" validate:"required"`   // Student IDs who are present
	Absent    []uuid.UUID `json:"absent" validate:"required"`    // Student IDs who are absent
}

// MarkAttendance marks attendance for a period
func (s *ProgressService) MarkAttendance(
	ctx context.Context,
	tenantID, courseID, termID, classID uuid.UUID,
	req *MarkAttendanceRequest,
	markedBy uuid.UUID,
) error {
	now := time.Now()

	// Mark present students
	for _, studentID := range req.Present {
		progress, err := s.GetOrCreateProgress(ctx, tenantID, studentID, courseID, termID, classID)
		if err != nil {
			return fmt.Errorf("failed to get/create progress for student %s: %w", studentID, err)
		}

		entry := &postgres.AttendanceEntry{
			ID:         uuid.New(),
			TenantID:   tenantID,
			ProgressID: progress.ID,
			PeriodID:   req.PeriodID,
			Date:       req.Date,
			IsPresent:  true,
			MarkedBy:   markedBy,
			MarkedAt:   now,
		}

		if err := s.progressRepo.CreateAttendanceEntry(ctx, entry, nil); err != nil {
			return fmt.Errorf("failed to create attendance entry: %w", err)
		}
	}

	// Mark absent students
	for _, studentID := range req.Absent {
		progress, err := s.GetOrCreateProgress(ctx, tenantID, studentID, courseID, termID, classID)
		if err != nil {
			return fmt.Errorf("failed to get/create progress for student %s: %w", studentID, err)
		}

		entry := &postgres.AttendanceEntry{
			ID:         uuid.New(),
			TenantID:   tenantID,
			ProgressID: progress.ID,
			PeriodID:   req.PeriodID,
			Date:       req.Date,
			IsPresent:  false,
			MarkedBy:   markedBy,
			MarkedAt:   now,
		}

		if err := s.progressRepo.CreateAttendanceEntry(ctx, entry, nil); err != nil {
			return fmt.Errorf("failed to create attendance entry: %w", err)
		}
	}

	return nil
}

// UpdateAttendanceSummary updates the attendance summary for a progress record
func (s *ProgressService) UpdateAttendanceSummary(ctx context.Context, progressID uuid.UUID) (*domain.Progress, error) {
	progress, err := s.progressRepo.Get(ctx, progressID)
	if err != nil {
		return nil, fmt.Errorf("failed to get progress: %w", err)
	}

	attendance, err := s.progressRepo.ComputeAttendanceSummary(ctx, progressID)
	if err != nil {
		return nil, fmt.Errorf("failed to compute attendance summary: %w", err)
	}

	progress.Attendance = *attendance
	progress.UpdatedAt = time.Now()

	// Flag if attendance is below threshold (e.g., 75%)
	if attendance.Percentage < 75 && attendance.TotalPeriods >= 5 {
		reason := fmt.Sprintf("Attendance below 75%% (%.1f%%)", attendance.Percentage)
		progress.FlagForLowPerformance(reason)
	}

	if err := s.progressRepo.Update(ctx, progress, nil); err != nil {
		return nil, fmt.Errorf("failed to update progress: %w", err)
	}

	return progress, nil
}

// AddTutorRemarksRequest represents a request to add tutor remarks
type AddTutorRemarksRequest struct {
	Remarks string `json:"remarks" validate:"required,min=5,max=1000"`
}

// AddTutorRemarks adds tutor remarks to a progress record
func (s *ProgressService) AddTutorRemarks(
	ctx context.Context,
	progressID uuid.UUID,
	req *AddTutorRemarksRequest,
) (*domain.Progress, error) {
	progress, err := s.progressRepo.Get(ctx, progressID)
	if err != nil {
		return nil, fmt.Errorf("failed to get progress: %w", err)
	}

	progress.TutorRemarks = &req.Remarks
	progress.UpdatedAt = time.Now()

	if err := s.progressRepo.Update(ctx, progress, nil); err != nil {
		return nil, fmt.Errorf("failed to update progress: %w", err)
	}

	return progress, nil
}

// AddPrincipalRemarksRequest represents a request to add principal remarks
type AddPrincipalRemarksRequest struct {
	Remarks string `json:"remarks" validate:"required,min=5,max=1000"`
}

// AddPrincipalRemarks adds principal remarks to a progress record
func (s *ProgressService) AddPrincipalRemarks(
	ctx context.Context,
	progressID uuid.UUID,
	req *AddPrincipalRemarksRequest,
) (*domain.Progress, error) {
	progress, err := s.progressRepo.Get(ctx, progressID)
	if err != nil {
		return nil, fmt.Errorf("failed to get progress: %w", err)
	}

	progress.PrincipalRemarks = &req.Remarks
	progress.UpdatedAt = time.Now()

	if err := s.progressRepo.Update(ctx, progress, nil); err != nil {
		return nil, fmt.Errorf("failed to update progress: %w", err)
	}

	return progress, nil
}

// ListStudentProgress lists all progress records for a student
func (s *ProgressService) ListStudentProgress(
	ctx context.Context,
	studentID uuid.UUID,
	params repository.PaginationParams,
) ([]*domain.Progress, *repository.PaginatedResult, error) {
	progresses, err := s.progressRepo.ListByStudent(ctx, studentID, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list progress: %w", err)
	}

	ids := make([]uuid.UUID, len(progresses))
	for i, p := range progresses {
		ids[i] = p.ID
	}
	pagination := repository.BuildPaginatedResult(ids, params.Limit)

	return progresses, &pagination, nil
}

// ListCourseProgress lists all progress records for a course
func (s *ProgressService) ListCourseProgress(
	ctx context.Context,
	courseID uuid.UUID,
	params repository.PaginationParams,
) ([]*domain.Progress, *repository.PaginatedResult, error) {
	progresses, err := s.progressRepo.ListByCourse(ctx, courseID, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list progress: %w", err)
	}

	ids := make([]uuid.UUID, len(progresses))
	for i, p := range progresses {
		ids[i] = p.ID
	}
	pagination := repository.BuildPaginatedResult(ids, params.Limit)

	return progresses, &pagination, nil
}

// ListClassProgress lists all progress records for a class
func (s *ProgressService) ListClassProgress(
	ctx context.Context,
	classID uuid.UUID,
	params repository.PaginationParams,
) ([]*domain.Progress, *repository.PaginatedResult, error) {
	progresses, err := s.progressRepo.ListByClass(ctx, classID, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list progress: %w", err)
	}

	ids := make([]uuid.UUID, len(progresses))
	for i, p := range progresses {
		ids[i] = p.ID
	}
	pagination := repository.BuildPaginatedResult(ids, params.Limit)

	return progresses, &pagination, nil
}

// ListFlaggedStudents lists all flagged progress records for a tenant
func (s *ProgressService) ListFlaggedStudents(
	ctx context.Context,
	tenantID uuid.UUID,
	params repository.PaginationParams,
) ([]*domain.Progress, *repository.PaginatedResult, error) {
	progresses, err := s.progressRepo.ListFlaggedByTenant(ctx, tenantID, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list flagged students: %w", err)
	}

	ids := make([]uuid.UUID, len(progresses))
	for i, p := range progresses {
		ids[i] = p.ID
	}
	pagination := repository.BuildPaginatedResult(ids, params.Limit)

	return progresses, &pagination, nil
}

// ===================== Report Card Operations =====================

// GenerateReportCardRequest represents a request to generate a report card
type GenerateReportCardRequest struct {
	StudentID        uuid.UUID  `json:"student_id" validate:"required,uuid"`
	TermID           uuid.UUID  `json:"term_id" validate:"required,uuid"`
	PrincipalRemarks *string    `json:"principal_remarks,omitempty" validate:"omitempty,max=1000"`
	NextTermBegins   *time.Time `json:"next_term_begins,omitempty"`
}

// GenerateReportCard generates a report card for a student for a term
func (s *ProgressService) GenerateReportCard(
	ctx context.Context,
	tenantID uuid.UUID,
	req *GenerateReportCardRequest,
	generatedBy uuid.UUID,
	weighting GradeWeighting,
	ipAddress string,
) (*domain.ReportCard, error) {
	// Get all progress records for the student in the term
	progresses, err := s.progressRepo.ListByStudentAndTerm(ctx, req.StudentID, req.TermID)
	if err != nil {
		return nil, fmt.Errorf("failed to list progress records: %w", err)
	}

	if len(progresses) == 0 {
		return nil, fmt.Errorf("no progress records found for student in this term")
	}

	// Compute grades for all progress records
	for _, p := range progresses {
		p.ComputeGrade(weighting.ContinuousAssessmentWeight, weighting.ExaminationWeight)
		p.UpdatedAt = time.Now()
		if err := s.progressRepo.Update(ctx, p, nil); err != nil {
			return nil, fmt.Errorf("failed to update progress: %w", err)
		}
	}

	// Get class info from first progress record
	classID := progresses[0].ClassID

	// Compute class position for this student
	// First, compute positions for all students in the class
	if err := s.ComputeClassPositions(ctx, classID, req.TermID); err != nil {
		return nil, fmt.Errorf("failed to compute class positions: %w", err)
	}

	// Refresh progress records to get updated positions
	progresses, err = s.progressRepo.ListByStudentAndTerm(ctx, req.StudentID, req.TermID)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh progress records: %w", err)
	}

	// Get total students in class
	allClassProgress, err := s.progressRepo.ListByClassAndTerm(ctx, classID, req.TermID)
	if err != nil {
		return nil, fmt.Errorf("failed to get class progress: %w", err)
	}

	// Count unique students
	uniqueStudents := make(map[uuid.UUID]bool)
	for _, p := range allClassProgress {
		uniqueStudents[p.StudentID] = true
	}
	totalStudents := len(uniqueStudents)

	// Determine class position (average position across courses)
	totalPosition := 0
	for _, p := range progresses {
		if p.ClassPosition != nil {
			totalPosition += *p.ClassPosition
		}
	}
	avgPosition := 1
	if len(progresses) > 0 {
		avgPosition = totalPosition / len(progresses)
	}

	// Create report card
	now := time.Now()
	reportCard := &domain.ReportCard{
		ID:               uuid.New(),
		TenantID:         tenantID,
		StudentID:        req.StudentID,
		TermID:           req.TermID,
		ClassID:          classID,
		CourseProgresses: make([]domain.Progress, len(progresses)),
		ClassPosition:    avgPosition,
		TotalStudents:    totalStudents,
		PrincipalRemarks: req.PrincipalRemarks,
		NextTermBegins:   req.NextTermBegins,
		GeneratedAt:      now,
		GeneratedBy:      generatedBy,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	// Copy progress records
	for i, p := range progresses {
		reportCard.CourseProgresses[i] = *p
	}

	// Compute overall grade
	reportCard.ComputeOverallGrade()

	// Save report card
	if err := s.progressRepo.CreateReportCard(ctx, reportCard, nil); err != nil {
		return nil, fmt.Errorf("failed to create report card: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionReportCardGenerated,
		generatedBy,
		domain.RoleAdmin,
		&tenantID,
		domain.AuditResourceProgress,
		reportCard.ID,
		nil,
		reportCard,
		ipAddress,
	)

	return reportCard, nil
}

// GetReportCard retrieves a report card by ID
func (s *ProgressService) GetReportCard(ctx context.Context, id uuid.UUID) (*domain.ReportCard, error) {
	return s.progressRepo.GetReportCard(ctx, id)
}

// GetStudentReportCard retrieves a student's report card for a term
func (s *ProgressService) GetStudentReportCard(ctx context.Context, studentID, termID uuid.UUID) (*domain.ReportCard, error) {
	return s.progressRepo.GetReportCardByStudentAndTerm(ctx, studentID, termID)
}

// ListStudentReportCards lists all report cards for a student
func (s *ProgressService) ListStudentReportCards(ctx context.Context, studentID uuid.UUID) ([]*domain.ReportCard, error) {
	return s.progressRepo.ListReportCardsByStudent(ctx, studentID)
}

// UpdateReportCardRemarksRequest represents a request to update report card remarks
type UpdateReportCardRemarksRequest struct {
	PrincipalRemarks *string    `json:"principal_remarks,omitempty" validate:"omitempty,max=1000"`
	NextTermBegins   *time.Time `json:"next_term_begins,omitempty"`
}

// UpdateReportCardRemarks updates the remarks on a report card
func (s *ProgressService) UpdateReportCardRemarks(
	ctx context.Context,
	reportCardID uuid.UUID,
	req *UpdateReportCardRemarksRequest,
) (*domain.ReportCard, error) {
	reportCard, err := s.progressRepo.GetReportCard(ctx, reportCardID)
	if err != nil {
		return nil, fmt.Errorf("failed to get report card: %w", err)
	}

	if req.PrincipalRemarks != nil {
		reportCard.PrincipalRemarks = req.PrincipalRemarks
	}
	if req.NextTermBegins != nil {
		reportCard.NextTermBegins = req.NextTermBegins
	}
	reportCard.UpdatedAt = time.Now()

	if err := s.progressRepo.UpdateReportCard(ctx, reportCard, nil); err != nil {
		return nil, fmt.Errorf("failed to update report card: %w", err)
	}

	return reportCard, nil
}

// GenerateClassReportCards generates report cards for all students in a class
func (s *ProgressService) GenerateClassReportCards(
	ctx context.Context,
	tenantID, classID, termID uuid.UUID,
	principalRemarks *string,
	nextTermBegins *time.Time,
	generatedBy uuid.UUID,
	weighting GradeWeighting,
	ipAddress string,
) ([]*domain.ReportCard, error) {
	// Get all progress records for the class
	progresses, err := s.progressRepo.ListByClassAndTerm(ctx, classID, termID)
	if err != nil {
		return nil, fmt.Errorf("failed to list class progress: %w", err)
	}

	// Get unique student IDs
	studentIDs := make(map[uuid.UUID]bool)
	for _, p := range progresses {
		studentIDs[p.StudentID] = true
	}

	// Generate report card for each student
	reportCards := make([]*domain.ReportCard, 0, len(studentIDs))
	for studentID := range studentIDs {
		req := &GenerateReportCardRequest{
			StudentID:        studentID,
			TermID:           termID,
			PrincipalRemarks: principalRemarks,
			NextTermBegins:   nextTermBegins,
		}

		reportCard, err := s.GenerateReportCard(ctx, tenantID, req, generatedBy, weighting, ipAddress)
		if err != nil {
			// Log error but continue with other students
			continue
		}

		reportCards = append(reportCards, reportCard)
	}

	return reportCards, nil
}

// ===================== Statistics =====================

// GetCourseStatistics retrieves statistics for a course
func (s *ProgressService) GetCourseStatistics(ctx context.Context, courseID, termID uuid.UUID) (*postgres.CourseStatistics, error) {
	return s.progressRepo.GetCourseStatistics(ctx, courseID, termID)
}

// GetClassStatistics retrieves statistics for a class
func (s *ProgressService) GetClassStatistics(ctx context.Context, classID, termID uuid.UUID) (*postgres.ClassStatistics, error) {
	return s.progressRepo.GetClassStatistics(ctx, classID, termID)
}

// CompleteProgress marks a progress record as completed
func (s *ProgressService) CompleteProgress(ctx context.Context, progressID uuid.UUID) (*domain.Progress, error) {
	progress, err := s.progressRepo.Get(ctx, progressID)
	if err != nil {
		return nil, fmt.Errorf("failed to get progress: %w", err)
	}

	progress.Complete()

	if err := s.progressRepo.Update(ctx, progress, nil); err != nil {
		return nil, fmt.Errorf("failed to update progress: %w", err)
	}

	return progress, nil
}

// UnflagProgress removes the flag from a progress record
func (s *ProgressService) UnflagProgress(ctx context.Context, progressID uuid.UUID) (*domain.Progress, error) {
	progress, err := s.progressRepo.Get(ctx, progressID)
	if err != nil {
		return nil, fmt.Errorf("failed to get progress: %w", err)
	}

	progress.IsFlagged = false
	progress.FlagReason = nil
	progress.Status = domain.ProgressStatusInProgress
	progress.UpdatedAt = time.Now()

	if err := s.progressRepo.Update(ctx, progress, nil); err != nil {
		return nil, fmt.Errorf("failed to update progress: %w", err)
	}

	return progress, nil
}
