package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// ---------- Состав группы ----------

type Student struct {
	ID       int64  `json:"id"`
	GroupID  int64  `json:"group_id"`
	FullName string `json:"full_name"`
	Login    string `json:"login"`
	HasLogin bool   `json:"has_login"`
}

func (s *Store) ListStudents(ctx context.Context, groupID int64) ([]Student, error) {
	const q = `select id, group_id, full_name, coalesce(login, '')
	             from students where group_id = $1 order by full_name`

	rows, err := s.Pool.Query(ctx, q, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Student{}
	for rows.Next() {
		var x Student
		if err := rows.Scan(&x.ID, &x.GroupID, &x.FullName, &x.Login); err != nil {
			return nil, err
		}
		x.HasLogin = x.Login != ""
		out = append(out, x)
	}
	return out, rows.Err()
}

// AddStudents добавляет сразу список имён — состав группы удобнее заводить
// одним списком, а не по одному.
func (s *Store) AddStudents(ctx context.Context, teacherID, groupID int64, names []string) (int, error) {
	var owns bool
	const qOwn = `select exists(select 1 from groups where id = $1 and teacher_id = $2)`
	if err := s.Pool.QueryRow(ctx, qOwn, groupID, teacherID).Scan(&owns); err != nil {
		return 0, err
	}
	if !owns {
		return 0, ErrNotFound
	}

	const q = `insert into students (group_id, full_name) values ($1, $2)
	      on conflict (group_id, full_name) do nothing`

	added := 0
	for _, n := range names {
		tag, err := s.Pool.Exec(ctx, q, groupID, n)
		if err != nil {
			return added, err
		}
		added += int(tag.RowsAffected())
	}
	return added, nil
}

func (s *Store) DeleteStudent(ctx context.Context, teacherID, id int64) error {
	const q = `delete from students s using groups g
	            where s.id = $1 and s.group_id = g.id and g.teacher_id = $2`

	tag, err := s.Pool.Exec(ctx, q, id, teacherID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// StudentByName ищет ученика в составе группы занятия.
func (s *Store) StudentByName(ctx context.Context, groupID int64, name string) (*Student, error) {
	const q = `select id, group_id, full_name, coalesce(login, '')
	             from students where group_id = $1 and lower(full_name) = lower($2)`

	var x Student
	err := s.Pool.QueryRow(ctx, q, groupID, name).Scan(&x.ID, &x.GroupID, &x.FullName, &x.Login)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &x, err
}

// ---------- Журнал ----------

type JournalWork struct {
	SessionID int64     `json:"session_id"`
	Title     string    `json:"title"`
	Mode      string    `json:"mode"`
	CreatedAt time.Time `json:"created_at"`
	MaxScore  float64   `json:"max_score"`
}

type JournalCell struct {
	SessionID   int64    `json:"session_id"`
	Score       *float64 `json:"score"`
	Percent     *float64 `json:"percent"`
	Status      string   `json:"status"`
	Preliminary bool     `json:"preliminary"`
}

type JournalRow struct {
	StudentID int64                  `json:"student_id"`
	Name      string                 `json:"name"`
	Cells     map[int64]*JournalCell `json:"cells"`
	Average   *float64               `json:"average"`
	Done      int                    `json:"done"`
}

type Journal struct {
	GroupID   int64         `json:"group_id"`
	GroupName string        `json:"group_name"`
	Works     []JournalWork `json:"works"`
	Rows      []JournalRow  `json:"rows"`
}

// GroupJournal собирает сквозную ведомость: все работы группы по горизонтали,
// ученики по вертикали. Это то, чего не видно внутри одного занятия.
func (s *Store) GroupJournal(ctx context.Context, teacherID, groupID int64) (*Journal, error) {
	out := &Journal{GroupID: groupID, Works: []JournalWork{}, Rows: []JournalRow{}}

	const qGroup = `select name from groups where id = $1 and teacher_id = $2`
	if err := s.Pool.QueryRow(ctx, qGroup, groupID, teacherID).Scan(&out.GroupName); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Работы группы: занятия с тестом, у которых состав уже зафиксирован.
	const qWorks = `select s.id, coalesce(t.title, l.title, 'Sapak'), s.mode, s.created_at,
	                       coalesce(s.frozen_questions, '[]'::jsonb)
	                  from sessions s
	                  left join tests t on t.id = s.test_id
	                  left join lessons l on l.id = s.lesson_id
	                 where s.group_id = $1 and s.test_id is not null
	                 order by s.created_at`

	rows, err := s.Pool.Query(ctx, qWorks, groupID)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var w JournalWork
		var blob []byte
		if err := rows.Scan(&w.SessionID, &w.Title, &w.Mode, &w.CreatedAt, &blob); err != nil {
			rows.Close()
			return nil, err
		}
		w.MaxScore = sumPoints(blob)
		out.Works = append(out.Works, w)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	students, err := s.ListStudents(ctx, groupID)
	if err != nil {
		return nil, err
	}

	maxByWork := make(map[int64]float64, len(out.Works))
	for _, w := range out.Works {
		maxByWork[w.SessionID] = w.MaxScore
	}

	// Баллы всех учеников группы одним запросом.
	const qCells = `select p.student_id, p.session_id,
	                       coalesce(sum(a.final_score), 0),
	                       count(*) filter (where a.needs_manual and a.manual_score is null),
	                       coalesce(sub.status, 'in_progress')
	                  from participants p
	                  join sessions s on s.id = p.session_id
	                  left join answers a on a.participant_id = p.id
	                  left join submissions sub on sub.participant_id = p.id
	                 where s.group_id = $1 and p.student_id is not null
	                 group by p.student_id, p.session_id, sub.status`

	cellRows, err := s.Pool.Query(ctx, qCells, groupID)
	if err != nil {
		return nil, err
	}
	defer cellRows.Close()

	byStudent := map[int64]map[int64]*JournalCell{}
	for cellRows.Next() {
		var sid, sessID int64
		var score float64
		var pending int
		var status string
		if err := cellRows.Scan(&sid, &sessID, &score, &pending, &status); err != nil {
			return nil, err
		}

		cell := &JournalCell{SessionID: sessID, Score: &score, Status: status,
			Preliminary: pending > 0}
		if m := maxByWork[sessID]; m > 0 {
			pct := score / m * 100
			cell.Percent = &pct
		}
		if byStudent[sid] == nil {
			byStudent[sid] = map[int64]*JournalCell{}
		}
		byStudent[sid][sessID] = cell
	}
	if err := cellRows.Err(); err != nil {
		return nil, err
	}

	for _, st := range students {
		row := JournalRow{StudentID: st.ID, Name: st.FullName,
			Cells: byStudent[st.ID]}
		if row.Cells == nil {
			row.Cells = map[int64]*JournalCell{}
		}

		sum := 0.0
		for _, c := range row.Cells {
			if c.Percent != nil {
				sum += *c.Percent
				row.Done++
			}
		}
		if row.Done > 0 {
			avg := sum / float64(row.Done)
			row.Average = &avg
		}
		out.Rows = append(out.Rows, row)
	}
	return out, nil
}

// sumPoints считает максимальный балл по замороженному составу работы.
func sumPoints(blob []byte) float64 {
	var qs []FrozenQuestion
	if len(blob) == 0 || jsonUnmarshal(blob, &qs) != nil {
		return 0
	}
	total := 0.0
	for _, q := range qs {
		total += q.Points
	}
	return total
}

func jsonUnmarshal(b []byte, v any) error { return json.Unmarshal(b, v) }
