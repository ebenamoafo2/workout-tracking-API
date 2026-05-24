package main

import (
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/ebenamoafo2/workout-tracking/internal/app"
	"github.com/ebenamoafo2/workout-tracking/internal/routes"
)

func main() {
	var port int
	flag.IntVar(&port, "port", 8080, "go backend server port")
	flag.Parse()

	app, err := app.NewApplication()
	if err != nil {
		panic(err)
	}

	// Ensure the database connection is closed when the application exits
	defer func() {
		if err := app.DB.Close(); err != nil {
			app.Logger.Printf("ERROR: failed to close database connection: %v", err)
		}
	}()

	r := routes.SetupRoutes(app)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      r,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	app.Logger.Printf("we are running on port %d\n", port)

	err = server.ListenAndServe()
	if err != nil {
		app.Logger.Fatal(err)
	}

}
