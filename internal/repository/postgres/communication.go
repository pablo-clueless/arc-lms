package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"arc-lms/internal/domain"
	"arc-lms/internal/repository"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// CommunicationRepository handles email persistence operations
type CommunicationRepository struct {
	db *sql.DB
}

// NewCommunicationRepository creates a new communication repository
func NewCommunicationRepository(db *sql.DB) *CommunicationRepository {
	return &CommunicationRepository{db: db}
}

// Create creates a new email record
func (r *CommunicationRepository) Create(ctx context.Context, email *domain.Email) error {
	recipientsJSON, err := json.Marshal(email.Recipients)
	if err != nil {
		return fmt.Errorf("failed to marshal recipients: %w", err)
	}

	specificUserIDsJSON, err := json.Marshal(email.SpecificUserIDs)
	if err != nil {
		return fmt.Errorf("failed to marshal specific user IDs: %w", err)
	}

	attachmentURLsJSON, err := json.Marshal(email.AttachmentURLs)
	if err != nil {
		return fmt.Errorf("failed to marshal attachment URLs: %w", err)
	}

	query := `
		INSERT INTO emails (
			id, tenant_id, sender_id, subject, body, html_body,
			recipient_scope, target_class_id, target_course_id, specific_user_ids,
			recipients, status, scheduled_for, sent_at, attachment_urls,
			total_recipients, success_count, failure_count,
			created_at, updated_at, cancelled_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10,
			$11, $12, $13, $14, $15,
			$16, $17, $18,
			$19, $20, $21
		)
	`

	_, err = r.db.ExecContext(ctx, query,
		email.ID, email.TenantID, email.SenderID, email.Subject, email.Body, email.HTMLBody,
		email.RecipientScope, email.TargetClassID, email.TargetCourseID, specificUserIDsJSON,
		recipientsJSON, email.Status, email.ScheduledFor, email.SentAt, attachmentURLsJSON,
		email.TotalRecipients, email.SuccessCount, email.FailureCount,
		email.CreatedAt, email.UpdatedAt, email.CancelledAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create email: %w", err)
	}

	return nil
}

// Get retrieves an email by ID
func (r *CommunicationRepository) Get(ctx context.Context, id uuid.UUID) (*domain.Email, error) {
	query := `
		SELECT
			id, tenant_id, sender_id, subject, body, html_body,
			recipient_scope, target_class_id, target_course_id, specific_user_ids,
			recipients, status, scheduled_for, sent_at, attachment_urls,
			total_recipients, success_count, failure_count,
			created_at, updated_at, cancelled_at
		FROM emails
		WHERE id = $1
	`

	var email domain.Email
	var recipientsJSON, specificUserIDsJSON, attachmentURLsJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&email.ID, &email.TenantID, &email.SenderID, &email.Subject, &email.Body, &email.HTMLBody,
		&email.RecipientScope, &email.TargetClassID, &email.TargetCourseID, &specificUserIDsJSON,
		&recipientsJSON, &email.Status, &email.ScheduledFor, &email.SentAt, &attachmentURLsJSON,
		&email.TotalRecipients, &email.SuccessCount, &email.FailureCount,
		&email.CreatedAt, &email.UpdatedAt, &email.CancelledAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("email not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get email: %w", err)
	}

	if err := json.Unmarshal(recipientsJSON, &email.Recipients); err != nil {
		return nil, fmt.Errorf("failed to unmarshal recipients: %w", err)
	}
	if err := json.Unmarshal(specificUserIDsJSON, &email.SpecificUserIDs); err != nil {
		email.SpecificUserIDs = []uuid.UUID{}
	}
	if err := json.Unmarshal(attachmentURLsJSON, &email.AttachmentURLs); err != nil {
		email.AttachmentURLs = []string{}
	}

	return &email, nil
}

// Update updates an email record
func (r *CommunicationRepository) Update(ctx context.Context, email *domain.Email) error {
	recipientsJSON, err := json.Marshal(email.Recipients)
	if err != nil {
		return fmt.Errorf("failed to marshal recipients: %w", err)
	}

	query := `
		UPDATE emails SET
			status = $1,
			recipients = $2,
			sent_at = $3,
			total_recipients = $4,
			success_count = $5,
			failure_count = $6,
			updated_at = $7,
			cancelled_at = $8
		WHERE id = $9
	`

	_, err = r.db.ExecContext(ctx, query,
		email.Status, recipientsJSON, email.SentAt,
		email.TotalRecipients, email.SuccessCount, email.FailureCount,
		email.UpdatedAt, email.CancelledAt, email.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update email: %w", err)
	}

	return nil
}

