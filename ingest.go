package main

import (
	"log"
	"time"

	"github.com/signalstoerung/reader/internal/feeds"
)

// periodicUpdates waits for a tick to be transmitted from a time.Ticker and then triggers an update of the feeds.
// It terminates when receiving anything on the q (quit) channel (or if the channel closes).
func periodicUpdates(t *time.Ticker, q chan int) {
	for {
		select {
		case <-t.C:
			log.Print("Periodic feed update triggered.")
			feeds.UpdateFeeds()
			if globalConfig.AIActive {
				triggerScoring()
			}
		case <-q:
			t.Stop()
			return
		}
	}
}
