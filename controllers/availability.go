package controllers

import (
	"net/http"
	"strconv"
	"time"

	"go-potensia/config"
	"go-potensia/models"
	"go-potensia/utils"

	"github.com/gin-gonic/gin"
)

// bookedBlock adalah satu sesi mentor yang sudah terisi di hari tersebut.
type bookedBlock struct {
	SessionID uint
	Slot      utils.TimeSlot
}

// SlotResult adalah satu slot waktu yang dikembalikan ke frontend.
type SlotResult struct {
	Time      string `json:"time"`                 // "09:00"
	Available bool   `json:"available"`
	BlockedBy *uint  `json:"blocked_by,omitempty"` // session_id yang memblokir, untuk debugging
	FreeFrom  string `json:"free_from,omitempty"`  // jam berakhirnya sesi yang memblokir, misal "10:30"
}

// getMentorBookedBlocks mengambil semua sesi aktif mentor pada tanggal tertentu dari DB.
// Hanya status upcoming dan ongoing yang dianggap memblokir slot.
func getMentorBookedBlocks(mentorProfileID uint, date time.Time) ([]bookedBlock, error) {
	// Query tepat 1 hari — gunakan UTC midnight agar konsisten dengan penyimpanan DB
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	dayEnd := dayStart.Add(24 * time.Hour)

	type rawSession struct {
		ID          uint
		ScheduledAt time.Time
		Duration    int
	}

	var rows []rawSession
	err := config.DB.
		Model(&models.Session{}).
		Select("id, scheduled_at, duration").
		Where(
			"mentor_id = ? AND scheduled_at >= ? AND scheduled_at < ? AND status IN ?",
			mentorProfileID,
			dayStart,
			dayEnd,
			[]models.SessionStatus{models.SessionUpcoming, models.SessionOngoing},
		).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	blocks := make([]bookedBlock, len(rows))
	for i, r := range rows {
		blocks[i] = bookedBlock{
			SessionID: r.ID,
			Slot:      utils.NewSlot(r.ScheduledAt, r.Duration),
		}
	}
	return blocks, nil
}

// firstConflict memeriksa proposed slot terhadap semua blok yang terbooking.
// excludeID digunakan saat mentor reschedule sesinya sendiri (agar tidak konflik dengan diri sendiri).
// Mengembalikan pointer ke blok konflik, atau nil jika aman.
func firstConflict(blocks []bookedBlock, proposed utils.TimeSlot, excludeID uint) *bookedBlock {
	for i := range blocks {
		if blocks[i].SessionID == excludeID {
			continue
		}
		if proposed.Overlaps(blocks[i].Slot) {
			return &blocks[i]
		}
	}
	return nil
}

