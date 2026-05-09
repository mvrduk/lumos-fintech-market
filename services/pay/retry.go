package main

import (
	"fmt"
	"time"
)

func retry(attempts int, sleep time.Duration, operation func() error) error {
	var err error

	for i := 0; i < attempts; i++ {
		err = operation()

		if err == nil {
			return nil
		}

		if i < attempts-1 {
			time.Sleep(sleep)
		}
	}

	return fmt.Errorf("after %d attempts, last error: %w")
}
