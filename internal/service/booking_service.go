package service

import (
	"errors"
	"time"

	"studyroom/internal/repo"
)

type BookingService interface {
	CreateRoom(name string, capacity int) (string, error)
	ListRooms() ([]repo.RoomRow, error)
	SetRoomSchedule(roomID string, start, end string, isOpen bool) error
	CreateBooking(roomID, userID string, start, end string) (string, error)
	CancelBooking(bookingID, userID string) error
	JoinWaitlist(roomID, userID string, start, end string) error
}

type bookingService struct {
	rooms repo.RoomRepo
	book  repo.BookingRepo
	wait  repo.WaitlistRepo
}

func NewBookingService(r repo.RoomRepo, b repo.BookingRepo, w repo.WaitlistRepo) BookingService {
	return &bookingService{rooms: r, book: b, wait: w}
}

func (s *bookingService) CreateRoom(name string, capacity int) (string, error) {
	if name == "" || capacity <= 0 { return "", errors.New("invalid room") }
	return s.rooms.Create(name, capacity)
}
func (s *bookingService) ListRooms() ([]repo.RoomRow, error) { return s.rooms.List() }

func (s *bookingService) SetRoomSchedule(roomID string, start, end string, isOpen bool) error {
	if _, err := time.Parse(time.RFC3339, start); err != nil { return err }
	if _, err := time.Parse(time.RFC3339, end); err != nil { return err }
	if end <= start { return errors.New("end must be after start") }
	return s.rooms.SetSchedule(roomID, start, end, isOpen)
}

func (s *bookingService) CreateBooking(roomID, userID string, start, end string) (string, error) {
	if end <= start { return "", errors.New("invalid time range") }
	ok, err := s.rooms.IsWithinOpenSchedule(roomID, start, end)
	if err != nil { return "", err }
	if !ok { return "", errors.New("room not open in this interval") }
	over, err := s.book.HasOverlap(roomID, start, end)
	if err != nil { return "", err }
	if over { return "", errors.New("room already booked in this interval") }
	return s.book.Create(roomID, userID, start, end)
}

func (s *bookingService) CancelBooking(bookingID, userID string) error {
	roomID, _, start, end, status, err := s.book.GetByID(bookingID)
	if err != nil { return err }
	if status != "confirmed" { return nil }
	if err := s.book.Cancel(bookingID, userID); err != nil { return err }
	if uid, ok, err := s.wait.DequeueFirst(roomID, start, end); err == nil && ok {
		if over, _ := s.book.HasOverlap(roomID, start, end); !over {
			_, _ = s.book.Create(roomID, uid, start, end)
		}
	}
	return nil
}

func (s *bookingService) JoinWaitlist(roomID, userID string, start, end string) error {
	if end <= start { return errors.New("invalid time range") }
	return s.wait.Enqueue(roomID, userID, start, end)
}
