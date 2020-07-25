package routes

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"html/template"
	"log"
	"strconv"

	"github.com/hw-cs-reps/platform/config"
	"github.com/hw-cs-reps/platform/mailer"
	"github.com/hw-cs-reps/platform/models"

	"github.com/go-macaron/csrf"
	"github.com/go-macaron/session"
	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	macaron "gopkg.in/macaron.v1"
)

// HomepageHandler response for the home page.
func HomepageHandler(ctx *macaron.Context, sess session.Store, f *session.Flash) {
	ctx.Data["Config"] = config.Config.InstanceConfig
	ctx.Data["IsHome"] = 1
	ctx.Data["Title"] = "Class Reps"
	ctx.HTML(200, "index")
}

// ComplaintsHandler response for the complaints page.
func ComplaintsHandler(ctx *macaron.Context, sess session.Store, f *session.Flash) {
	ctx.Data["Title"] = "Complaints"
	ctx.Data["IsComplaints"] = 1
	ctx.Data["Courses"] = config.Config.InstanceConfig.Courses
	ctx.HTML(200, "complaints")
}

// ComplaintsConfirmHandler response for the complaints send confirmation page.
func ComplaintsConfirmHandler(ctx *macaron.Context, sess session.Store) {

}

// PostComplaintsHandler response for the complaints page.
func PostComplaintsHandler(ctx *macaron.Context, sess session.Store, f *session.Flash) {
	if ctx.Query("confirm") == "1" { // confirm sending
		var sender string
		if ctx.Query("email") == "" {
			sender = "anonymous"
		} else {
			sender = ctx.Query("email")
		}
		// TODO send to respective class reps
		mailer.Email("TODO", "Complaint submission", `A complaint submission
From: `+sender+`
Category: `+ctx.Query("category")+`
Title: `+ctx.Query("title")+`
Message:
`+ctx.Query("message"))

		f.Success("Your complaint was sent!")
		ctx.Redirect("/complaints")
		return
	}

	ctx.Data["Category"] = ctx.Query("category")
	ctx.Data["Subject"] = ctx.Query("subject")
	ctx.Data["Message"] = ctx.Query("message")
	ctx.Data["Email"] = ctx.Query("Email")

	var degrees []string
	for _, c := range config.Config.InstanceConfig.Courses {
		if c.Code == ctx.Query("category") {
			degrees = c.DegreeCode
			break
		}
	}

	var recipients []string

	for _, c := range config.Config.InstanceConfig.ClassReps {
		for _, d := range degrees {
			if c.DegreeCode == d {
				recipients = append(recipients, c.Name)
				break
			}
		}
	}

	ctx.Data["Recipients"] = recipients

	ctx.HTML(200, "complaints-confirm")
}

// TicketsHandler response for the tickets listing page.
func TicketsHandler(ctx *macaron.Context, sess session.Store, f *session.Flash) {
	ctx.Data["Tickets"] = models.GetTickets()
	ctx.Data["IsTickets"] = 1
	ctx.Data["Title"] = "Tickets"
	ctx.HTML(200, "tickets")
}

// markdownToHTML converts a string (in Markdown) and outputs (X)HTML.
// The input may also contain HTML, and the output is sanitized.
func markdownToHTML(s string) string {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithXHTML(),
			html.WithUnsafe(),
		),
	)
	var buf bytes.Buffer
	if err := md.Convert([]byte(s), &buf); err != nil {
		panic(err)
	}
	return string(bluemonday.UGCPolicy().SanitizeBytes(buf.Bytes()))
}

// CoursesHandler gets courses page
func CoursesHandler(ctx *macaron.Context, sess session.Store, f *session.Flash) {
	ctx.Data["Courses"] = config.Config.InstanceConfig.Courses
	ctx.Data["Title"] = "Courses"
	ctx.HTML(200, "courses")
}

// LecturerHandler gets courses page
func LecturerHandler(ctx *macaron.Context, sess session.Store, f *session.Flash) {
	ctx.Data["Lecturers"] = config.Config.InstanceConfig.Lecturers
	ctx.Data["Title"] = "Lecturers"
	ctx.HTML(200, "lecturers")
}

// PrivacyHandler gets the privacy policy page
func PrivacyHandler(ctx *macaron.Context, sess session.Store, f *session.Flash) {
	ctx.Data["Title"] = "Privacy Policy"
	ctx.HTML(200, "privacy")
}

