package http

// Option configures the HTTP server.
type Option func(*Server)

// WithModuleBuilders registers module HTTP server builders.
func WithModuleBuilders(builders ...ModuleBuilder) Option {
	return func(s *Server) {
		s.moduleBuilders = append(s.moduleBuilders, builders...)
	}
}
