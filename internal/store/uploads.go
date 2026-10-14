package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

type Upload struct {
	ID            string    `json:"id"`
	SessionID     int64     `json:"session_id"`
	ParticipantID int64     `json:"participant_id"`
	FileName      string    `json:"file_name"`
	ContentType   string    `json:"content_type"`
	SizeBytes     int64     `json:"size_bytes"`
	StoredPath    string    `json:"-"`
	CreatedAt     time.Time `json:"created_at"`
}

func (s *Store) AddUpload(ctx context.Context, u Upload) error {
	const q = `insert into uploads (id, session_id, participant_id, file_name,
	                                content_type, size_bytes, stored_path)
	           values ($1, $2, $3, $4, $5, $6, $7)`

	_, err := s.Pool.Exec(ctx, q, u.ID, u.SessionID, u.ParticipantID, u.FileName,
		u.ContentType, u.SizeBytes, u.StoredPath)
	return err
}

func (s *Store) GetUpload(ctx context.Context, id string) (*Upload, error) {
	const q = `select id, session_id, participant_id, file_name, content_type,
	                  size_bytes, stored_path, created_at
	             from uploads where id = $1`

	var u Upload
	err := s.Pool.QueryRow(ctx, q, id).Scan(&u.ID, &u.SessionID, &u.ParticipantID,
		&u.FileName, &u.ContentType, &u.SizeBytes, &u.StoredPath, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &u, err
}

// NewUploadID — случайный идентификатор файла: имя на диске не зависит от
// того, что прислал ученик.
func NewUploadID() (string, error) { return randomToken() }
