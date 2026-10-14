package store

import (
	"context"
	crand "crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	ErrJoinClosed   = errors.New("sapaga goşulmak ýapyk")
	ErrTestNotLive  = errors.New("synag işjeň däl")
	ErrDeadlinePast = errors.New("synagyň möhleti tamamlandy")
	ErrNotInRoster  = errors.New("adyňyz toparyň sanawynda ýok")
)

// FrozenQuestion — снимок вопроса на момент старта теста. Последующие правки
// в банке вопросов не меняют результаты уже сданных работ (раздел 6 ТЗ).
type FrozenQuestion struct {
	ID          int64           `json:"id"`
	Kind        string          `json:"kind"`
	Prompt      string          `json:"prompt"`
	Payload     json.RawMessage `json:"payload"`
	AnswerKey   json.RawMessage `json:"answer_key"`
	Points      float64         `json:"points"`
	Hint        string          `json:"hint"`
	Explanation string          `json:"explanation"`
	Position    int             `json:"position"`
}

// ForStudent отдаёт вопрос без ключа правильных ответов.
func (q FrozenQuestion) ForStudent() map[string]any {
	return map[string]any{
		"id":       q.ID,
		"kind":     q.Kind,
		"prompt":   q.Prompt,
		"payload":  q.Payload,
		"points":   q.Points,
		"hint":     q.Hint,
		"position": q.Position,
	}
}

