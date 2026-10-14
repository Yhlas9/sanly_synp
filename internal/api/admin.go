package api

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"sanlysynp/internal/store"
)

// adminOnly пропускает только администратора системы.
func (s *Server) adminOnly(next http.HandlerFunc) http.HandlerFunc {
	return s.auth(func(w http.ResponseWriter, r *http.Request) {
		if teacherOf(r).Role != "admin" {
			fail(w, http.StatusForbidden, "Bu bölüm diňe ulgam dolandyryjysy üçin",
				"Dolandyryjynyň hasaby bilen giriň")
			return
		}
		next(w, r)
	})
}

// ---------- Учётные записи ----------

func (s *Server) adminTeachers(w http.ResponseWriter, r *http.Request) {
	out, err := s.st.ListTeachers(r.Context())
	if err != nil {
		fail(w, http.StatusInternalServerError, "Hasaplary okap bolmady", "Sahypany täzeläň")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) adminCreateTeacher(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Login    string `json:"login"`
		Password string `json:"password"`
		FullName string `json:"full_name"`
		Role     string `json:"role"`
	}
	if err := readJSON(r, &in); err != nil || in.Login == "" || in.FullName == "" {
		fail(w, http.StatusBadRequest, "Login we doly at gerek", "Meýdanlary dolduryň")
		return
	}
	// Açar söz boş goýulsa, ulgam özi döredýär — ony bir gezek görkezýäris.
	password := in.Password
	if password == "" {
		generated, err := store.NewPassword()
		if err != nil {
			fail(w, http.StatusInternalServerError, "Içerki ýalňyşlyk", "Gaýtadan synanyşyň")
			return
		}
		password = generated
	}
	if len(password) < 4 {
		fail(w, http.StatusBadRequest, "Açar söz gysga",
			"Azyndan 4 belgi ýazyň ýa-da boş goýuň — ulgam özi döreder")
		return
	}

	out, err := s.st.CreateTeacher(r.Context(), in.Login, password, in.FullName, in.Role)
	if err != nil {
		fail(w, http.StatusBadRequest, err.Error(), "Başga login saýlaň")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"teacher": out, "password": password})
}

func (s *Server) adminSetPassword(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		fail(w, http.StatusBadRequest, "Belgi nädogry", "Sanawa dolanyň")
		return
	}

	var in struct {
		Password string `json:"password"`
	}
	if err := readJSON(r, &in); err != nil || len(in.Password) < 4 {
		fail(w, http.StatusBadRequest, "Açar söz gysga", "Azyndan 4 belgi ýazyň")
		return
	}

	if err := s.st.SetPassword(r.Context(), id, in.Password); errors.Is(err, store.ErrNotFound) {
		fail(w, http.StatusNotFound, "Hasap tapylmady", "Sanawa dolanyň")
		return
	} else if err != nil {
		fail(w, http.StatusInternalServerError, "Üýtgedilmedi", "Gaýtadan synanyşyň")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) adminDeleteTeacher(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		fail(w, http.StatusBadRequest, "Belgi nädogry", "Sanawa dolanyň")
		return
	}
	if id == teacherOf(r).ID {
		fail(w, http.StatusBadRequest, "Öz hasabyňyzy pozup bolmaýar",
			"Başga dolandyryjydan haýyş ediň")
		return
	}

	switch err := s.st.DeleteTeacher(r.Context(), id); {
	case errors.Is(err, store.ErrNotFound):
		fail(w, http.StatusNotFound, "Hasap tapylmady", "Sanawa dolanyň")
	case errors.Is(err, store.ErrLastAdmin):
		fail(w, http.StatusBadRequest, "Soňky dolandyryjyny pozup bolmaýar",
			"Ilki başga dolandyryjy dörediň")
	case err != nil:
		fail(w, http.StatusInternalServerError, "Pozulmady", "Gaýtadan synanyşyň")
	default:
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}

