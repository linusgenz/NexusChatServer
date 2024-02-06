package croc

import (
	"fmt"
	"github.com/robfig/cron/v3"
	"webserver/internal/helper"
)

func DeleteExpiredInviteLinks() {
	c := cron.New()

	_, err := c.AddFunc("0 2 * * 0", func() {
		helper.DeleteExpiredInviteLinks()
	})
	if err != nil {
		fmt.Println("Error scheduling cron job:", err)
		return
	}

	// Start the cron scheduler
	c.Start()

	// Keep the application running
	select {}
}
