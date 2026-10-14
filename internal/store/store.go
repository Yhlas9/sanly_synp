// Package store — доступ к PostgreSQL на чистом SQL, без ORM.
package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("tapylmady")

type Store struct{ Pool *pgxpool.Pool }

func Open(ctx context.Context, dsn string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("dsn: %w", err)
	}
	// 30 учеников одновременно — норматив приёмки; запас держим вчетверо.
	cfg.MaxConns = 40
	cfg.MinConns = 4

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}
	return &Store{Pool: pool}, nil
}

func (s *Store) Close() { s.Pool.Close() }

// Migrate прогоняет .sql файлы каталога по порядку имён.
func (s *Store) Migrate(ctx context.Context, dir string) error {
	entries, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		return err
	}
	sort.Strings(entries)

	for _, path := range entries {
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if _, err := s.Pool.Exec(ctx, string(body)); err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(path), err)
		}
	}
	return nil
}

// ---------- Типы ----------

type Teacher struct {
	ID           int64  `json:"id"`
	Login        string `json:"login"`
	FullName     string `json:"full_name"`
	Role         string `json:"role"`
	PasswordHash string `json:"-"`
}

type Subject struct {
	ID     int64  `json:"id"`
	Code   string `json:"code"`
	NameTK string `json:"name_tk"`
}

type Lesson struct {
	ID            int64           `json:"id"`
	SubjectID     int64           `json:"subject_id"`
	SubjectCode   string          `json:"subject_code"`
	Title         string          `json:"title"`
	Topic         string          `json:"topic"`
	Goal          string          `json:"goal"`
	BodyMD        string          `json:"body_md"`
	DemoKind      string          `json:"demo_kind"`
	DemoConfig    json.RawMessage `json:"demo_config"`
	PracticalTask string          `json:"practical_task"`
	TeacherID     int64           `json:"teacher_id"`
	IsSample      bool            `json:"is_sample"`
	Mine          bool            `json:"mine"`
}

// Question несёт answer_key, который никогда не сериализуется клиенту:
// json-тег "-" закрывает ключ правильных ответов на уровне типа.
type Question struct {
	ID          int64           `json:"id"`
	SubjectID   int64           `json:"subject_id"`
	Kind        string          `json:"kind"`
	Prompt      string          `json:"prompt"`
	Payload     json.RawMessage `json:"payload"`
	AnswerKey   json.RawMessage `json:"-"`
	Points      float64         `json:"points"`
	Hint        string          `json:"hint"`
	Explanation string          `json:"explanation"`
	Position    int             `json:"position"`
}

type Session struct {
	ID         int64      `json:"id"`
	TeacherID  int64      `json:"teacher_id"`
	LessonID   *int64     `json:"lesson_id"`
	TestID     *int64     `json:"test_id"`
	JoinCode   string     `json:"join_code"`
	State      string     `json:"state"`
	JoinOpen   bool       `json:"join_open"`
	StartedAt  *time.Time `json:"test_started_at"`
	Deadline   *time.Time `json:"test_deadline"`
	TestTitle  string     `json:"test_title"`
	LessonName string     `json:"lesson_title"`

	// Mode: "live" — mugallym synagy özi başladýar;
	//       "homework" — synag bellenen aralykda hemişe açyk.
	Mode        string     `json:"mode"`
	OpensAt     *time.Time `json:"opens_at"`
	ClosesAt    *time.Time `json:"closes_at"`
	DurationSec int        `json:"duration_sec"`

	GroupID   *int64 `json:"group_id"`
	GroupName string `json:"group_name"`
}

// IsHomework — öý işi görnüşindäki tabşyryk.
func (s Session) IsHomework() bool { return s.Mode == "homework" }

// Window — öý işiniň açyk wagty gelip ýetdimi.
func (s Session) Window(now time.Time) (open bool, reason string) {
	if s.OpensAt != nil && now.Before(*s.OpensAt) {
		return false, "tabşyryk entek açylmady"
	}
	if s.ClosesAt != nil && now.After(*s.ClosesAt) {
		return false, "tabşyrygyň möhleti tamamlandy"
	}
	return true, ""
}

type Participant struct {
	ID          int64      `json:"id"`
	SessionID   int64      `json:"session_id"`
	DisplayName string     `json:"display_name"`
	Token       string     `json:"-"`
	Connected   bool       `json:"connected"`
	LastSeen    *time.Time `json:"last_seen"`
	Answered    int        `json:"answered"`
	Submitted   bool       `json:"submitted"`
	// StartedAt — öý işinde okuwçynyň işe başlan pursaty: şondan
	// başlap onuň şahsy möhleti hasaplanýar.
	StartedAt *time.Time `json:"started_at"`
}