// Delete deletes an email record (only drafts can be deleted)
func (r *CommunicationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM emails WHERE id = $1 AND status = 'DRAFT'`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete email: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("email not found or cannot be deleted")
	}

	return nil
}

// ListByTenant retrieves emails for a tenant with pagination
func (r *CommunicationRepository) ListByTenant(
	ctx context.Context,
	tenantID uuid.UUID,
	status *domain.CommunicationStatus,
	params repository.PaginationParams,
) ([]*domain.Email, error) {
	query := `
		SELECT
			id, tenant_id, sender_id, subject, body, html_body,
			recipient_scope, target_class_id, target_course_id, specific_user_ids,
			recipients, status, scheduled_for, sent_at, attachment_urls,
			total_recipients, success_count, failure_count,
			created_at, updated_at, cancelled_at
		FROM emails
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
		query += fmt.Sprintf(" AND created_at < (SELECT created_at FROM emails WHERE id = $%d)", argIndex)
		args = append(args, *params.Cursor)
		argIndex++
	}

	query += " ORDER BY created_at DESC"

	if params.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", params.Limit)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list emails: %w", err)
	}
	defer rows.Close()

	return r.scanEmails(rows)
}

// ListBySender retrieves emails sent by a specific user
func (r *CommunicationRepository) ListBySender(
	ctx context.Context,
	senderID uuid.UUID,
	params repository.PaginationParams,
) ([]*domain.Email, error) {
	query := `
		SELECT
			id, tenant_id, sender_id, subject, body, html_body,
			recipient_scope, target_class_id, target_course_id, specific_user_ids,
			recipients, status, scheduled_for, sent_at, attachment_urls,
			total_recipients, success_count, failure_count,
			created_at, updated_at, cancelled_at
		FROM emails
		WHERE sender_id = $1
	`

	args := []interface{}{senderID}

	if params.Cursor != nil {
		query += " AND created_at < (SELECT created_at FROM emails WHERE id = $2)"
		args = append(args, *params.Cursor)
	}

	query += " ORDER BY created_at DESC"

	if params.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", params.Limit)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list emails: %w", err)
	}
	defer rows.Close()

	return r.scanEmails(rows)
}

// ListScheduled retrieves scheduled emails that are ready to be sent
func (r *CommunicationRepository) ListScheduled(ctx context.Context, before time.Time) ([]*domain.Email, error) {
	query := `
		SELECT
			id, tenant_id, sender_id, subject, body, html_body,
			recipient_scope, target_class_id, target_course_id, specific_user_ids,
			recipients, status, scheduled_for, sent_at, attachment_urls,
			total_recipients, success_count, failure_count,
			created_at, updated_at, cancelled_at
		FROM emails
		WHERE status = 'SCHEDULED' AND scheduled_for <= $1
		ORDER BY scheduled_for ASC
	`

	rows, err := r.db.QueryContext(ctx, query, before)
	if err != nil {
		return nil, fmt.Errorf("failed to list scheduled emails: %w", err)
	}
	defer rows.Close()

	return r.scanEmails(rows)
}

