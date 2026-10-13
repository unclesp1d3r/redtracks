package app

// DiagnosticsService lets the frontend write to redtracks.log.
type DiagnosticsService struct{ app *App }

// NewDiagnosticsService returns the diagnostics service bound to a.
func NewDiagnosticsService(a *App) *DiagnosticsService { return &DiagnosticsService{app: a} }

// LogError records a frontend error; where names the screen or action.
func (s *DiagnosticsService) LogError(where, message string) {
	s.app.LogError(where, message)
}
