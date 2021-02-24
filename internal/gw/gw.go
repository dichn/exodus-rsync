package gw

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/release-engineering/exodus-rsync/internal/conf"
	"github.com/release-engineering/exodus-rsync/internal/log"
	"github.com/release-engineering/exodus-rsync/internal/walk"
)

// Client provides a client for the exodus-gw service.
type Client struct {
	env        conf.Environment
	httpClient *http.Client
	s3         *s3.S3
	uploader   *s3manager.Uploader
}

func (c *Client) doJSONRequest(ctx context.Context, method string, url string, body io.Reader, target interface{}) error {
	fullURL := c.env.Config.GwURL + url
	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)

	if err != nil {
		return fmt.Errorf("preparing request to %s: %w", fullURL, err)
	}

	req.Header["Accept"] = []string{"application/json"}
	req.Header["Content-Type"] = []string{"application/json"}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", req.Method, req.URL, err)
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("%s %s: %s %v", req.Method, req.URL, resp.Status, resp.Body)
	}

	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(target)
	if err != nil {
		return fmt.Errorf("%s %s: %w", req.Method, req.URL, err)
	}

	return nil
}

func (c *Client) haveBlob(ctx context.Context, item walk.SyncItem) (bool, error) {
	logger := log.FromContext(ctx)

	_, err := c.s3.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(c.env.GwEnv),
		Key:    aws.String(item.Key),
	})

	if err == nil {
		logger.F("key", item.Key).Debug("blob is present")
		return true, nil
	}

	awsErr, isAwsErr := err.(awserr.Error)

	if isAwsErr && awsErr.Code() == "NotFound" {
		// Fine, object doesn't exist yet
		logger.F("key", item.Key).Debug("blob is not present")
		return false, nil
	}

	// Anything else is unusual
	logger.F("error", err, "key", item.Key).Warn("S3 HEAD unexpected error")

	return false, err
}

func (c *Client) uploadBlob(ctx context.Context, item walk.SyncItem) error {
	logger := log.FromContext(ctx)

	var err error

	defer logger.F("src", item.SrcPath, "key", item.Key).Trace("Uploading").Stop(&err)

	file, err := os.Open(item.SrcPath)
	if err != nil {
		return fmt.Errorf("upload (open) %s: %w", item.SrcPath, err)
	}
	defer file.Close()

	res, err := c.uploader.UploadWithContext(ctx, &s3manager.UploadInput{
		Bucket: &c.env.GwEnv,
		Key:    &item.Key,
		Body:   file,
	})

	if err != nil {
		return fmt.Errorf("upload (s3) %s: %w", item.SrcPath, err)
	}

	logger.F("location", res.Location).Debug("uploaded blob")

	return nil
}

// EnsureUploaded will process every given item for sync and ensure that the content
// is present in the target exodus-gw environment.
//
// For each item, onUploaded is invoked if the item was uploaded during the call,
// while onPresent is invoked if the item was already present prior to the call.
//
// In either case, returning from the callback with an error will cause EnsureUploaded
// to stop and return the same error.
func (c *Client) EnsureUploaded(
	ctx context.Context,
	items []walk.SyncItem,
	onUploaded func(walk.SyncItem) error,
	onPresent func(walk.SyncItem) error,
) error {
	// TODO: concurrency
	for _, item := range items {
		// Check if it's present
		have, err := c.haveBlob(ctx, item)
		if err != nil {
			return fmt.Errorf("checking for presence of %s: %w", item.Key, err)
		}

		if have {
			if err = onPresent(item); err != nil {
				return err
			}
			continue
		}

		if err = c.uploadBlob(ctx, item); err != nil {
			return err
		}
		if err = onUploaded(item); err != nil {
			return err
		}
	}

	return nil
}

// NewClient creates and returns a new exodus-gw client pointing at the
// given Environment (from the configuration file).
func NewClient(env conf.Environment) (*Client, error) {
	// TODO: should support loading these from environment too.
	cert, err := tls.LoadX509KeyPair(env.Config.GwCert, env.Config.GwKey)
	if err != nil {
		return nil, fmt.Errorf("can't load cert/key: %w", err)
	}

	out := &Client{env: env}

	transport := http.Transport{
		TLSClientConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
		},
	}
	out.httpClient = &http.Client{Transport: &transport}

	sess, err := session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigDisable,
		Config: aws.Config{
			Endpoint:         aws.String(env.Config.GwURL + "/upload"),
			S3ForcePathStyle: aws.Bool(true),

			Region:      aws.String("us-east-1"),
			Credentials: credentials.AnonymousCredentials,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create AWS session: %w", err)
	}

	out.s3 = s3.New(sess)
	out.uploader = s3manager.NewUploaderWithClient(out.s3)

	return out, nil
}

func (c *Client) url(path string) string {
	return c.env.Config.GwURL + "/" + path
}
