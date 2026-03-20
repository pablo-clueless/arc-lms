package seed

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"arc-lms/internal/domain"
	"arc-lms/internal/pkg/crypto"

	"github.com/google/uuid"
)

// SampleSchoolConfig holds sample school seed configuration
type SampleSchoolConfig struct {
	SchoolName     string
	ContactEmail   string
	DefaultPassword string
}

// DefaultSampleSchoolConfig returns default configuration
func DefaultSampleSchoolConfig() SampleSchoolConfig {
	return SampleSchoolConfig{
		SchoolName:     "Greenfield International Academy",
		ContactEmail:   "admin@greenfield.edu.ng",
		DefaultPassword: "Password123!",
	}
}

// SeedSampleSchool creates a complete sample school with all data
func SeedSampleSchool(db *sql.DB, config SampleSchoolConfig) error {
	ctx := context.Background()

	// Check if sample school already exists
	var exists bool
	err := db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM tenants WHERE name = $1)", config.SchoolName).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check if sample school exists: %w", err)
	}
	if exists {
		log.Printf("✅ Sample school '%s' already exists (skipping seed)", config.SchoolName)
		return nil
	}

	log.Printf("🏫 Creating sample school: %s", config.SchoolName)

	// Hash default password
	hashedPassword, err := crypto.HashPassword(config.DefaultPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Start transaction
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Create tenant
	tenantID := uuid.New()
	principalAdminID := uuid.New()

	if err := createTenant(ctx, tx, tenantID, principalAdminID, config); err != nil {
		return err
	}

	// 2. Create admins (3 total, including principal admin)
	adminIDs, err := createAdmins(ctx, tx, tenantID, principalAdminID, hashedPassword, 3)
	if err != nil {
		return err
	}
	log.Printf("  ✅ Created %d admins", len(adminIDs))

	// 3. Create tutors (35)
	tutorIDs, err := createTutors(ctx, tx, tenantID, hashedPassword, 35)
	if err != nil {
		return err
	}
	log.Printf("  ✅ Created %d tutors", len(tutorIDs))

	// 4. Create session
	sessionID := uuid.New()
	if err := createSession(ctx, tx, tenantID, sessionID); err != nil {
		return err
	}
	log.Printf("  ✅ Created session 2025/2026")

	// 5. Create term
	termID := uuid.New()
	if err := createTerm(ctx, tx, tenantID, sessionID, termID); err != nil {
		return err
	}
	log.Printf("  ✅ Created First Term")

	// 6. Create classes (Primary 1-6, JSS1-3, SS1-3)
	classIDs, err := createClasses(ctx, tx, tenantID, sessionID)
	if err != nil {
		return err
	}
	log.Printf("  ✅ Created %d classes", len(classIDs))

	// 7. Create students and enrollments (min 15 per class)
	studentCount, err := createStudentsAndEnrollments(ctx, tx, tenantID, sessionID, classIDs, hashedPassword)
	if err != nil {
		return err
	}
	log.Printf("  ✅ Created %d students with enrollments", studentCount)

	// 8. Create courses
	courseCount, err := createCourses(ctx, tx, tenantID, sessionID, termID, classIDs, tutorIDs)
	if err != nil {
		return err
	}
	log.Printf("  ✅ Created %d courses", courseCount)

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("✅ Sample school '%s' created successfully!", config.SchoolName)
	log.Printf("   Login credentials: any user email with password '%s'", config.DefaultPassword)

	return nil
}

