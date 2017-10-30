package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"

	"cloud.google.com/go/storage"
)

var (
	bucketName string
	port       int
	bkt        *storage.BucketHandle
)

func handleFlags() {
	flag.StringVar(&bucketName, "bucket", "", "GCS bucket to use as a bazel cache.")
	flag.IntVar(&port, "port", 8080, "Port to listen on.")
	flag.Parse()

	if bucketName == "" {
		log.Fatalln("Please provide a value for the --bucket flag.")
	}
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
	fmt.Printf("gcs-bazel-cache listening on %s\n", addr)
	http.ListenAndServe(addr, nil)
}

func doGet(ctx context.Context, path string, w io.Writer) error {
	obj := bkt.Object(path)
	r, err := obj.NewReader(ctx)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, r)
	return err
}

func doPut(ctx context.Context, path string, r io.Reader) error {
	obj := bkt.Object(path)
	w := obj.NewWriter(ctx)
	_, err := io.Copy(w, r)
	return err

}

func cacheHandler(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		if err := doGet(req.Context(), req.URL.Path, w); err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
	case "PUT":
		if err := doPut(req.Context(), req.URL.Path, req.Body); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		req.Body.Close()
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
