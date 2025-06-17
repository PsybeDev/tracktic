package sims

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/mpapenbr/goirsdk/irsdk"
)

type RaceStrategyData struct {
	// session stats and time
	IsConnected          bool    `json:"is_connected"`           // IsConnected indicates if the app is connected to the sim
	SessionTimeRemaining float64 `json:"session_time_remaining"` // SessionTimeRemaining is the time remaining in the session in seconds
	SessionLapsRemaining int     `json:"session_laps_remaining"` // SessionLapsRemaining is the number of laps remaining in the session
	CurrentLap           int     `json:"current_lap"`            // CurrentLap is the current lap number
	TotalLaps            int     `json:"total_laps"`             // TotalLaps is the total number of laps in the session
	SessionFlags         string  `json:"session_flags"`          // SessionFlags is the current session flags
	SessionType          string  `json:"session_type"`           // SessionType is the current session type
	IsInPitStall         bool    `json:"is_in_pit_stall"`        // IsInPitStall indicates if the car is in the pit stall
	OnTrack              bool    `json:"on_track"`               // OnTrack indicates if the car is on track

	// player stats
	PlayerCarIdx      int `json:"player_car_idx"`      // PlayerCarIdx is the index of the player car
	FuelLevel         int `json:"fuel_level"`          // FuelLevel is the fuel level in liters
	FuelUsedPerHour   int `json:"fuel_used_per_hour"`  // FuelUsedPerHour is the fuel used per hour in liters
	LastLapTime       int `json:"last_lap_time"`       // LastLapTime is the last lap time in seconds
	CurrentPosition   int `json:"current_position"`    // CurrentPosition is the current position of the player car
	LapDistPercentage int `json:"lap_dist_percentage"` // LapDistPercentage is the lap distance percentage of the player car

	// opponent context
	// TODO: Add opponent context
}

func MainLoop() {
	client := http.Client{Timeout: 10 * time.Second}
	// parent loop
	for {
		// establish connection to panic
		for {
			fmt.Println("Connecting to IRacing")
			if simIsRunning, err := irsdk.IsSimRunning(context.Background(), &client); err != nil {
				panic(err)
			} else if simIsRunning {
				fmt.Println("Connected to IRacing")
				break
			}
			time.Sleep(2 * time.Second)
		}

		tick := time.NewTicker(250 * time.Millisecond)
		defer tick.Stop()

		api := irsdk.NewIrsdk()
		// game loop
		for range tick.C {
			if api.WaitForValidData() {
				api.GetData()
				if sessionTime, err := api.GetDoubleValue("SessionTime"); err == nil {
					fmt.Printf("Session time: %f\n", sessionTime)
				} else {
					fmt.Println("Error getting session time")
				}
			}
		}
	}
}
