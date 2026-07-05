package booking

import "time"

const DefaultModifyCutoffHours = 4

func (s *Service) canModifyBooking(slotStart time.Time) bool {
	cutoff := s.slotSvc.Now().Add(DefaultModifyCutoffHours * time.Hour)
	return cutoff.Before(slotStart)
}
