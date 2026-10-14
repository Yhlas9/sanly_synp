package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// ResultRow — строка итоговой таблицы из раздела 7 ТЗ.
type ResultRow struct {
	ParticipantID int64      `json:"participant_id"`
	Name          string     `json:"name"`
	Score         float64    `json:"score"`
	MaxScore      float64    `json:"max_score"`
	Percent       float64    `json:"percent"`
	SubmittedAt   *time.Time `json:"submitted_at"`
	Status        string     `json:"status"`
	PendingManual int        `json:"pending_manual"`
	// Preliminary — «deslapky»: есть задания, ожидающие ручной проверки.
	Preliminary bool `json:"preliminary"`
}

type ResultsPayload struct {
	SessionID  int64       `json:"session_id"`
	TestTitle  string      `json:"test_title"`
	MaxScore   float64     `json:"max_score"`
	Rows       []ResultRow `json:"rows"`
	HardestQs  []HardQ     `json:"hardest_questions"`
	TotalTaken int         `json:"total_taken"`
}

// HardQ — вопрос, который чаще всего заваливают: раздел 7 требует показывать
// темы для повторного объяснения, одного общего балла недостаточно.
type HardQ struct {
	QuestionID  int64   `json:"question_id"`
	Prompt      string  `json:"prompt"`
	WrongCount  int     `json:"wrong_count"`
	TotalCount  int     `json:"total_count"`
	WrongShare  float64 `json:"wrong_share"`
	Explanation string  `json:"explanation"`
}

func (s *Store) Results(ctx context.Context, sessionID int64) (*ResultsPayload, error) {
	sess, err := s.SessionByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	questions, err := s.FrozenQuestions(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	maxScore := 0.0
	prompts := make(map[int64]FrozenQuestion, len(questions))
	for _, q := range questions {
		maxScore += q.Points
		prompts[q.ID] = q
	}

	const q = `select p.id, p.display_name,
	                  coalesce(sum(a.final_score), 0),
	                  count(*) filter (where a.needs_manual and a.manual_score is null),
	                  sub.status, sub.submitted_at
	             from participants p
	             left join answers a on a.participant_id = p.id
	             left join submissions sub on sub.participant_id = p.id
	            where p.session_id = $1
	            group by p.id, p.display_name, sub.status, sub.submitted_at
	            order by coalesce(sum(a.final_score), 0) desc, p.display_name`

	rows, err := s.Pool.Query(ctx, q, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := &ResultsPayload{
		SessionID: sessionID,
		TestTitle: sess.TestTitle,
		MaxScore:  maxScore,
		Rows:      []ResultRow{},
	}

	for rows.Next() {
		var r ResultRow
		var status *string
		if err := rows.Scan(&r.ParticipantID, &r.Name, &r.Score,
			&r.PendingManual, &status, &r.SubmittedAt); err != nil {
			return nil, err
		}

		r.MaxScore = maxScore
		if maxScore > 0 {
			r.Percent = r.Score / maxScore * 100
		}
		r.Status = "in_progress"
		if status != nil {
			r.Status = *status
			out.TotalTaken++
		}
		r.Preliminary = r.PendingManual > 0

		out.Rows = append(out.Rows, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out.HardestQs, err = s.hardestQuestions(ctx, sessionID, prompts)
	return out, err
}

func (s *Store) hardestQuestions(ctx context.Context, sessionID int64,
	prompts map[int64]FrozenQuestion) ([]HardQ, error) {

	const q = `select a.question_id,
	                  count(*) filter (where a.is_correct is false) as wrong,
	                  count(*) as total
	             from answers a
	            where a.session_id = $1 and a.is_correct is not null
	            group by a.question_id
	           having count(*) filter (where a.is_correct is false) > 0
	            order by wrong desc, total desc
	            limit 5`

	rows, err := s.Pool.Query(ctx, q, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []HardQ{}
	for rows.Next() {
		var h HardQ
		if err := rows.Scan(&h.QuestionID, &h.WrongCount, &h.TotalCount); err != nil {
			return nil, err
		}
		if src, ok := prompts[h.QuestionID]; ok {
			h.Prompt, h.Explanation = src.Prompt, src.Explanation
		}
		if h.TotalCount > 0 {
			h.WrongShare = float64(h.WrongCount) / float64(h.TotalCount) * 100
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// AnswerDetail — разбор по одному ученику: его ответ, правильный, балл, пояснение.
type AnswerDetail struct {
	AnswerID   int64  `json:"answer_id"`
	QuestionID int64  `json:"question_id"`
	Kind       string `json:"kind"`
	Prompt     string `json:"prompt"`
	// Payload нужен, чтобы показать текст вариантов, а не их идентификаторы:
	// в отчёте «b» ничего не значит, а «I = U / R» — значит.
	Payload     json.RawMessage `json:"payload"`
	Value       json.RawMessage `json:"value"`
	AnswerKey   json.RawMessage `json:"answer_key"`
	Points      float64         `json:"points"`
	Score       *float64        `json:"score"`
	IsCorrect   *bool           `json:"is_correct"`
	NeedsManual bool            `json:"needs_manual"`
	Explanation string          `json:"explanation"`
	Comment     string          `json:"teacher_comment"`
}

// StudentDetail отдаёт разбор работы. Вызывается только учителем: здесь
// ключ правильных ответов раскрывается намеренно.
func (s *Store) StudentDetail(ctx context.Context, sessionID, participantID int64) ([]AnswerDetail, error) {
	questions, err := s.FrozenQuestions(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	byID := make(map[int64]FrozenQuestion, len(questions))
	for _, q := range questions {
		byID[q.ID] = q
	}

	const q = `select a.id, a.question_id, a.value, a.final_score, a.is_correct,
	                  a.needs_manual, a.teacher_comment
	             from answers a
	            where a.participant_id = $1
	            order by a.question_id`

	rows, err := s.Pool.Query(ctx, q, participantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []AnswerDetail{}
	for rows.Next() {
		var d AnswerDetail
		if err := rows.Scan(&d.AnswerID, &d.QuestionID, &d.Value, &d.Score,
			&d.IsCorrect, &d.NeedsManual, &d.Comment); err != nil {
			return nil, err
		}
		if src, ok := byID[d.QuestionID]; ok {
			d.Kind, d.Prompt, d.Points = src.Kind, src.Prompt, src.Points
			d.AnswerKey, d.Explanation = src.AnswerKey, src.Explanation
			d.Payload = src.Payload
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// AdjustGrade сохраняет старую оценку, новую и причину (требование раздела 7).
func (s *Store) AdjustGrade(ctx context.Context, answerID int64, newScore float64, reason, comment string) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var old *float64
	const qOld = `select final_score from answers where id = $1 for update`
	if err := tx.QueryRow(ctx, qOld, answerID).Scan(&old); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}

	const qUpd = `update answers
	                 set manual_score = $2, final_score = $2,
	                     teacher_comment = $3, needs_manual = false, updated_at = now()
	               where id = $1`
	if _, err := tx.Exec(ctx, qUpd, answerID, newScore, comment); err != nil {
		return err
	}

	const qLog = `insert into grade_adjustments (answer_id, old_score, new_score, reason)
	              values ($1, $2, $3, $4)`
	if _, err := tx.Exec(ctx, qLog, answerID, old, newScore, reason); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