// StartTest фиксирует состав теста и ставит срок по серверным часам.
func (s *Store) StartTest(ctx context.Context, sessionID int64) (*Session, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Блокируем строку занятия: FOR UPDATE нельзя применять к внешнему
	// соединению, поэтому длительность теста читаем отдельным запросом.
	var testID *int64
	const qLock = `select test_id from sessions where id = $1 for update`
	if err := tx.QueryRow(ctx, qLock, sessionID).Scan(&testID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if testID == nil {
		return nil, errors.New("sapaga synag berkidilmedik")
	}

	var duration int
	var shuffleQuestions, shuffleOptions bool
	const qDuration = `select duration_sec, shuffle_questions, shuffle_options
	                     from tests where id = $1`
	if err := tx.QueryRow(ctx, qDuration, *testID).
		Scan(&duration, &shuffleQuestions, &shuffleOptions); err != nil {
		return nil, err
	}

	const qQuestions = `select q.id, q.kind, q.prompt, q.payload, q.answer_key,
	                           coalesce(tq.points_override, q.points), q.hint,
	                           q.explanation, tq.position
	                      from test_questions tq
	                      join questions q on q.id = tq.question_id
	                     where tq.test_id = $1
	                     order by tq.position, q.id`

	rows, err := tx.Query(ctx, qQuestions, *testID)
	if err != nil {
		return nil, err
	}

	var frozen []FrozenQuestion
	for rows.Next() {
		var q FrozenQuestion
		if err := rows.Scan(&q.ID, &q.Kind, &q.Prompt, &q.Payload, &q.AnswerKey,
			&q.Points, &q.Hint, &q.Explanation, &q.Position); err != nil {
			rows.Close()
			return nil, err
		}
		frozen = append(frozen, q)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(frozen) == 0 {
		return nil, errors.New("synagda sorag ýok")
	}

	if shuffleQuestions {
		shuffle(len(frozen), func(i, j int) { frozen[i], frozen[j] = frozen[j], frozen[i] })
		for i := range frozen {
			frozen[i].Position = i + 1
		}
	}
	if shuffleOptions {
		for i := range frozen {
			shuffleChoiceOptions(&frozen[i])
		}
	}

	blob, err := json.Marshal(frozen)
	if err != nil {
		return nil, err
	}

	const qStart = `update sessions
	                   set state = 'test',
	                       test_started_at = now(),
	                       test_deadline = now() + make_interval(secs => $2),
	                       frozen_questions = $3
	                 where id = $1`
	if _, err := tx.Exec(ctx, qStart, sessionID, duration, blob); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.SessionByID(ctx, sessionID)
}

func (s *Store) FrozenQuestions(ctx context.Context, sessionID int64) ([]FrozenQuestion, error) {
	const q = `select coalesce(frozen_questions, '[]'::jsonb) from sessions where id = $1`

	var blob []byte
	if err := s.Pool.QueryRow(ctx, q, sessionID).Scan(&blob); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	var out []FrozenQuestion
	if err := json.Unmarshal(blob, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ---------- Участники ----------

func (s *Store) JoinParticipant(ctx context.Context, sess *Session, name string) (*Participant, error) {
	if !sess.JoinOpen {
		return nil, ErrJoinClosed
	}

	token, err := randomToken()
	if err != nil {
		return nil, err
	}

	// Sapak topara berkidilen bolsa, goşulýan adam şol toparyň sanawynda
	// bolmaly: şonda onuň işleri žurnalda bir okuwça ýygnanýar.
	var studentID *int64
	if sess.GroupID != nil {
		st, err := s.StudentByName(ctx, *sess.GroupID, name)
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotInRoster
		}
		if err != nil {
			return nil, err
		}
		studentID = &st.ID
		name = st.FullName
	}

	const q = `insert into participants (session_id, display_name, token, student_id)
	           values ($1, $2, $3, $4)
	        returning id, session_id, display_name, token, connected, joined_at`

	// Одинаковые имена встречаются, поэтому тёзку не отбиваем ошибкой,
	// а нумеруем: ученик не должен застрять на входе.
	candidate := name
	for attempt := 1; attempt <= 20; attempt++ {
		var p Participant
		var joined time.Time
		err := s.Pool.QueryRow(ctx, q, sess.ID, candidate, token, studentID).
			Scan(&p.ID, &p.SessionID, &p.DisplayName, &p.Token, &p.Connected, &joined)
		if err == nil {
			return &p, nil
		}
		if !isUniqueViolation(err) {
			return nil, err
		}
		candidate = fmt.Sprintf("%s (%d)", name, attempt+1)
	}
	return nil, errors.New("at saýlap bolmady")
}

func (s *Store) ParticipantByToken(ctx context.Context, token string) (*Participant, error) {
	const q = `select id, session_id, display_name, token, connected, last_seen, started_at
	             from participants where token = $1`

	var p Participant
	err := s.Pool.QueryRow(ctx, q, token).
		Scan(&p.ID, &p.SessionID, &p.DisplayName, &p.Token, &p.Connected, &p.LastSeen, &p.StartedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &p, err
}

func (s *Store) SetPresence(ctx context.Context, participantID int64, online bool) error {
	const q = `update participants set connected = $2, last_seen = now() where id = $1`
	_, err := s.Pool.Exec(ctx, q, participantID, online)
	return err
}

func (s *Store) ListParticipants(ctx context.Context, sessionID int64) ([]Participant, error) {
	const q = `select p.id, p.session_id, p.display_name, p.connected, p.last_seen,
	                  (select count(*) from answers a where a.participant_id = p.id),
	                  coalesce((select sub.status = 'submitted' from submissions sub
	                             where sub.participant_id = p.id), false)
	             from participants p
	            where p.session_id = $1
	            order by p.display_name`

	rows, err := s.Pool.Query(ctx, q, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Participant{}
	for rows.Next() {
		var p Participant
		if err := rows.Scan(&p.ID, &p.SessionID, &p.DisplayName, &p.Connected,
			&p.LastSeen, &p.Answered, &p.Submitted); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ---------- Ответы ----------

type SaveResult struct {
	Status      string  `json:"status"`
	Score       float64 `json:"-"`
	NeedsManual bool    `json:"-"`
	Duplicate   bool    `json:"duplicate"`
}

// SaveAnswer идемпотентен по client_key: повторная отправка после обрыва сети
// не создаёт второй ответ и не засчитывается дважды.
func (s *Store) SaveAnswer(ctx context.Context, sessionID, participantID, questionID int64,
	clientKey string, value json.RawMessage, questions []FrozenQuestion) (*SaveResult, error) {

	var q *FrozenQuestion
	for i := range questions {
		if questions[i].ID == questionID {
			q = &questions[i]
			break
		}
	}
	if q == nil {
		return nil, ErrNotFound
	}

	ratio, correct, needsManual := Grade(q.Kind, q.AnswerKey, value)
	score := ratio * q.Points

	var autoScore, finalScore *float64
	var isCorrect *bool
	if !needsManual {
		autoScore, finalScore, isCorrect = &score, &score, &correct
	}

	// Повторная отправка того же client_key не меняет строку:
	// условие where отсекает дубликат, а returning показывает, была ли запись.
	const q1 = `insert into answers (session_id, participant_id, question_id, client_key,
	                                 value, status, is_correct, auto_score, final_score, needs_manual)
	            values ($1, $2, $3, $4, $5, 'saved', $6, $7, $8, $9)
	       on conflict (participant_id, question_id) do update
	               set client_key   = excluded.client_key,
	                   value        = excluded.value,
	                   is_correct   = excluded.is_correct,
	                   auto_score   = excluded.auto_score,
	                   final_score  = coalesce(answers.manual_score, excluded.final_score),
	                   needs_manual = excluded.needs_manual,
	                   updated_at   = now()
	             where answers.client_key <> excluded.client_key
	         returning id`

	var id int64
	err := s.Pool.QueryRow(ctx, q1, sessionID, participantID, questionID, clientKey,
		value, isCorrect, autoScore, finalScore, needsManual).Scan(&id)

	if errors.Is(err, pgx.ErrNoRows) {
		// Строка уже содержит этот client_key — ответ принят ранее.
		return &SaveResult{Status: "saved", Duplicate: true}, nil
	}
	if err != nil {
		return nil, err
	}
	return &SaveResult{Status: "saved", Score: score, NeedsManual: needsManual}, nil
}

func (s *Store) SubmitWork(ctx context.Context, sessionID, participantID int64, late bool) error {
	status := "submitted"
	if late {
		status = "late"
	}

	const q = `insert into submissions (session_id, participant_id, status, submitted_at)
	           values ($1, $2, $3, now())
	      on conflict (session_id, participant_id) do update
	              set status = excluded.status, submitted_at = now()`
	if _, err := s.Pool.Exec(ctx, q, sessionID, participantID, status); err != nil {
		return err
	}

	const qa = `update answers set status = 'submitted', updated_at = now()
	             where participant_id = $1`
	_, err := s.Pool.Exec(ctx, qa, participantID)
	return err
}

// AnswersOf возвращает сохранённые ответы ученика — для восстановления работы
// после обрыва связи.
func (s *Store) AnswersOf(ctx context.Context, participantID int64) (map[int64]json.RawMessage, error) {
	const q = `select question_id, value from answers where participant_id = $1`

	rows, err := s.Pool.Query(ctx, q, participantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[int64]json.RawMessage{}
	for rows.Next() {
		var qid int64
		var val json.RawMessage
		if err := rows.Scan(&qid, &val); err != nil {
			return nil, err
		}
		out[qid] = val
	}
	return out, rows.Err()
}

func (s *Store) ParticipantByID(ctx context.Context, id int64) (*Participant, error) {
	const q = `select id, session_id, display_name, token, connected, last_seen, started_at
	             from participants where id = $1`

	var p Participant
	err := s.Pool.QueryRow(ctx, q, id).
		Scan(&p.ID, &p.SessionID, &p.DisplayName, &p.Token, &p.Connected, &p.LastSeen, &p.StartedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &p, err
}

// shuffle — перестановка Фишера — Йетса на криптографическом источнике.
func shuffle(n int, swap func(i, j int)) {
	for i := n - 1; i > 0; i-- {
		j, err := crand.Int(crand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return
		}
		swap(i, int(j.Int64()))
	}
}

// shuffleChoiceOptions меняет порядок вариантов. Ключ правильных ответов
// хранит идентификаторы, а не позиции, поэтому оценка не страдает.
func shuffleChoiceOptions(q *FrozenQuestion) {
	if q.Kind != KindChoice {
		return
	}

	var payload struct {
		Options  []json.RawMessage `json:"options"`
		Multiple bool              `json:"multiple"`
	}
	if json.Unmarshal(q.Payload, &payload) != nil || len(payload.Options) < 2 {
		return
	}

	shuffle(len(payload.Options), func(i, j int) {
		payload.Options[i], payload.Options[j] = payload.Options[j], payload.Options[i]
	})

	if blob, err := json.Marshal(payload); err == nil {
		q.Payload = blob
	}
}

// GrantExtraTime продлевает срок теста. Приём опоздавшей работы и выдача
// дополнительного времени — отдельное действие учителя (раздел 6 ТЗ).
func (s *Store) GrantExtraTime(ctx context.Context, sessionID int64, seconds int) error {
	const q = `update sessions
	              set test_deadline = greatest(test_deadline, now()) + make_interval(secs => $2)
	            where id = $1 and test_deadline is not null`

	tag, err := s.Pool.Exec(ctx, q, sessionID, seconds)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ReleaseResults открывает ученикам их собственные результаты.
func (s *Store) ReleaseResults(ctx context.Context, sessionID int64) error {
	const q = `update sessions set state = 'closed' where id = $1`
	_, err := s.Pool.Exec(ctx, q, sessionID)
	return err
}

// MyResult — то, что видит сам ученик: только своя работа.
type MyResult struct {
	Score       float64 `json:"score"`
	MaxScore    float64 `json:"max_score"`
	Percent     float64 `json:"percent"`
	Preliminary bool    `json:"preliminary"`
	Status      string  `json:"status"`
}

func (s *Store) MyResult(ctx context.Context, sessionID, participantID int64) (*MyResult, error) {
	questions, err := s.FrozenQuestions(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	out := &MyResult{Status: "in_progress"}
	for _, q := range questions {
		out.MaxScore += q.Points
	}

	const q = `select coalesce(sum(a.final_score), 0),
	                  count(*) filter (where a.needs_manual and a.manual_score is null),
	                  coalesce((select status from submissions
	                             where participant_id = $1), 'in_progress')
	             from answers a where a.participant_id = $1`

	var pending int
	if err := s.Pool.QueryRow(ctx, q, participantID).
		Scan(&out.Score, &pending, &out.Status); err != nil {
		return nil, err
	}

	out.Preliminary = pending > 0
	if out.MaxScore > 0 {
		out.Percent = out.Score / out.MaxScore * 100
	}
	return out, nil
}

// freezeQuestions делает снимок состава теста. Используется и живым занятием
// (в момент старта), и домашним заданием (в момент создания).
func (s *Store) freezeQuestions(ctx context.Context, sessionID, testID int64) error {
	var shuffleQuestions, shuffleOptions bool
	const qCfg = `select shuffle_questions, shuffle_options from tests where id = $1`
	if err := s.Pool.QueryRow(ctx, qCfg, testID).Scan(&shuffleQuestions, &shuffleOptions); err != nil {
		return err
	}

	const qQuestions = `select q.id, q.kind, q.prompt, q.payload, q.answer_key,
	                           coalesce(tq.points_override, q.points), q.hint,
	                           q.explanation, tq.position
	                      from test_questions tq
	                      join questions q on q.id = tq.question_id
	                     where tq.test_id = $1
	                     order by tq.position, q.id`

	rows, err := s.Pool.Query(ctx, qQuestions, testID)
	if err != nil {
		return err
	}

	var frozen []FrozenQuestion
	for rows.Next() {
		var q FrozenQuestion
		if err := rows.Scan(&q.ID, &q.Kind, &q.Prompt, &q.Payload, &q.AnswerKey,
			&q.Points, &q.Hint, &q.Explanation, &q.Position); err != nil {
			rows.Close()
			return err
		}
		frozen = append(frozen, q)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if len(frozen) == 0 {
		return errors.New("synagda sorag ýok")
	}

	if shuffleQuestions {
		shuffle(len(frozen), func(i, j int) { frozen[i], frozen[j] = frozen[j], frozen[i] })
		for i := range frozen {
			frozen[i].Position = i + 1
		}
	}
	if shuffleOptions {
		for i := range frozen {
			shuffleChoiceOptions(&frozen[i])
		}
	}

	blob, err := json.Marshal(frozen)
	if err != nil {
		return err
	}

	_, err = s.Pool.Exec(ctx, `update sessions set frozen_questions = $2 where id = $1`,
		sessionID, blob)
	return err
}

// MarkStarted фиксирует момент, когда ученик открыл домашнее задание.
// Личный срок отсчитывается отсюда.
func (s *Store) MarkStarted(ctx context.Context, participantID int64) (*time.Time, error) {
	const q = `update participants set started_at = coalesce(started_at, now())
	            where id = $1 returning started_at`

	var at *time.Time
	err := s.Pool.QueryRow(ctx, q, participantID).Scan(&at)
	return at, err
}

// EffectiveDeadline — личный срок ученика.
// Живое занятие: общий срок, который задал учитель.
// Домашнее задание: начало работы + длительность, но не позже закрытия.
func EffectiveDeadline(sess *Session, p *Participant) *time.Time {
	if !sess.IsHomework() {
		return sess.Deadline
	}
	if p == nil || p.StartedAt == nil {
		return sess.ClosesAt
	}

	personal := p.StartedAt.Add(time.Duration(sess.DurationSec) * time.Second)
	if sess.ClosesAt != nil && sess.ClosesAt.Before(personal) {
		return sess.ClosesAt
	}
	return &personal
}

// SubmissionGate сообщает, можно ли ученику ещё менять ответы.
// Если работа сдана, а повторная попытка не разрешена, правки запрещены.
func (s *Store) SubmissionGate(ctx context.Context, sess *Session, participantID int64) (allowed bool, err error) {
	var submitted bool
	const qSub = `select exists(select 1 from submissions
	                             where participant_id = $1 and status in ('submitted', 'late'))`
	if err := s.Pool.QueryRow(ctx, qSub, participantID).Scan(&submitted); err != nil {
		return false, err
	}
	if !submitted {
		return true, nil
	}
	if sess.TestID == nil {
		return false, nil
	}

	var retake bool
	const qTest = `select allow_retake from tests where id = $1`
	if err := s.Pool.QueryRow(ctx, qTest, *sess.TestID).Scan(&retake); err != nil {
		return false, err
	}
	return retake, nil
}
