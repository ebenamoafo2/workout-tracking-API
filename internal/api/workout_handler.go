package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type WorkoutHander struct{}

func NewWorkoutHandler() *WorkoutHander {
	return &WorkoutHander{}
}

func (wh *WorkoutHander) HandleGetWorkoutById(w http.ResponseWriter, r *http.Request) {
	paramsWrokoutID := chi.URLParam(r,"id")

	if paramsWrokoutID == ""{
		http.NotFound(w,r)
		return
	}

	workoutID, err := strconv.ParseInt(paramsWrokoutID,10,64)
	if err != nil{
		http.NotFound(w,r)
		return
	}

	fmt.Fprintf(w, "This is the workout id %d\n", workoutID)
}

func (wh *WorkoutHander) HandleCreateWorkout(w http.ResponseWriter, r *http.Request){
	fmt.Fprintf(w,"Created a workout\n")	
}