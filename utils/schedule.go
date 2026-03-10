package utils

import "time"

// TimeSlot merepresentasikan rentang waktu [Start, End).
// End tidak termasuk (half-open interval), sehingga slot bersebelahan tidak konflik.
type TimeSlot struct {
	Start time.Time
	End   time.Time
}

// NewSlot membuat TimeSlot dari waktu mulai dan durasi dalam menit.
func NewSlot(start time.Time, durationMinutes int) TimeSlot {
	return TimeSlot{
		Start: start,
		End:   start.Add(time.Duration(durationMinutes) * time.Minute),
	}
}

// Overlaps mengembalikan true jika dua slot saling bertabrakan.
// Dua slot bersebelahan tepat (a.End == b.Start) TIDAK dianggap konflik.
// Contoh: [09:00-10:00) dan [10:00-11:00) → tidak konflik.
//         [09:00-10:30) dan [10:00-11:00) → konflik (overlap 30 menit).
func (a TimeSlot) Overlaps(b TimeSlot) bool {
	return a.Start.Before(b.End) && b.Start.Before(a.End)
}