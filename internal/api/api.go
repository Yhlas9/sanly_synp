// Package api — HTTP и WebSocket слой платформы.
package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/crypto/bcrypt"

	"sanlysynp/internal/hub"
	"sanlysynp/internal/store"
)

const cookieName = "sanly_synp"

type Server struct {
	st     *store.Store
	hb     *hub.Hub
	webDir string
	dsn    string

	mu       sync.RWMutex
	sessions map[string]int64 // cookie -> teacher id
	okuwcy   map[string]int64 // cookie -> student id
}

func New(st *store.Store, hb *hub.Hub, webDir, dsn string) *Server {
	s := &Server{st: st, hb: hb, webDir: webDir, dsn: dsn,
		sessions: map[string]int64{}, okuwcy: map[string]int64{}}

	// Подключение и отключение ученика сразу видно на панели учителя.
	hb.OnStudentPresence = func(participantID int64, online bool) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		if err := st.SetPresence(ctx, participantID, online); err != nil {
			log.Printf("presence: %v", err)
			return
		}
		if p, err := st.ParticipantByID(ctx, participantID); err == nil {
			s.pushParticipants(ctx, p.SessionID)
		}
	}

	// Кадр демонстрации экрана уходит ученикам как есть, без переупаковки.
	hb.OnTeacherFrame = func(sessionID int64, payload []byte) {
		hb.BroadcastRaw(sessionID, payload, hub.RoleStudent)
	}

	return s
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	// --- Учитель ---
	mux.HandleFunc("POST /api/login", s.login)
	mux.HandleFunc("POST /api/logout", s.logout)
	mux.HandleFunc("GET /api/me", s.auth(s.me))
	mux.HandleFunc("GET /api/me/profile", s.auth(s.myProfile))
	mux.HandleFunc("POST /api/me/password", s.auth(s.myPassword))
	mux.HandleFunc("GET /api/subjects", s.auth(s.listSubjects))
	mux.HandleFunc("GET /api/lessons", s.auth(s.listLessons))
	mux.HandleFunc("GET /api/tests", s.auth(s.listTests))
	mux.HandleFunc("POST /api/sessions", s.auth(s.createSession))
	mux.HandleFunc("GET /api/sessions/{id}", s.auth(s.getSession))
	mux.HandleFunc("GET /api/sessions/{id}/participants", s.auth(s.participants))
	mux.HandleFunc("POST /api/sessions/{id}/join-open", s.auth(s.joinOpen))
	mux.HandleFunc("POST /api/sessions/{id}/start-test", s.auth(s.startTest))
	mux.HandleFunc("POST /api/sessions/{id}/state", s.auth(s.setState))
	mux.HandleFunc("GET /api/sessions/{id}/results", s.auth(s.results))
	mux.HandleFunc("GET /api/sessions/{id}/results.csv", s.auth(s.resultsCSV))
	mux.HandleFunc("GET /api/sessions/{id}/students/{pid}", s.auth(s.studentDetail))
	mux.HandleFunc("POST /api/answers/{id}/grade", s.auth(s.gradeAnswer))

	// --- Подготовка материала: группы, уроки, вопросы, тесты ---
	mux.HandleFunc("GET /api/groups", s.auth(s.listGroups))
	mux.HandleFunc("POST /api/groups", s.auth(s.createGroup))
	mux.HandleFunc("DELETE /api/groups/{id}", s.auth(s.deleteGroup))
	mux.HandleFunc("GET /api/groups/{id}/students", s.auth(s.listStudents))
	mux.HandleFunc("POST /api/groups/{id}/students", s.auth(s.addStudents))
	mux.HandleFunc("DELETE /api/students/{id}", s.auth(s.deleteStudent))
	mux.HandleFunc("GET /api/groups/{id}/journal", s.auth(s.journal))

	mux.HandleFunc("POST /api/lessons", s.auth(s.createLesson))
	mux.HandleFunc("PUT /api/lessons/{id}", s.auth(s.updateLesson))
	mux.HandleFunc("POST /api/lessons/{id}/copy", s.auth(s.copyLesson))
	mux.HandleFunc("DELETE /api/lessons/{id}", s.auth(s.deleteLesson))
	mux.HandleFunc("GET /api/lessons/{id}/questions", s.auth(s.lessonQuestions))

	mux.HandleFunc("POST /api/questions", s.auth(s.createQuestion))
	mux.HandleFunc("PUT /api/questions/{id}", s.auth(s.updateQuestion))
	mux.HandleFunc("DELETE /api/questions/{id}", s.auth(s.deleteQuestion))

	mux.HandleFunc("GET /api/tests/{id}", s.auth(s.getTest))
	mux.HandleFunc("POST /api/tests", s.auth(s.saveTest))
	mux.HandleFunc("PUT /api/tests/{id}", s.auth(s.saveTest))
	mux.HandleFunc("DELETE /api/tests/{id}", s.auth(s.deleteTest))

	mux.HandleFunc("POST /api/sessions/{id}/extra-time", s.auth(s.extraTime))
	mux.HandleFunc("POST /api/sessions/{id}/release", s.auth(s.releaseResults))

	// --- Ученик ---
	mux.HandleFunc("POST /api/groups/{id}/accounts", s.auth(s.createAccounts))
	mux.HandleFunc("POST /api/students/{id}/reset", s.auth(s.resetAccount))

	// --- Okuwçynyň hasaby ---
	mux.HandleFunc("POST /api/okuwcy/login", s.okuwcyLogin)
	mux.HandleFunc("POST /api/okuwcy/logout", s.okuwcyLogout)
	mux.HandleFunc("GET /api/okuwcy/me", s.okuwcyAuth(s.okuwcyMe))
	mux.HandleFunc("GET /api/okuwcy/profile", s.okuwcyAuth(s.okuwcyProfile))

	mux.HandleFunc("GET /api/roster", s.roster)
	mux.HandleFunc("POST /api/join", s.join)
	mux.HandleFunc("GET /api/student/state", s.studentState)
	mux.HandleFunc("POST /api/student/answer", s.studentAnswer)
	mux.HandleFunc("POST /api/student/submit", s.studentSubmit)
	mux.HandleFunc("GET /api/student/result", s.studentResult)
	mux.HandleFunc("POST /api/student/upload", s.studentUpload)
	mux.HandleFunc("GET /api/files/{id}", s.serveUpload)

	// --- Живое состояние и демонстрация экрана ---
	mux.HandleFunc("GET /ws/teacher", s.wsTeacher)
	mux.HandleFunc("GET /ws/student", s.wsStudent)

	mux.HandleFunc("GET /hasabat", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, s.webDir+"/hasabat.html")
	})

	mux.HandleFunc("GET /dolandyrys", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, s.webDir+"/dolandyrys.html")
	})

	mux.HandleFunc("GET /materiallar", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, s.webDir+"/materiallar.html")
	})

	mux.HandleFunc("GET /mugallym", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, s.webDir+"/mugallym.html")
	})

	mux.HandleFunc("GET /profil", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, s.webDir+"/profil.html")
	})

	// Отдельный адрес для учеников: короткий, его легко продиктовать вслух.
	mux.HandleFunc("GET /okuwcy", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, s.webDir+"/okuwcy.html")
	})

	// --- Ulgam dolandyryjysy ---
	mux.HandleFunc("GET /api/admin/stats", s.adminOnly(s.adminStats))
	mux.HandleFunc("GET /api/admin/system", s.adminOnly(s.adminSystem))
	mux.HandleFunc("GET /api/admin/teachers", s.adminOnly(s.adminTeachers))
	mux.HandleFunc("GET /api/admin/students", s.adminOnly(s.adminStudents))
	mux.HandleFunc("GET /api/admin/groups", s.adminOnly(s.adminGroups))
	mux.HandleFunc("POST /api/admin/groups/{id}/accounts", s.adminOnly(s.adminGroupAccounts))
	mux.HandleFunc("POST /api/admin/students/{id}/reset", s.adminOnly(s.adminResetStudent))
	mux.HandleFunc("POST /api/admin/teachers", s.adminOnly(s.adminCreateTeacher))
	mux.HandleFunc("POST /api/admin/teachers/{id}/password", s.adminOnly(s.adminSetPassword))
	mux.HandleFunc("DELETE /api/admin/teachers/{id}", s.adminOnly(s.adminDeleteTeacher))
	mux.HandleFunc("GET /api/admin/backups", s.adminOnly(s.adminListBackups))
	mux.HandleFunc("POST /api/admin/backups", s.adminOnly(s.adminCreateBackup))
	mux.HandleFunc("GET /api/admin/backups/{name}", s.adminOnly(s.adminDownloadBackup))
	mux.HandleFunc("POST /api/admin/restore", s.adminOnly(s.adminRestore))

	mux.HandleFunc("GET /api/net", s.netInfo)
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "time": time.Now()})
	})

	// Статика отдаётся с диска — все шрифты и библиотеки лежат локально,
	// ни одной ссылки наружу (требование раздела 10).
	mux.Handle("/", http.FileServer(http.Dir(s.webDir)))

	return noCache(mux)
}

