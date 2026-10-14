package api

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"sanlysynp/internal/hub"
	"sanlysynp/internal/store"
)

// ---------- Занятие: сторона учителя ----------

func (s *Server) createSession(w http.ResponseWriter, r *http.Request) {
	var in store.NewSession
	if err := readJSON(r, &in); err != nil {
		fail(w, http.StatusBadRequest, "Maglumat nädogry", "Sapagy we synagy saýlaň")
		return
	}
	if in.Mode == "homework" && in.TestID == nil {
		fail(w, http.StatusBadRequest, "Öý işi üçin synag gerek", "Synagy saýlaň")
		return
	}

	sess, err := s.st.CreateSession(r.Context(), teacherOf(r).ID, in)
	if err != nil {
		fail(w, http.StatusInternalServerError, "Sapak döredilmedi: "+err.Error(), "Gaýtadan synanyşyň")
		return
	}
	writeJSON(w, http.StatusOK, sess)
}

func (s *Server) loadOwnedSession(w http.ResponseWriter, r *http.Request) *store.Session {
	id, err := pathInt(r, "id")
	if err != nil {
		fail(w, http.StatusBadRequest, "Sapagyň belgisi nädogry", "Sapaklar sahypasyna dolanyň")
		return nil
	}

	sess, err := s.st.SessionByID(r.Context(), id)
	if err != nil {
		fail(w, http.StatusNotFound, "Sapak tapylmady", "Sapaklar sahypasyna dolanyň")
		return nil
	}
	// Права проверяет сервер, а не скрытая кнопка в интерфейсе.
	if sess.TeacherID != teacherOf(r).ID {
		fail(w, http.StatusForbidden, "Bu sapak size degişli däl", "Öz sapaklaryňyzy açyň")
		return nil
	}
	return sess
}

func (s *Server) getSession(w http.ResponseWriter, r *http.Request) {
	if sess := s.loadOwnedSession(w, r); sess != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"session":     sess,
			"server_time": time.Now(),
			"online":      s.hb.CountStudents(sess.ID),
		})
	}
}

