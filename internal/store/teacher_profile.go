package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

var ErrWrongPassword = errors.New("häzirki açar söz nädogry")

// TeacherStats — то, что учитель сделал в системе.
type TeacherStats struct {
	Lessons   int `json:"lessons"`
	Questions int `json:"questions"`
	Tests     int `json:"tests"`
	Groups    int `json:"groups"`
	Students  int `json:"students"`
	Sessions  int `json:"sessions"`
	Answers   int `json:"answers"`
}

// TeacherWork — одно проведённое занятие в истории учителя.
type TeacherWork struct {
	SessionID    int64     `json:"session_id"`
	Title        string    `json:"title"`
	Subject      string    `json:"subject"`
	GroupName    string    `json:"group_name"`
	Mode         string    `json:"mode"`
	State        string    `json:"state"`
	Date         time.Time `json:"date"`
	Participants int       `json:"participants"`
	Submitted    int       `json:"submitted"`
	Average      *float64  `json:"average"`
}

type TeacherProfile struct {
	Teacher Teacher       `json:"teacher"`
	Stats   TeacherStats  `json:"stats"`
	Works   []TeacherWork `json:"works"`
}

func (s *Store) TeacherProfile(ctx context.Context, teacherID int64) (*TeacherProfile, error) {
	t, err := s.TeacherByID(ctx, teacherID)
	if err != nil {
		return nil, err
	}
	out := &TeacherProfile{Teacher: *t, Works: []TeacherWork{}}

	const qStats = `select
	   (select count(*) from lessons   where teacher_id = $1),
	   (select count(*) from questions q join lessons l on l.id = q.lesson_id
	                                     where l.teacher_id = $1),
	   (select count(*) from tests     where teacher_id = $1),
	   (select count(*) from groups    where teacher_id = $1),
	   (select count(*) from students st join groups g on g.id = st.group_id
	                                     where g.teacher_id = $1),
	   (select count(*) from sessions  where teacher_id = $1),
	   (select count(*) from answers a join sessions se on se.id = a.session_id
	                                   where se.teacher_id = $1)`

	if err := s.Pool.QueryRow(ctx, qStats, teacherID).Scan(
		&out.Stats.Lessons, &out.Stats.Questions, &out.Stats.Tests, &out.Stats.Groups,
		&out.Stats.Students, &out.Stats.Sessions, &out.Stats.Answers); err != nil {
		return nil, err
	}

	// История занятий: сколько подключилось, сколько сдало и средний процент.
	const qWorks = `select se.id, coalesce(t.title, l.title, 'Sapak'),
	                       coalesce(sub.name_tk, ''), coalesce(g.name, ''),
	                       se.mode, se.state, se.created_at,
	                       coalesce(se.frozen_questions, '[]'::jsonb),
	                       (select count(*) from participants p where p.session_id = se.id),
	                       (select count(*) from submissions sm
	                         where sm.session_id = se.id and sm.status in ('submitted', 'late')),
	                       (select coalesce(sum(a.final_score), 0) from answers a
	                         where a.session_id = se.id)
	                  from sessions se
	                  left join tests t on t.id = se.test_id
	                  left join lessons l on l.id = se.lesson_id
	                  left join subjects sub on sub.id = coalesce(t.subject_id, l.subject_id)
	                  left join groups g on g.id = se.group_id
	                 where se.teacher_id = $1
	                 order by se.created_at desc
	                 limit 50`

	rows, err := s.Pool.Query(ctx, qWorks, teacherID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var w TeacherWork
		var blob []byte
		var totalScore float64
		if err := rows.Scan(&w.SessionID, &w.Title, &w.Subject, &w.GroupName,
			&w.Mode, &w.State, &w.Date, &blob,
			&w.Participants, &w.Submitted, &totalScore); err != nil {
			return nil, err
		}

		// Средний процент считаем только когда есть и работы, и баллы.
		if maxScore := sumPoints(blob); maxScore > 0 && w.Participants > 0 {
			avg := totalScore / (maxScore * float64(w.Participants)) * 100
			w.Average = &avg
		}
		out.Works = append(out.Works, w)
	}
	return out, rows.Err()
}

// ChangeOwnPassword меняет пароль самому себе: нужно знать текущий.
func (s *Store) ChangeOwnPassword(ctx context.Context, teacherID int64, current, next string) error {
	var hash string
	const q = `select password_hash from teachers where id = $1`
	if err := s.Pool.QueryRow(ctx, q, teacherID).Scan(&hash); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(current)) != nil {
		return ErrWrongPassword
	}
	return s.SetPassword(ctx, teacherID, next)
}
