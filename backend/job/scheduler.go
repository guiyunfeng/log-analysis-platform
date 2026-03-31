package job

import (
	"log"
	"time"

	"log-analysis-platform/service"
)

// Scheduler runs periodic tasks
type Scheduler struct {
	ticker   *time.Ticker
	quit     chan struct{}
	warnBuf  []service.BatchWarningItem
}

var DefaultScheduler *Scheduler

func NewScheduler() *Scheduler {
	return &Scheduler{
		quit: make(chan struct{}),
	}
}

// Start begins the scheduler goroutines
func (s *Scheduler) Start() {
	// Run analysis every minute
	s.ticker = time.NewTicker(1 * time.Minute)

	go func() {
		log.Println("Scheduler started")
		for {
			select {
			case <-s.ticker.C:
				s.runAnalysis()
			case <-s.quit:
				s.ticker.Stop()
				log.Println("Scheduler stopped")
				return
			}
		}
	}()
}

// Stop halts the scheduler
func (s *Scheduler) Stop() {
	close(s.quit)
}

func (s *Scheduler) runAnalysis() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Analysis panic recovered: %v", r)
		}
	}()

	if service.DefaultAnalyzer != nil {
		service.DefaultAnalyzer.RunAnalysis()
	}
}
