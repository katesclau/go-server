package main

import (
	"context"
	"fmt"
	"errors"
	"net/http"
	"net"
	"io"
	"os"
	"time"
	"log"

	"github.com/joho/godotenv"
)

const keyServerAddr = "serverAddr"

type Logger struct {
	handler http.Handler
}

func (l *Logger) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	start := time.Now()
	log.Printf("Received request from %s\n", ctx.Value(keyServerAddr))
	l.handler.ServeHTTP(w, r)
	log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
}

func NewLogger(handler http.Handler) *Logger {
	return &Logger{handler}
}

func getRoot(w http.ResponseWriter, r *http.Request) {
	ret := "Received request: \n"

	// Echo headers
	for key, vals := range r.Header {
		for _, val := range vals {
			ret = fmt.Sprintf("%sHeader: %s, Value: %s\n", ret, key, val)
		}
	}

	// Echo method
	ret = fmt.Sprintf("%sMethod: %s\n", ret, r.Method)

	// Echo params
	values := r.URL.Query()
	for key, vals := range values {
		for _, val := range vals {
			ret = fmt.Sprintf("%sQuery param: %s, Value: %s\n", ret, key, val)
		}
	}

	// Echo body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		ret = fmt.Sprintf("%sFailed to read request body: %v\n", ret, err)
	} else {
		ret = fmt.Sprintf("%sRequest body: %s\n", ret, string(body))
	}	

	time.Sleep(1*time.Second)

	io.WriteString(w, ret)
}

func main() {
	fmt.Println("Starting server...")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", getRoot)
	wrappedMux := NewLogger(mux)

	port := os.Getenv("PORT")
	port2 := os.Getenv("PORT2")

	serverOne := &http.Server{
		Addr: fmt.Sprintf(":%s", port),
		Handler: wrappedMux,
		BaseContext: func(listener net.Listener) context.Context {
			return context.WithValue(ctx, keyServerAddr, listener.Addr().String())
		},
	}

	serverTwo := &http.Server{
		Addr: fmt.Sprintf(":%s", port2),
		Handler: wrappedMux,
		BaseContext: func(listener net.Listener) context.Context {
			return context.WithValue(ctx, keyServerAddr, listener.Addr().String())
		},
	}

	go func() {
		err := serverOne.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			fmt.Println("Server one closed")
		} else if err != nil {
			fmt.Println("Error starting server two", err)
			cancel()
		}
	}()

	go func() {
		err := serverTwo.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			fmt.Println("Server two closed")
		} else if err != nil {
			fmt.Println("Error starting server two", err)
			cancel()
		}
	}()

	<-ctx.Done()
}