func (s *Server) adminStats(w http.ResponseWriter, r *http.Request) {
	out, err := s.st.Stats(r.Context())
	if err != nil {
		fail(w, http.StatusInternalServerError, "Maglumat okalmady", "Sahypany täzeläň")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// ---------- Резервные копии ----------

const backupDir = "backups"

type backupFile struct {
	Name      string    `json:"name"`
	SizeBytes int64     `json:"size_bytes"`
	CreatedAt time.Time `json:"created_at"`
}

// safeBackupName не выпускает имя файла за пределы каталога копий.
func safeBackupName(name string) (string, bool) {
	if name == "" || name != filepath.Base(name) || strings.Contains(name, "..") {
		return "", false
	}
	if !strings.HasSuffix(name, ".sql") {
		return "", false
	}
	return filepath.Join(backupDir, name), true
}

func (s *Server) adminListBackups(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir(backupDir)
	if err != nil && !os.IsNotExist(err) {
		fail(w, http.StatusInternalServerError, "Nusgalary okap bolmady", "Gaýtadan synanyşyň")
		return
	}

	out := []backupFile{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, backupFile{e.Name(), info.Size(), info.ModTime()})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) adminCreateBackup(w http.ResponseWriter, r *http.Request) {
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		fail(w, http.StatusInternalServerError, "Nusga döredilmedi", "Gaýtadan synanyşyň")
		return
	}

	name := fmt.Sprintf("sanly-synp-%s.sql", time.Now().Format("20060102-150405"))
	path := filepath.Join(backupDir, name)

	file, err := os.Create(path)
	if err != nil {
		fail(w, http.StatusInternalServerError, "Faýl döredilmedi", "Gaýtadan synanyşyň")
		return
	}
	defer file.Close()

	ctx, cancel := contextWithTimeout(r, 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "pg_dump", "--clean", "--if-exists", s.dsn)
	cmd.Stdout = file
	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		_ = os.Remove(path)
		fail(w, http.StatusInternalServerError,
			"pg_dump ýerine ýetirilmedi: "+strings.TrimSpace(stderr.String()),
			"PostgreSQL gurallarynyň PATH-da bardygyny barlaň")
		return
	}

	info, _ := file.Stat()
	writeJSON(w, http.StatusOK, backupFile{name, info.Size(), time.Now()})
}

func (s *Server) adminDownloadBackup(w http.ResponseWriter, r *http.Request) {
	path, ok := safeBackupName(r.PathValue("name"))
	if !ok {
		fail(w, http.StatusBadRequest, "Faýlyň ady nädogry", "Sanawdan saýlaň")
		return
	}
	if _, err := os.Stat(path); err != nil {
		fail(w, http.StatusNotFound, "Nusga tapylmady", "Sanawy täzeläň")
		return
	}

	w.Header().Set("Content-Type", "application/sql")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filepath.Base(path)+"\"")
	http.ServeFile(w, r, path)
}

// adminRestore возвращает систему к состоянию из выбранной копии.
// Действие необратимое, поэтому требует явного подтверждения в теле запроса.
func (s *Server) adminRestore(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name    string `json:"name"`
		Confirm string `json:"confirm"`
	}
	if err := readJSON(r, &in); err != nil {
		fail(w, http.StatusBadRequest, "Maglumat nädogry", "Nusgany saýlaň")
		return
	}
	if in.Confirm != "DIKELT" {
		fail(w, http.StatusBadRequest, "Tassyklama gerek",
			"Tassyklamak üçin DIKELT sözüni ýazyň")
		return
	}

	path, ok := safeBackupName(in.Name)
	if !ok {
		fail(w, http.StatusBadRequest, "Faýlyň ady nädogry", "Sanawdan saýlaň")
		return
	}

	ctx, cancel := contextWithTimeout(r, 120*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "psql", "--quiet", "--dbname", s.dsn, "--file", path)
	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		fail(w, http.StatusInternalServerError,
			"Dikeldip bolmady: "+strings.TrimSpace(lastLines(stderr.String(), 2)),
			"Nusganyň bitewidigini barlaň")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "restored": in.Name})
}

func lastLines(s string, n int) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "; ")
}

// startedAt фиксирует момент запуска: администратору полезно видеть,
// сколько система работает без перезапуска.
var startedAt = time.Now()

func dirSize(path string) int64 {
	var total int64
	entries, err := os.ReadDir(path)
	if err != nil {
		return 0
	}
	for _, e := range entries {
		if info, err := e.Info(); err == nil && !e.IsDir() {
			total += info.Size()
		}
	}
	return total
}

