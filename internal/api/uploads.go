package api

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"sanlysynp/internal/store"
)

// Amaly iş üçin faýlyň çägi. Ýerli torda uly faýl geçirmek sapagy haýalladýar.
const maxUploadBytes = 10 << 20 // 10 MB

// Rugsat berilýän görnüşler: tejribäniň suraty, tablisa, çyzgy, hasabat.
var allowedUploadExt = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true,
	".pdf": true, ".txt": true, ".csv": true,
	".xlsx": true, ".xls": true, ".docx": true, ".odt": true, ".ods": true,
}

func (s *Server) uploadDir() string {
	if v := os.Getenv("UPLOAD_DIR"); v != "" {
		return v
	}
	return "uploads"
}

// studentUpload принимает файл практической работы от ученика.
func (s *Server) studentUpload(w http.ResponseWriter, r *http.Request) {
	p, sess := s.currentStudent(w, r)
	if p == nil {
		return
	}

	if dl := store.EffectiveDeadline(sess, p); dl != nil && nowAfter(dl) {
		fail(w, http.StatusConflict, "Möhlet tamamlandy", "Faýl kabul edilmeýär")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes+1<<20)
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		fail(w, http.StatusBadRequest, "Faýly okap bolmady",
			"Faýlyň ölçegi 10 MB-dan uly bolmaly däl")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		fail(w, http.StatusBadRequest, "Faýl saýlanmady", "Faýly saýlap gaýtadan ugradyň")
		return
	}
	defer file.Close()

	if header.Size > maxUploadBytes {
		fail(w, http.StatusRequestEntityTooLarge, "Faýl örän uly",
			"10 MB-dan kiçi faýl saýlaň")
		return
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedUploadExt[ext] {
		fail(w, http.StatusUnsupportedMediaType, "Bu görnüşli faýl kabul edilmeýär",
			"Surat, PDF, tablisa ýa-da tekst faýlyny ugradyň")
		return
	}

	id, err := store.NewUploadID()
	if err != nil {
		fail(w, http.StatusInternalServerError, "Içerki ýalňyşlyk", "Gaýtadan synanyşyň")
		return
	}

	dir := s.uploadDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fail(w, http.StatusInternalServerError, "Faýl saklanmady", "Gaýtadan synanyşyň")
		return
	}

	// Diskdäki at tötänleýin: okuwçynyň iberen ady ýola täsir etmeýär.
	stored := filepath.Join(dir, id+ext)
	dst, err := os.Create(stored)
	if err != nil {
		fail(w, http.StatusInternalServerError, "Faýl saklanmady", "Gaýtadan synanyşyň")
		return
	}

	written, err := io.Copy(dst, io.LimitReader(file, maxUploadBytes))
	closeErr := dst.Close()
	if err != nil || closeErr != nil {
		_ = os.Remove(stored)
		fail(w, http.StatusInternalServerError, "Faýl doly ýazylmady", "Gaýtadan ugradyň")
		return
	}

	rec := store.Upload{
		ID: id, SessionID: sess.ID, ParticipantID: p.ID,
		FileName:    filepath.Base(header.Filename),
		ContentType: header.Header.Get("Content-Type"),
		SizeBytes:   written, StoredPath: stored,
	}
	if err := s.st.AddUpload(r.Context(), rec); err != nil {
		_ = os.Remove(stored)
		fail(w, http.StatusInternalServerError, "Faýl ýazga alynmady", "Gaýtadan ugradyň")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id": rec.ID, "file_name": rec.FileName, "size_bytes": rec.SizeBytes,
	})
}

// serveUpload отдаёт файл. Смотреть может учитель, которому принадлежит
// занятие, и сам ученик, который файл загрузил.
func (s *Server) serveUpload(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	up, err := s.st.GetUpload(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		fail(w, http.StatusNotFound, "Faýl tapylmady", "Sanawa dolanyň")
		return
	}
	if err != nil {
		fail(w, http.StatusInternalServerError, "Faýly okap bolmady", "Gaýtadan synanyşyň")
		return
	}

	if !s.mayReadUpload(r, up) {
		fail(w, http.StatusForbidden, "Bu faýly görmäge rugsat ýok",
			"Diňe mugallym we faýly ugradan okuwçy görüp bilýär")
		return
	}

	w.Header().Set("Content-Disposition",
		"inline; filename*=UTF-8''"+urlEscape(up.FileName))
	http.ServeFile(w, r, up.StoredPath)
}

func (s *Server) mayReadUpload(r *http.Request, up *store.Upload) bool {
	if t, err := s.currentTeacher(r); err == nil {
		if sess, err := s.st.SessionByID(r.Context(), up.SessionID); err == nil &&
			sess.TeacherID == t.ID {
			return true
		}
	}
	if tok := s.studentToken(r); tok != "" {
		if p, err := s.st.ParticipantByToken(r.Context(), tok); err == nil && p.ID == up.ParticipantID {
			return true
		}
	}
	return false
}
