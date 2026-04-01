package postgres

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"arc-lms/internal/domain"
	"arc-lms/internal/repository"
)

// CourseContentRepository handles database operations for course content
type CourseContentRepository struct {
	*repository.BaseRepository
}

// NewCourseContentRepository creates a new course content repository
func NewCourseContentRepository(db *sql.DB) *CourseContentRepository {
	return &CourseContentRepository{
		BaseRepository: repository.NewBaseRepository(db),
	}
}

// Create creates a new course content item
func (r *CourseContentRepository) Create(ctx context.Context, content *domain.CourseContent, tx *sql.Tx) error {
	query := `
		INSERT INTO course_contents (
			id, course_id, title, content_type, content,
			description, order_index, duration, file_size,
			mime_type, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	execer := repository.GetExecer(r.GetDB(), tx)
	_, err := execer.ExecContext(ctx, query,
		content.ID,
		content.CourseID,
		content.Title,
		content.ContentType,
		content.Content,
		repository.ToNullString(content.Description),
		content.OrderIndex,
		repository.ToNullInt(content.Duration),
		repository.ToNullInt64(content.FileSize),
		repository.ToNullString(content.MimeType),
		content.CreatedAt,
		content.UpdatedAt,
	)

	if err != nil {
		return repository.ParseError(err)
	}

	return nil
}

// Get retrieves a course content item by ID
func (r *CourseContentRepository) Get(ctx context.Context, id uuid.UUID) (*domain.CourseContent, error) {
	query := `
		SELECT
			id, course_id, title, content_type, content,
			description, order_index, duration, file_size,
			mime_type, created_at, updated_at
		FROM course_contents
		WHERE id = $1
	`

	return r.scanContent(r.GetDB().QueryRowContext(ctx, query, id))
}

// scanContent scans a course content item from a database row
func (r *CourseContentRepository) scanContent(row *sql.Row) (*domain.CourseContent, error) {
	var content domain.CourseContent
	var description, mimeType sql.NullString
	var duration sql.NullInt32
	var fileSize sql.NullInt64

	err := row.Scan(
		&content.ID,
		&content.CourseID,
		&content.Title,
		&content.ContentType,
		&content.Content,
		&description,
		&content.OrderIndex,
		&duration,
		&fileSize,
		&mimeType,
		&content.CreatedAt,
		&content.UpdatedAt,
	)

	if err != nil {
		return nil, repository.ParseError(err)
	}

	content.Description = repository.FromNullString(description)
	content.MimeType = repository.FromNullString(mimeType)

	if duration.Valid {
		d := int(duration.Int32)
		content.Duration = &d
	}
	if fileSize.Valid {
		content.FileSize = &fileSize.Int64
	}

	return &content, nil
}

// Update updates an existing course content item
func (r *CourseContentRepository) Update(ctx context.Context, content *domain.CourseContent, tx *sql.Tx) error {
	query := `
		UPDATE course_contents
		SET
			title = $2,
			content_type = $3,
			content = $4,
			description = $5,
			order_index = $6,
			duration = $7,
			file_size = $8,
			mime_type = $9,
			updated_at = $10
		WHERE id = $1
	`

	execer := repository.GetExecer(r.GetDB(), tx)
	result, err := execer.ExecContext(ctx, query,
		content.ID,
		content.Title,
		content.ContentType,
		content.Content,
		repository.ToNullString(content.Description),
		content.OrderIndex,
		repository.ToNullInt(content.Duration),
		repository.ToNullInt64(content.FileSize),
		repository.ToNullString(content.MimeType),
		content.UpdatedAt,
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

// Delete deletes a course content item
func (r *CourseContentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM course_contents WHERE id = $1`

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

// ListByCourse retrieves all content items for a course, ordered by order_index
func (r *CourseContentRepository) ListByCourse(ctx context.Context, courseID uuid.UUID, params repository.PaginationParams) ([]*domain.CourseContent, int, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, 0, err
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM course_contents WHERE course_id = $1"
	var total int
	if err := r.GetDB().QueryRowContext(ctx, countQuery, courseID).Scan(&total); err != nil {
		return nil, 0, repository.ParseError(err)
	}

	// Get paginated results ordered by order_index
	query := `
		SELECT
			id, course_id, title, content_type, content,
			description, order_index, duration, file_size,
			mime_type, created_at, updated_at
		FROM course_contents
		WHERE course_id = $1
		ORDER BY order_index ASC, created_at ASC
		LIMIT $2 OFFSET $3
	`

	contents, err := r.queryContents(ctx, query, courseID, params.Limit, params.Offset())
	if err != nil {
		return nil, 0, err
	}

	return contents, total, nil
}

// ListByCourseAndType retrieves content items for a course filtered by type
func (r *CourseContentRepository) ListByCourseAndType(ctx context.Context, courseID uuid.UUID, contentType domain.ContentType, params repository.PaginationParams) ([]*domain.CourseContent, int, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, 0, err
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM course_contents WHERE course_id = $1 AND content_type = $2"
	var total int
	if err := r.GetDB().QueryRowContext(ctx, countQuery, courseID, contentType).Scan(&total); err != nil {
		return nil, 0, repository.ParseError(err)
	}

	// Get paginated results
	query := `
		SELECT
			id, course_id, title, content_type, content,
			description, order_index, duration, file_size,
			mime_type, created_at, updated_at
		FROM course_contents
		WHERE course_id = $1 AND content_type = $2
		ORDER BY order_index ASC, created_at ASC
		LIMIT $3 OFFSET $4
	`

	contents, err := r.queryContents(ctx, query, courseID, contentType, params.Limit, params.Offset())
	if err != nil {
		return nil, 0, err
	}

	return contents, total, nil
}

// queryContents executes a query and returns a list of course contents
func (r *CourseContentRepository) queryContents(ctx context.Context, query string, args ...interface{}) ([]*domain.CourseContent, error) {
	rows, err := r.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, repository.ParseError(err)
	}
	defer rows.Close()

	contents := make([]*domain.CourseContent, 0)
	for rows.Next() {
		content, err := r.scanContentFromRows(rows)
		if err != nil {
			return nil, err
		}
		contents = append(contents, content)
	}

	if err := rows.Err(); err != nil {
		return nil, repository.ParseError(err)
	}

	return contents, nil
}