// dbNameFromDSN достаёт имя базы, не раскрывая пароль из строки подключения.
func dbNameFromDSN(dsn string) string {
	if i := strings.LastIndex(dsn, "/"); i >= 0 && i+1 < len(dsn) {
		name := dsn[i+1:]
		if j := strings.Index(name, "?"); j >= 0 {
			name = name[:j]
		}
		return name
	}
	return "—"
}

func (s *Server) adminSystem(w http.ResponseWriter, r *http.Request) {
	up := time.Since(startedAt)

	writeJSON(w, http.StatusOK, map[string]any{
		"started_at":    startedAt,
		"uptime_sec":    int(up.Seconds()),
		"go_version":    runtime.Version(),
		"platform":      runtime.GOOS + "/" + runtime.GOARCH,
		"db_name":       dbNameFromDSN(s.dsn),
		"upload_dir":    s.uploadDir(),
		"upload_bytes":  dirSize(s.uploadDir()),
		"backup_bytes":  dirSize(backupDir),
		"student_paths": localIPv4(),
		"max_upload_mb": maxUploadBytes / (1 << 20),
	})
}

// ---------- Профиль учителя ----------

func (s *Server) myProfile(w http.ResponseWriter, r *http.Request) {
	out, err := s.st.TeacherProfile(r.Context(), teacherOf(r).ID)
	if err != nil {
		fail(w, http.StatusInternalServerError, "Maglumat okalmady", "Sahypany täzeläň")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// myPassword даёт учителю сменить пароль самому, зная текущий.
// Раньше это умел только администратор.
func (s *Server) myPassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Current string `json:"current"`
		Next    string `json:"next"`
	}
	if err := readJSON(r, &in); err != nil {
		fail(w, http.StatusBadRequest, "Maglumat nädogry", "Formany gaýtadan dolduryň")
		return
	}
	if len(in.Next) < 4 {
		fail(w, http.StatusBadRequest, "Täze açar söz gysga", "Azyndan 4 belgi ýazyň")
		return
	}
	if in.Next == in.Current {
		fail(w, http.StatusBadRequest, "Täze açar söz köneden tapawutly bolmaly",
			"Başga açar söz saýlaň")
		return
	}

	switch err := s.st.ChangeOwnPassword(r.Context(), teacherOf(r).ID, in.Current, in.Next); {
	case errors.Is(err, store.ErrWrongPassword):
		fail(w, http.StatusForbidden, "Häzirki açar söz nädogry", "Gaýtadan synanyşyň")
	case err != nil:
		fail(w, http.StatusInternalServerError, "Üýtgedilmedi", "Gaýtadan synanyşyň")
	default:
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}

// ---------- Все ученики и группы: взгляд администратора ----------

func (s *Server) adminStudents(w http.ResponseWriter, r *http.Request) {
	out, err := s.st.ListAllStudents(r.Context())
	if err != nil {
		fail(w, http.StatusInternalServerError, "Okuwçylary okap bolmady", "Sahypany täzeläň")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) adminGroups(w http.ResponseWriter, r *http.Request) {
	out, err := s.st.ListAllGroups(r.Context())
	if err != nil {
		fail(w, http.StatusInternalServerError, "Toparlary okap bolmady", "Sahypany täzeläň")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) adminGroupAccounts(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		fail(w, http.StatusBadRequest, "Belgi nädogry", "Sanawa dolanyň")
		return
	}

	out, err := s.st.AdminCreateAccounts(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		fail(w, http.StatusNotFound, "Topar tapylmady", "Sanawy täzeläň")
		return
	}
	if err != nil {
		fail(w, http.StatusInternalServerError, "Hasaplar döredilmedi", "Gaýtadan synanyşyň")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) adminResetStudent(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		fail(w, http.StatusBadRequest, "Belgi nädogry", "Sanawa dolanyň")
		return
	}

	out, err := s.st.AdminResetAccount(r.Context(), id)
	switch {
	case errors.Is(err, store.ErrNotFound):
		fail(w, http.StatusNotFound, "Okuwçy tapylmady", "Sanawy täzeläň")
	case errors.Is(err, store.ErrNoAccount):
		fail(w, http.StatusBadRequest, "Bu okuwçynyň hasaby ýok",
			"Ilki topara hasap dörediň")
	case err != nil:
		fail(w, http.StatusInternalServerError, "Täzelenmedi", "Gaýtadan synanyşyň")
	default:
		writeJSON(w, http.StatusOK, out)
	}
}