// GetRecipientsForScope retrieves the list of recipients based on scope
func (r *CommunicationRepository) GetRecipientsForScope(
	ctx context.Context,
	tenantID uuid.UUID,
	scope domain.RecipientScope,
	targetClassID *uuid.UUID,
	targetCourseID *uuid.UUID,
	specificUserIDs []uuid.UUID,
) ([]domain.DeliveryRecipient, error) {
	var query string
	var args []interface{}

	switch scope {
	case domain.RecipientScopeAllUsers:
		query = `
			SELECT id, email FROM users
			WHERE tenant_id = $1 AND status = 'ACTIVE'
		`
		args = []interface{}{tenantID}

	case domain.RecipientScopeAllTutors:
		query = `
			SELECT id, email FROM users
			WHERE tenant_id = $1 AND role = 'TUTOR' AND status = 'ACTIVE'
		`
		args = []interface{}{tenantID}

	case domain.RecipientScopeAllStudents:
		query = `
			SELECT id, email FROM users
			WHERE tenant_id = $1 AND role = 'STUDENT' AND status = 'ACTIVE'
		`
		args = []interface{}{tenantID}

	case domain.RecipientScopeClass:
		if targetClassID == nil {
			return nil, fmt.Errorf("target_class_id is required for CLASS scope")
		}
		// Get tutors assigned to courses in this class + enrolled students
		query = `
			SELECT DISTINCT u.id, u.email FROM users u
			WHERE u.status = 'ACTIVE' AND (
				-- Tutors assigned to courses in this class
				u.id IN (
					SELECT c.tutor_id FROM courses c
					WHERE c.class_id = $1 AND c.tutor_id IS NOT NULL
				)
				OR
				-- Students enrolled in this class
				u.id IN (
					SELECT e.student_id FROM enrollments e
					WHERE e.class_id = $1 AND e.status = 'ACTIVE'
				)
			)
		`
		args = []interface{}{*targetClassID}

	case domain.RecipientScopeCourse:
		if targetCourseID == nil {
			return nil, fmt.Errorf("target_course_id is required for COURSE scope")
		}
		// Get students enrolled in the class that this course belongs to
		query = `
			SELECT DISTINCT u.id, u.email FROM users u
			INNER JOIN enrollments e ON e.student_id = u.id
			INNER JOIN courses c ON c.class_id = e.class_id
			WHERE c.id = $1 AND e.status = 'ACTIVE' AND u.status = 'ACTIVE'
		`
		args = []interface{}{*targetCourseID}

	case domain.RecipientScopeSpecific:
		if len(specificUserIDs) == 0 {
			return nil, fmt.Errorf("specific_user_ids is required for SPECIFIC_USERS scope")
		}
		query = `
			SELECT id, email FROM users
			WHERE id = ANY($1) AND status = 'ACTIVE'
		`
		args = []interface{}{pq.Array(specificUserIDs)}

	default:
		return nil, fmt.Errorf("unsupported recipient scope: %s", scope)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get recipients: %w", err)
	}
	defer rows.Close()

	var recipients []domain.DeliveryRecipient
	for rows.Next() {
		var recipient domain.DeliveryRecipient
		if err := rows.Scan(&recipient.UserID, &recipient.Email); err != nil {
			return nil, fmt.Errorf("failed to scan recipient: %w", err)
		}
		recipient.Status = domain.CommunicationStatusScheduled
		recipients = append(recipients, recipient)
	}

	return recipients, nil
}

// ValidateTutorAccess validates that a tutor can send to the specified scope
func (r *CommunicationRepository) ValidateTutorAccess(
	ctx context.Context,
	tutorID uuid.UUID,
	scope domain.RecipientScope,
	targetClassID *uuid.UUID,
	targetCourseID *uuid.UUID,
) (bool, error) {
	switch scope {
	case domain.RecipientScopeClass:
		if targetClassID == nil {
			return false, nil
		}
		// Check if tutor teaches any course in this class
		query := `
			SELECT EXISTS(
				SELECT 1 FROM courses
				WHERE class_id = $1 AND tutor_id = $2
			)
		`
		var exists bool
		err := r.db.QueryRowContext(ctx, query, *targetClassID, tutorID).Scan(&exists)
		return exists, err

	case domain.RecipientScopeCourse:
		if targetCourseID == nil {
			return false, nil
		}
		// Check if tutor teaches this course
		query := `
			SELECT EXISTS(
				SELECT 1 FROM courses
				WHERE id = $1 AND tutor_id = $2
			)
		`
		var exists bool
		err := r.db.QueryRowContext(ctx, query, *targetCourseID, tutorID).Scan(&exists)
		return exists, err

	default:
		// Tutors can only send to class or course scope
		return false, nil
	}
}

