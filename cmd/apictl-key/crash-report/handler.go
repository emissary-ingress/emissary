package crash_report

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/datawire/apro/cmd/apictl-key/datasource"
)

type client struct {
	l          *logrus.Logger
	s3Bucket   string
	s3Region   string
	datasource datasource.Datasource
}

type crashReportCreationResponse struct {
	ReportId  string
	Method    string
	UploadURL string
}

func NewClient(l *logrus.Logger, s3Bucket string, s3Region string, datasource datasource.Datasource) http.Handler {
	return &client{
		l:          l,
		s3Bucket:   s3Bucket,
		s3Region:   s3Region,
		datasource: datasource,
	}
}

func (c *client) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if c.s3Bucket == "" {
		http.NotFound(w, r)
		return
	}

	// TODO: Read request information metadata, extracing product, install_ids, etc.

	crashReportId := uuid.New().String()
	signedUploadURL, err := c.generateSignedURL(crashReportId)

	if err != nil {
		http.Error(w, "crash-report creation failure", http.StatusInternalServerError)
		return
	}

	// TODO: Save crashReportId and metadata in database
	c.l.Infof("generated crash-report %s", crashReportId)

	response := crashReportCreationResponse{
		ReportId:  crashReportId,
		Method:    "PUT",
		UploadURL: signedUploadURL,
	}

	// TODO: Remove this sample usage. Leaving it here to serve as an example for edgectl
	c.exampleUpload(response)

	// If all is good, return 201 and the generated upload URL.
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(&response)
}

func (c *client) generateSignedURL(uniqueBucketObjectKey string) (string, error) {
	// Start an S3 session
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(c.s3Region),
	})
	if err != nil {
		c.l.WithError(err).Error("error creating aws s3 session")
		return "", err
	}
	s3Session := s3.New(sess)

	req, _ := s3Session.PutObjectRequest(&s3.PutObjectInput{
		Bucket: aws.String(c.s3Bucket),
		Key:    aws.String(uniqueBucketObjectKey),
	})
	signedUrl, err := req.Presign(1 * time.Hour)

	if err != nil {
		c.l.WithError(err).Error("error creating aws s3 signed url")
		return "", err
	}

	return signedUrl, nil
}

func (c *client) exampleUpload(response crashReportCreationResponse) {
	uploadContent := "Hello world!"

	client := &http.Client{}
	req, err := http.NewRequest(response.Method, response.UploadURL, strings.NewReader(uploadContent))
	if err != nil {
		c.l.Error(err)
		return
	}

	res, err := client.Do(req)
	if err != nil {
		c.l.Error(err)
		return
	}
	defer res.Body.Close()
	bodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		c.l.Error(err)
		return
	}
	bodyString := string(bodyBytes)
	c.l.Infof("%s: %s", res.Status, bodyString)
}
