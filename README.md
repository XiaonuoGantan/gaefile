# gaefile
`gaefile` is an open source golang project that implements simple file uploads and downlloads using Google Cloud Storage.

## How to use this project?

First, clone it into your $GOPATH.

```
$ cd $GOPATH/src
$ git clone git@github.com:XiaonuoGantan/gaefile.git
```

Second, create two files under `gaeserver/`:

```
$ cd gaefile
$ touch gaeserver/ServiceAccountAccessID.txt
$ touch gaeserver/ServiceAccountPrivateKey.pem
```

The content of `ServiceAccountAccessID.txt` is a one line text which is your Google Service Account's email address that looks like `...@developer.gserviceaccount.com`.

The content of `ServiceAccountPrivateKey.pem` is the Google service account private key. It is obtainable from the Google Developer Console. Details about how to obtain it is available at `https://godoc.org/google.golang.org/cloud/storage#SignedURLOptions`.

Third, change the option `GOOGLE_CLOUD_STAORAGE_BUCKET` in `gaeserver/app.yaml` to your own cloud storage bucket.

Fourth, run the server `$ goapp serve -port 8090 -host 0.0.0.0 gaeserver`.

## How to upload and download files?

Upload files:

```
$ cd gaeserver/sample_images
$ curl -X PUT -H 'X-Gaefile-Filename: image.jpeg' -H 'Content-Type: image/jpeg' --data-binary "@sample_image_1.jpg"  http://localhost:8090
{"key":"zHcmkYxgCZVw7A==_image.jpeg"}
```

Download files:

```
$ cd gaeserver/sample_images
$ curl -X GET -L http://localhost:8090/zHcmkYxgCZVw7A==_image.jpeg > downloaded.jpg
$ diff sample_image_1.jpg downloaded.jpg
```

If everything works, then the `diff` command above should output nothing since the downloaded image and the sample image are identical.
