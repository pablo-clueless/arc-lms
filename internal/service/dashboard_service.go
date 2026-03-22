package service

import (
	"context"
	"fmt"

	"arc-lms/internal/domain"
	"arc-lms/internal/pkg/metrics"
	"arc-lms/internal/repository"
	"arc-lms/internal/repository/postgres"

	"github.com/google/uuid"
)

// DashboardService handles dashboard data operations
type DashboardService struct {
	tenantRepo     *postgres.TenantRepository
	userRepo       *postgres.UserRepository
	sessionRepo    *postgres.SessionRepository
	classRepo      *postgres.ClassRepository
	courseRepo     *postgres.CourseRepository
	enrollmentRepo *postgres.EnrollmentRepository
	invoiceRepo    *postgres.InvoiceRepository
}

// NewDashboardService creates a new dashboard service
func NewDashboardService(
	tenantRepo *postgres.TenantRepository,
	userRepo *postgres.UserRepository,
	sessionRepo *postgres.SessionRepository,
	classRepo *postgres.ClassRepository,
	courseRepo *postgres.CourseRepository,
	enrollmentRepo *postgres.EnrollmentRepository,
	invoiceRepo *postgres.InvoiceRepository,
) *DashboardService {
	return &DashboardService{
		tenantRepo:     tenantRepo,
		userRepo:       userRepo,
		sessionRepo:    sessionRepo,
		classRepo:      classRepo,
		courseRepo:     courseRepo,
		enrollmentRepo: enrollmentRepo,
		invoiceRepo:    invoiceRepo,
	}
}

// SuperAdminDashboard represents dashboard data for SUPER_ADMIN
type SuperAdminDashboard struct {
	TotalTenants     int                         `json:"total_tenants"`
	ActiveTenants    int                         `json:"active_tenants"`
	SuspendedTenants int                         `json:"suspended_tenants"`
	TotalUsers       int                         `json:"total_users"`
	UsersByRole      map[string]int              `json:"users_by_role"`
	RecentTenants    []*domain.Tenant            `json:"recent_tenants"`
	UserGrowth       []postgres.UserGrowthPoint  `json:"user_growth"`
	SystemMetrics    *metrics.SystemMetrics      `json:"system_metrics"`
	DBStats          *repository.DBStats         `json:"db_stats"`
	// Billing metrics
	BillingMetrics   *postgres.BillingMetrics    `json:"billing_metrics"`
}

// AdminDashboard represents dashboard data for ADMIN
type AdminDashboard struct {
	TenantInfo         *domain.Tenant            `json:"tenant_info"`
	TotalUsers         int                       `json:"total_users"`
	UsersByRole        map[string]int            `json:"users_by_role"`
	TotalClasses       int                       `json:"total_classes"`
	TotalCourses       int                       `json:"total_courses"`
	TotalSessions      int                       `json:"total_sessions"`
	ActiveSession      *domain.Session           `json:"active_session,omitempty"`
	TotalEnrollments   int                       `json:"total_enrollments"`
	ActiveEnrollments  int                       `json:"active_enrollments"`
}

// TutorDashboard represents dashboard data for TUTOR
type TutorDashboard struct {
	TotalCourses       int                       `json:"total_courses"`
	Courses            []*domain.Course          `json:"courses"`
	TotalStudents      int                       `json:"total_students"`
	ActiveSession      *domain.Session           `json:"active_session,omitempty"`
}

// StudentDashboard represents dashboard data for STUDENT
type StudentDashboard struct {
	TotalEnrollments   int                       `json:"total_enrollments"`
	Enrollments        []*domain.Enrollment      `json:"enrollments"`
	TotalCourses       int                       `json:"total_courses"`
	Courses            []*domain.Course          `json:"courses"`
	ActiveSession      *domain.Session           `json:"active_session,omitempty"`
}

// GetDashboard returns role-specific dashboard data
func (s *DashboardService) GetDashboard(
	ctx context.Context,
	userID uuid.UUID,
	tenantID *uuid.UUID,
	role domain.Role,
) (interface{}, error) {
	switch role {
	case domain.RoleSuperAdmin:
		return s.getSuperAdminDashboard(ctx)
	case domain.RoleAdmin:
		if tenantID == nil {
			return nil, fmt.Errorf("tenant ID required for ADMIN role")
		}
		return s.getAdminDashboard(ctx, *tenantID)
	case domain.RoleTutor:
		if tenantID == nil {
			return nil, fmt.Errorf("tenant ID required for TUTOR role")
		}
		return s.getTutorDashboard(ctx, userID, *tenantID)
	case domain.RoleStudent:
		if tenantID == nil {
			return nil, fmt.Errorf("tenant ID required for STUDENT role")
		}
		return s.getStudentDashboard(ctx, userID, *tenantID)
	default:
		return nil, fmt.Errorf("unknown role: %s", role)
	}
}

