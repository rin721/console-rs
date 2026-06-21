package config

import "fmt"

func errInteractiveUnavailable() error {
	return fmt.Errorf("interactive UI is not available")
}
