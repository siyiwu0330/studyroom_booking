package service

import "studyroom/internal/repo"

type SearchService interface {
	FindAvailable(minCapacity int, start, end string) ([]repo.RoomRow, error)
}

type searchService struct{ rooms repo.RoomRepo }

func NewSearchService(r repo.RoomRepo, _ repo.BookingRepo) SearchService {
	return &searchService{rooms: r}
}

func (s *searchService) FindAvailable(minCapacity int, start, end string) ([]repo.RoomRow, error) {
	return s.rooms.FindAvailable(minCapacity, start, end)
}