// scanContentFromRows scans a course content item from a Rows object
func (r *CourseContentRepository) scanContentFromRows(rows *sql.Rows) (*domain.CourseContent, error) {
	var content domain.CourseContent
	var description, mimeType sql.NullString
	var duration sql.NullInt32
	var fileSize sql.NullInt64

	err := rows.Scan(
		&content.ID,
		&content.CourseID,
		&content.Title,
		&content.ContentType,
		&content.Content,
		&description,
		&content.OrderIndex,
		&duration,
		&fileSize,
		&mimeType,
		&content.CreatedAt,
		&content.UpdatedAt,
	)

	if err != nil {
		return nil, repository.ParseError(err)
	}

	content.Description = repository.FromNullString(description)
	content.MimeType = repository.FromNullString(mimeType)

	if duration.Valid {
		d := int(duration.Int32)
		content.Duration = &d
	}
	if fileSize.Valid {
		content.FileSize = &fileSize.Int64
	}

	return &content, nil
}

// ReorderContents updates the order_index for multiple content items
func (r *CourseContentRepository) ReorderContents(ctx context.Context, courseID uuid.UUID, contentIDs []uuid.UUID, tx *sql.Tx) error {
	execer := repository.GetExecer(r.GetDB(), tx)

	for i, contentID := range contentIDs {
		query := `
			UPDATE course_contents
			SET order_index = $1, updated_at = NOW()
			WHERE id = $2 AND course_id = $3
		`
		_, err := execer.ExecContext(ctx, query, i, contentID, courseID)
		if err != nil {
			return repository.ParseError(err)
		}
	}

	return nil
}

// GetNextOrderIndex returns the next order index for a course
func (r *CourseContentRepository) GetNextOrderIndex(ctx context.Context, courseID uuid.UUID) (int, error) {
	query := `SELECT COALESCE(MAX(order_index), -1) + 1 FROM course_contents WHERE course_id = $1`

	var nextIndex int
	err := r.GetDB().QueryRowContext(ctx, query, courseID).Scan(&nextIndex)
	if err != nil {
		return 0, repository.ParseError(err)
	}

	return nextIndex, nil
}

// DeleteByCourse deletes all content items for a course
func (r *CourseContentRepository) DeleteByCourse(ctx context.Context, courseID uuid.UUID, tx *sql.Tx) error {
	query := `DELETE FROM course_contents WHERE course_id = $1`

	execer := repository.GetExecer(r.GetDB(), tx)
	_, err := execer.ExecContext(ctx, query, courseID)
	if err != nil {
		return repository.ParseError(err)
	}

	return nil
}

// CountByCourse returns the number of content items for a course
func (r *CourseContentRepository) CountByCourse(ctx context.Context, courseID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM course_contents WHERE course_id = $1`

	var count int
	err := r.GetDB().QueryRowContext(ctx, query, courseID).Scan(&count)
	if err != nil {
		return 0, repository.ParseError(err)
	}

	return count, nil
}