// ---------- Вспомогательное ----------

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writeJSON: %v", err)
	}
}

// fail отдаёт причину ошибки и подсказку о следующем шаге (раздел 9 ТЗ).
func fail(w http.ResponseWriter, code int, msg, next string) {
	writeJSON(w, code, map[string]any{"error": msg, "next": next})
}

func readJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 4<<20))
	return dec.Decode(dst)
}

func pathInt(r *http.Request, name string) (int64, error) {
	return strconv.ParseInt(r.PathValue(name), 10, 64)
}

func noCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(r.URL.Path) >= 5 && r.URL.Path[:5] == "/api/" {
			w.Header().Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(w, r)
	})
}

// ---------- Авторизация учителя ----------

type ctxKey string

const ctxTeacher ctxKey = "teacher"

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t, err := s.currentTeacher(r)
		if err != nil {
			fail(w, http.StatusUnauthorized, "Giriş gerek", "Mugallym hökmünde giriň")
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), ctxTeacher, t)))
	}
}

func teacherOf(r *http.Request) *store.Teacher {
	t, _ := r.Context().Value(ctxTeacher).(*store.Teacher)
	return t
}

func (s *Server) currentTeacher(r *http.Request) (*store.Teacher, error) {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	id, ok := s.sessions[c.Value]
	s.mu.RUnlock()
	if !ok {
		return nil, errors.New("session expired")
	}
	return s.st.TeacherByID(r.Context(), id)
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}
	if err := readJSON(r, &in); err != nil {
		fail(w, http.StatusBadRequest, "Maglumat nädogry", "Formany gaýtadan dolduryň")
		return
	}

	t, err := s.st.TeacherByLogin(r.Context(), in.Login)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(t.PasswordHash), []byte(in.Password)) != nil {
		fail(w, http.StatusUnauthorized, "Login ýa-da açar söz nädogry", "Maglumatlary barlap gaýtadan synanyşyň")
		return
	}

	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		fail(w, http.StatusInternalServerError, "Içerki ýalňyşlyk", "Gaýtadan synanyşyň")
		return
	}
	token := hex.EncodeToString(raw)

	s.mu.Lock()
	s.sessions[token] = t.ID
	s.mu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name: cookieName, Value: token, Path: "/",
		HttpOnly: true, SameSite: http.SameSiteLaxMode,
		MaxAge: 12 * 3600,
	})
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(cookieName); err == nil {
		s.mu.Lock()
		delete(s.sessions, c.Value)
		s.mu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: "", Path: "/", MaxAge: -1})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, teacherOf(r))
}

