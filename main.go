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
	bkt        *storage.BucketHandle
)

func handleFlags() {
	flag.StringVar(&bucketName, "bucket", "", "GCS bucket to use as a bazel cache.")
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
	http.ListenAndServe(":8080", nil)
}

func doGet(ctx context.Context, path string, w io.Writer) error {
	obj := bkt.Object(path)
	r, err := obj.NewReader(ctx)
	if err != nil {
		fmt.Println("Cache miss!")
		return err
	}
	fmt.Println("Cache hit!")
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