// GetMentorAvailability mengembalikan daftar slot waktu (granularitas 30 menit)
// yang tersedia/tidak tersedia untuk mentor pada tanggal tertentu.
//
// Digunakan frontend untuk merender kalender pemilihan jadwal sesi:
//   - slot available → bisa dipilih
//   - slot unavailable → di-disable di UI
//
// GET /api/v1/mentors/:mentor_id/availability
//
// Query params:
//
//	date     string  YYYY-MM-DD  (wajib)
//	duration int     menit       (wajib — ambil dari course_package.duration_per_session)
func GetMentorAvailability(c *gin.Context) {
	// ── Parse path param ──────────────────────────────────────────────────────
	mentorIDStr := c.Param("mentor_id")
	mentorID, err := strconv.ParseUint(mentorIDStr, 10, 32)
	if err != nil || mentorID == 0 {
		utils.BadRequest(c, "mentor_id tidak valid")
		return
	}

	// ── Parse query params ────────────────────────────────────────────────────
	dateStr := c.Query("date")
	if dateStr == "" {
		utils.BadRequest(c, "Query param 'date' wajib diisi (format: YYYY-MM-DD)")
		return
	}
	durationStr := c.Query("duration")
	if durationStr == "" {
		utils.BadRequest(c, "Query param 'duration' wajib diisi (menit, sesuai paket kursus)")
		return
	}

	duration, convErr := strconv.Atoi(durationStr)
	if convErr != nil || duration <= 0 || duration > 480 {
		utils.BadRequest(c, "duration tidak valid (harus 1–480 menit)")
		return
	}

	date, parseErr := time.Parse("2006-01-02", dateStr)
	if parseErr != nil {
		utils.BadRequest(c, "Format date tidak valid, gunakan YYYY-MM-DD")
		return
	}

	// Jangan izinkan cek hari yang sudah lewat (midnight UTC hari ini)
	todayUTC := time.Now().UTC().Truncate(24 * time.Hour)
	dateUTC := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	if dateUTC.Before(todayUTC) {
		utils.BadRequest(c, "Tidak dapat mengecek availability untuk tanggal yang sudah lewat")
		return
	}

	// ── Validasi mentor ada ───────────────────────────────────────────────────
	var mentor models.MentorProfile
	if err := config.DB.Select("id").Where("id = ?", mentorID).First(&mentor).Error; err != nil {
		utils.NotFound(c, "Mentor tidak ditemukan")
		return
	}

	// ── Ambil sesi yang sudah terbooking ─────────────────────────────────────
	blocks, err := getMentorBookedBlocks(uint(mentorID), dateUTC)
	if err != nil {
		utils.InternalError(c, "Gagal mengambil data jadwal mentor")
		return
	}

	// ── Generate slot per 30 menit, jam 07:00 – 21:00 ────────────────────────
	// Batas akhir: sesi HARUS selesai paling lambat jam 21:00
	// Contoh: duration 60 menit → slot terakhir yang boleh dimulai = 20:00
	// Contoh: duration 90 menit → slot terakhir = 19:30
	const opStartHour = 7
	const opEndHour = 21 // sesi harus sudah berakhir sebelum/tepat jam ini

	operationalEnd := time.Date(dateUTC.Year(), dateUTC.Month(), dateUTC.Day(),
		opEndHour, 0, 0, 0, time.UTC)

	var slots []SlotResult

	for h := opStartHour; h < opEndHour; h++ {
		for _, m := range []int{0, 30} {
			slotStart := time.Date(dateUTC.Year(), dateUTC.Month(), dateUTC.Day(),
				h, m, 0, 0, time.UTC)
			proposed := utils.NewSlot(slotStart, duration)

			// Slot hanya valid jika sesi akan selesai tepat/sebelum jam operasional berakhir
			if proposed.End.After(operationalEnd) {
				goto nextDay // semua slot setelah ini pun akan melewati batas, stop
			}

			{
				conflict := firstConflict(blocks, proposed, 0 /* tidak ada exclusion */)
				slot := SlotResult{
					Time:      slotStart.Format("15:04"),
					Available: conflict == nil,
				}
				if conflict != nil {
					slot.BlockedBy = &conflict.SessionID
					slot.FreeFrom = conflict.Slot.End.Format("15:04")
				}
				slots = append(slots, slot)
			}
		}
	}
nextDay:

	c.JSON(http.StatusOK, gin.H{
		"mentor_id":    mentorID,
		"date":         dateStr,
		"duration_min": duration,
		"slots":        slots,
		"total_booked": len(blocks),
	})
}

// CheckMentorConflict adalah helper internal yang dipakai oleh MentorUpdateSession
// untuk memvalidasi reschedule sebelum menyimpan ke DB.
//
// mentorProfileID : ID dari tabel mentor_profiles (bukan users.id)
// proposedStart   : waktu mulai yang diusulkan
// durationMinutes : durasi sesi dalam menit
// excludeSessionID: ID sesi yang sedang di-reschedule (dikecualikan dari cek konflik)
//
// Mengembalikan:
//
//	ok=true, nil, nil          → tidak ada konflik, aman disimpan
//	ok=false, &conflict, nil   → ada konflik, conflict berisi detail sesi yang menghalangi
//	ok=false, nil, err         → error DB
func CheckMentorConflict(mentorProfileID uint, proposedStart time.Time, durationMinutes int, excludeSessionID uint) (ok bool, conflict *bookedBlock, err error) {
	blocks, err := getMentorBookedBlocks(mentorProfileID, proposedStart)
	if err != nil {
		return false, nil, err
	}
	proposed := utils.NewSlot(proposedStart, durationMinutes)
	c := firstConflict(blocks, proposed, excludeSessionID)
	if c != nil {
		return false, c, nil
	}
	return true, nil, nil
}