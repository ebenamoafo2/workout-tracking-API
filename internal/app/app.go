package app

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/ebenamoafo2/workout-tracking/internal/api"
)

type Application struct {
	Logger *log.Logger
	WorkoutHandler *api.WorkoutHander
}

func NewApplication() (*Application, error) {
	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)


	// stores will go here 


	//handlers will go here
	workoutHandler := api.NewWorkoutHandler()

	
	app := &Application{
		Logger: logger,
		WorkoutHandler: workoutHandler,
	}

	return app, nil
}

// HealthCheck function
func (a *Application) HealthCheck(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Status is available")
}
