package service

import (
	"context"
	"fmt"
	"time"

	"arc-lms/internal/domain"
	"arc-lms/internal/repository"
	"arc-lms/internal/repository/postgres"

	"github.com/google/uuid"
)

// CourseService handles course management operations
type CourseService struct {
	courseRepo        *postgres.CourseRepository
	courseContentRepo *postgres.CourseContentRepository
	classRepo         *postgres.ClassRepository
	userRepo          *postgres.UserRepository
	auditService      *AuditService
}

// NewCourseService creates a new course service
func NewCourseService(
	courseRepo *postgres.CourseRepository,
	courseContentRepo *postgres.CourseContentRepository,
	classRepo *postgres.ClassRepository,
	userRepo *postgres.UserRepository,
	auditService *AuditService,
) *CourseService {
	return &CourseService{
		courseRepo:        courseRepo,
		courseContentRepo: courseContentRepo,
		classRepo:         classRepo,
		userRepo:          userRepo,
		auditService:      auditService,
	}
}

// CreateCourseRequest represents course creation data
type CreateCourseRequest struct {
	ClassID      uuid.UUID  `json:"class_id" validate:"required,uuid"`
	TermID       uuid.UUID  `json:"term_id" validate:"required,uuid"`
	Name         string     `json:"name" validate:"required,min=2,max=200"`
	SubjectCode  string     `json:"subject_code" validate:"required,min=2,max=20"`
	TutorID      uuid.UUID  `json:"tutor_id" validate:"required,uuid"`
	Description  *string    `json:"description,omitempty" validate:"omitempty,max=1000"`
	GradeWeights *string    `json:"grade_weights,omitempty"` // JSON string
}

// UpdateCourseRequest represents course update data
type UpdateCourseRequest struct {
	Name         *string    `json:"name,omitempty" validate:"omitempty,min=2,max=200"`
	Description  *string    `json:"description,omitempty" validate:"omitempty,max=1000"`
	GradeWeights *string    `json:"grade_weights,omitempty"`
}

// ReassignTutorRequest represents tutor reassignment data
type ReassignTutorRequest struct {
	TutorID uuid.UUID `json:"tutor_id" validate:"required,uuid"`
	Reason  string    `json:"reason" validate:"required,min=10,max=500"`
}

// CreateCourse creates a new course
func (s *CourseService) CreateCourse(
	ctx context.Context,
	tenantID uuid.UUID,
	req *CreateCourseRequest,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.Course, error) {
	// Verify class exists and belongs to tenant
	class, err := s.classRepo.Get(ctx, req.ClassID)
	if err != nil {
		return nil, fmt.Errorf("failed to get class: %w", err)
	}

	if class.TenantID != tenantID {
		return nil, fmt.Errorf("class does not belong to this tenant")
	}

	// Verify tutor exists and belongs to tenant
	tutor, err := s.userRepo.GetByID(ctx, req.TutorID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tutor: %w", err)
	}

	if tutor.TenantID == nil || *tutor.TenantID != tenantID {
		return nil, fmt.Errorf("tutor does not belong to this tenant")
	}

	if tutor.Role != domain.RoleTutor && tutor.Role != domain.RoleAdmin {
		return nil, fmt.Errorf("user is not a tutor")
	}

	// Create course
	course := &domain.Course{
		ID:              uuid.New(),
		TenantID:        tenantID,
		SessionID:       class.SessionID,
		ClassID:         req.ClassID,
		TermID:          req.TermID,
		Name:            req.Name,
		SubjectCode:     req.SubjectCode,
		AssignedTutorID: req.TutorID,
		Description:     req.Description,
		Status:          domain.CourseStatusActive,
		Materials:       []string{},
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// TODO: Parse and set CustomGradeWeighting from req.GradeWeights if provided

	if err := s.courseRepo.Create(ctx, course, nil); err != nil {
		return nil, fmt.Errorf("failed to create course: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionCourseCreated,
		actorID,
		actorRole,
		&tenantID,
		domain.AuditResourceCourse,
		course.ID,
		nil,
		course,
		ipAddress,
	)

	return course, nil
}

// GetCourse gets a course by ID
func (s *CourseService) GetCourse(ctx context.Context, id uuid.UUID) (*domain.Course, error) {
	course, err := s.courseRepo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get course: %w", err)
	}
	return course, nil
}

// UpdateCourse updates a course
func (s *CourseService) UpdateCourse(
	ctx context.Context,
	id uuid.UUID,
	req *UpdateCourseRequest,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.Course, error) {
	// Get existing course
	course, err := s.courseRepo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get course: %w", err)
	}

	// Store before state for audit
	beforeState := *course

	// Update fields
	if req.Name != nil {
		course.Name = *req.Name
	}
	if req.Description != nil {
		course.Description = req.Description
	}
	// TODO: Parse and set CustomGradeWeighting from req.GradeWeights if provided
	course.UpdatedAt = time.Now()

	if err := s.courseRepo.Update(ctx, course, nil); err != nil {
		return nil, fmt.Errorf("failed to update course: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionCourseUpdated,
		actorID,
		actorRole,
		&course.TenantID,
		domain.AuditResourceCourse,
		course.ID,
		&beforeState,
		course,
		ipAddress,
	)

	return course, nil
}

// DeleteCourse deletes a course
func (s *CourseService) DeleteCourse(
	ctx context.Context,
	id uuid.UUID,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) error {
	// Get course for audit
	course, err := s.courseRepo.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get course: %w", err)
	}

	// Delete course
	if err := s.courseRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete course: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionCourseDeleted,
		actorID,
		actorRole,
		&course.TenantID,
		domain.AuditResourceCourse,
		course.ID,
		course,
		nil,
		ipAddress,
	)

	return nil
}

