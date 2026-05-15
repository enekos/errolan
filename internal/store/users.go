package store

import (
	"database/sql"
	"time"

	"github.com/enekos/errolan/internal/models"
)

const userCols = `id, email, name, password_hash, is_admin, is_banned, created_at`

func scanUser(row scanner) (*models.User, error) {
	var u models.User
	var admin, banned int
	if err := row.Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &admin, &banned, &u.CreatedAt); err != nil {
		return nil, err
	}
	u.IsAdmin = admin != 0
	u.IsBanned = banned != 0
	return &u, nil
}

func (s *Store) CreateUser(email, name, passwordHash string, isAdmin bool) (*models.User, error) {
	u := &models.User{
		Email:        email,
		Name:         name,
		PasswordHash: passwordHash,
		IsAdmin:      isAdmin,
		CreatedAt:    time.Now().Unix(),
	}
	res, err := s.DB.Exec(
		`INSERT INTO users (email, name, password_hash, is_admin, created_at) VALUES (?, ?, ?, ?, ?)`,
		u.Email, u.Name, u.PasswordHash, boolInt(u.IsAdmin), u.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	u.ID = id
	return u, nil
}

func (s *Store) UserByEmail(email string) (*models.User, error) {
	row := s.DB.QueryRow(`SELECT `+userCols+` FROM users WHERE email = ?`, email)
	u, err := scanUser(row)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return u, err
}

func (s *Store) UserByID(id int64) (*models.User, error) {
	row := s.DB.QueryRow(`SELECT `+userCols+` FROM users WHERE id = ?`, id)
	u, err := scanUser(row)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return u, err
}

func (s *Store) ListUsers(limit, offset int) ([]*models.User, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.DB.Query(
		`SELECT `+userCols+` FROM users ORDER BY id DESC LIMIT ? OFFSET ?`,
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *Store) SetUserBanned(id int64, banned bool) error {
	_, err := s.DB.Exec(`UPDATE users SET is_banned = ? WHERE id = ?`, boolInt(banned), id)
	return err
}

func (s *Store) CountUsers() (int, error) {
	var n int
	err := s.DB.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}
