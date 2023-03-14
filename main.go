package main

import (
	_ "embed"
	"encoding/json"
	"html/template"
	"net/http"
	"net/mail"
	"os"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const DEFAULT_LISTEN = ":8080"
const DEFAULT_DATABASE_FILE = "nivenly-clae.db"

type Contributor struct {
	gorm.Model
	LegalName      string
	Email          string
	GithubUsername string `gorm:"index:idx_ghuser"`
	Agreed         bool
	RemoteAddr     string
}

func main() {
	var err error
	log.Infof("Starting clae")

	dbfile := DEFAULT_DATABASE_FILE
	if len(os.Getenv("DATABASE")) > 0 {
		dbfile = os.Getenv("DATABASE")
	}

	listen := DEFAULT_LISTEN
	if len(os.Getenv("LISTEN")) > 0 {
		listen = os.Getenv("LISTEN")
	}

	clae := CLAE{}

	log.Infof("Connecting to sqlite")
	clae.DB, err = gorm.Open(sqlite.Open(dbfile), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("Cannot open sqlite db: %v", err)
	}

	err = clae.DB.AutoMigrate(&Contributor{})
	if err != nil {
		log.Fatalf("AutoMigrate failed: %v", err)
	}

	http.HandleFunc("/", clae.FormHandler)
	http.HandleFunc("/logo", LogoHandler)
	http.HandleFunc("/contributor", clae.ContributorHandler)
	http.HandleFunc("/dump", clae.DumpHandler)

	log.Infof("listening on %s", listen)
	http.ListenAndServe(listen, nil)
}

//go:embed html/nivenly.png
var logo []byte

func LogoHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Write(logo)
}

type CLAE struct {
	DB *gorm.DB
}

func (c *CLAE) FormHandler(w http.ResponseWriter, r *http.Request) {
	remoteAddr := r.RemoteAddr
	// Note: the "Novaproxy-For" header is a value set on Alice in the water tower!
	if forwardedFor := r.Header.Get("Novaproxy-For"); len(forwardedFor) > 0 {
		remoteAddr = forwardedFor
	}

	switch r.Method {
	case http.MethodGet:
		renderForm(w, "")
		break
	case http.MethodPost:
		if len(r.FormValue("legalname")) > 128 {
			log.WithField("RemoteAddr", remoteAddr).Infof("legal name too long")
			renderForm(w, "Legal name too long (max 128 chars)")
			return
		}

		if len(r.FormValue("email")) > 128 {
			log.WithField("RemoteAddr", remoteAddr).Infof("email too long")
			renderForm(w, "Email too long (max 128 chars)")
			return
		}

		if _, err := mail.ParseAddress(r.FormValue("email")); err != nil {
			log.WithField("RemoteAddr", remoteAddr).Infof("email did not parse; %v", err)
			renderForm(w, "Invalid email address")
			return
		}

		if len(r.FormValue("ghusername")) > 128 {
			log.WithField("RemoteAddr", remoteAddr).Infof("github username too long")
			renderForm(w, "GitHub Username too long (max 128 chars)")
			return
		}

		rx, _ := regexp.Compile("[a-zA-Z0-9-_]*")
		if !rx.MatchString(r.FormValue("ghusername")) {
			log.WithField("RemoteAddr", remoteAddr).Infof("github name didn't match regex")
			renderForm(w, "Invalid GitHub Username")
			return
		}

		resp, err := http.Get("https://github.com/" + r.FormValue("ghusername"))
		if err != nil || resp.StatusCode != 200 {
			log.WithField("RemoteAddr", remoteAddr).Infof("invalid github username")
			renderForm(w, "Invalid GitHub Username")
			return
		}

		if r.FormValue("agreed") != "on" {
			log.WithField("RemoteAddr", remoteAddr).Infof("did not accept CLA")
			renderForm(w, "Please tick the checkbox to agree the CLA.")
			return
		}

		cont := Contributor{
			LegalName:      r.FormValue("legalname"),
			Email:          r.FormValue("email"),
			GithubUsername: r.FormValue("ghusername"),
			Agreed:         true,
			RemoteAddr:     remoteAddr,
		}

		txres := c.DB.Save(&cont)
		if err := txres.Error; err != nil {
			log.Errorf("Could not save to database: %v", err)
			renderForm(w, "Internal Server Error")
			return
		}

		log.WithField("RemoteAddr", remoteAddr).Infof("%s signed the CLA", cont.GithubUsername)

		renderOK(w, "")
		break
	default:
		w.WriteHeader(405)
	}
}

