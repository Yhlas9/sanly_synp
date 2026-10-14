package store

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
)

// ---------- Доступ к материалам ----------

// ErrReadOnly — материал видно, но менять его нельзя: он чужой или образцовый.
var ErrReadOnly = errors.New("bu material diňe okamak üçin")

// lessonAccess сообщает, может ли учитель читать и менять урок.
func (s *Store) lessonAccess(ctx context.Context, teacherID, lessonID int64) (read, write bool, err error) {
	const q = `select teacher_id, is_sample from lessons where id = $1`

	var owner int64
	var sample bool
	if err := s.Pool.QueryRow(ctx, q, lessonID).Scan(&owner, &sample); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, false, ErrNotFound
		}
		return false, false, err
	}
	own := owner == teacherID
	return own || sample, own && !sample, nil
}

// questionAccess — права на вопрос определяются правами на его урок.
func (s *Store) questionAccess(ctx context.Context, teacherID, questionID int64) (read, write bool, err error) {
	const q = `select coalesce(l.teacher_id, 0), coalesce(l.is_sample, false)
	             from questions q left join lessons l on l.id = q.lesson_id
	            where q.id = $1`

	var owner int64
	var sample bool
	if err := s.Pool.QueryRow(ctx, q, questionID).Scan(&owner, &sample); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, false, ErrNotFound
		}
		return false, false, err
	}
	own := owner == teacherID
	return own || sample, own && !sample, nil
}

// ---------- Группы ----------

