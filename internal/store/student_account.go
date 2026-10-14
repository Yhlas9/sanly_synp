package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

var ErrNoAccount = errors.New("okuwçynyň hasaby ýok")

// StudentAccount — то, что видит сам ученик о себе.
type StudentAccount struct {
	ID        int64  `json:"id"`
	FullName  string `json:"full_name"`
	Login     string `json:"login"`
	GroupID   int64  `json:"group_id"`
	GroupName string `json:"group_name"`
}

// translit переводит туркменские буквы в латиницу без диакритики,
// чтобы логин набирался с любой клавиатуры.
var translitMap = map[rune]string{
	'ä': "a", 'ç': "c", 'ž': "j", 'ň': "n", 'ö': "o",
	'ş': "s", 'ü': "u", 'ý': "y",
}

func slugify(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case translitMap[r] != "":
			b.WriteString(translitMap[r])
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case unicode.IsSpace(r), r == '-':
			b.WriteByte('.')
		}
	}

	out := strings.Trim(b.String(), ".")
	for strings.Contains(out, "..") {
		out = strings.ReplaceAll(out, "..", ".")
	}
	if out == "" {
		out = "okuwcy"
	}
	return out
}

// Açar sözde biri-birine meňzeş harplar ulanylmaýar: 0/O, 1/l.
const pwAlphabet = "abcdefghjkmnpqrstuvwxyz23456789"

func randomPassword(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := randRead(buf); err != nil {
		return "", err
	}
	for i := range buf {
		buf[i] = pwAlphabet[int(buf[i])%len(pwAlphabet)]
	}
	return string(buf), nil
}

// Credential — выданные ученику данные для входа. Пароль возвращается
// один раз, в открытом виде: учитель распечатывает и раздаёт.
type Credential struct {
	StudentID int64  `json:"student_id"`
	FullName  string `json:"full_name"`
	Login     string `json:"login"`
	Password  string `json:"password"`
}

// CreateAccounts выдаёт учётные записи ученикам группы учителя.
func (s *Store) CreateAccounts(ctx context.Context, teacherID, groupID int64) ([]Credential, error) {
	var owns bool
	const qOwn = `select exists(select 1 from groups where id = $1 and teacher_id = $2)`
	if err := s.Pool.QueryRow(ctx, qOwn, groupID, teacherID).Scan(&owns); err != nil {
		return nil, err
	}
	if !owns {
		return nil, ErrNotFound
	}
	return s.createAccountsForGroup(ctx, groupID)
}

// AdminCreateAccounts — то же самое, но для администратора: он отвечает
// за все группы, поэтому проверка владения не нужна.
func (s *Store) AdminCreateAccounts(ctx context.Context, groupID int64) ([]Credential, error) {
	var exists bool
	if err := s.Pool.QueryRow(ctx,
		`select exists(select 1 from groups where id = $1)`, groupID).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNotFound
	}
	return s.createAccountsForGroup(ctx, groupID)
}

func (s *Store) createAccountsForGroup(ctx context.Context, groupID int64) ([]Credential, error) {
	const qList = `select id, full_name from students
	                where group_id = $1 and login is null order by full_name`

	rows, err := s.Pool.Query(ctx, qList, groupID)
	if err != nil {
		return nil, err
	}

	type pending struct {
		id   int64
		name string
	}
	var todo []pending
	for rows.Next() {
		var p pending
		if err := rows.Scan(&p.id, &p.name); err != nil {
			rows.Close()
			return nil, err
		}
		todo = append(todo, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]Credential, 0, len(todo))
	for _, p := range todo {
		pw, err := randomPassword(6)
		if err != nil {
			return nil, err
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}

		base := slugify(p.name)
		// Meňzeş atlar duşsa, logina san goşulýar.
		var login string
		for attempt := 0; attempt < 30; attempt++ {
			login = base
			if attempt > 0 {
				login = fmt.Sprintf("%s%d", base, attempt+1)
			}
			const qSet = `update students set login = $2, password_hash = $3
			               where id = $1 and login is null`
			_, err := s.Pool.Exec(ctx, qSet, p.id, login, string(hash))
			if err == nil {
				break
			}
			if !isUniqueViolation(err) {
				return nil, err
			}
			login = ""
		}
		if login == "" {
			return nil, errors.New("login saýlap bolmady: " + p.name)
		}

		out = append(out, Credential{StudentID: p.id, FullName: p.name, Login: login, Password: pw})
	}
	return out, nil
}

