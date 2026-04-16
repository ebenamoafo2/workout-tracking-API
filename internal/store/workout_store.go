package store

import "database/sql"

// Workout represents a workout session in the database
type Workout struct {
	ID              int            `json:"id"`
	UserID          int            `json:"user_id"`
	Title           string         `json:"title"`
	Description     string         `json:"description"`
	DurationMinutes int            `json:"duration_minutes"`
	CaloriesBurned  int            `json:"calories_burned"`
	Entries         []WorkoutEntry `json:"entries"`
}

// WorkoutEntry represents a single exercise within a workout
type WorkoutEntry struct {
	ID              int      `json:"id"`
	ExerciseName    string   `json:"exercise_name"`
	Sets            int      `json:"sets"`
	Reps            *int     `json:"reps"`             // pointer so it can be nil (e.g. for time-based exercises)
	DurationSeconds *int     `json:"duration_seconds"` // pointer so it can be nil (e.g. for rep-based exercises)
	Weight          *float64 `json:"weight"`           // pointer so it can be nil (e.g. bodyweight exercises)
	Notes           string   `json:"notes"`
	OrderIndex      int      `json:"order_index"` // controls the display order of exercises in a workout
}

// PostgresWorkoutStore holds a reference to the database
type PostgresWorkoutStore struct {
	db *sql.DB
}

// NewPostgresWorkoutStore creates a new store using the given database connection
func NewPostgresWorkoutStore(db *sql.DB) *PostgresWorkoutStore {
	return &PostgresWorkoutStore{db: db}
}

// WorkoutStore defines the operations available for managing workouts
type WorkoutStore interface {
	CreateWorkout(*Workout) (*Workout, error)
	GetWorkoutByID(id int64) (*Workout, error)
	UpdateWorkout(*Workout) error
	DeleteWorkout(id int64) error
	GetWorkoutOwner(id int64) (int, error)
}

// CreateWorkout inserts a new workout and its entries into the database.
// It uses a transaction so that if anything fails, nothing is saved.
func (pg *PostgresWorkoutStore) CreateWorkout(workout *Workout) (*Workout, error) {
	// start a transaction — all inserts succeed together or roll back together
	tx, err := pg.db.Begin()
	if err != nil {
		return nil, err
	}
	// if anything goes wrong before Commit(), this will undo all changes
	defer tx.Rollback()

	query :=
		`
  INSERT INTO workouts (user_id, title, description, duration_minutes, calories_burned)
  VALUES ($1, $2, $3, $4, $5)
  RETURNING id 
  `

	// insert the workout and get back the auto-generated ID
	err = tx.QueryRow(query, workout.UserID, workout.Title, workout.Description, workout.DurationMinutes, workout.CaloriesBurned).Scan(&workout.ID)
	if err != nil {
		return nil, err
	}

	// insert each exercise entry linked to the workout
	for _, entry := range workout.Entries {
		query := `
    INSERT INTO workout_entries (workout_id, exercise_name, sets, reps, duration_seconds, weight, notes, order_index)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    RETURNING id
    `
		err = tx.QueryRow(query, workout.ID, entry.ExerciseName, entry.Sets, entry.Reps, entry.DurationSeconds, entry.Weight, entry.Notes, entry.OrderIndex).Scan(&entry.ID)
		if err != nil {
			return nil, err
		}
	}

	// commit the transaction — makes all changes permanent
	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return workout, nil
}

// GetWorkoutByID fetches a workout and all its entries by the workout's ID.
// Returns nil if no workout is found.
func (pg *PostgresWorkoutStore) GetWorkoutByID(id int64) (*Workout, error) {
	workout := &Workout{}

	query := `
  SELECT id, title, description, duration_minutes, calories_burned
  FROM workouts
  WHERE id = $1
  `
	err := pg.db.QueryRow(query, id).Scan(&workout.ID, &workout.Title, &workout.Description, &workout.DurationMinutes, &workout.CaloriesBurned)

	// no workout found — return nil instead of an error
	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	// fetch all entries belonging to this workout, ordered by their position
	entryQuery := `
  SELECT id, exercise_name, sets, reps, duration_seconds, weight, notes, order_index
  FROM workout_entries
  WHERE workout_id = $1
  ORDER BY order_index
  `

	rows, err := pg.db.Query(entryQuery, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close() // always close rows when done to free up the connection

	// loop through each row and build the entries slice
	for rows.Next() {
		var entry WorkoutEntry
		err = rows.Scan(
			&entry.ID,
			&entry.ExerciseName,
			&entry.Sets,
			&entry.Reps,
			&entry.DurationSeconds,
			&entry.Weight,
			&entry.Notes,
			&entry.OrderIndex,
		)
		if err != nil {
			return nil, err
		}
		workout.Entries = append(workout.Entries, entry)
	}

	return workout, nil
}

// UpdateWorkout updates a workout's details and replaces all its entries.
// It uses a transaction to ensure the update is atomic.
func (pg *PostgresWorkoutStore) UpdateWorkout(workout *Workout) error {
	tx, err := pg.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// update the main workout record
	query := `
  UPDATE workouts
  SET title = $1, description = $2, duration_minutes = $3, calories_burned = $4
  WHERE id = $5
  `
	result, err := tx.Exec(query, workout.Title, workout.Description, workout.DurationMinutes, workout.CaloriesBurned, workout.ID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	// if no rows were updated, the workout doesn't exist
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	// delete old entries and re-insert the new ones (simpler than diffing changes)
	_, err = tx.Exec(`DELETE FROM workout_entries WHERE workout_id = $1`, workout.ID)
	if err != nil {
		return err
	}

	// insert the updated entries
	for _, entry := range workout.Entries {
		query := `
    INSERT INTO workout_entries (workout_id, exercise_name, sets, reps, duration_seconds, weight, notes, order_index)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    `

		_, err := tx.Exec(query,
			workout.ID,
			entry.ExerciseName,
			entry.Sets,
			entry.Reps,
			entry.DurationSeconds,
			entry.Weight,
			entry.Notes,
			entry.OrderIndex,
		)

		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// DeleteWorkout removes a workout by ID.
// Returns an error if no workout with that ID exists.
func (pg *PostgresWorkoutStore) DeleteWorkout(id int64) error {
	query := `
  DELETE from workouts
  WHERE id = $1
  `

	result, err := pg.db.Exec(query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	// if nothing was deleted, the workout didn't exist
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// GetWorkoutOwner returns the user ID of whoever owns the given workout.
// Useful for checking if the current user is authorized to modify it.
func (pg *PostgresWorkoutStore) GetWorkoutOwner(workoutID int64) (int, error) {
	var userID int

	query := `
  SELECT user_id
  FROM workouts
  WHERE id = $1
  `

	err := pg.db.QueryRow(query, workoutID).Scan(&userID)
	if err != nil {
		return 0, err
	}

	return userID, nil
}
