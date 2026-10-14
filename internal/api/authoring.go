package api

import (
	"errors"
	"net/http"

	"sanlysynp/internal/store"
)

// ---------- Группы ----------

func (s *Server) listGroups(w http.ResponseWriter, r *http.Request) {
	out, err := s.st.ListGroups(r.Context(), teacherOf(r).ID)
	if err != nil {
		fail(w, http.StatusInternalServerError, "Toparlary okap bolmady", "Sahypany täzeläň")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) createGroup(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name string `json:"name"`
	}
	if err := readJSON(r, &in); err != nil || in.Name == "" {
		fail(w, http.StatusBadRequest, "Toparyň ady gerek", "Ady ýazyp gaýtadan synanyşyň")
		return
	}

	g, err := s.st.CreateGroup(r.Context(), teacherOf(r).ID, in.Name)
	if err != nil {
		fail(w, http.StatusInternalServerError, "Topar döredilmedi", "Gaýtadan synanyşyň")
		return
	}
	writeJSON(w, http.StatusOK, g)
}

func (s *Server) deleteGroup(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		fail(w, http.StatusBadRequest, "Belgi nädogry", "Sanawa dolanyň")
		return
	}
	if err := s.st.DeleteGroup(r.Context(), teacherOf(r).ID, id); err != nil {
		fail(w, http.StatusInternalServerError, "Pozulmady", "Gaýtadan synanyşyň")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---------- Уроки ----------

func (s *Server) createLesson(w http.ResponseWriter, r *http.Request) {
	var in store.LessonInput
	if err := readJSON(r, &in); err != nil || in.Title == "" || in.SubjectID == 0 {
		fail(w, http.StatusBadRequest, "Ady we dersi hökman gerek", "Meýdanlary dolduryň")
		return
	}

	out, err := s.st.CreateLesson(r.Context(), teacherOf(r).ID, in)
	if err != nil {
		fail(w, http.StatusInternalServerError, "Sapak döredilmedi", "Gaýtadan synanyşyň")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) updateLesson(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		fail(w, http.StatusBadRequest, "Belgi nädogry", "Sanawa dolanyň")
		return
	}

	var in store.LessonInput
	if err := readJSON(r, &in); err != nil || in.Title == "" {
		fail(w, http.StatusBadRequest, "Sapagyň ady gerek", "Meýdanlary dolduryň")
		return
	}

	out, err := s.st.UpdateLesson(r.Context(), teacherOf(r).ID, id, in)
	if errors.Is(err, store.ErrNotFound) {
		fail(w, http.StatusNotFound, "Nusgalyk sapagy üýtgedip bolmaýar ýa-da ol size degişli däl",
			"«Nusga» düwmesi bilen öz nusgaňyzy dörediň")
		return
	}
	if err != nil {
		fail(w, http.StatusInternalServerError, "Üýtgedilmedi", "Gaýtadan synanyşyň")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) copyLesson(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		fail(w, http.StatusBadRequest, "Belgi nädogry", "Sanawa dolanyň")
		return
	}

	out, err := s.st.CopyLesson(r.Context(), teacherOf(r).ID, id)
	if errors.Is(err, store.ErrNotFound) {
		fail(w, http.StatusNotFound, "Sapak tapylmady", "Sanawa dolanyň")
		return
	}
	if err != nil {
		fail(w, http.StatusInternalServerError, "Nusga alynmady", "Gaýtadan synanyşyň")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) deleteLesson(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		fail(w, http.StatusBadRequest, "Belgi nädogry", "Sanawa dolanyň")
		return
	}

	if err := s.st.DeleteLesson(r.Context(), teacherOf(r).ID, id); errors.Is(err, store.ErrNotFound) {
		fail(w, http.StatusNotFound, "Nusgalyk sapagy pozup bolmaýar ýa-da ol size degişli däl",
			"Öz sapaklaryňyzy açyň")
		return
	} else if err != nil {
		fail(w, http.StatusInternalServerError, "Pozulmady", "Gaýtadan synanyşyň")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---------- Вопросы ----------

func (s *Server) lessonQuestions(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		fail(w, http.StatusBadRequest, "Belgi nädogry", "Sanawa dolanyň")
		return
	}

	out, err := s.st.ListQuestions(r.Context(), teacherOf(r).ID, id)
	if errors.Is(err, store.ErrNotFound) {
		fail(w, http.StatusNotFound, "Sapak tapylmady", "Öz sapaklaryňyzy açyň")
		return
	}
	if err != nil {
		fail(w, http.StatusInternalServerError, "Soraglary okap bolmady", "Sahypany täzeläň")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) createQuestion(w http.ResponseWriter, r *http.Request) {
	var in store.QuestionInput
	if err := readJSON(r, &in); err != nil || in.Prompt == "" || in.Kind == "" {
		fail(w, http.StatusBadRequest, "Soragyň teksti we görnüşi gerek", "Meýdanlary dolduryň")
		return
	}
	if in.Points <= 0 {
		in.Points = 1
	}

	id, err := s.st.CreateQuestion(r.Context(), teacherOf(r).ID, in)
	if errors.Is(err, store.ErrReadOnly) {
		fail(w, http.StatusForbidden, "Nusgalyk sapagy üýtgedip bolmaýar",
			"Ilki «Nusga» düwmesi bilen öz nusgaňyzy dörediň")
		return
	}
	if errors.Is(err, store.ErrNotFound) {
		fail(w, http.StatusNotFound, "Sapak tapylmady", "Öz sapaklaryňyzy açyň")
		return
	}
	if err != nil {
		fail(w, http.StatusInternalServerError, "Sorag goşulmady", "Gaýtadan synanyşyň")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id})
}

func (s *Server) updateQuestion(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		fail(w, http.StatusBadRequest, "Belgi nädogry", "Sanawa dolanyň")
		return
	}

	var in store.QuestionInput
	if err := readJSON(r, &in); err != nil || in.Prompt == "" {
		fail(w, http.StatusBadRequest, "Soragyň teksti gerek", "Meýdany dolduryň")
		return
	}
	if in.Points <= 0 {
		in.Points = 1
	}

	if err := s.st.UpdateQuestion(r.Context(), teacherOf(r).ID, id, in); errors.Is(err, store.ErrNotFound) {
		fail(w, http.StatusNotFound, "Sorag tapylmady ýa-da size degişli däl",
			"Öz sapaklaryňyzyň soraglaryny açyň")
		return
	} else if errors.Is(err, store.ErrReadOnly) {
		fail(w, http.StatusForbidden, "Nusgalyk soragy üýtgedip bolmaýar",
			"«Nusga» düwmesi bilen öz nusgaňyzy dörediň we ony üýtgediň")
		return
	} else if err != nil {
		fail(w, http.StatusInternalServerError, "Üýtgedilmedi", "Gaýtadan synanyşyň")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) deleteQuestion(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		fail(w, http.StatusBadRequest, "Belgi nädogry", "Sanawa dolanyň")
		return
	}
	switch err := s.st.DeleteQuestion(r.Context(), teacherOf(r).ID, id); {
	case errors.Is(err, store.ErrReadOnly):
		fail(w, http.StatusForbidden, "Nusgalyk soragy pozup bolmaýar",
			"Ilki sapagyň öz nusgaňyzy dörediň")
		return
	case errors.Is(err, store.ErrNotFound):
		fail(w, http.StatusNotFound, "Sorag tapylmady ýa-da size degişli däl",
			"Öz sapaklaryňyzyň soraglaryny açyň")
		return
	case err != nil:
		fail(w, http.StatusInternalServerError, "Pozulmady", "Gaýtadan synanyşyň")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---------- Тесты ----------

func (s *Server) getTest(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		fail(w, http.StatusBadRequest, "Belgi nädogry", "Sanawa dolanyň")
		return
	}

	out, err := s.st.GetTest(r.Context(), teacherOf(r).ID, id)
	if errors.Is(err, store.ErrNotFound) {
		fail(w, http.StatusNotFound, "Synag tapylmady", "Sanawa dolanyň")
		return
	}
	if err != nil {
		fail(w, http.StatusInternalServerError, "Okap bolmady", "Sahypany täzeläň")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) saveTest(w http.ResponseWriter, r *http.Request) {
	var id *int64
	if raw := r.PathValue("id"); raw != "" {
		v, err := pathInt(r, "id")
		if err != nil {
			fail(w, http.StatusBadRequest, "Belgi nädogry", "Sanawa dolanyň")
			return
		}
		id = &v
	}

	var in store.TestInput
	if err := readJSON(r, &in); err != nil || in.Title == "" || in.SubjectID == 0 {
		fail(w, http.StatusBadRequest, "Synagyň ady we dersi gerek", "Meýdanlary dolduryň")
		return
	}
	if in.DurationSec <= 0 {
		in.DurationSec = 600
	}
	if in.ResultsPolicy == "" {
		in.ResultsPolicy = "after_close"
	}

	testID, err := s.st.SaveTest(r.Context(), teacherOf(r).ID, id, in)
	if errors.Is(err, store.ErrNotFound) {
		fail(w, http.StatusNotFound, "Synag tapylmady ýa-da size degişli däl", "Öz synaglaryňyzy açyň")
		return
	}
	if errors.Is(err, store.ErrReadOnly) {
		fail(w, http.StatusForbidden, "Synaga diňe öz soraglaryňyzy goşup bilersiňiz",
			"Nusgalyk synagy üýtgetmek hem bolmaýar — ilki nusga alyň")
		return
	}
	if err != nil {
		fail(w, http.StatusInternalServerError, "Saklanmady", "Gaýtadan synanyşyň")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": testID})
}

func (s *Server) deleteTest(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		fail(w, http.StatusBadRequest, "Belgi nädogry", "Sanawa dolanyň")
		return
	}

	if err := s.st.DeleteTest(r.Context(), teacherOf(r).ID, id); errors.Is(err, store.ErrNotFound) {
		fail(w, http.StatusNotFound, "Synag tapylmady ýa-da size degişli däl", "Öz synaglaryňyzy açyň")
		return
	} else if err != nil {
		fail(w, http.StatusInternalServerError, "Pozulmady", "Gaýtadan synanyşyň")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---------- Дополнительное время и выдача результатов ----------

func (s *Server) extraTime(w http.ResponseWriter, r *http.Request) {
	sess := s.loadOwnedSession(w, r)
	if sess == nil {
		return
	}

	var in struct {
		Minutes int `json:"minutes"`
	}
	if err := readJSON(r, &in); err != nil || in.Minutes <= 0 {
		fail(w, http.StatusBadRequest, "Goşmaça wagt nädogry", "Minutlaryň sanyny ýazyň")
		return
	}

	if err := s.st.GrantExtraTime(r.Context(), sess.ID, in.Minutes*60); err != nil {
		fail(w, http.StatusBadRequest, "Wagt goşulmady", "Synagyň başlandygyny barlaň")
		return
	}

	updated, _ := s.st.SessionByID(r.Context(), sess.ID)
	s.hb.Broadcast(sess.ID, hubEvent("test_started", map[string]any{
		"deadline": updated.Deadline,
	}))
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) releaseResults(w http.ResponseWriter, r *http.Request) {
	sess := s.loadOwnedSession(w, r)
	if sess == nil {
		return
	}

	if err := s.st.ReleaseResults(r.Context(), sess.ID); err != nil {
		fail(w, http.StatusInternalServerError, "Açylmady", "Gaýtadan synanyşyň")
		return
	}

	s.hb.Broadcast(sess.ID, hubEvent("state", "closed"))
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// studentResult отдаёт ученику ТОЛЬКО его собственный результат и только
// после того, как учитель закрыл занятие (раздел 7 ТЗ).
func (s *Server) studentResult(w http.ResponseWriter, r *http.Request) {
	p, sess := s.currentStudent(w, r)
	if p == nil {
		return
	}
	if sess.State != "closed" {
		fail(w, http.StatusForbidden, "Netijeler entek açyk däl", "Mugallymyň açmagyna garaşyň")
		return
	}

	out, err := s.st.MyResult(r.Context(), sess.ID, p.ID)
	if err != nil {
		fail(w, http.StatusInternalServerError, "Netijäni okap bolmady", "Sahypany täzeläň")
		return
	}
	writeJSON(w, http.StatusOK, out)
}
