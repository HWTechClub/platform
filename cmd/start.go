package cmd

import (
	"fmt"

	"github.com/hw-cs-reps/platform/models"
	"github.com/hw-cs-reps/platform/modules/settings"
	"github.com/hw-cs-reps/platform/routes"

	"github.com/go-macaron/cache"
	"github.com/go-macaron/captcha"
	"github.com/go-macaron/csrf"
	"github.com/go-macaron/session"
	_ "github.com/go-macaron/session/mysql" // MySQL driver for persistent sessions
	"github.com/hako/durafmt"
	"github.com/urfave/cli/v2"
	macaron "gopkg.in/macaron.v1"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"
)

// CmdStart represents a command-line command
// which starts the bot.
var CmdStart = &cli.Command{
	Name:    "run",
	Aliases: []string{"start", "web"},
	Usage:   "Start the web server",
	Action:  start,
}

func start(clx *cli.Context) (err error) {
	settings.LoadConfig()
	engine := models.SetupEngine()
	defer engine.Close()

	// Run macaron
	m := macaron.Classic()
	m.Use(macaron.Renderer(macaron.RenderOptions{
		Funcs: []template.FuncMap{map[string]interface{}{
			"CalcTime": func(sTime time.Time) string {
				return fmt.Sprint(time.Since(sTime).Nanoseconds() / int64(time.Millisecond))
			},
			"EmailToUser": func(s string) string {
				if strings.Contains(s, "@") {
					return strings.Split(s, "@")[0]
				} else {
					return s
				}
			},
			"CalcDurationShort": func(unix int64) string {
				return durafmt.Parse(time.Now().Sub(time.Unix(unix, 0))).LimitFirstN(1).String()
			},
		}},
		IndentJSON: true,
	}))

	if settings.Config.DevMode {
		fmt.Println("In development mode.")
		macaron.Env = macaron.DEV
	} else {
		fmt.Println("In production mode.")
		macaron.Env = macaron.PROD
	}

	m.Use(cache.Cacher())
	sessOpt := session.Options{
		CookieLifeTime: 15778800, // 6 months
		Gclifetime:     15778800,
		CookieName:     "hithereimacookie",
	}
	if settings.Config.DBConfig.Type == settings.MySQL {
		sqlConfig := fmt.Sprintf("%s:%s@tcp(%s)/%s",
			settings.Config.DBConfig.User, settings.Config.DBConfig.Password,
			settings.Config.DBConfig.Host, settings.Config.DBConfig.Name)
		sessOpt.Provider = "mysql"
		sessOpt.ProviderConfig = sqlConfig
		sessOpt.CookieLifeTime = 0
	}
	m.Use(session.Sessioner(sessOpt))
	m.Use(csrf.Csrfer())
	m.Use(captcha.Captchaer())
	m.Use(routes.ContextInit())

	m.Get("/", routes.HomepageHandler)

	log.Printf("Starting web server on port %s\n", settings.Config.SitePort)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("0.0.0.0:%s", settings.Config.SitePort), m))
	return nil
}
