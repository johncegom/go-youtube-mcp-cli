package cli

import (
	"fmt"
	"os"
	"time"

	"golang.org/x/term"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// spinner starts a braille-frame progress spinner on stderr and returns a
// stop function. It is a no-op (stop does nothing) when quiet is true or
// stderr is not a terminal, mirroring the TS CLI's spinner() helper.
func spinner(msg string, quiet bool) func() {
	if quiet || !term.IsTerminal(int(os.Stderr.Fd())) {
		return func() {}
	}

	stop := make(chan struct{})
	done := make(chan struct{})

	go func() {
		defer close(done)
		fmt.Fprintf(os.Stderr, "%s %s", spinnerFrames[0], msg)
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()
		i := 0
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				i++
				fmt.Fprintf(os.Stderr, "\r%s %s", spinnerFrames[i%len(spinnerFrames)], msg)
			}
		}
	}()

	return func() {
		close(stop)
		<-done
		fmt.Fprint(os.Stderr, "\r\x1b[K")
	}
}
