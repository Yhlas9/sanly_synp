package api

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"

	"sanlysynp/internal/store"
)

const studentCookie = "sanly_okuwcy"

// ---------- Вход ученика ----------

func (s *Server) okuwcyLogin(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}
	if err := readJSON(r, &in); err != nil {
		fail(w, http.StatusBadRequest, "Maglumat nädogry", "Formany gaýtadan dolduryň")
		return
	}

	acc, err := s.st.StudentLogin(r.Context(), in.Login, in.Password)
	if err != nil {
		fail(w, http.StatusUnauthorized, "Login ýa-da açar söz nädogry",
			"Mugallymdan beren kagyzyňyzy barlaň")
		return
	}

	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		fail(w, http.StatusInternalServerError, "Içerki ýalňyşlyk", "Gaýtadan synanyşyň")
		return
	}
	token := hex.EncodeToString(raw)

	s.mu.Lock()
	s.okuwcy[token] = acc.ID
	s.mu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name: studentCookie, Value: token, Path: "/",
		HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: 12 * 3600,
	})
	writeJSON(w, http.StatusOK, acc)
}

func (s *Server) okuwcyLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(studentCookie); err == nil {
		s.mu.Lock()
		delete(s.okuwcy, c.Value)
		s.mu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{Name: studentCookie, Value: "", Path: "/", MaxAge: -1})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// currentOkuwcy возвращает вошедшего ученика, если он есть.
func (s *Server) currentOkuwcy(r *http.Request) (*store.StudentAccount, error) {
	c, err := r.Cookie(studentCookie)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	id, ok := s.okuwcy[c.Value]
	s.mu.RUnlock()
	if !ok {
		return nil, errors.New("session expired")
	}
	return s.st.StudentAccountByID(r.Context(), id)
}

func (s *Server) okuwcyAuth(next func(http.ResponseWriter, *http.Request, *store.StudentAccount)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		acc, err := s.currentOkuwcy(r)
		if err != nil {
			fail(w, http.StatusUnauthorized, "Giriş gerek", "Login we açar söz bilen giriň")
			return
		}
		next(w, r, acc)
	}
}

func (s *Server) okuwcyMe(w http.ResponseWriter, r *http.Request, acc *store.StudentAccount) {
	writeJSON(w, http.StatusOK, acc)
}

func (s *Server) okuwcyProfile(w http.ResponseWriter, r *http.Request, acc *store.StudentAccount) {
	out, err := s.st.MyProfile(r.Context(), acc.ID)
	if err != nil {
		fail(w, http.StatusInternalServerError, "Maglumat okalmady", "Sahypany täzeläň")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// ---------- Учётные записи: сторона учителя ----------

func (s *Server) createAccounts(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		fail(w, http.StatusBadRequest, "Belgi nädogry", "Sanawa dolanyň")
		return
	}

	out, err := s.st.CreateAccounts(r.Context(), teacherOf(r).ID, id)
	if errors.Is(err, store.ErrNotFound) {
		fail(w, http.StatusNotFound, "Topar tapylmady", "Öz toparlaryňyzy açyň")
		return
	}
	if err != nil {
		fail(w, http.StatusInternalServerError, "Hasaplar döredilmedi", "Gaýtadan synanyşyň")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) resetAccount(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		fail(w, http.StatusBadRequest, "Belgi nädogry", "Sanawa dolanyň")
		return
	}

	out, err := s.st.ResetAccount(r.Context(), teacherOf(r).ID, id)
	switch {
	case errors.Is(err, store.ErrNotFound):
		fail(w, http.StatusNotFound, "Okuwçy tapylmady", "Sanawa dolanyň")
	case errors.Is(err, store.ErrNoAccount):
		fail(w, http.StatusBadRequest, "Bu okuwçynyň hasaby ýok",
			"Ilki topara hasap dörediň")
	case err != nil:
		fail(w, http.StatusInternalServerError, "Täzelenmedi", "Gaýtadan synanyşyň")
	default:
		writeJSON(w, http.StatusOK, out)
	}
}
