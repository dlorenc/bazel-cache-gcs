package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"

	"cloud.google.com/go/storage"
	log "github.com/sirupsen/logrus"
)

var (
	bucketName string
	port       int
	bkt        *storage.BucketHandle
	verbosity  string
)

func handleFlags() {
	flag.StringVar(&bucketName, "bucket", "", "GCS bucket to use as a bazel cache.")
	flag.IntVar(&port, "port", 8080, "Port to listen on.")
	flag.StringVar(&verbosity, "verbosity", "warn", "Logging verbosity.")
	flag.Parse()

	if bucketName == "" {
		log.Fatalln("Please provide a value for the --bucket flag.")
	}

	lvl, err := log.ParseLevel(verbosity)
	if err != nil {
		log.Fatalln("Unable to parse verbosity flag.")
	}
	log.SetLevel(lvl)
}

func main() {
	handleFlags()

	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalln("Error creating storage client: ", err)
	}
	bkt = client.Bucket(bucketName)

	http.HandleFunc("/", cacheHandler)
	addr := fmt.Sprintf("localhost:%d", port)
	fmt.Printf("gcs-bazel-cache listening on %s for bucket %s\n", addr, bucketName)
	http.ListenAndServe(addr, nil)
}

func doGet(ctx context.Context, path string, w io.Writer) error {
	obj := bkt.Object(path)
	r, err := obj.NewReader(ctx)
	if err != nil {
		log.Infof("Cache miss on %s.", path)
		return err
	}
	log.Infof("Cache hit on %s.", path)
	n, err = io.Copy(w, r)
	if err != nil {
		return err
	}
	log.Infof("Bytes copied from cache: %d", n)
	return nil
}

func doPut(ctx context.Context, path string, r io.Reader) error {
	log.Infof("Adding %s to cache.", path)
	obj := bkt.Object(path)

	w := obj.NewWriter(ctx)
	n, err := io.Copy(w, r)
	if err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	log.Infof("Bytes written: %d", n)
	return err

}

func cacheHandler(w http.ResponseWriter, req *http.Request) {
	log.Debug("Incoming http request: %v", req)
	switch req.Method {
	case "GET":
		if err := doGet(req.Context(), req.URL.Path, w); err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
	case "PUT":
		if err := doPut(req.Context(), req.URL.Path, req.Body); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Errorf("Put object: ", err)
			return
		}
		req.Body.Close()
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