func (s *Server) participants(w http.ResponseWriter, r *http.Request) {
	sess := s.loadOwnedSession(w, r)
	if sess == nil {
		return
	}

	list, err := s.st.ListParticipants(r.Context(), sess.ID)
	if err != nil {
		fail(w, http.StatusInternalServerError, "Okuwçylary okap bolmady", "Sahypany täzeläň")
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) joinOpen(w http.ResponseWriter, r *http.Request) {
	sess := s.loadOwnedSession(w, r)
	if sess == nil {
		return
	}

	var in struct {
		Open bool `json:"open"`
	}
	if err := readJSON(r, &in); err != nil {
		fail(w, http.StatusBadRequest, "Maglumat nädogry", "Gaýtadan synanyşyň")
		return
	}
	if err := s.st.SetJoinOpen(r.Context(), sess.ID, in.Open); err != nil {
		fail(w, http.StatusInternalServerError, "Üýtgedip bolmady", "Gaýtadan synanyşyň")
		return
	}

	s.hb.Broadcast(sess.ID, hub.Event{Type: "join_open", Data: in.Open})
	writeJSON(w, http.StatusOK, map[string]any{"join_open": in.Open})
}

func (s *Server) setState(w http.ResponseWriter, r *http.Request) {
	sess := s.loadOwnedSession(w, r)
	if sess == nil {
		return
	}

	var in struct {
		State string `json:"state"`
	}
	if err := readJSON(r, &in); err != nil {
		fail(w, http.StatusBadRequest, "Maglumat nädogry", "Gaýtadan synanyşyň")
		return
	}
	switch in.State {
	case "lobby", "lesson", "test", "closed":
	default:
		fail(w, http.StatusBadRequest, "Ýagdaý nädogry", "lobby, lesson, test ýa-da closed saýlaň")
		return
	}

	if err := s.st.SetSessionState(r.Context(), sess.ID, in.State); err != nil {
		fail(w, http.StatusInternalServerError, "Üýtgedip bolmady", "Gaýtadan synanyşyň")
		return
	}

	s.hb.Broadcast(sess.ID, hub.Event{Type: "state", Data: in.State})
	writeJSON(w, http.StatusOK, map[string]any{"state": in.State})
}

func (s *Server) startTest(w http.ResponseWriter, r *http.Request) {
	sess := s.loadOwnedSession(w, r)
	if sess == nil {
		return
	}

	updated, err := s.st.StartTest(r.Context(), sess.ID)
	if err != nil {
		fail(w, http.StatusBadRequest, "Synagy başlap bolmady: "+err.Error(),
			"Sapaga synag berkidilendigini barlaň")
		return
	}

	s.hb.Broadcast(sess.ID, hub.Event{Type: "test_started", Data: map[string]any{
		"deadline":    updated.Deadline,
		"server_time": time.Now(),
	}})
	writeJSON(w, http.StatusOK, updated)
}

// ---------- Ученик ----------

func (s *Server) join(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}
	if err := readJSON(r, &in); err != nil || in.Name == "" || in.Code == "" {
		fail(w, http.StatusBadRequest, "Ady we sapak kody gerek", "Iki meýdany hem dolduryň")
		return
	}

	sess, err := s.st.SessionByCode(r.Context(), normalizeCode(in.Code))
	if err != nil {
		fail(w, http.StatusNotFound, "Beýle kodly sapak ýok", "Kody mugallymdan soraň")
		return
	}

	p, err := s.st.JoinParticipant(r.Context(), sess, in.Name)
	if errors.Is(err, store.ErrJoinClosed) {
		fail(w, http.StatusForbidden, "Sapaga goşulmak ýapyk", "Mugallymdan açmagyny soraň")
		return
	}
	if errors.Is(err, store.ErrNotInRoster) {
		fail(w, http.StatusForbidden, "Adyňyz toparyň sanawynda ýok",
			"Sanawdan öz adyňyzy saýlaň ýa-da mugallyma ýüz tutuň")
		return
	}
	if err != nil {
		fail(w, http.StatusInternalServerError, "Goşulyp bolmady", "Gaýtadan synanyşyň")
		return
	}

	s.pushParticipants(r.Context(), sess.ID)
	writeJSON(w, http.StatusOK, map[string]any{
		"token":   p.Token,
		"name":    p.DisplayName,
		"session": sess,
	})
}

func normalizeCode(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			out = append(out, r-32)
		case (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'):
			out = append(out, r)
		}
	}
	return string(out)
}

func (s *Server) studentToken(r *http.Request) string {
	if t := r.Header.Get("X-Student-Token"); t != "" {
		return t
	}
	return r.URL.Query().Get("token")
}

func (s *Server) currentStudent(w http.ResponseWriter, r *http.Request) (*store.Participant, *store.Session) {
	p, err := s.st.ParticipantByToken(r.Context(), s.studentToken(r))
	if err != nil {
		fail(w, http.StatusUnauthorized, "Sapaga goşulmadyk", "Kod bilen gaýtadan giriň")
		return nil, nil
	}

	sess, err := s.st.SessionByID(r.Context(), p.SessionID)
	if err != nil {
		fail(w, http.StatusNotFound, "Sapak tapylmady", "Kod bilen gaýtadan giriň")
		return nil, nil
	}
	return p, sess
}

