package helper

import (
	"fmt"
	"log"
	"webserver/internal/config"
)

func DeleteExpiredInviteLinks() {
	tx, err := config.UseDBPool().DB.Begin()
	if err != nil {
		log.Println(err)
		return
	}

	_, err = tx.Exec("DELETE FROM invite_links WHERE created_at < datetime('now', '-7 days');")
	if err != nil {
		fmt.Println("Error deleting expired entries:", err)
		return
	}

	fmt.Println("Expired entries deleted successfully")
}