// getSuperAdminDashboard returns dashboard for SUPER_ADMIN
func (s *DashboardService) getSuperAdminDashboard(ctx context.Context) (*SuperAdminDashboard, error) {
	// Get all tenants
	paginationParams := repository.PaginationParams{
		Limit:     1000, // High limit for dashboard
		SortOrder: "DESC",
	}
	allTenants, _, err := s.tenantRepo.List(ctx, nil, paginationParams)
	if err != nil {
		return nil, fmt.Errorf("failed to list tenants: %w", err)
	}

	// Count active and suspended tenants
	activeTenants := 0
	suspendedTenants := 0
	for _, tenant := range allTenants {
		if tenant.Status == domain.TenantStatusActive {
			activeTenants++
		} else if tenant.Status == domain.TenantStatusSuspended {
			suspendedTenants++
		}
	}

	// Get all users (across all tenants)
	allUsers, _, err := s.userRepo.List(ctx, nil, nil, nil, paginationParams)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	// Count users by role
	usersByRole := map[string]int{
		string(domain.RoleSuperAdmin): 0,
		string(domain.RoleAdmin):      0,
		string(domain.RoleTutor):      0,
		string(domain.RoleStudent):    0,
	}
	for _, user := range allUsers {
		usersByRole[string(user.Role)]++
	}

	// Get recent tenants (last 5)
	recentTenants := allTenants
	if len(recentTenants) > 5 {
		recentTenants = recentTenants[:5]
	}

	// Get user growth data (last 30 days)
	userGrowth, err := s.userRepo.GetUserGrowth(ctx, 30)
	if err != nil {
		// Non-critical, continue without growth data
		userGrowth = []postgres.UserGrowthPoint{}
	}

	// Get system metrics
	systemMetrics := metrics.GetCollector().GetMetrics()

	// Get DB stats
	dbStats := s.userRepo.GetDBStats()

	// Get billing metrics
	var billingMetrics *postgres.BillingMetrics
	if s.invoiceRepo != nil {
		billingMetrics, _ = s.invoiceRepo.GetBillingMetrics(ctx)
	}

	return &SuperAdminDashboard{
		TotalTenants:     len(allTenants),
		ActiveTenants:    activeTenants,
		SuspendedTenants: suspendedTenants,
		TotalUsers:       len(allUsers),
		UsersByRole:      usersByRole,
		RecentTenants:    recentTenants,
		UserGrowth:       userGrowth,
		SystemMetrics:    systemMetrics,
		DBStats:          dbStats,
		BillingMetrics:   billingMetrics,
	}, nil
}

