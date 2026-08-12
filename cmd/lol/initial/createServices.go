package initial

import (
	"strconv"

	"lol/internal/config"
	"lol/internal/database"
	"lol/internal/reminder"
	"lol/internal/server"
	"lol/internal/sms"

	"github.com/go-dev-frame/sponge/pkg/app"
)

// CreateServices create http service
func CreateServices() []app.IServer {
	var cfg = config.Get()
	var servers []app.IServer

	// create a http service
	httpAddr := ":" + strconv.Itoa(cfg.HTTP.Port)
	httpServer := server.NewHTTPServer(httpAddr,
		server.WithHTTPIsProd(cfg.App.Env == "prod"),
	)
	servers = append(servers, httpServer)

	if cfg.SMS.Enabled {
		sender, err := sms.NewTencentSender(cfg.SMS)
		if err != nil {
			panic("init Tencent SMS sender error: " + err.Error())
		}
		reminderService, err := reminder.NewService(database.GetDB(), sender, cfg.SMS)
		if err != nil {
			panic("init SMS reminder scheduler error: " + err.Error())
		}
		servers = append(servers, reminderService)
	}

	return servers
}
