package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

var ErrLastAdmin = errors.New("ulgamda iň bolmanda bir dolandyryjy galmaly")

type TeacherRow struct {
	ID        int64     `json:"id"`
	Login     string    `json:"login"`
	FullName  string    `json:"full_name"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	Lessons   int       `json:"lesson_count"`
	Sessions  int       `json:"session_count"`
	Groups    int       `json:"group_count"`
	Students  int       `json:"student_count"`
}

func (s *Store) ListTeachers(ctx context.Context) ([]TeacherRow, error) {
	const q = `select t.id, t.login, t.full_name, t.role, t.created_at,
	                  (select count(*) from lessons l where l.teacher_id = t.id),
	                  (select count(*) from sessions se where se.teacher_id = t.id),
	                  (select count(*) from groups g where g.teacher_id = t.id),
	                  (select count(*) from students st join groups g on g.id = st.group_id
	                    where g.teacher_id = t.id)
	             from teachers t order by (t.role = 'admin') desc, t.login`

	rows, err := s.Pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []TeacherRow{}
	for rows.Next() {
		var x TeacherRow
		if err := rows.Scan(&x.ID, &x.Login, &x.FullName, &x.Role,
			&x.CreatedAt, &x.Lessons, &x.Sessions, &x.Groups, &x.Students); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (s *Store) CreateTeacher(ctx context.Context, login, password, fullName, role string) (*TeacherRow, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	if role != "admin" {
		role = "teacher"
	}

	const q = `insert into teachers (login, password_hash, full_name, role)
	           values ($1, $2, $3, $4)
	        returning id, login, full_name, role, created_at`

	var x TeacherRow
	err = s.Pool.QueryRow(ctx, q, login, string(hash), fullName, role).
		Scan(&x.ID, &x.Login, &x.FullName, &x.Role, &x.CreatedAt)
	if isUniqueViolation(err) {
		return nil, errors.New("beýle login eýýäm bar")
	}
	return &x, err
}

func (s *Store) SetPassword(ctx context.Context, id int64, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	tag, err := s.Pool.Exec(ctx,
		`update teachers set password_hash = $2 where id = $1`, id, string(hash))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteTeacher не даёт удалить последнего администратора: иначе в систему
// станет некому войти.
func (s *Store) DeleteTeacher(ctx context.Context, id int64) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var role string
	if err := tx.QueryRow(ctx, `select role from teachers where id = $1 for update`, id).
		Scan(&role); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}

	if role == "admin" {
		var admins int
		if err := tx.QueryRow(ctx, `select count(*) from teachers where role = 'admin'`).
			Scan(&admins); err != nil {
			return err
		}
		if admins <= 1 {
			return ErrLastAdmin
		}
	}

	if _, err := tx.Exec(ctx, `delete from teachers where id = $1`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Stats — краткая сводка для панели администратора.
type Stats struct {
	Teachers  int `json:"teachers"`
	Groups    int `json:"groups"`
	Students  int `json:"students"`
	Lessons   int `json:"lessons"`
	Questions int `json:"questions"`
	Tests     int `json:"tests"`
	Sessions  int `json:"sessions"`
	Answers   int `json:"answers"`
	Uploads   int `json:"uploads"`
}

func (s *Store) Stats(ctx context.Context) (*Stats, error) {
	const q = `select (select count(*) from teachers),  (select count(*) from groups),
	                  (select count(*) from students),  (select count(*) from lessons),
	                  (select count(*) from questions), (select count(*) from tests),
	                  (select count(*) from sessions),  (select count(*) from answers),
	                  (select count(*) from uploads)`

	var x Stats
	err := s.Pool.QueryRow(ctx, q).Scan(&x.Teachers, &x.Groups, &x.Students, &x.Lessons,
		&x.Questions, &x.Tests, &x.Sessions, &x.Answers, &x.Uploads)
	return &x, err
}