// ---------- Учителя ----------

func (s *Store) TeacherByLogin(ctx context.Context, login string) (*Teacher, error) {
	const q = `select id, login, password_hash, full_name, role
	             from teachers where login = $1`

	var t Teacher
	err := s.Pool.QueryRow(ctx, q, login).
		Scan(&t.ID, &t.Login, &t.PasswordHash, &t.FullName, &t.Role)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &t, err
}

func (s *Store) TeacherByID(ctx context.Context, id int64) (*Teacher, error) {
	const q = `select id, login, password_hash, full_name, role
	             from teachers where id = $1`

	var t Teacher
	err := s.Pool.QueryRow(ctx, q, id).
		Scan(&t.ID, &t.Login, &t.PasswordHash, &t.FullName, &t.Role)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &t, err
}

// ---------- Предметы и уроки ----------

func (s *Store) ListSubjects(ctx context.Context) ([]Subject, error) {
	const q = `select id, code, name_tk from subjects order by sort_order, id`

	rows, err := s.Pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Subject
	for rows.Next() {
		var x Subject
		if err := rows.Scan(&x.ID, &x.Code, &x.NameTK); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

// ListLessons отдаёт собственные уроки учителя плюс образцовые,
// которые идут с платформой. Чужие уроки не видны.
func (s *Store) ListLessons(ctx context.Context, teacherID int64, subjectCode string) ([]Lesson, error) {
	const q = `select l.id, l.subject_id, s.code, l.title, l.topic, l.goal,
	                  l.body_md, l.demo_kind, l.demo_config, l.practical_task,
	                  l.teacher_id, l.is_sample
	             from lessons l
	             join subjects s on s.id = l.subject_id
	            where ($2 = '' or s.code = $2)
	              and (l.teacher_id = $1 or l.is_sample)
	            order by l.is_sample, s.sort_order, l.id`

	rows, err := s.Pool.Query(ctx, q, teacherID, subjectCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Lesson
	for rows.Next() {
		var x Lesson
		if err := rows.Scan(&x.ID, &x.SubjectID, &x.SubjectCode, &x.Title, &x.Topic,
			&x.Goal, &x.BodyMD, &x.DemoKind, &x.DemoConfig, &x.PracticalTask,
			&x.TeacherID, &x.IsSample); err != nil {
			return nil, err
		}
		x.Mine = x.TeacherID == teacherID
		out = append(out, x)
	}
	return out, rows.Err()
}

func (s *Store) GetLesson(ctx context.Context, id int64) (*Lesson, error) {
	const q = `select l.id, l.subject_id, s.code, l.title, l.topic, l.goal,
	                  l.body_md, l.demo_kind, l.demo_config, l.practical_task,
	                  l.teacher_id, l.is_sample
	             from lessons l
	             join subjects s on s.id = l.subject_id
	            where l.id = $1`

	var x Lesson
	err := s.Pool.QueryRow(ctx, q, id).Scan(&x.ID, &x.SubjectID, &x.SubjectCode,
		&x.Title, &x.Topic, &x.Goal, &x.BodyMD, &x.DemoKind, &x.DemoConfig,
		&x.PracticalTask, &x.TeacherID, &x.IsSample)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &x, err
}

type TestInfo struct {
	ID            int64  `json:"id"`
	Title         string `json:"title"`
	SubjectCode   string `json:"subject_code"`
	DurationSec   int    `json:"duration_sec"`
	QuestionCount int    `json:"question_count"`
	TeacherID     int64  `json:"teacher_id"`
	IsSample      bool   `json:"is_sample"`
	Mine          bool   `json:"mine"`
}

func (s *Store) ListTests(ctx context.Context, teacherID int64) ([]TestInfo, error) {
	const q = `select t.id, t.title, s.code, t.duration_sec,
	                  (select count(*) from test_questions tq where tq.test_id = t.id),
	                  t.teacher_id, t.is_sample
	             from tests t
	             join subjects s on s.id = t.subject_id
	            where t.teacher_id = $1 or t.is_sample
	            order by t.is_sample, s.sort_order, t.id`

	rows, err := s.Pool.Query(ctx, q, teacherID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []TestInfo
	for rows.Next() {
		var x TestInfo
		if err := rows.Scan(&x.ID, &x.Title, &x.SubjectCode, &x.DurationSec,
			&x.QuestionCount, &x.TeacherID, &x.IsSample); err != nil {
			return nil, err
		}
		x.Mine = x.TeacherID == teacherID
		out = append(out, x)
	}
	return out, rows.Err()
}

// ---------- Занятие ----------

// Алфавит кода без похожих знаков: 0/O и 1/I на телефоне путают.
const codeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func randomCode(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	for i := range buf {
		buf[i] = codeAlphabet[int(buf[i])%len(codeAlphabet)]
	}
	return string(buf), nil
}

func randomToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// NewSession — параметры создаваемого занятия.
type NewSession struct {
	LessonID *int64     `json:"lesson_id"`
	TestID   *int64     `json:"test_id"`
	GroupID  *int64     `json:"group_id"`
	Mode     string     `json:"mode"` // live | homework
	OpensAt  *time.Time `json:"opens_at"`
	ClosesAt *time.Time `json:"closes_at"`
}

func (s *Store) CreateSession(ctx context.Context, teacherID int64, in NewSession) (*Session, error) {
	mode := in.Mode
	if mode != "homework" {
		mode = "live"
	}
	// Öý işi döredilen badyna açyk: mugallymyň "başlat" düwmesine basmagy
	// gerek däl, okuwçy möhletiň içinde islän wagty girip bilýär.
	state := "lobby"
	if mode == "homework" {
		state = "test"
	}

	const q = `insert into sessions (teacher_id, lesson_id, test_id, join_code,
	                                 state, mode, opens_at, closes_at, group_id)
	           values ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	        returning id`

	// Код занятия короткий, поэтому на коллизию отвечаем повтором.
	for attempt := 0; attempt < 8; attempt++ {
		code, err := randomCode(5)
		if err != nil {
			return nil, err
		}

		var id int64
		err = s.Pool.QueryRow(ctx, q, teacherID, in.LessonID, in.TestID, code,
			state, mode, in.OpensAt, in.ClosesAt, in.GroupID).Scan(&id)
		if err == nil {
			if mode == "homework" && in.TestID != nil {
				// Состав задания фиксируется сразу: все ученики получат
				// один и тот же набор вопросов, когда бы ни зашли.
				if err := s.freezeQuestions(ctx, id, *in.TestID); err != nil {
					return nil, err
				}
			}
			return s.SessionByID(ctx, id)
		}
		if !isUniqueViolation(err) {
			return nil, err
		}
	}
	return nil, errors.New("sapak kody döredip bolmady")
}

const sessionCols = `s.id, s.teacher_id, s.lesson_id, s.test_id, s.join_code, s.state,
                     s.join_open, s.test_started_at, s.test_deadline,
                     coalesce(t.title, ''), coalesce(l.title, ''),
                     s.mode, s.opens_at, s.closes_at, coalesce(t.duration_sec, 600),
                     s.group_id, coalesce(g.name, '')`

func scanSession(row pgx.Row) (*Session, error) {
	var x Session
	err := row.Scan(&x.ID, &x.TeacherID, &x.LessonID, &x.TestID, &x.JoinCode, &x.State,
		&x.JoinOpen, &x.StartedAt, &x.Deadline, &x.TestTitle, &x.LessonName,
		&x.Mode, &x.OpensAt, &x.ClosesAt, &x.DurationSec, &x.GroupID, &x.GroupName)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &x, err
}

func (s *Store) SessionByID(ctx context.Context, id int64) (*Session, error) {
	q := `select ` + sessionCols + `
	        from sessions s
	        left join tests t on t.id = s.test_id
	        left join lessons l on l.id = s.lesson_id
	        left join groups g on g.id = s.group_id
	       where s.id = $1`
	return scanSession(s.Pool.QueryRow(ctx, q, id))
}

func (s *Store) SessionByCode(ctx context.Context, code string) (*Session, error) {
	q := `select ` + sessionCols + `
	        from sessions s
	        left join tests t on t.id = s.test_id
	        left join lessons l on l.id = s.lesson_id
	        left join groups g on g.id = s.group_id
	       where s.join_code = $1`
	return scanSession(s.Pool.QueryRow(ctx, q, code))
}

func (s *Store) SetJoinOpen(ctx context.Context, sessionID int64, open bool) error {
	const q = `update sessions set join_open = $2 where id = $1`
	_, err := s.Pool.Exec(ctx, q, sessionID, open)
	return err
}

func (s *Store) SetSessionState(ctx context.Context, sessionID int64, state string) error {
	const q = `update sessions set state = $2 where id = $1`
	_, err := s.Pool.Exec(ctx, q, sessionID, state)
	return err
}

func isUniqueViolation(err error) bool {
	var pgErr interface{ SQLState() string }
	return errors.As(err, &pgErr) && pgErr.SQLState() == "23505"
}

// randRead вынесен отдельно, чтобы генераторы паролей и кодов
// пользовались одним источником случайности.
func randRead(b []byte) (int, error) { return rand.Read(b) }
