package sims

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/mpapenbr/goirsdk/irsdk"
)

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

		api := irsdk.NewIrsdk()
		// game loop
		for {
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
