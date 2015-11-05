package gaeserver

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	appengine "google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
	"google.golang.org/cloud"
	"google.golang.org/cloud/storage"

	"gaefile/gaeserver/util"
	"gaefile/ioext"
)

const DEFAULT_CONTENT_TYPE = "image/jpeg"

type cloudBucketContext struct {
	ctx      context.Context
	bucket   string
	filename string
	req      *http.Request
}

type createFileResponse struct {
	Key string `json:"key"`
}

func getCloudContext(gae context.Context) context.Context {
	hc := &http.Client{
		Transport: &oauth2.Transport{
			Source: google.AppEngineTokenSource(gae, storage.ScopeFullControl),
			Base:   &urlfetch.Transport{Context: gae},
		},
	}
	return cloud.NewContext(appengine.AppID(gae), hc)
}

// cc.ctx and gae are two different context.Context objects
func createFile(cc cloudBucketContext, gae context.Context) error {
	var wc = storage.NewWriter(cc.ctx, cc.bucket, cc.filename)
	var contentType = DEFAULT_CONTENT_TYPE
	if ctype := cc.req.Header.Get("Content-Type"); len(ctype) > 0 {
		contentType = ctype
	}
	wc.ContentType = contentType
	// Make sure that signed URLs are downloadable by everyone, thus sharable.
	wc.ACL = []storage.ACLRule{{storage.AllUsers, storage.RoleReader}}
	var timeout = time.Now().UTC().Add(28 * time.Second)
	if bytesCopied, err := ioext.Copy(wc, cc.req.Body, timeout); err != nil {
		return err
	} else {
		log.Infof(gae, "bytesCopied = %d", bytesCopied)
	}
	if err := wc.Close(); err != nil {
		return err
	}
	return nil
}

func uploadFile(
	wri http.ResponseWriter,
	req *http.Request,
) {
	var gae = appengine.NewContext(req)
	var bucket = util.GetBucketName()
	log.Infof(gae, "uploading file to bucket %s", bucket)
	cloudCtx := getCloudContext(gae)
	var numOfBytes = 10
	var randBytes = make([]byte, numOfBytes)
	if _, err := rand.Read(randBytes); err != nil {
		log.Errorf(gae, "crypto/rand.Read() error: %s", err)
		wri.WriteHeader(http.StatusInternalServerError)
		return
	}
	var filename = req.Header.Get("X-Gaefile-Filename")
	log.Infof(gae, "before filename = %s", string(filename))
	var randStr = base64.URLEncoding.EncodeToString(randBytes)
	filename = randStr + "_" + filename
	log.Infof(gae, "after filename = %s", string(filename))
	cc := cloudBucketContext{
		ctx:      cloudCtx,
		bucket:   bucket,
		filename: filename,
		req:      req,
	}
	if err := createFile(cc, gae); err != nil {
		wri.WriteHeader(http.StatusInternalServerError)
		log.Errorf(gae, "createFile error: %s", err)
		return
	} else {
		var resp = createFileResponse{
			Key: filename,
		}
		log.Infof(gae, "createFileResponse = %+v", resp)
		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(&resp); err != nil {
			wri.WriteHeader(http.StatusInternalServerError)
			log.Errorf(gae, "createFileResponse error: %s", err)
			return
		}
		if _, err := io.Copy(wri, &buf); err != nil {
			wri.WriteHeader(http.StatusInternalServerError)
			log.Errorf(gae, "createFileResponse io.Copy error: %s", err)
			return
		}
	}
}

func downloadFile(
	wri http.ResponseWriter,
	req *http.Request,
) {
	var gae = appengine.NewContext(req)
	var bucket = util.GetBucketName()
	var vars = mux.Vars(req)
	log.Infof(gae, "download file %s from bucket %s", vars["key"], bucket)
	var cloudCtx = getCloudContext(gae)
	rc, err := storage.NewReader(cloudCtx, bucket, vars["key"])
	if err != nil {
		log.Infof(gae, "unable to open file %q from bucket %q", vars["key"], bucket)
		wri.WriteHeader(http.StatusNotFound)
		return
	}
	defer rc.Close()
	// Return an signed URL from which the client can download a file
	accessIdFile := os.Getenv("GOOGLE_ACCESS_ID")
	if accessIdFile == "" {
		log.Errorf(gae, "Missing GOOGLE_ACCESS_ID from app.yaml")
		wri.WriteHeader(http.StatusInternalServerError)
		return
	}
	accessId, err := ioutil.ReadFile(accessIdFile)
	if err != nil {
		log.Errorf(gae, "Cannot read service account access id from %s", accessIdFile)
		wri.WriteHeader(http.StatusInternalServerError)
	}
	privateKeyFile := os.Getenv("GOOGLE_PRIVATE_KEY")
	if privateKeyFile == "" {
		log.Errorf(gae, "Missing GOOGLE_PRIVATE_KEY from app.yaml")
		wri.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Infof(gae, "Private key file %s", privateKeyFile)
	privateKey, err := ioutil.ReadFile(privateKeyFile)
	if err != nil {
		log.Errorf(gae, "Cannot read service account private key from %s", privateKeyFile)
		wri.WriteHeader(http.StatusInternalServerError)
		return
	}
	var opts = storage.SignedURLOptions{
		GoogleAccessID: string(accessId),
		PrivateKey:     privateKey,
		Method:         "GET",
		Expires:        time.Now().UTC().Add(300 * time.Second),
	}
	signedURL, err := storage.SignedURL(bucket, vars["key"], &opts)
	if err != nil {
		log.Errorf(gae, "Unable to generate a signedURL: %s", err)
		wri.WriteHeader(http.StatusInternalServerError)
	}
	http.Redirect(wri, req, signedURL, http.StatusTemporaryRedirect)
}

func UploadInit(r *mux.Router) {
	r.HandleFunc("/", uploadFile).Methods("PUT")
	r.HandleFunc("/{key}", downloadFile).Methods("GET")
}
