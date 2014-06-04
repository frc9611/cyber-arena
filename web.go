// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Configuration and functions for the event server web interface.

package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"html/template"
	"log"
	"net/http"
	"os/exec"
	"runtime"
	"strconv"
)

const httpPort = 8080

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	template, err := template.ParseFiles("templates/index.html", "templates/base.html")
	if err != nil {
		handleWebErr(w, err)
		return
	}
	err = template.ExecuteTemplate(w, "base", nil)
	if err != nil {
		handleWebErr(w, err)
		return
	}
}

func ServeWebInterface() {
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))
	http.Handle("/", newHandler())
	log.Printf("Serving HTTP requests on port %d", httpPort)

	// Open in Default Web Browser
	// Necessary to Authenticate
	url := "http://localhost:"+strconv.Itoa(httpPort)
	var err error
	switch runtime.GOOS {
		case "linux":
		    err = exec.Command("xdg-open", url).Start()
		case "darwin":
		    err = exec.Command("open", url).Start()
		case "windows":
		    err = exec.Command(`rundll32.exe`, "url.dll,FileProtocolHandler", url).Start()
		default:
		    err = fmt.Errorf("unsupported platform")
	}
    if err != nil {
        println(err.Error())
    }

	// Start Server
	http.ListenAndServe(fmt.Sprintf(":%d", httpPort), nil)
}

func newHandler() http.Handler {
	router := mux.NewRouter()
	router.HandleFunc("/reports/csv/rankings", RankingsCsvReportHandler)
	router.HandleFunc("/reports/pdf/rankings", RankingsPdfReportHandler)
	router.HandleFunc("/reports/json/rankings", RankingsJSONReportHandler)
	router.HandleFunc("/reports/csv/schedule/{type}", ScheduleCsvReportHandler)
	router.HandleFunc("/reports/pdf/schedule/{type}", SchedulePdfReportHandler)
	router.HandleFunc("/reports/csv/teams", TeamsCsvReportHandler)
	router.HandleFunc("/reports/pdf/teams", TeamsPdfReportHandler)
	router.HandleFunc("/", IndexHandler)
	return router
}

func handleWebErr(w http.ResponseWriter, err error) {
	http.Error(w, "Internal server error: "+err.Error(), 500)
}