// ResetAccount выдаёт новый пароль ученику, который свой забыл.
func (s *Store) ResetAccount(ctx context.Context, teacherID, studentID int64) (*Credential, error) {
	const qGet = `select s.full_name, s.login from students s
	              join groups g on g.id = s.group_id
	             where s.id = $1 and g.teacher_id = $2`

	var name string
	var login *string
	if err := s.Pool.QueryRow(ctx, qGet, studentID, teacherID).Scan(&name, &login); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if login == nil {
		return nil, ErrNoAccount
	}

	pw, err := randomPassword(6)
	if err != nil {
		return nil, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	if _, err := s.Pool.Exec(ctx,
		`update students set password_hash = $2 where id = $1`, studentID, string(hash)); err != nil {
		return nil, err
	}
	return &Credential{StudentID: studentID, FullName: name, Login: *login, Password: pw}, nil
}

func (s *Store) StudentLogin(ctx context.Context, login, password string) (*StudentAccount, error) {
	const q = `select s.id, s.full_name, s.login, s.password_hash, s.group_id, g.name
	             from students s join groups g on g.id = s.group_id
	            where lower(s.login) = lower($1)`

	var a StudentAccount
	var hash string
	err := s.Pool.QueryRow(ctx, q, login).
		Scan(&a.ID, &a.FullName, &a.Login, &hash, &a.GroupID, &a.GroupName)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return nil, ErrNotFound
	}

	_, _ = s.Pool.Exec(ctx, `update students set last_login_at = now() where id = $1`, a.ID)
	return &a, nil
}

func (s *Store) StudentAccountByID(ctx context.Context, id int64) (*StudentAccount, error) {
	const q = `select s.id, s.full_name, coalesce(s.login, ''), s.group_id, g.name
	             from students s join groups g on g.id = s.group_id
	            where s.id = $1`

	var a StudentAccount
	err := s.Pool.QueryRow(ctx, q, id).
		Scan(&a.ID, &a.FullName, &a.Login, &a.GroupID, &a.GroupName)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &a, err
}

// ---------- Личный кабинет ----------

type MyWork struct {
	SessionID   int64      `json:"session_id"`
	Title       string     `json:"title"`
	Subject     string     `json:"subject"`
	Mode        string     `json:"mode"`
	Date        time.Time  `json:"date"`
	Score       float64    `json:"score"`
	MaxScore    float64    `json:"max_score"`
	Percent     float64    `json:"percent"`
	Status      string     `json:"status"`
	Preliminary bool       `json:"preliminary"`
	SubmittedAt *time.Time `json:"submitted_at"`
	Released    bool       `json:"released"`
}

type MyProfile struct {
	Account StudentAccount `json:"account"`
	Works   []MyWork       `json:"works"`
	Average *float64       `json:"average"`
	Done    int            `json:"done"`
}