// TicketPageHandler response for the a specific ticket..
func TicketPageHandler(ctx *macaron.Context, sess session.Store, f *session.Flash, x csrf.CSRF) {
	ticket, err := models.GetTicket(ctx.ParamsInt64("id"))
	if err != nil {
		log.Println(err)
		ctx.Redirect("/tickets")
		return
	}
	ctx.Data["csrf_token"] = x.GetToken()
	ctx.Data["FormattedPost"] = template.HTML(markdownToHTML(ticket.Description))
	ticket.LoadComments()
	ctx.Data["Ticket"] = ticket
	voterHash := userHash(getIP(ctx), ctx.Req.Header.Get("User-Agent"))
	ctx.Data["Upvoted"] = containsString(voterHash, ticket.Voters)
	ctx.HTML(200, "ticket")
}

// PostTicketPageHandler handles posting a new comment on a ticket.
func PostTicketPageHandler(ctx *macaron.Context, sess session.Store, f *session.Flash) {
	ticket, err := models.GetTicket(ctx.ParamsInt64("id"))
	if err != nil {
		log.Println(err)
		ctx.Redirect("/tickets")
		return
	}
	if ticket.IsResolved {
		ctx.Redirect("/tickets/" + ctx.Params("id"))
		return
	}

	comment := models.Comment{
		TicketID: ticket.TicketID,
		PosterID: sess.Get("id").(string),
		Text:     ctx.Query("text"),
	}

	err = models.AddComment(&comment)
	if err != nil {
		log.Println(err)
	}

	ctx.Redirect("/tickets/" + ctx.Params("id"))
}

// NewTicketsHandler response for posting new ticket.
func NewTicketHandler(ctx *macaron.Context, sess session.Store, f *session.Flash) {
	ctx.HTML(200, "new-ticket")
}

// PostNewTicketsHandler post response for posting new ticket.
func PostNewTicketHandler(ctx *macaron.Context, sess session.Store, f *session.Flash) {
	title := ctx.Query("title")
	text := ctx.Query("text")
	ticket := models.Ticket{
		Title:       title,
		Description: text,
	}
	err := models.AddTicket(&ticket)
	if err != nil {
		log.Println(err)
		f.Error("Failed to add ticket")
		ctx.Redirect("/tickets")
		return
	}
	ctx.Redirect(fmt.Sprintf("/tickets/%d", ticket.TicketID))
}

func userHash(ip string, useragent string) string {
	h := sha256.New()
	h.Write([]byte(ip + useragent + config.Config.VoterPepper))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func getIP(ctx *macaron.Context) string {
	xf := ctx.Header().Get("X-Forwarded-For")
	if len(xf) > 0 {
		return xf
	}
	return ctx.RemoteAddr()
}

func containsString(s string, arr []string) bool {
	for _, v := range arr {
		if v == s {
			return true
		}
	}
	return false
}

// UpvoteTicketHandler response for upvoting a specific ticket.
func UpvoteTicketHandler(ctx *macaron.Context, sess session.Store, f *session.Flash) {
	ticket, err := models.GetTicket(ctx.ParamsInt64("id"))
	if err != nil {
		log.Println(err)
		ctx.Redirect("/tickets")
		return
	}

	voterHash := userHash(getIP(ctx), ctx.Req.Header.Get("User-Agent"))
	if !ticket.IsResolved && !containsString(voterHash, ticket.Voters) {
		ticket.Voters = append(ticket.Voters, voterHash)
		models.UpdateTicketCols(ticket, "voters")
	}

	ctx.Redirect("/tickets/" + strconv.Itoa(ctx.ParamsInt("id")))
}

// TicketEditHandler response for adding posting new ticket.
func TicketEditHandler(ctx *macaron.Context, sess session.Store, f *session.Flash) {
	ctx.HTML(200, "new-ticket")
}

// PostTicketEditHandler response for adding posting new ticket.
func PostTicketEditHandler(ctx *macaron.Context, sess session.Store, f *session.Flash) {
}

// PostTicketDeleteHandler response for deleting a ticket.
func PostTicketDeleteHandler(ctx *macaron.Context, sess session.Store, f *session.Flash) {
}
