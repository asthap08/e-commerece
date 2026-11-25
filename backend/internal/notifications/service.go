package notifications

import "log/slog"

type Service struct{ log *slog.Logger }

func NewService(log *slog.Logger) *Service { return &Service{log: log} }

func (s *Service) Send(n Notification) {
	s.log.Info("notification", "to", n.To, "subject", n.Subject)
}