func (c *CLAE) DumpHandler(w http.ResponseWriter, r *http.Request) {
	remoteAddr := r.RemoteAddr
	if forwardedFor := r.Header.Get("Novaproxy-For"); len(forwardedFor) > 0 {
		remoteAddr = forwardedFor
	}

	providedToken := strings.TrimSpace(r.URL.Query().Get("token"))
	expectedToken := strings.TrimSpace(os.Getenv("TOKEN"))
	if providedToken != expectedToken {
		log.WithFields(log.Fields{"RemoteAddr": remoteAddr}).Errorf("invalid token for GET /dump")
		w.WriteHeader(403)
		return
	}

	results := []Contributor{}
	txres := c.DB.Find(&results)
	if err := txres.Error; err != nil {
		log.Errorf("Could not query database for contributors: %v", err)
		w.WriteHeader(500)
	}

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	err := enc.Encode(results)
	if err != nil {
		log.Errorf("Could not marshal github usernames: %v", err)
		w.WriteHeader(500)
	}
}

func (c *CLAE) ContributorHandler(w http.ResponseWriter, r *http.Request) {
	remoteAddr := r.RemoteAddr
	if forwardedFor := r.Header.Get("Novaproxy-For"); len(forwardedFor) > 0 {
		remoteAddr = forwardedFor
	}

	providedToken := strings.TrimSpace(r.URL.Query().Get("token"))
	expectedToken := strings.TrimSpace(os.Getenv("TOKEN"))
	if providedToken != expectedToken {
		log.WithField("RemoteAddr", remoteAddr).Errorf("invalid token for GET /contributor")
		w.WriteHeader(403)
		return
	}

	res := map[string]bool{}

	ghname := r.URL.Query().Get("checkContributor")
	if len(ghname) < 1 || len(ghname) > 128 {
		log.Errorf("invalid checkContributor URL param")
		w.WriteHeader(400)
		return
	}

	rx, _ := regexp.Compile("[a-zA-Z0-9-_]*")
	if !rx.MatchString(r.FormValue("ghusername")) {
		log.Errorf("invalid checkContributor GH username")
		w.WriteHeader(400)
		return
	}

	cont := Contributor{}
	txres := c.DB.Where("github_username LIKE ?", ghname).First(&cont)
	res["isContributor"] = true
	if err := txres.Error; err != nil {
		log.Errorf("Could not find contributor: %v", err)
		res["isContributor"] = false
	}

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(res)
	if err != nil {
		log.Errorf("Could not marshal github usernames: %v", err)
		w.WriteHeader(500)
	}
}

func renderForm(w http.ResponseWriter, errMsg string) {
	tmpl, err := template.ParseFiles("html/form.html")
	if err != nil {
		log.Errorf("Could not read html/form.html")
		w.WriteHeader(500)
		w.Write([]byte("Internal Server Error"))
	}

	w.WriteHeader(200)
	tmpl.Execute(w, map[string]string{"ErrMsg": errMsg})
}

func renderOK(w http.ResponseWriter, redirectUrl string) {
	tmpl, err := template.ParseFiles("html/ok.html")
	if err != nil {
		log.Errorf("Could not read html/ok.html")
		w.WriteHeader(500)
		w.Write([]byte("Internal Server Error"))
	}

	w.WriteHeader(200)
	tmpl.Execute(w, map[string]string{"RedirectUrl": redirectUrl})
}
