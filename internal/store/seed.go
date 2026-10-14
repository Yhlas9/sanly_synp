package store

import (
	"context"
	"os"

	"golang.org/x/crypto/bcrypt"
)

// Пять предметов из раздела 8 ТЗ.
var defaultSubjects = []struct {
	Code, NameTK string
	Order        int
}{
	{"matematika", "Matematika", 1},
	{"fizika", "Fizika", 2},
	{"himiya", "Himiýa", 3},
	{"biologiya", "Biologiýa", 4},
	{"informatika", "Informatika", 5},
}

// EnsureSeed создаёт справочники и учётную запись учителя при первом запуске.
func (s *Store) EnsureSeed(ctx context.Context) error {
	const qSubject = `insert into subjects (code, name_tk, sort_order)
	                  values ($1, $2, $3)
	             on conflict (code) do update set name_tk = excluded.name_tk`

	for _, sub := range defaultSubjects {
		if _, err := s.Pool.Exec(ctx, qSubject, sub.Code, sub.NameTK, sub.Order); err != nil {
			return err
		}
	}

	var exists bool
	if err := s.Pool.QueryRow(ctx, `select exists(select 1 from teachers)`).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("mugallym"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	const qTeacher = `insert into teachers (login, password_hash, full_name, role)
	                  values ($1, $2, $3, $4) on conflict (login) do nothing`

	if _, err := s.Pool.Exec(ctx, qTeacher, "mugallym", string(hash), "Synag mugallymy", "teacher"); err != nil {
		return err
	}

	adminHash, err := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = s.Pool.Exec(ctx, qTeacher, "admin", string(adminHash), "Ulgam dolandyryjysy", "admin")
	return err
}

// SeedContent заливает образцовый учебный материал по пяти предметам.
// Выполняется один раз: скрипт сам проверяет, что уроков ещё нет.
func (s *Store) SeedContent(ctx context.Context, path string) error {
	body, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = s.Pool.Exec(ctx, string(body))
	return err
}