// GetEmailStatistics retrieves email statistics for a tenant
func (r *CommunicationRepository) GetEmailStatistics(
	ctx context.Context,
	tenantID uuid.UUID,
	startDate, endDate time.Time,
) (*EmailStatistics, error) {
	query := `
		SELECT
			COUNT(*) as total_emails,
			COUNT(*) FILTER (WHERE status = 'SENT') as sent_count,
			COUNT(*) FILTER (WHERE status = 'SCHEDULED') as scheduled_count,
			COUNT(*) FILTER (WHERE status = 'FAILED') as failed_count,
			COUNT(*) FILTER (WHERE status = 'CANCELLED') as cancelled_count,
			COALESCE(SUM(total_recipients), 0) as total_recipients,
			COALESCE(SUM(success_count), 0) as total_delivered,
			COALESCE(SUM(failure_count), 0) as total_failed_deliveries
		FROM emails
		WHERE tenant_id = $1 AND created_at BETWEEN $2 AND $3
	`

	var stats EmailStatistics
	stats.TenantID = tenantID

	err := r.db.QueryRowContext(ctx, query, tenantID, startDate, endDate).Scan(
		&stats.TotalEmails,
		&stats.SentCount,
		&stats.ScheduledCount,
		&stats.FailedCount,
		&stats.CancelledCount,
		&stats.TotalRecipients,
		&stats.TotalDelivered,
		&stats.TotalFailedDeliveries,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get email statistics: %w", err)
	}

	if stats.TotalRecipients > 0 {
		stats.DeliveryRate = float64(stats.TotalDelivered) / float64(stats.TotalRecipients) * 100
	}

	return &stats, nil
}

// EmailStatistics represents email statistics
type EmailStatistics struct {
	TenantID              uuid.UUID `json:"tenant_id"`
	TotalEmails           int       `json:"total_emails"`
	SentCount             int       `json:"sent_count"`
	ScheduledCount        int       `json:"scheduled_count"`
	FailedCount           int       `json:"failed_count"`
	CancelledCount        int       `json:"cancelled_count"`
	TotalRecipients       int       `json:"total_recipients"`
	TotalDelivered        int       `json:"total_delivered"`
	TotalFailedDeliveries int       `json:"total_failed_deliveries"`
	DeliveryRate          float64   `json:"delivery_rate"`
}

// Helper method to scan email rows
func (r *CommunicationRepository) scanEmails(rows *sql.Rows) ([]*domain.Email, error) {
	var emails []*domain.Email

	for rows.Next() {
		var email domain.Email
		var recipientsJSON, specificUserIDsJSON, attachmentURLsJSON []byte

		err := rows.Scan(
			&email.ID, &email.TenantID, &email.SenderID, &email.Subject, &email.Body, &email.HTMLBody,
			&email.RecipientScope, &email.TargetClassID, &email.TargetCourseID, &specificUserIDsJSON,
			&recipientsJSON, &email.Status, &email.ScheduledFor, &email.SentAt, &attachmentURLsJSON,
			&email.TotalRecipients, &email.SuccessCount, &email.FailureCount,
			&email.CreatedAt, &email.UpdatedAt, &email.CancelledAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan email: %w", err)
		}

		if err := json.Unmarshal(recipientsJSON, &email.Recipients); err != nil {
			email.Recipients = []domain.DeliveryRecipient{}
		}
		if err := json.Unmarshal(specificUserIDsJSON, &email.SpecificUserIDs); err != nil {
			email.SpecificUserIDs = []uuid.UUID{}
		}
		if err := json.Unmarshal(attachmentURLsJSON, &email.AttachmentURLs); err != nil {
			email.AttachmentURLs = []string{}
		}

		emails = append(emails, &email)
	}

	return emails, nil
}

// GetCoTutorsForClass retrieves tutors who also teach in the same class
func (r *CommunicationRepository) GetCoTutorsForClass(ctx context.Context, classID uuid.UUID, excludeTutorID uuid.UUID) ([]domain.DeliveryRecipient, error) {
	query := `
		SELECT DISTINCT u.id, u.email FROM users u
		INNER JOIN courses c ON c.tutor_id = u.id
		WHERE c.class_id = $1 AND u.id != $2 AND u.status = 'ACTIVE'
	`

	rows, err := r.db.QueryContext(ctx, query, classID, excludeTutorID)
	if err != nil {
		return nil, fmt.Errorf("failed to get co-tutors: %w", err)
	}
	defer rows.Close()

	var recipients []domain.DeliveryRecipient
	for rows.Next() {
		var recipient domain.DeliveryRecipient
		if err := rows.Scan(&recipient.UserID, &recipient.Email); err != nil {
			return nil, fmt.Errorf("failed to scan recipient: %w", err)
		}
		recipient.Status = domain.CommunicationStatusScheduled
		recipients = append(recipients, recipient)
	}

	return recipients, nil
}

