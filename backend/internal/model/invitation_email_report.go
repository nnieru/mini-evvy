package model

type InvitationEmailReportRow struct {
	GuestName    string
	CategoryName string
	SeatCode     string
	GuestEmail   string
	TicketCode   string
	JobStatus    JobStatus
}
