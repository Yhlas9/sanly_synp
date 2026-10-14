package api

import (
	"errors"
	"net/http"
	"strings"

	"sanlysynp/internal/store"
)

// ---------- Состав группы ----------

func (s *Server) listStudents(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		fail(w, http.StatusBadRequest, "Belgi nädogry", "Sanawa dolanyň")
		return
	}

	out, err := s.st.ListStudents(r.Context(), id)
	if err != nil {
		fail(w, http.StatusInternalServerError, "Sanawy okap bolmady", "Sahypany täzeläň")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) addStudents(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		fail(w, http.StatusBadRequest, "Belgi nädogry", "Sanawa dolanyň")
		return
	}

	var in struct {
		Names string `json:"names"`
	}
	if err := readJSON(r, &in); err != nil {
		fail(w, http.StatusBadRequest, "Maglumat nädogry", "Atlary ýazyň")
		return
	}

	// Atlar setirme-setir ýa-da otur bilen ýazylyp bilner.
	var names []string
	for _, raw := range strings.FieldsFunc(in.Names, func(r rune) bool {
		return r == '\n' || r == ',' || r == ';'
	}) {
		if n := strings.TrimSpace(raw); n != "" {
			names = append(names, n)
		}
	}
	if len(names) == 0 {
		fail(w, http.StatusBadRequest, "At ýazylmady", "Her setire bir at ýazyň")
		return
	}

	added, err := s.st.AddStudents(r.Context(), teacherOf(r).ID, id, names)
	if errors.Is(err, store.ErrNotFound) {
		fail(w, http.StatusNotFound, "Topar tapylmady", "Öz toparlaryňyzy açyň")
		return
	}
	if err != nil {
		fail(w, http.StatusInternalServerError, "Goşulmady", "Gaýtadan synanyşyň")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"added": added, "total": len(names)})
}

func (s *Server) deleteStudent(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		fail(w, http.StatusBadRequest, "Belgi nädogry", "Sanawa dolanyň")
		return
	}
	if err := s.st.DeleteStudent(r.Context(), teacherOf(r).ID, id); errors.Is(err, store.ErrNotFound) {
		fail(w, http.StatusNotFound, "Okuwçy tapylmady", "Sanawa dolanyň")
		return
	} else if err != nil {
		fail(w, http.StatusInternalServerError, "Pozulmady", "Gaýtadan synanyşyň")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---------- Журнал ----------

func (s *Server) journal(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		fail(w, http.StatusBadRequest, "Belgi nädogry", "Topary saýlaň")
		return
	}

	out, err := s.st.GroupJournal(r.Context(), teacherOf(r).ID, id)
	if errors.Is(err, store.ErrNotFound) {
		fail(w, http.StatusNotFound, "Topar tapylmady", "Öz toparlaryňyzy açyň")
		return
	}
	if err != nil {
		fail(w, http.StatusInternalServerError, "Žurnaly okap bolmady", "Sahypany täzeläň")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// roster отдаёт список имён для выбора на странице ученика.
// Доступен без входа: иначе ученик не сможет назваться, чтобы войти.
func (s *Server) roster(w http.ResponseWriter, r *http.Request) {
	sess, err := s.st.SessionByCode(r.Context(), normalizeCode(r.URL.Query().Get("code")))
	if err != nil {
		fail(w, http.StatusNotFound, "Beýle kodly sapak ýok", "Kody mugallymdan soraň")
		return
	}
	if sess.GroupID == nil {
		writeJSON(w, http.StatusOK, map[string]any{"group_name": "", "names": []string{}})
		return
	}

	students, err := s.st.ListStudents(r.Context(), *sess.GroupID)
	if err != nil {
		fail(w, http.StatusInternalServerError, "Sanawy okap bolmady", "Sahypany täzeläň")
		return
	}

	names := make([]string, 0, len(students))
	for _, st := range students {
		names = append(names, st.FullName)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"group_name": sess.GroupName, "names": names,
	})
}