// studentState отдаёт текущее задание, срок по серверным часам и ранее
// сохранённые ответы — этого достаточно, чтобы восстановить работу после обрыва.
func (s *Server) studentState(w http.ResponseWriter, r *http.Request) {
	p, sess := s.currentStudent(w, r)
	if p == nil {
		return
	}

	// Öý işinde ilkinji açylyşda şahsy möhlet başlaýar.
	if sess.IsHomework() {
		if open, reason := sess.Window(time.Now()); !open {
			writeJSON(w, http.StatusOK, map[string]any{
				"participant": p, "session": sess,
				"server_time": time.Now(), "closed_reason": reason,
			})
			return
		}
		if at, err := s.st.MarkStarted(r.Context(), p.ID); err == nil {
			p.StartedAt = at
		}
	}

	out := map[string]any{
		"participant": p,
		"session":     sess,
		"server_time": time.Now(),
	}
	// Личный срок: у домашнего задания он свой у каждого ученика.
	if dl := store.EffectiveDeadline(sess, p); dl != nil {
		out["deadline"] = dl
	}

	if sess.State == "test" {
		questions, err := s.st.FrozenQuestions(r.Context(), sess.ID)
		if err != nil {
			fail(w, http.StatusInternalServerError, "Soraglary okap bolmady", "Sahypany täzeläň")
			return
		}
		// Ключ правильных ответов на устройство ученика не уходит.
		safe := make([]map[string]any, 0, len(questions))
		for _, q := range questions {
			safe = append(safe, q.ForStudent())
		}
		out["questions"] = safe

		saved, err := s.st.AnswersOf(r.Context(), p.ID)
		if err == nil {
			out["saved_answers"] = saved
		}
	}

	if sess.LessonID != nil {
		if lesson, err := s.st.GetLesson(r.Context(), *sess.LessonID); err == nil {
			out["lesson"] = lesson
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) studentAnswer(w http.ResponseWriter, r *http.Request) {
	p, sess := s.currentStudent(w, r)
	if p == nil {
		return
	}

	var in struct {
		QuestionID int64           `json:"question_id"`
		ClientKey  string          `json:"client_key"`
		Value      json.RawMessage `json:"value"`
	}
	if err := readJSON(r, &in); err != nil || in.ClientKey == "" {
		fail(w, http.StatusBadRequest, "Jogap kabul edilmedi", "Gaýtadan ugradyň")
		return
	}

	if sess.State != "test" {
		fail(w, http.StatusConflict, "Synag işjeň däl", "Mugallymyň synagy başlamagyna garaşyň")
		return
	}
	// Срок считает сервер: перевод часов на телефоне ничего не меняет.
	// У домашнего задания срок персональный — от момента, когда ученик начал.
	if dl := store.EffectiveDeadline(sess, p); dl != nil && time.Now().After(*dl) {
		fail(w, http.StatusConflict, "Tabşyrygyň möhleti tamamlandy", "Saklanan jogaplaryňyz hasaba alyndy")
		return
	}

	// Tabşyrylan işi üýtgetmek diňe gaýtadan tabşyrmaga rugsat berlende mümkin.
	allowed, err := s.st.SubmissionGate(r.Context(), sess, p.ID)
	if err != nil {
		fail(w, http.StatusInternalServerError, "Ýagdaýy barlap bolmady", "Gaýtadan synanyşyň")
		return
	}
	if !allowed {
		fail(w, http.StatusConflict, "Iş eýýäm tabşyryldy",
			"Jogaplary üýtgetmek üçin mugallymdan gaýtadan tabşyrmaga rugsat soraň")
		return
	}

	questions, err := s.st.FrozenQuestions(r.Context(), sess.ID)
	if err != nil {
		fail(w, http.StatusInternalServerError, "Soraglary okap bolmady", "Sahypany täzeläň")
		return
	}

	res, err := s.st.SaveAnswer(r.Context(), sess.ID, p.ID, in.QuestionID, in.ClientKey, in.Value, questions)
	if errors.Is(err, store.ErrNotFound) {
		fail(w, http.StatusBadRequest, "Beýle sorag ýok", "Sahypany täzeläň")
		return
	}
	if err != nil {
		fail(w, http.StatusInternalServerError, "Jogap saklanmady", "Gaýtadan ugradyň")
		return
	}

	s.pushParticipants(r.Context(), sess.ID)
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) studentSubmit(w http.ResponseWriter, r *http.Request) {
	p, sess := s.currentStudent(w, r)
	if p == nil {
		return
	}

	dl := store.EffectiveDeadline(sess, p)
	late := dl != nil && time.Now().After(*dl)
	if err := s.st.SubmitWork(r.Context(), sess.ID, p.ID, late); err != nil {
		fail(w, http.StatusInternalServerError, "Işi tabşyryp bolmady", "Gaýtadan synanyşyň")
		return
	}

	s.pushParticipants(r.Context(), sess.ID)
	writeJSON(w, http.StatusOK, map[string]any{"status": "tabşyryldy", "late": late})
}

// ---------- Результаты ----------

func (s *Server) results(w http.ResponseWriter, r *http.Request) {
	sess := s.loadOwnedSession(w, r)
	if sess == nil {
		return
	}

	out, err := s.st.Results(r.Context(), sess.ID)
	if err != nil {
		fail(w, http.StatusInternalServerError, "Netijeleri okap bolmady", "Sahypany täzeläň")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) resultsCSV(w http.ResponseWriter, r *http.Request) {
	sess := s.loadOwnedSession(w, r)
	if sess == nil {
		return
	}

	out, err := s.st.Results(r.Context(), sess.ID)
	if err != nil {
		fail(w, http.StatusInternalServerError, "Netijeleri okap bolmady", "Sahypany täzeläň")
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition",
		fmt.Sprintf("attachment; filename=\"netijeler-%d.csv\"", sess.ID))
	// BOM: без него Excel открывает туркменские буквы как мусор.
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})

	cw := csv.NewWriter(w)
	defer cw.Flush()

	_ = cw.Write([]string{"Okuwçy", "Bal", "Iň ýokary bal", "Göterim", "Ýagdaý", "El bilen barlanýan", "Tabşyran wagty"})
	for _, row := range out.Rows {
		submitted := ""
		if row.SubmittedAt != nil {
			submitted = row.SubmittedAt.Format("2006-01-02 15:04:05")
		}
		status := row.Status
		if row.Preliminary {
			status += " (deslapky)"
		}
		_ = cw.Write([]string{
			row.Name,
			strconv.FormatFloat(row.Score, 'f', 2, 64),
			strconv.FormatFloat(row.MaxScore, 'f', 2, 64),
			strconv.FormatFloat(row.Percent, 'f', 1, 64),
			status,
			strconv.Itoa(row.PendingManual),
			submitted,
		})
	}
}

func (s *Server) studentDetail(w http.ResponseWriter, r *http.Request) {
	sess := s.loadOwnedSession(w, r)
	if sess == nil {
		return
	}

	pid, err := pathInt(r, "pid")
	if err != nil {
		fail(w, http.StatusBadRequest, "Okuwçynyň belgisi nädogry", "Sanawa dolanyň")
		return
	}

	out, err := s.st.StudentDetail(r.Context(), sess.ID, pid)
	if err != nil {
		fail(w, http.StatusInternalServerError, "Işi okap bolmady", "Sahypany täzeläň")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) gradeAnswer(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		fail(w, http.StatusBadRequest, "Jogabyň belgisi nädogry", "Sanawa dolanyň")
		return
	}

	var in struct {
		Score   float64 `json:"score"`
		Reason  string  `json:"reason"`
		Comment string  `json:"comment"`
	}
	if err := readJSON(r, &in); err != nil {
		fail(w, http.StatusBadRequest, "Maglumat nädogry", "Baly we sebäbi ýazyň")
		return
	}

	if err := s.st.AdjustGrade(r.Context(), id, in.Score, in.Reason, in.Comment); err != nil {
		fail(w, http.StatusInternalServerError, "Bal üýtgedilmedi", "Gaýtadan synanyşyň")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---------- WebSocket ----------

func (s *Server) wsTeacher(w http.ResponseWriter, r *http.Request) {
	t, err := s.currentTeacher(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	sessionID, err := strconv.ParseInt(r.URL.Query().Get("session"), 10, 64)
	if err != nil {
		http.Error(w, "session gerek", http.StatusBadRequest)
		return
	}
	sess, err := s.st.SessionByID(r.Context(), sessionID)
	if err != nil || sess.TeacherID != t.ID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	s.hb.Serve(conn, sessionID, hub.RoleTeacher, t.ID)
}

func (s *Server) wsStudent(w http.ResponseWriter, r *http.Request) {
	p, err := s.st.ParticipantByToken(r.Context(), s.studentToken(r))
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	s.hb.Serve(conn, p.SessionID, hub.RoleStudent, p.ID)
}

// pushParticipants обновляет панель учителя: кто подключён и сколько ответил.
func (s *Server) pushParticipants(ctx context.Context, sessionID int64) {
	list, err := s.st.ListParticipants(ctx, sessionID)
	if err != nil {
		log.Printf("pushParticipants: %v", err)
		return
	}
	s.hb.Broadcast(sessionID, hub.Event{Type: "participants", Data: list}, hub.RoleTeacher)
}