// CreateEmailsTable creates the emails table if it doesn't exist
func (r *CommunicationRepository) CreateEmailsTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS emails (
			id UUID PRIMARY KEY,
			tenant_id UUID NOT NULL REFERENCES tenants(id),
			sender_id UUID NOT NULL REFERENCES users(id),
			subject VARCHAR(200) NOT NULL,
			body TEXT NOT NULL,
			html_body TEXT,
			recipient_scope VARCHAR(50) NOT NULL,
			target_class_id UUID REFERENCES classes(id),
			target_course_id UUID REFERENCES courses(id),
			specific_user_ids JSONB DEFAULT '[]',
			recipients JSONB DEFAULT '[]',
			status VARCHAR(20) NOT NULL DEFAULT 'DRAFT',
			scheduled_for TIMESTAMP WITH TIME ZONE,
			sent_at TIMESTAMP WITH TIME ZONE,
			attachment_urls JSONB DEFAULT '[]',
			total_recipients INTEGER DEFAULT 0,
			success_count INTEGER DEFAULT 0,
			failure_count INTEGER DEFAULT 0,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			cancelled_at TIMESTAMP WITH TIME ZONE
		);

		CREATE INDEX IF NOT EXISTS idx_emails_tenant_id ON emails(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_emails_sender_id ON emails(sender_id);
		CREATE INDEX IF NOT EXISTS idx_emails_status ON emails(status);
		CREATE INDEX IF NOT EXISTS idx_emails_scheduled_for ON emails(scheduled_for) WHERE status = 'SCHEDULED';
	`

	_, err := r.db.ExecContext(ctx, query)
	return err
}

// UpdateRecipientStatus updates the delivery status for a specific recipient
func (r *CommunicationRepository) UpdateRecipientStatus(
	ctx context.Context,
	emailID uuid.UUID,
	userID uuid.UUID,
	status domain.CommunicationStatus,
	failureReason *string,
) error {
	// Get current email
	email, err := r.Get(ctx, emailID)
	if err != nil {
		return err
	}

	// Find and update recipient
	now := time.Now()
	for i := range email.Recipients {
		if email.Recipients[i].UserID == userID {
			email.Recipients[i].Status = status
			if status == domain.CommunicationStatusSent {
				email.Recipients[i].SentAt = &now
				email.SuccessCount++
			} else if status == domain.CommunicationStatusFailed {
				email.Recipients[i].FailedAt = &now
				email.Recipients[i].FailureReason = failureReason
				email.FailureCount++
			}
			break
		}
	}

	email.UpdatedAt = now

	// Check if all recipients have been processed
	allProcessed := true
	for _, r := range email.Recipients {
		if r.Status == domain.CommunicationStatusScheduled || r.Status == domain.CommunicationStatusSending {
			allProcessed = false
			break
		}
	}

	if allProcessed {
		email.Status = domain.CommunicationStatusSent
		email.SentAt = &now
	}

	return r.Update(ctx, email)
}

// SearchEmails searches emails by subject or body content
func (r *CommunicationRepository) SearchEmails(
	ctx context.Context,
	tenantID uuid.UUID,
	searchTerm string,
	params repository.PaginationParams,
) ([]*domain.Email, error) {
	searchPattern := "%" + strings.ToLower(searchTerm) + "%"

	query := `
		SELECT
			id, tenant_id, sender_id, subject, body, html_body,
			recipient_scope, target_class_id, target_course_id, specific_user_ids,
			recipients, status, scheduled_for, sent_at, attachment_urls,
			total_recipients, success_count, failure_count,
			created_at, updated_at, cancelled_at
		FROM emails
		WHERE tenant_id = $1 AND (LOWER(subject) LIKE $2 OR LOWER(body) LIKE $2)
		ORDER BY created_at DESC
	`

	args := []interface{}{tenantID, searchPattern}

	if params.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", params.Limit)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search emails: %w", err)
	}
	defer rows.Close()

	return r.scanEmails(rows)
}