// ---------- Справочники ----------

func (s *Server) listSubjects(w http.ResponseWriter, r *http.Request) {
	out, err := s.st.ListSubjects(r.Context())
	if err != nil {
		fail(w, http.StatusInternalServerError, "Dersleri okap bolmady", "Sahypany täzeläň")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) listLessons(w http.ResponseWriter, r *http.Request) {
	out, err := s.st.ListLessons(r.Context(), teacherOf(r).ID, r.URL.Query().Get("subject"))
	if err != nil {
		fail(w, http.StatusInternalServerError, "Sapaklary okap bolmady", "Sahypany täzeläň")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) listTests(w http.ResponseWriter, r *http.Request) {
	out, err := s.st.ListTests(r.Context(), teacherOf(r).ID)
	if err != nil {
		fail(w, http.StatusInternalServerError, "Synaglary okap bolmady", "Sahypany täzeläň")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	// Локальная сеть: адрес хоста меняется (router, hotspot), origin не ограничиваем.
	CheckOrigin: func(*http.Request) bool { return true },
}

func hubEvent(t string, data any) hub.Event { return hub.Event{Type: t, Data: data} }

// localIPv4 — адреса этого компьютера в локальной сети. Учитель открывает
// панель через localhost (иначе браузер не даёт показывать экран), поэтому
// адрес для учеников подсказывает сервер, а не строка браузера.
func localIPv4() []string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil
	}

	var out []string
	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if !ok || ipnet.IP.IsLoopback() || ipnet.IP.IsLinkLocalUnicast() {
			continue
		}
		if ip4 := ipnet.IP.To4(); ip4 != nil {
			out = append(out, ip4.String())
		}
	}
	return out
}

func (s *Server) netInfo(w http.ResponseWriter, r *http.Request) {
	_, port, err := net.SplitHostPort(r.Host)
	if err != nil || port == "" {
		port = "8080"
	}

	ips := localIPv4()
	urls := make([]string, 0, len(ips))
	for _, ip := range ips {
		urls = append(urls, net.JoinHostPort(ip, port)+"/okuwcy")
	}
	writeJSON(w, http.StatusOK, map[string]any{"student_urls": urls})
}

func nowAfter(t *time.Time) bool { return t != nil && time.Now().After(*t) }

// urlEscape кодирует имя файла для заголовка Content-Disposition.
func urlEscape(s string) string { return url.PathEscape(s) }

func contextWithTimeout(r *http.Request, d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), d)
}