// getAdminDashboard returns dashboard for ADMIN
func (s *DashboardService) getAdminDashboard(ctx context.Context, tenantID uuid.UUID) (*AdminDashboard, error) {
	// Get tenant info
	tenant, err := s.tenantRepo.Get(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	// Get users in tenant
	paginationParams := repository.PaginationParams{
		Limit:     1000, // High limit for dashboard
		SortOrder: "DESC",
	}
	users, _, err := s.userRepo.List(ctx, &tenantID, nil, nil, paginationParams)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	// Count users by role
	usersByRole := map[string]int{
		string(domain.RoleAdmin):   0,
		string(domain.RoleTutor):   0,
		string(domain.RoleStudent): 0,
	}
	for _, user := range users {
		usersByRole[string(user.Role)]++
	}

	// Get classes
	classes, _, err := s.classRepo.ListByTenant(ctx, tenantID, paginationParams)
	if err != nil {
		return nil, fmt.Errorf("failed to list classes: %w", err)
	}

	// Get sessions
	sessions, _, err := s.sessionRepo.ListByTenant(ctx, tenantID, nil, paginationParams)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	// Get courses (count by iterating through all classes)
	allCourses := []*domain.Course{}
	for _, class := range classes {
		classCourses, _, err := s.courseRepo.ListByClass(ctx, class.ID, paginationParams)
		if err != nil {
			continue
		}
		allCourses = append(allCourses, classCourses...)
	}

	// Get active session
	var activeSession *domain.Session
	for _, session := range sessions {
		if session.Status == domain.SessionStatusActive {
			activeSession = session
			break
		}
	}

	// Get enrollments (iterate through classes to get all enrollments)
	allEnrollments := []*domain.Enrollment{}
	for _, class := range classes {
		classEnrollments, _, err := s.enrollmentRepo.ListByClass(ctx, class.ID, paginationParams)
		if err != nil {
			continue
		}
		allEnrollments = append(allEnrollments, classEnrollments...)
	}

	// Count active enrollments
	activeEnrollments := 0
	for _, enrollment := range allEnrollments {
		if enrollment.Status == domain.EnrollmentStatusActive {
			activeEnrollments++
		}
	}

	return &AdminDashboard{
		TenantInfo:        tenant,
		TotalUsers:        len(users),
		UsersByRole:       usersByRole,
		TotalClasses:      len(classes),
		TotalCourses:      len(allCourses),
		TotalSessions:     len(sessions),
		ActiveSession:     activeSession,
		TotalEnrollments:  len(allEnrollments),
		ActiveEnrollments: activeEnrollments,
	}, nil
}

// getTutorDashboard returns dashboard for TUTOR
func (s *DashboardService) getTutorDashboard(ctx context.Context, tutorID uuid.UUID, tenantID uuid.UUID) (*TutorDashboard, error) {
	// Get courses assigned to this tutor
	paginationParams := repository.PaginationParams{
		Limit:     1000, // High limit for dashboard
		SortOrder: "DESC",
	}
	courses, _, err := s.courseRepo.ListByTutor(ctx, tutorID, paginationParams)
	if err != nil {
		return nil, fmt.Errorf("failed to list tutor courses: %w", err)
	}

	// Get unique students from enrollments in the tutor's classes
	studentMap := make(map[uuid.UUID]bool)
	for _, course := range courses {
		enrollments, _, err := s.enrollmentRepo.ListByClass(ctx, course.ClassID, paginationParams)
		if err != nil {
			continue
		}
		for _, enrollment := range enrollments {
			if enrollment.Status == domain.EnrollmentStatusActive {
				studentMap[enrollment.StudentID] = true
			}
		}
	}

	// Get active session
	sessions, _, err := s.sessionRepo.ListByTenant(ctx, tenantID, nil, paginationParams)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	var activeSession *domain.Session
	for _, session := range sessions {
		if session.Status == domain.SessionStatusActive {
			activeSession = session
			break
		}
	}

	return &TutorDashboard{
		TotalCourses:  len(courses),
		Courses:       courses,
		TotalStudents: len(studentMap),
		ActiveSession: activeSession,
	}, nil
}

// getStudentDashboard returns dashboard for STUDENT
func (s *DashboardService) getStudentDashboard(ctx context.Context, studentID uuid.UUID, tenantID uuid.UUID) (*StudentDashboard, error) {
	// Get student's enrollments
	paginationParams := repository.PaginationParams{
		Limit:     1000, // High limit for dashboard
		SortOrder: "DESC",
	}
	enrollments, _, err := s.enrollmentRepo.ListByStudent(ctx, studentID, paginationParams)
	if err != nil {
		return nil, fmt.Errorf("failed to list enrollments: %w", err)
	}

	// Get courses for the enrolled classes
	var courses []*domain.Course
	classIDs := make(map[uuid.UUID]bool)
	for _, enrollment := range enrollments {
		if enrollment.Status == domain.EnrollmentStatusActive {
			classIDs[enrollment.ClassID] = true
		}
	}

	for classID := range classIDs {
		classCourses, _, err := s.courseRepo.ListByClass(ctx, classID, paginationParams)
		if err != nil {
			continue
		}
		courses = append(courses, classCourses...)
	}

	// Get active session
	sessions, _, err := s.sessionRepo.ListByTenant(ctx, tenantID, nil, paginationParams)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	var activeSession *domain.Session
	for _, session := range sessions {
		if session.Status == domain.SessionStatusActive {
			activeSession = session
			break
		}
	}

	return &StudentDashboard{
		TotalEnrollments: len(enrollments),
		Enrollments:      enrollments,
		TotalCourses:     len(courses),
		Courses:          courses,
		ActiveSession:    activeSession,
	}, nil
}
