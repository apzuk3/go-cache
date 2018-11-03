package cache

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type s3storage struct {
	instance *s3.S3
	bucket   string
}

func NewS3(sess *session.Session, bucket string) Storage {
	return s3storage{
		instance: s3.New(sess),
		bucket:   bucket,
	}
}

func (s s3storage) Write(key string, v interface{}, d time.Duration) error {
	var buf = bytes.NewBuffer(nil)
	if err := json.NewEncoder(buf).Encode(v); err != nil {
		return ErrNotJSONMarshalable
	}

	// _, err := s.instance.PutObject(&s3.PutObjectInput{
	// 	Key:     aws.String(relativePath(key)),
	// 	Body:    bytes.NewReader(buf.Bytes()),
	// 	Bucket:  aws.String(s.bucket),
	// 	Tagging: aws.String("go-cache"),
	// })
	return nil
}

func (s s3storage) Read(key string) (interface{}, error) {
	out, err := s.instance.GetObject(&s3.GetObjectInput{
		// Bucket: aws.String(s.bucket),
		// Key:    aws.String(relativePath(key)),
	})

	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(out.Body)
}

func (s s3storage) Delete(key string) error {
	// _, err := s.instance.DeleteObject(&s3.DeleteObjectInput{
	// 	Bucket: aws.String(s.bucket),
	// 	Key:    aws.String(relativePath(key)),
	// })
	return nil
}

func (s s3storage) Flush() error {
	return errors.New("S3 does not support flush")
}
