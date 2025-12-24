package classifier

import "fmt"

// InsufficientDataError is returned when Classify is called but the classifier
// has not yet learned enough URLs to produce reliable patterns.
type InsufficientDataError struct {
	Count int
}

func (e *InsufficientDataError) Error() string {
	return fmt.Sprintf("insufficient data: only %d URLs learned", e.Count)
}