func createTenant(ctx context.Context, tx *sql.Tx, tenantID, principalAdminID uuid.UUID, config SampleSchoolConfig) error {
	configuration := map[string]interface{}{
		"timezone":                     "Africa/Lagos",
		"school_level":                 "COMBINED",
		"period_duration":              45,
		"daily_period_limit":           8,
		"max_periods_per_week":         map[string]int{"Mathematics": 5, "English Language": 5},
		"grade_weighting":              map[string]int{"continuous_assessment": 40, "examination": 60},
		"attendance_threshold":         75,
		"invoice_grace_period":         14,
		"suspension_threshold":         30,
		"branding_assets":              map[string]string{},
		"communication_prefs":          map[string]bool{"email_enabled": true, "sms_enabled": true},
		"supported_classes":            []string{"Primary 1", "Primary 2", "Primary 3", "Primary 4", "Primary 5", "Primary 6", "JSS1", "JSS2", "JSS3", "SS1", "SS2", "SS3"},
		"notification_settings":        map[string]bool{"grades_published": true, "fee_reminder": true},
		"meeting_recording_retention": 30,
	}

	configJSON, _ := json.Marshal(configuration)

	billingContact := map[string]string{
		"name":  "Finance Office",
		"email": "finance@greenfield.edu.ng",
		"phone": "+2348012345678",
	}
	billingJSON, _ := json.Marshal(billingContact)

	query := `
		INSERT INTO tenants (
			id, name, school_type, contact_email, address, logo, status,
			configuration, billing_contact, principal_admin_id, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	now := time.Now()
	_, err := tx.ExecContext(ctx, query,
		tenantID,
		config.SchoolName,
		domain.SchoolTypeCombined,
		config.ContactEmail,
		"123 Education Avenue, Victoria Island, Lagos, Nigeria",
		"",
		domain.TenantStatusActive,
		configJSON,
		billingJSON,
		principalAdminID,
		now,
		now,
	)

	return err
}

func createAdmins(ctx context.Context, tx *sql.Tx, tenantID, principalAdminID uuid.UUID, hashedPassword string, count int) ([]uuid.UUID, error) {
	admins := []struct {
		firstName string
		lastName  string
		email     string
	}{
		{"Adebayo", "Okonkwo", "adebayo.okonkwo@greenfield.edu.ng"},
		{"Chioma", "Nnamdi", "chioma.nnamdi@greenfield.edu.ng"},
		{"Emeka", "Adeyemi", "emeka.adeyemi@greenfield.edu.ng"},
	}

	adminIDs := make([]uuid.UUID, count)
	now := time.Now()
	permissions := []string{"users:*", "classes:*", "courses:*", "enrollments:*", "sessions:*", "terms:*"}
	permissionsJSON, _ := json.Marshal(permissions)
	prefsJSON, _ := json.Marshal([]domain.NotificationPreference{})

	for i := 0; i < count && i < len(admins); i++ {
		var id uuid.UUID
		if i == 0 {
			id = principalAdminID
		} else {
			id = uuid.New()
		}
		adminIDs[i] = id

		query := `
			INSERT INTO users (
				id, tenant_id, role, email, password_hash,
				first_name, last_name, status, permissions, notification_preferences,
				created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		`

		_, err := tx.ExecContext(ctx, query,
			id,
			tenantID,
			domain.RoleAdmin,
			admins[i].email,
			hashedPassword,
			admins[i].firstName,
			admins[i].lastName,
			domain.UserStatusActive,
			permissionsJSON,
			prefsJSON,
			now,
			now,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create admin %s: %w", admins[i].email, err)
		}
	}

	return adminIDs, nil
}

func createTutors(ctx context.Context, tx *sql.Tx, tenantID uuid.UUID, hashedPassword string, count int) ([]uuid.UUID, error) {
	tutorNames := []struct {
		firstName string
		lastName  string
	}{
		{"Oluwaseun", "Afolabi"}, {"Ngozi", "Okafor"}, {"Babatunde", "Adewale"},
		{"Funke", "Akindele"}, {"Chukwuemeka", "Eze"}, {"Yetunde", "Bakare"},
		{"Obiora", "Nwosu"}, {"Adaeze", "Igwe"}, {"Kayode", "Ogundimu"},
		{"Chiamaka", "Okwu"}, {"Segun", "Onifade"}, {"Nneka", "Uche"},
		{"Tunde", "Fashola"}, {"Ifeoma", "Chukwu"}, {"Gbenga", "Adeniyi"},
		{"Amaka", "Obi"}, {"Dele", "Momodu"}, {"Chidinma", "Ezeh"},
		{"Kunle", "Ajayi"}, {"Ugochi", "Nwachukwu"}, {"Rotimi", "Adeleke"},
		{"Adanna", "Onyeka"}, {"Femi", "Otedola"}, {"Ebele", "Azubuike"},
		{"Bode", "Thomas"}, {"Chinwe", "Achebe"}, {"Sola", "Kosoko"},
		{"Uchenna", "Ikenna"}, {"Wale", "Adenuga"}, {"Kelechi", "Iheanacho"},
		{"Yinka", "Davies"}, {"Nkechi", "Maduka"}, {"Jide", "Sanwo"},
		{"Oluchi", "Okoro"}, {"Dapo", "Oyebanjo"},
	}

	tutorIDs := make([]uuid.UUID, count)
	now := time.Now()
	permissionsJSON, _ := json.Marshal([]string{"courses:read", "courses:update", "students:read"})
	prefsJSON, _ := json.Marshal([]domain.NotificationPreference{})

	for i := 0; i < count && i < len(tutorNames); i++ {
		id := uuid.New()
		tutorIDs[i] = id
		email := fmt.Sprintf("%s.%s@greenfield.edu.ng",
			toLowerCase(tutorNames[i].firstName),
			toLowerCase(tutorNames[i].lastName))

		query := `
			INSERT INTO users (
				id, tenant_id, role, email, password_hash,
				first_name, last_name, status, permissions, notification_preferences,
				created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		`

		_, err := tx.ExecContext(ctx, query,
			id,
			tenantID,
			domain.RoleTutor,
			email,
			hashedPassword,
			tutorNames[i].firstName,
			tutorNames[i].lastName,
			domain.UserStatusActive,
			permissionsJSON,
			prefsJSON,
			now,
			now,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create tutor: %w", err)
		}
	}

	return tutorIDs, nil
}

func createSession(ctx context.Context, tx *sql.Tx, tenantID, sessionID uuid.UUID) error {
	now := time.Now()
	query := `
		INSERT INTO sessions (
			id, tenant_id, label, start_year, end_year, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := tx.ExecContext(ctx, query,
		sessionID,
		tenantID,
		"2025/2026",
		2025,
		2026,
		domain.SessionStatusActive,
		now,
		now,
	)

	return err
}

func createTerm(ctx context.Context, tx *sql.Tx, tenantID, sessionID, termID uuid.UUID) error {
	now := time.Now()
	holidaysJSON, _ := json.Marshal([]domain.Holiday{})
	nonInstructionalJSON, _ := json.Marshal([]time.Time{})

	query := `
		INSERT INTO terms (
			id, tenant_id, session_id, ordinal, start_date, end_date, status,
			holidays, non_instructional_days, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	startDate := time.Date(2025, 9, 8, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, 12, 13, 0, 0, 0, 0, time.UTC)

	_, err := tx.ExecContext(ctx, query,
		termID,
		tenantID,
		sessionID,
		domain.TermOrdinalFirst,
		startDate,
		endDate,
		domain.TermStatusActive,
		holidaysJSON,
		nonInstructionalJSON,
		now,
		now,
	)

	return err
}

type classInfo struct {
	id    uuid.UUID
	name  string
	level domain.SchoolLevel
}

func createClasses(ctx context.Context, tx *sql.Tx, tenantID, sessionID uuid.UUID) ([]classInfo, error) {
	classes := []struct {
		name  string
		level domain.SchoolLevel
	}{
		{"Primary 1", domain.SchoolLevelPrimary},
		{"Primary 2", domain.SchoolLevelPrimary},
		{"Primary 3", domain.SchoolLevelPrimary},
		{"Primary 4", domain.SchoolLevelPrimary},
		{"Primary 5", domain.SchoolLevelPrimary},
		{"Primary 6", domain.SchoolLevelPrimary},
		{"JSS1", domain.SchoolLevelSecondary},
		{"JSS2", domain.SchoolLevelSecondary},
		{"JSS3", domain.SchoolLevelSecondary},
		{"SS1", domain.SchoolLevelSecondary},
		{"SS2", domain.SchoolLevelSecondary},
		{"SS3", domain.SchoolLevelSecondary},
	}

	classInfos := make([]classInfo, len(classes))
	now := time.Now()
	capacity := 30

	for i, c := range classes {
		id := uuid.New()
		classInfos[i] = classInfo{id: id, name: c.name, level: c.level}

		query := `
			INSERT INTO classes (
				id, tenant_id, session_id, name, arm, level, capacity, status, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`

		_, err := tx.ExecContext(ctx, query,
			id,
			tenantID,
			sessionID,
			c.name,
			"A",
			c.level,
			capacity,
			domain.ClassStatusActive,
			now,
			now,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create class %s: %w", c.name, err)
		}
	}

	return classInfos, nil
}

func createStudentsAndEnrollments(ctx context.Context, tx *sql.Tx, tenantID, sessionID uuid.UUID, classes []classInfo, hashedPassword string) (int, error) {
	firstNames := []string{
		"Adaora", "Chidi", "Olumide", "Funmi", "Emeka", "Yetunde", "Obiora", "Chioma",
		"Kayode", "Ngozi", "Segun", "Nneka", "Tunde", "Amara", "Gbenga", "Ifeoma",
		"Dele", "Adanna", "Femi", "Ebele", "Bode", "Chinwe", "Wale", "Uchenna",
		"Jide", "Kelechi", "Dapo", "Oluchi", "Rotimi", "Chiamaka",
	}
	lastNames := []string{
		"Okonkwo", "Adeyemi", "Nnamdi", "Bakare", "Eze", "Okafor", "Nwosu", "Igwe",
		"Ogundimu", "Okwu", "Onifade", "Uche", "Fashola", "Chukwu", "Adeniyi", "Obi",
		"Momodu", "Ezeh", "Ajayi", "Nwachukwu", "Adeleke", "Onyeka", "Otedola", "Azubuike",
		"Thomas", "Achebe", "Kosoko", "Ikenna", "Adenuga", "Iheanacho",
	}

	now := time.Now()
	totalStudents := 0
	permissionsJSON, _ := json.Marshal([]string{})
	prefsJSON, _ := json.Marshal([]domain.NotificationPreference{})

	studentQuery := `
		INSERT INTO users (
			id, tenant_id, role, email, password_hash,
			first_name, last_name, status, permissions, notification_preferences,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	enrollmentQuery := `
		INSERT INTO enrollments (
			id, tenant_id, student_id, class_id, session_id, status,
			enrollment_date, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	for _, class := range classes {
		// Create 15-20 students per class
		studentsPerClass := 15 + (totalStudents % 6) // Varies between 15-20

		for j := 0; j < studentsPerClass; j++ {
			studentID := uuid.New()
			firstName := firstNames[(totalStudents+j)%len(firstNames)]
			lastName := lastNames[(totalStudents+j+7)%len(lastNames)]
			email := fmt.Sprintf("%s.%s.%s@student.greenfield.edu.ng",
				toLowerCase(firstName),
				toLowerCase(lastName),
				studentID.String()[:8])

			// Create student
			_, err := tx.ExecContext(ctx, studentQuery,
				studentID,
				tenantID,
				domain.RoleStudent,
				email,
				hashedPassword,
				firstName,
				lastName,
				domain.UserStatusActive,
				permissionsJSON,
				prefsJSON,
				now,
				now,
			)
			if err != nil {
				return 0, fmt.Errorf("failed to create student: %w", err)
			}

			// Create enrollment
			enrollmentID := uuid.New()
			_, err = tx.ExecContext(ctx, enrollmentQuery,
				enrollmentID,
				tenantID,
				studentID,
				class.id,
				sessionID,
				domain.EnrollmentStatusActive,
				now,
				now,
				now,
			)
			if err != nil {
				return 0, fmt.Errorf("failed to create enrollment: %w", err)
			}

			totalStudents++
		}
	}

	return totalStudents, nil
}

func createCourses(ctx context.Context, tx *sql.Tx, tenantID, sessionID, termID uuid.UUID, classes []classInfo, tutorIDs []uuid.UUID) (int, error) {
	primaryCourses := []struct {
		name string
		code string
	}{
		{"English Language", "ENG"},
		{"Mathematics", "MTH"},
		{"Basic Science", "BSC"},
		{"Social Studies", "SST"},
		{"Civic Education", "CVE"},
		{"Computer Studies", "CMP"},
		{"Creative Arts", "CRA"},
		{"Physical & Health Education", "PHE"},
		{"Religious Studies", "REL"},
		{"Home Economics", "HEC"},
	}

	juniorSecondaryCourses := []struct {
		name string
		code string
	}{
		{"English Language", "ENG"},
		{"Mathematics", "MTH"},
		{"Basic Science", "BSC"},
		{"Basic Technology", "BTC"},
		{"Social Studies", "SST"},
		{"Civic Education", "CVE"},
		{"Computer Studies", "CMP"},
		{"Physical & Health Education", "PHE"},
		{"French", "FRN"},
		{"Business Studies", "BUS"},
		{"Home Economics", "HEC"},
		{"Agricultural Science", "AGR"},
	}

	seniorSecondaryCourses := []struct {
		name string
		code string
	}{
		{"English Language", "ENG"},
		{"Mathematics", "MTH"},
		{"Physics", "PHY"},
		{"Chemistry", "CHM"},
		{"Biology", "BIO"},
		{"Geography", "GEO"},
		{"Economics", "ECO"},
		{"Government", "GOV"},
		{"Literature in English", "LIT"},
		{"Computer Science", "CSC"},
		{"Civic Education", "CVE"},
		{"Further Mathematics", "FMT"},
		{"Technical Drawing", "TDR"},
		{"French", "FRN"},
	}

	now := time.Now()
	totalCourses := 0
	tutorIndex := 0
	materialsJSON, _ := json.Marshal([]string{})

	query := `
		INSERT INTO courses (
			id, tenant_id, session_id, class_id, term_id, name, subject_code,
			assigned_tutor_id, status, materials, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	for _, class := range classes {
		var courses []struct {
			name string
			code string
		}

		// Select appropriate courses based on class level
		if class.level == domain.SchoolLevelPrimary {
			courses = primaryCourses
		} else if class.name == "JSS1" || class.name == "JSS2" || class.name == "JSS3" {
			courses = juniorSecondaryCourses
		} else {
			courses = seniorSecondaryCourses
		}

		for _, course := range courses {
			courseID := uuid.New()
			tutorID := tutorIDs[tutorIndex%len(tutorIDs)]
			tutorIndex++

			_, err := tx.ExecContext(ctx, query,
				courseID,
				tenantID,
				sessionID,
				class.id,
				termID,
				course.name,
				course.code,
				tutorID,
				domain.CourseStatusActive,
				materialsJSON,
				now,
				now,
			)
			if err != nil {
				return 0, fmt.Errorf("failed to create course %s for %s: %w", course.name, class.name, err)
			}
			totalCourses++
		}
	}

	return totalCourses, nil
}

func toLowerCase(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}