// ListCourses lists courses with filters and pagination
func (s *CourseService) ListCourses(
	ctx context.Context,
	tenantID uuid.UUID,
	classID *uuid.UUID,
	termID *uuid.UUID,
	tutorID *uuid.UUID,
	params repository.PaginationParams,
) ([]*domain.Course, *repository.PaginatedResult, error) {
	var courses []*domain.Course
	var total int
	var err error

	// Use most specific filter available
	if classID != nil {
		courses, total, err = s.courseRepo.ListByClass(ctx, *classID, params)
	} else if termID != nil {
		courses, total, err = s.courseRepo.ListByTerm(ctx, *termID, params)
	} else if tutorID != nil {
		courses, total, err = s.courseRepo.ListByTutor(ctx, *tutorID, params)
	} else {
		// TODO: Implement ListByTenant in repository or filter results by tenantID
		return nil, nil, fmt.Errorf("at least one filter (classID, termID, or tutorID) is required")
	}

	if err != nil {
		return nil, nil, fmt.Errorf("failed to list courses: %w", err)
	}

	pagination := repository.BuildPaginatedResult(total, params)

	return courses, &pagination, nil
}

// ReassignTutor reassigns a course to a different tutor (FR-ACA-005)
func (s *CourseService) ReassignTutor(
	ctx context.Context,
	id uuid.UUID,
	req *ReassignTutorRequest,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.Course, error) {
	// Get course
	course, err := s.courseRepo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get course: %w", err)
	}

	// Verify new tutor exists and belongs to tenant
	tutor, err := s.userRepo.GetByID(ctx, req.TutorID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tutor: %w", err)
	}

	if tutor.TenantID == nil || *tutor.TenantID != course.TenantID {
		return nil, fmt.Errorf("tutor does not belong to this tenant")
	}

	if tutor.Role != domain.RoleTutor && tutor.Role != domain.RoleAdmin {
		return nil, fmt.Errorf("user is not a tutor")
	}

	// Store before state
	beforeState := *course

	// Reassign tutor
	course.ReassignTutor(req.TutorID)

	if err := s.courseRepo.Update(ctx, course, nil); err != nil {
		return nil, fmt.Errorf("failed to reassign tutor: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionCourseUpdated,
		actorID,
		actorRole,
		&course.TenantID,
		domain.AuditResourceCourse,
		course.ID,
		&beforeState,
		course,
		ipAddress,
	)

	return course, nil
}

// ==================== Course Content Methods ====================

// CreateContentRequest represents content creation data
type CreateContentRequest struct {
	Title       string             `json:"title" validate:"required,min=2,max=200"`
	ContentType domain.ContentType `json:"content_type" validate:"required,oneof=TEXT VIDEO IMAGE AUDIO DOCUMENT LINK"`
	Content     string             `json:"content" validate:"required"`
	Description *string            `json:"description,omitempty" validate:"omitempty,max=1000"`
	Duration    *int               `json:"duration,omitempty"`
	FileSize    *int64             `json:"file_size,omitempty"`
	MimeType    *string            `json:"mime_type,omitempty"`
}

// UpdateContentRequest represents content update data
type UpdateContentRequest struct {
	Title       *string             `json:"title,omitempty" validate:"omitempty,min=2,max=200"`
	ContentType *domain.ContentType `json:"content_type,omitempty" validate:"omitempty,oneof=TEXT VIDEO IMAGE AUDIO DOCUMENT LINK"`
	Content     *string             `json:"content,omitempty"`
	Description *string             `json:"description,omitempty" validate:"omitempty,max=1000"`
	Duration    *int                `json:"duration,omitempty"`
	FileSize    *int64              `json:"file_size,omitempty"`
	MimeType    *string             `json:"mime_type,omitempty"`
}

// ReorderContentRequest represents content reordering data
type ReorderContentRequest struct {
	ContentIDs []uuid.UUID `json:"content_ids" validate:"required,min=1"`
}

