package store

import (
	"database/sql"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           int64  `json:"id"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	PasswordHash password `json:"-"`
	Bio          string `json:"bio"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type password struct {
	plaintext *string
	hash      []byte
}

func (p *password) Set(plaintextPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintextPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	p.plaintext = &plaintextPassword
	p.hash = hash
	return nil
}

func (p *password) Matches(plaintextPassword string) (bool, error) {
   err := bcrypt.CompareHashAndPassword(p.hash, []byte(plaintextPassword))
   if err != nil {
	switch {
	case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
		return false, nil
	default:
		return false, err
	}
   }

   return true, nil
}


type PostgresUserStore struct {
	db *sql.DB
}

func NewPostgresUserStore(db *sql.DB) *PostgresUserStore {
	return &PostgresUserStore{db: db}
}


type UserStore interface {
	CreateUser(*User) error
	GetUserByUsername(username string) (*User, error)
	UpdateUser(*User) error
}


func (s *PostgresUserStore) CreateUser(user *User) error {
	query := `
		INSERT INTO users (username, email, password_hash, bio) 
		VALUES ($1, $2, $3, $4) 
		RETURNING id, created_at, updated_at
	`
	err := s.db.QueryRow(query,
		user.Username,
		user.Email,
		user.PasswordHash.hash,
		user.Bio,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return err
	}
	return nil
}

func (s *PostgresUserStore) GetUserByUsername(username string) (*User, error) {
	user := &User{
		PasswordHash: password{},
	}
	query := `SELECT id, username, email, password_hash, bio, created_at, updated_at FROM users WHERE username = $1`
	err := s.db.QueryRow(query, username).Scan(
		&user.ID, 
		&user.Username, 
		&user.Email, 
		&user.PasswordHash.hash, 
		&user.Bio, 
		&user.CreatedAt, 
		&user.UpdatedAt)

	// no user found — return nil instead of an error
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return user, nil
}


func (s *PostgresUserStore) UpdateUser(user *User) error {
	query := `
	UPDATE users
	SET username = $1, email = $2, bio = $3, updated_at = CURRENT_TIMESTAMP
	WHERE id = $4 
	RETURNING updated_at
	`
	result, err := s.db.Exec(query, user.Username, user.Email, user.Bio, user.ID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}