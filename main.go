package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	redirectCode, _ := strconv.Atoi(os.Getenv("REDIRECT_CODE"))
	if redirectCode < 300 || redirectCode > 300 {
		redirectCode = http.StatusFound
	}

	redirectTo := os.Getenv("REDIRECT_TO")

	log.Println("redirect-backend")
	log.Println("code:", redirectCode)
	log.Println("to:", redirectTo)

	srv := http.Server{
		Addr:    ":"+port,
		Handler: http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, redirectTo + r.RequestURI, redirectCode)
		}),
	}

	log.Printf("start server on %s\n", port)
	go func() {
		err := srv.ListenAndServe()
		if err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	srv.Shutdown(ctx)
}