// CreateContent creates new content for a course
func (s *CourseService) CreateContent(
	ctx context.Context,
	courseID uuid.UUID,
	req *CreateContentRequest,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.CourseContent, error) {
	// Verify course exists
	course, err := s.courseRepo.Get(ctx, courseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get course: %w", err)
	}

	// Get next order index
	orderIndex, err := s.courseContentRepo.GetNextOrderIndex(ctx, courseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get next order index: %w", err)
	}

	content := &domain.CourseContent{
		ID:          uuid.New(),
		CourseID:    courseID,
		Title:       req.Title,
		ContentType: req.ContentType,
		Content:     req.Content,
		Description: req.Description,
		OrderIndex:  orderIndex,
		Duration:    req.Duration,
		FileSize:    req.FileSize,
		MimeType:    req.MimeType,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.courseContentRepo.Create(ctx, content, nil); err != nil {
		return nil, fmt.Errorf("failed to create content: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionCourseUpdated,
		actorID,
		actorRole,
		&course.TenantID,
		"course_content",
		content.ID,
		nil,
		content,
		ipAddress,
	)

	return content, nil
}

// GetContent gets a content item by ID
func (s *CourseService) GetContent(ctx context.Context, id uuid.UUID) (*domain.CourseContent, error) {
	content, err := s.courseContentRepo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get content: %w", err)
	}
	return content, nil
}

// UpdateContent updates a content item
func (s *CourseService) UpdateContent(
	ctx context.Context,
	id uuid.UUID,
	req *UpdateContentRequest,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.CourseContent, error) {
	// Get existing content
	content, err := s.courseContentRepo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get content: %w", err)
	}

	// Get course for tenant ID
	course, err := s.courseRepo.Get(ctx, content.CourseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get course: %w", err)
	}

	// Store before state for audit
	beforeState := *content

	// Update fields
	if req.Title != nil {
		content.Title = *req.Title
	}
	if req.ContentType != nil {
		content.ContentType = *req.ContentType
	}
	if req.Content != nil {
		content.Content = *req.Content
	}
	if req.Description != nil {
		content.Description = req.Description
	}
	if req.Duration != nil {
		content.Duration = req.Duration
	}
	if req.FileSize != nil {
		content.FileSize = req.FileSize
	}
	if req.MimeType != nil {
		content.MimeType = req.MimeType
	}
	content.UpdatedAt = time.Now()

	if err := s.courseContentRepo.Update(ctx, content, nil); err != nil {
		return nil, fmt.Errorf("failed to update content: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionCourseUpdated,
		actorID,
		actorRole,
		&course.TenantID,
		"course_content",
		content.ID,
		&beforeState,
		content,
		ipAddress,
	)

	return content, nil
}

// DeleteContent deletes a content item
func (s *CourseService) DeleteContent(
	ctx context.Context,
	id uuid.UUID,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) error {
	// Get content for audit
	content, err := s.courseContentRepo.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get content: %w", err)
	}

	// Get course for tenant ID
	course, err := s.courseRepo.Get(ctx, content.CourseID)
	if err != nil {
		return fmt.Errorf("failed to get course: %w", err)
	}

	// Delete content
	if err := s.courseContentRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete content: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionCourseUpdated,
		actorID,
		actorRole,
		&course.TenantID,
		"course_content",
		content.ID,
		content,
		nil,
		ipAddress,
	)

	return nil
}

// ListContents lists all content items for a course
func (s *CourseService) ListContents(
	ctx context.Context,
	courseID uuid.UUID,
	contentType *domain.ContentType,
	params repository.PaginationParams,
) ([]*domain.CourseContent, *repository.PaginatedResult, error) {
	var contents []*domain.CourseContent
	var total int
	var err error

	if contentType != nil {
		contents, total, err = s.courseContentRepo.ListByCourseAndType(ctx, courseID, *contentType, params)
	} else {
		contents, total, err = s.courseContentRepo.ListByCourse(ctx, courseID, params)
	}

	if err != nil {
		return nil, nil, fmt.Errorf("failed to list contents: %w", err)
	}

	pagination := repository.BuildPaginatedResult(total, params)

	return contents, &pagination, nil
}

// ReorderContents reorders content items for a course
func (s *CourseService) ReorderContents(
	ctx context.Context,
	courseID uuid.UUID,
	req *ReorderContentRequest,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) error {
	// Verify course exists
	course, err := s.courseRepo.Get(ctx, courseID)
	if err != nil {
		return fmt.Errorf("failed to get course: %w", err)
	}

	// Reorder contents
	if err := s.courseContentRepo.ReorderContents(ctx, courseID, req.ContentIDs, nil); err != nil {
		return fmt.Errorf("failed to reorder contents: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionCourseUpdated,
		actorID,
		actorRole,
		&course.TenantID,
		"course_content",
		courseID,
		nil,
		map[string]interface{}{"action": "reorder", "content_ids": req.ContentIDs},
		ipAddress,
	)

	return nil
}

// GetCourseWithContents returns a course with all its content items
func (s *CourseService) GetCourseWithContents(ctx context.Context, courseID uuid.UUID) (*domain.Course, []*domain.CourseContent, error) {
	course, err := s.courseRepo.Get(ctx, courseID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get course: %w", err)
	}

	contents, _, err := s.courseContentRepo.ListByCourse(ctx, courseID, repository.PaginationParams{Limit: 1000})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list contents: %w", err)
	}

	return course, contents, nil
}
