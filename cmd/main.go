package main

import (
	"log"
	"net/http"

	"github.com/Vasu1712/scenyx-backend/internal/api/dms"
	"github.com/Vasu1712/scenyx-backend/internal/storage/memory"
	"github.com/Vasu1712/scenyx-backend/internal/ws"
)

func main() {
	dmStore := memory.NewDMStore()
	hub := ws.NewHub()
	go hub.Run()

	dmHandler := &dms.DMHandler{Store: dmStore, Hub: hub}

	http.HandleFunc("/api/v1/dms/start", dmHandler.StartOrGetConversation)
	http.HandleFunc("/api/v1/dms/list", dmHandler.ListConversations)
	http.HandleFunc("/api/v1/dms/messages", dmHandler.GetMessages)
	http.HandleFunc("/api/v1/dms/send", dmHandler.SendMessage)
	http.HandleFunc("/ws/dms", dmHandler.ServeWS)

	log.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
