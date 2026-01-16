package service

import "sync/atomic"

type HealthService struct {
	live  atomic.Bool
	ready atomic.Bool
}

func NewHealthService() *HealthService {
	s := &HealthService{}
	s.live.Store(true)
	s.ready.Store(false) // 啟動完成後再打開
	return s
}

func (s *HealthService) SetReady(v bool) {
	s.ready.Store(v)
}

func (s *HealthService) IsLive() bool {
	return s.live.Load()
}

func (s *HealthService) IsReady() bool {
	return s.ready.Load()
}
