package strategy

// Pass contract strategy uuid
type EventsCh struct {
	// Enable a strategy that will be launched immediately
	Enable chan string

	// Disable a strategy that will be halted immediately
	Disable chan string

	// Restart a strategy for reloading the new data from DB
	Restart chan string

	// To mark the strategy that is out of sync
	OutOfSync chan string

	// Reset contract strategy after fixing some data out of sync
	Reset chan string
}