// MyProfile собирает историю ученика по всем его работам.
// Результат показывается только по тем занятиям, которые учитель закрыл.
func (s *Store) MyProfile(ctx context.Context, studentID int64) (*MyProfile, error) {
	acc, err := s.StudentAccountByID(ctx, studentID)
	if err != nil {
		return nil, err
	}
	out := &MyProfile{Account: *acc, Works: []MyWork{}}

	const q = `select se.id, coalesce(t.title, l.title, 'Sapak'),
	                  coalesce(sub2.name_tk, ''), se.mode, se.created_at, se.state,
	                  coalesce(se.frozen_questions, '[]'::jsonb),
	                  coalesce(sum(a.final_score), 0),
	                  count(*) filter (where a.needs_manual and a.manual_score is null),
	                  coalesce(subm.status, 'in_progress'), subm.submitted_at
	             from participants p
	             join sessions se on se.id = p.session_id
	             left join tests t on t.id = se.test_id
	             left join lessons l on l.id = se.lesson_id
	             left join subjects sub2 on sub2.id = coalesce(t.subject_id, l.subject_id)
	             left join answers a on a.participant_id = p.id
	             left join submissions subm on subm.participant_id = p.id
	            where p.student_id = $1 and se.test_id is not null
	            group by se.id, t.title, l.title, sub2.name_tk, se.mode,
	                     se.created_at, se.state, se.frozen_questions,
	                     subm.status, subm.submitted_at
	            order by se.created_at desc`

	rows, err := s.Pool.Query(ctx, q, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sum := 0.0
	for rows.Next() {
		var w MyWork
		var state string
		var blob []byte
		var pending int
		if err := rows.Scan(&w.SessionID, &w.Title, &w.Subject, &w.Mode, &w.Date, &state,
			&blob, &w.Score, &pending, &w.Status, &w.SubmittedAt); err != nil {
			return nil, err
		}

		w.MaxScore = sumPoints(blob)
		w.Preliminary = pending > 0
		w.Released = state == "closed"
		if w.MaxScore > 0 {
			w.Percent = w.Score / w.MaxScore * 100
		}
		if w.Released {
			sum += w.Percent
			out.Done++
		}
		out.Works = append(out.Works, w)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if out.Done > 0 {
		avg := sum / float64(out.Done)
		out.Average = &avg
	}
	return out, nil
}

// AdminResetAccount — сброс пароля ученика администратором, без проверки
// принадлежности группы.
func (s *Store) AdminResetAccount(ctx context.Context, studentID int64) (*Credential, error) {
	const qGet = `select full_name, login from students where id = $1`

	var name string
	var login *string
	if err := s.Pool.QueryRow(ctx, qGet, studentID).Scan(&name, &login); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if login == nil {
		return nil, ErrNoAccount
	}

	pw, err := randomPassword(6)
	if err != nil {
		return nil, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	if _, err := s.Pool.Exec(ctx,
		`update students set password_hash = $2 where id = $1`, studentID, string(hash)); err != nil {
		return nil, err
	}
	return &Credential{StudentID: studentID, FullName: name, Login: *login, Password: pw}, nil
}

// AdminStudent — строка общего списка учеников для администратора.
type AdminStudent struct {
	ID          int64      `json:"id"`
	FullName    string     `json:"full_name"`
	Login       string     `json:"login"`
	HasLogin    bool       `json:"has_login"`
	GroupID     int64      `json:"group_id"`
	GroupName   string     `json:"group_name"`
	TeacherName string     `json:"teacher_name"`
	Works       int        `json:"works"`
	LastLoginAt *time.Time `json:"last_login_at"`
}

func (s *Store) ListAllStudents(ctx context.Context) ([]AdminStudent, error) {
	const q = `select st.id, st.full_name, coalesce(st.login, ''),
	                  g.id, g.name, t.full_name, st.last_login_at,
	                  (select count(*) from participants p where p.student_id = st.id)
	             from students st
	             join groups g on g.id = st.group_id
	             join teachers t on t.id = g.teacher_id
	            order by g.name, st.full_name`

	rows, err := s.Pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []AdminStudent{}
	for rows.Next() {
		var x AdminStudent
		if err := rows.Scan(&x.ID, &x.FullName, &x.Login, &x.GroupID, &x.GroupName,
			&x.TeacherName, &x.LastLoginAt, &x.Works); err != nil {
			return nil, err
		}
		x.HasLogin = x.Login != ""
		out = append(out, x)
	}
	return out, rows.Err()
}

// AdminGroup — группа в общем списке: чья она и сколько в ней учеников.
type AdminGroup struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	TeacherID   int64  `json:"teacher_id"`
	TeacherName string `json:"teacher_name"`
	Students    int    `json:"students"`
	WithAccount int    `json:"with_account"`
}

func (s *Store) ListAllGroups(ctx context.Context) ([]AdminGroup, error) {
	const q = `select g.id, g.name, t.id, t.full_name,
	                  (select count(*) from students st where st.group_id = g.id),
	                  (select count(*) from students st
	                    where st.group_id = g.id and st.login is not null)
	             from groups g join teachers t on t.id = g.teacher_id
	            order by t.full_name, g.name`

	rows, err := s.Pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []AdminGroup{}
	for rows.Next() {
		var x AdminGroup
		if err := rows.Scan(&x.ID, &x.Name, &x.TeacherID, &x.TeacherName,
			&x.Students, &x.WithAccount); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

// NewPassword отдаёт случайный пароль наружу: им пользуется и выдача
// учётных записей ученикам, и создание учителя администратором.
func NewPassword() (string, error) { return randomPassword(8) }
