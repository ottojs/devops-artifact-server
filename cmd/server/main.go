package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Settings
var dataDir string = "files"
var accessKey = os.Getenv("ACCESS_KEY")

var maxRam int64 = 1024 * 1024 * 512    // 512 MiB
var maxBytes int64 = 1024 * 1024 * 1500 // 1500 MiB

type UploadReqBodyMeta struct {
	Organization string `json:"organization"`
	Project      string `json:"project"`
	Type         string `json:"type"`
	// Unused for now
	Tags []string `json:"tags"`
	// Only Used for Downloading
	// Uploads are not allowed to specify versions yet
	Version string `json:"version"`
}

func upload(w http.ResponseWriter, r *http.Request) {

	// PUT is Required
	if r.Method != "PUT" {
		//w.WriteHeader(http.StatusMethodNotAllowed) // 405
		//fmt.Fprintf(w, "Use PUT Method, Thanks\n")
		redirect(w, r)
		return
	}

	// Set Max Body Size
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

	// Generate Timestamp
	timeobj := time.Now()
	y, mon, d := timeobj.Date()
	h, min, s := timeobj.Clock()
	timestamp := fmt.Sprintf("%d-%02d-%02d-%02d-%02d-%02d", y, mon, d, h, min, s)

	// Parse Form
	err := r.ParseMultipartForm(maxRam)
	//fmt.Println("FORM", r.Form)
	//fmt.Println("FORMVAL", r.FormValue("meta"))

	if err != nil {
		if err.Error() == "http: request body too large" {
			w.WriteHeader(http.StatusRequestEntityTooLarge) // 413
			fmt.Fprintf(w, fmt.Sprintf("File too large. Maximum is %d bytes\n", maxBytes))
		} else {
			w.WriteHeader(http.StatusBadRequest) // 400
			fmt.Fprintln(w, err)
		}
		return
	}

	// Extract Meta
	meta := &UploadReqBodyMeta{}
	decoder := json.NewDecoder(strings.NewReader(r.FormValue("meta")))
	decoder.DisallowUnknownFields()

	// Decode the request body to the destination.
	err = decoder.Decode(meta)
	if err != nil {
		// EOF if meta is undefined
		w.WriteHeader(http.StatusBadRequest)       // 400
		fmt.Fprintln(w, errors.New("bad request")) // err
		return
	}

	if meta.Organization == "" || meta.Project == "" || meta.Type == "" {
		fmt.Fprintln(w, errors.New("missing required parameters"))
		return
	}

	// Read File
	source, handler, err := r.FormFile("file")
	if err != nil {
		fmt.Println("Error Reading the File", err.Error())
		return
	}
	defer source.Close()

	// Open Local File
	filedir := fmt.Sprintf("./%s/%s/%s/%s", dataDir, meta.Organization, meta.Project, meta.Type)
	_ = os.MkdirAll(filedir, 0750) // os.ModeDir

	// Sanitize: Replace "/"
	rawname := strings.ReplaceAll(handler.Filename, "/", "")
	// Split on "."
	namearr := strings.Split(rawname, ".")
	// Grab Extension
	ext := namearr[len(namearr)-1]
	// Remove Extension
	namearr = namearr[:(len(namearr) - 1)]

	// Reassemble
	// TODO: No extension creates ".(date).name"
	dest2 := fmt.Sprintf("%s/%s.%s.%s", filedir, strings.Join(namearr, "."), timestamp, ext)
	//fmt.Println(dest)

	// Clean
	dest2 = filepath.Clean(dest2)

	// Open new file
	// TODO: Fix better
	/* #nosec */
	destination, err := os.Create(dest2)
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}
	defer func() {
		err := destination.Close()
		if err != nil {
			fmt.Println(err)
		}
	}()

	// Save File
	bytestotal, _ := io.Copy(destination, source)

	// Respond
	w.WriteHeader(http.StatusCreated)
	// TODO: Write out properly
	fmt.Fprintf(w, fmt.Sprintf("Successfully Uploaded File: %d bytes\n", bytestotal))

}

func download(w http.ResponseWriter, r *http.Request) {

	query := r.URL.Query()
	accessKeyProvided := query.Get("access_key")
	if accessKeyProvided != accessKey {
		fmt.Fprintln(w, errors.New("not authorized"))
		return
	}

	organization := filepath.Clean(query.Get("organization"))
	project := filepath.Clean(query.Get("project"))
	itemtype := filepath.Clean(query.Get("type"))
	version := filepath.Clean(query.Get("version"))

	if len(organization) == 0 || len(project) == 0 || len(itemtype) == 0 {
		fmt.Fprintln(w, errors.New("missing required parameters"))
		return
	}

	// Default to Latest
	if version == "." {
		version = "latest"
	}

	meta := &UploadReqBodyMeta{
		Organization: organization,
		Project:      project,
		Type:         itemtype,
		Version:      version,
	}
	fmt.Println(meta)
	latestfileName := ""
	if version == "latest" {

		// Directory to Scan
		filedir := fmt.Sprintf("./%s/%s/%s/%s", dataDir, meta.Organization, meta.Project, meta.Type)

		_, err := os.Stat(filedir)
		if os.IsNotExist(err) {
			fmt.Fprintln(w, errors.New("not found"))
			return
		}

		files, err := ioutil.ReadDir(filedir)
		if err != nil {
			fmt.Fprintln(w, err)
			return
		}

		// Grab last one (latest)
		latestfile := files[len(files)-1]
		latestfileName = filepath.Clean(latestfile.Name())

	} else {

		// TODO: not supported
		return

	}

	// Open File
	filepath2 := filepath.Join(dataDir, meta.Organization, meta.Project, meta.Type, latestfileName)
	f, err := os.Open(filepath.Clean(filepath2))
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}
	defer func() {
		err := f.Close()
		if err != nil {
			fmt.Println(err)
		}
	}()
	finfo, err := f.Stat()
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}

	// Response
	w.Header().Set("Content-Length", fmt.Sprintf("%d", finfo.Size()))
	w.Header().Set("Content-Type", "image/png")
	//w.Header().Set("Content-Type", "application/octet-stream")
	//w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, latestfileName))
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, f)

}

func health(w http.ResponseWriter, r *http.Request) {
	// Response
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK")
}

func redirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "https://www.google.com", 301)
}

func main() {
	if accessKey == "" {
		fmt.Println("BLANK ACCESS KEY!!! Set ACCESS_KEY to a secure value")
		os.Exit(1)
		return
	}
	http.HandleFunc("/", redirect)
	http.HandleFunc("/health", health)
	http.HandleFunc("/upload", upload)
	http.HandleFunc("/download", download)
	port := "8080"
	envPort := os.Getenv("PORT")
	if envPort != "" {
		port = envPort
	}
	server := &http.Server{
		// TODO: Dynamic Listen Address
		Addr: "127.0.0.1:" + port,
		//Handler: handler(),
		ReadTimeout:  300 * time.Second,
		WriteTimeout: 300 * time.Second,
	}

	fmt.Println("HTTP Server Running on", port)
	_ = server.ListenAndServe()
}