type Group struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"student_count"`
}

func (s *Store) ListGroups(ctx context.Context, teacherID int64) ([]Group, error) {
	const q = `select g.id, g.name,
	                  (select count(*) from students s where s.group_id = g.id)
	             from groups g where g.teacher_id = $1 order by g.name`

	rows, err := s.Pool.Query(ctx, q, teacherID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Group{}
	for rows.Next() {
		var g Group
		if err := rows.Scan(&g.ID, &g.Name, &g.Count); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// CreateGroup идемпотентна: повторный ввод того же имени возвращает
// существующую группу, а не плодит дубликат с отдельным журналом.
func (s *Store) CreateGroup(ctx context.Context, teacherID int64, name string) (*Group, error) {
	const q = `with upsert as (
	             insert into groups (teacher_id, name) values ($1, $2)
	        on conflict (teacher_id, name) do update set name = excluded.name
	          returning id, name)
	           select u.id, u.name,
	                  (select count(*) from students s where s.group_id = u.id)
	             from upsert u`

	var g Group
	err := s.Pool.QueryRow(ctx, q, teacherID, name).Scan(&g.ID, &g.Name, &g.Count)
	return &g, err
}

func (s *Store) DeleteGroup(ctx context.Context, teacherID, id int64) error {
	const q = `delete from groups where id = $1 and teacher_id = $2`
	_, err := s.Pool.Exec(ctx, q, id, teacherID)
	return err
}

// ---------- Уроки ----------

type LessonInput struct {
	SubjectID     int64           `json:"subject_id"`
	Title         string          `json:"title"`
	Topic         string          `json:"topic"`
	Goal          string          `json:"goal"`
	BodyMD        string          `json:"body_md"`
	DemoKind      string          `json:"demo_kind"`
	DemoConfig    json.RawMessage `json:"demo_config"`
	PracticalTask string          `json:"practical_task"`
}

func (in LessonInput) config() json.RawMessage {
	if len(in.DemoConfig) == 0 {
		return json.RawMessage(`{}`)
	}
	return in.DemoConfig
}

func (s *Store) CreateLesson(ctx context.Context, teacherID int64, in LessonInput) (*Lesson, error) {
	const q = `insert into lessons (teacher_id, subject_id, title, topic, goal, body_md,
	                                demo_kind, demo_config, practical_task)
	           values ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	        returning id`

	var id int64
	err := s.Pool.QueryRow(ctx, q, teacherID, in.SubjectID, in.Title, in.Topic, in.Goal,
		in.BodyMD, in.DemoKind, in.config(), in.PracticalTask).Scan(&id)
	if err != nil {
		return nil, err
	}
	return s.GetLesson(ctx, id)
}

func (s *Store) UpdateLesson(ctx context.Context, teacherID, id int64, in LessonInput) (*Lesson, error) {
	const q = `update lessons
	              set subject_id = $3, title = $4, topic = $5, goal = $6, body_md = $7,
	                  demo_kind = $8, demo_config = $9, practical_task = $10, updated_at = now()
	            where id = $1 and teacher_id = $2 and not is_sample`

	tag, err := s.Pool.Exec(ctx, q, id, teacherID, in.SubjectID, in.Title, in.Topic, in.Goal,
		in.BodyMD, in.DemoKind, in.config(), in.PracticalTask)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return s.getOwnLesson(ctx, teacherID, id)
}

// CopyLesson создаёт независимую копию урока вместе с его вопросами —
// ТЗ требует возможности переиспользовать и изменять готовый урок.
func (s *Store) CopyLesson(ctx context.Context, teacherID, id int64) (*Lesson, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const qLesson = `insert into lessons (teacher_id, subject_id, title, topic, goal, body_md,
	                                      demo_kind, demo_config, practical_task)
	                 select $2, subject_id, title || ' (nusga)', topic, goal, body_md,
	                        demo_kind, demo_config, practical_task
	                   from lessons where id = $1
	              returning id`

	var newID int64
	if err := tx.QueryRow(ctx, qLesson, id, teacherID).Scan(&newID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	const qQuestions = `insert into questions (lesson_id, subject_id, kind, prompt, payload,
	                                           answer_key, points, hint, explanation, position)
	                    select $2, subject_id, kind, prompt, payload, answer_key, points,
	                           hint, explanation, position
	                      from questions where lesson_id = $1`
	if _, err := tx.Exec(ctx, qQuestions, id, newID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.getOwnLesson(ctx, teacherID, newID)
}

func (s *Store) DeleteLesson(ctx context.Context, teacherID, id int64) error {
	const q = `delete from lessons where id = $1 and teacher_id = $2 and not is_sample`

	tag, err := s.Pool.Exec(ctx, q, id, teacherID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ---------- Вопросы ----------

type QuestionInput struct {
	LessonID    *int64          `json:"lesson_id"`
	SubjectID   int64           `json:"subject_id"`
	Kind        string          `json:"kind"`
	Prompt      string          `json:"prompt"`
	Payload     json.RawMessage `json:"payload"`
	AnswerKey   json.RawMessage `json:"answer_key"`
	Points      float64         `json:"points"`
	Hint        string          `json:"hint"`
	Explanation string          `json:"explanation"`
	Position    int             `json:"position"`
}

func raw(v json.RawMessage) json.RawMessage {
	if len(v) == 0 {
		return json.RawMessage(`{}`)
	}
	return v
}

// AuthoredQuestion отдаётся только учителю, поэтому ключ ответов здесь открыт.
type AuthoredQuestion struct {
	Question
	LessonID  *int64          `json:"lesson_id"`
	KeyJSON   json.RawMessage `json:"answer_key"`
	LessonTtl string          `json:"lesson_title"`
}

func (s *Store) ListQuestions(ctx context.Context, teacherID, lessonID int64) ([]AuthoredQuestion, error) {
	read, _, err := s.lessonAccess(ctx, teacherID, lessonID)
	if err != nil {
		return nil, err
	}
	if !read {
		return nil, ErrNotFound
	}

	const q = `select q.id, q.lesson_id, q.subject_id, q.kind, q.prompt, q.payload,
	                  q.answer_key, q.points, q.hint, q.explanation, q.position
	             from questions q
	            where q.lesson_id = $1
	            order by q.position, q.id`

	rows, err := s.Pool.Query(ctx, q, lessonID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []AuthoredQuestion{}
	for rows.Next() {
		var x AuthoredQuestion
		if err := rows.Scan(&x.ID, &x.LessonID, &x.SubjectID, &x.Kind, &x.Prompt,
			&x.Payload, &x.KeyJSON, &x.Points, &x.Hint, &x.Explanation, &x.Position); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (s *Store) CreateQuestion(ctx context.Context, teacherID int64, in QuestionInput) (int64, error) {
	if in.LessonID == nil {
		return 0, ErrNotFound
	}
	read, write, err := s.lessonAccess(ctx, teacherID, *in.LessonID)
	if err != nil {
		return 0, err
	}
	if !read {
		return 0, ErrNotFound // başga mugallymyň sapagy — görünmeýär
	}
	if !write {
		return 0, ErrReadOnly // nusgalyk sapak — diňe okamak üçin
	}

	const q = `insert into questions (lesson_id, subject_id, kind, prompt, payload,
	                                  answer_key, points, hint, explanation, position)
	           values ($1, $2, $3, $4, $5, $6, $7, $8, $9,
	                   coalesce(nullif($10, 0),
	                            (select coalesce(max(position), 0) + 1 from questions where lesson_id = $1)))
	        returning id`

	var id int64
	err = s.Pool.QueryRow(ctx, q, in.LessonID, in.SubjectID, in.Kind, in.Prompt,
		raw(in.Payload), raw(in.AnswerKey), in.Points, in.Hint, in.Explanation, in.Position).Scan(&id)
	return id, err
}

func (s *Store) UpdateQuestion(ctx context.Context, teacherID, id int64, in QuestionInput) error {
	if read, write, err := s.questionAccess(ctx, teacherID, id); err != nil {
		return err
	} else if !read {
		return ErrNotFound
	} else if !write {
		return ErrReadOnly
	}

	const q = `update questions
	              set kind = $2, prompt = $3, payload = $4, answer_key = $5,
	                  points = $6, hint = $7, explanation = $8
	            where id = $1`

	tag, err := s.Pool.Exec(ctx, q, id, in.Kind, in.Prompt, raw(in.Payload),
		raw(in.AnswerKey), in.Points, in.Hint, in.Explanation)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteQuestion(ctx context.Context, teacherID, id int64) error {
	if read, write, err := s.questionAccess(ctx, teacherID, id); err != nil {
		return err
	} else if !read {
		return ErrNotFound
	} else if !write {
		return ErrReadOnly
	}
	_, err := s.Pool.Exec(ctx, `delete from questions where id = $1`, id)
	return err
}

// ---------- Тесты ----------

type TestInput struct {
	SubjectID        int64   `json:"subject_id"`
	Title            string  `json:"title"`
	DurationSec      int     `json:"duration_sec"`
	ShuffleQuestions bool    `json:"shuffle_questions"`
	ShuffleOptions   bool    `json:"shuffle_options"`
	AllowRetake      bool    `json:"allow_retake"`
	ResultsPolicy    string  `json:"results_policy"`
	QuestionIDs      []int64 `json:"question_ids"`
}

func (s *Store) SaveTest(ctx context.Context, teacherID int64, id *int64, in TestInput) (int64, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Synaga diňe özüňe elýeterli soraglary goşup bolýar.
	if len(in.QuestionIDs) > 0 {
		const qCheck = `select count(*) from questions q
		                  join lessons l on l.id = q.lesson_id
		                 where q.id = any($1) and (l.teacher_id = $2 or l.is_sample)`
		var ok int
		if err := tx.QueryRow(ctx, qCheck, in.QuestionIDs, teacherID).Scan(&ok); err != nil {
			return 0, err
		}
		if ok != len(in.QuestionIDs) {
			return 0, ErrReadOnly
		}
	}

	var testID int64
	if id == nil {
		const q = `insert into tests (teacher_id, subject_id, title, duration_sec,
		                              shuffle_questions, shuffle_options, allow_retake, results_policy)
		           values ($1, $2, $3, $4, $5, $6, $7, $8) returning id`
		if err := tx.QueryRow(ctx, q, teacherID, in.SubjectID, in.Title, in.DurationSec,
			in.ShuffleQuestions, in.ShuffleOptions, in.AllowRetake, in.ResultsPolicy).Scan(&testID); err != nil {
			return 0, err
		}
	} else {
		testID = *id
		const q = `update tests set subject_id = $3, title = $4, duration_sec = $5,
		                  shuffle_questions = $6, shuffle_options = $7, allow_retake = $8,
		                  results_policy = $9
		            where id = $1 and teacher_id = $2 and not is_sample`
		tag, err := tx.Exec(ctx, q, testID, teacherID, in.SubjectID, in.Title, in.DurationSec,
			in.ShuffleQuestions, in.ShuffleOptions, in.AllowRetake, in.ResultsPolicy)
		if err != nil {
			return 0, err
		}
		if tag.RowsAffected() == 0 {
			return 0, ErrNotFound
		}
		if _, err := tx.Exec(ctx, `delete from test_questions where test_id = $1`, testID); err != nil {
			return 0, err
		}
	}

	const qLink = `insert into test_questions (test_id, question_id, position)
	               values ($1, $2, $3) on conflict do nothing`
	for i, qid := range in.QuestionIDs {
		if _, err := tx.Exec(ctx, qLink, testID, qid, i+1); err != nil {
			return 0, err
		}
	}
	return testID, tx.Commit(ctx)
}

// TestDetail — состав теста для редактирования.
type TestDetail struct {
	TestInfo
	ShuffleQuestions bool    `json:"shuffle_questions"`
	ShuffleOptions   bool    `json:"shuffle_options"`
	AllowRetake      bool    `json:"allow_retake"`
	ResultsPolicy    string  `json:"results_policy"`
	QuestionIDs      []int64 `json:"question_ids"`
}

func (s *Store) GetTest(ctx context.Context, teacherID, id int64) (*TestDetail, error) {
	const q = `select t.id, t.title, s.code, t.duration_sec, t.shuffle_questions,
	                  t.shuffle_options, t.allow_retake, t.results_policy,
	                  t.teacher_id, t.is_sample
	             from tests t join subjects s on s.id = t.subject_id
	            where t.id = $1 and (t.teacher_id = $2 or t.is_sample)`

	var d TestDetail
	err := s.Pool.QueryRow(ctx, q, id, teacherID).Scan(&d.ID, &d.Title, &d.SubjectCode,
		&d.DurationSec, &d.ShuffleQuestions, &d.ShuffleOptions, &d.AllowRetake,
		&d.ResultsPolicy, &d.TeacherID, &d.IsSample)
	d.Mine = d.TeacherID == teacherID
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	rows, err := s.Pool.Query(ctx,
		`select question_id from test_questions where test_id = $1 order by position`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	d.QuestionIDs = []int64{}
	for rows.Next() {
		var qid int64
		if err := rows.Scan(&qid); err != nil {
			return nil, err
		}
		d.QuestionIDs = append(d.QuestionIDs, qid)
	}
	d.QuestionCount = len(d.QuestionIDs)
	return &d, rows.Err()
}

func (s *Store) DeleteTest(ctx context.Context, teacherID, id int64) error {
	tag, err := s.Pool.Exec(ctx,
		`delete from tests where id = $1 and teacher_id = $2 and not is_sample`, id, teacherID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// getOwnLesson отдаёт урок с проставленным признаком владения.
func (s *Store) getOwnLesson(ctx context.Context, teacherID, id int64) (*Lesson, error) {
	l, err := s.GetLesson(ctx, id)
	if err != nil {
		return nil, err
	}
	l.Mine = l.TeacherID == teacherID
	return l, nil
}
