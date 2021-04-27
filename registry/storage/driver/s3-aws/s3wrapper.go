package s3

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"golang.org/x/time/rate"
)

// s3wrapper implements a subset of s3iface.S3API allowing us to rate limit,
// retry, add trace logging, or otherwise improve s3 calls made by the driver.
type s3wrapper struct {
	s3 s3iface.S3API
	*rate.Limiter
}

func newS3Wrapper(s3 s3iface.S3API, params DriverParameters) *s3wrapper {
	return &s3wrapper{s3, rate.NewLimiter(rate.Limit(params.MaxRequestsPerSecond), defaultBurst)}
}

func (w *s3wrapper) PutObjectWithContext(ctx aws.Context, input *s3.PutObjectInput, opts ...request.Option) (*s3.PutObjectOutput, error) {
	if err := w.Wait(ctx); err != nil {
		return nil, err
	}

	return w.s3.PutObjectWithContext(ctx, input, opts...)
}

func (w *s3wrapper) GetObjectWithContext(ctx aws.Context, input *s3.GetObjectInput, opts ...request.Option) (*s3.GetObjectOutput, error) {
	if err := w.Wait(ctx); err != nil {
		return nil, err
	}

	return w.s3.GetObjectWithContext(ctx, input, opts...)
}

func (w *s3wrapper) CreateMultipartUploadWithContext(ctx aws.Context, input *s3.CreateMultipartUploadInput, opts ...request.Option) (*s3.CreateMultipartUploadOutput, error) {
	if err := w.Wait(ctx); err != nil {
		return nil, err
	}

	return w.s3.CreateMultipartUploadWithContext(ctx, input, opts...)
}

func (w *s3wrapper) ListMultipartUploadsWithContext(ctx aws.Context, input *s3.ListMultipartUploadsInput, opts ...request.Option) (*s3.ListMultipartUploadsOutput, error) {
	if err := w.Wait(ctx); err != nil {
		return nil, err
	}

	return w.s3.ListMultipartUploadsWithContext(ctx, input, opts...)
}

func (w *s3wrapper) ListPartsWithContext(ctx aws.Context, input *s3.ListPartsInput, opts ...request.Option) (*s3.ListPartsOutput, error) {
	if err := w.Wait(ctx); err != nil {
		return nil, err
	}

	return w.s3.ListPartsWithContext(ctx, input, opts...)
}

func (w *s3wrapper) ListObjectsV2WithContext(ctx aws.Context, input *s3.ListObjectsV2Input, opts ...request.Option) (*s3.ListObjectsV2Output, error) {
	if err := w.Wait(ctx); err != nil {
		return nil, err
	}

	return w.s3.ListObjectsV2WithContext(ctx, input, opts...)
}

func (w *s3wrapper) CopyObjectWithContext(ctx aws.Context, input *s3.CopyObjectInput, opts ...request.Option) (*s3.CopyObjectOutput, error) {
	if err := w.Wait(ctx); err != nil {
		return nil, err
	}

	return w.s3.CopyObjectWithContext(ctx, input, opts...)
}

func (w *s3wrapper) UploadPartCopyWithContext(ctx aws.Context, input *s3.UploadPartCopyInput, opts ...request.Option) (*s3.UploadPartCopyOutput, error) {
	if err := w.Wait(ctx); err != nil {
		return nil, err
	}

	return w.s3.UploadPartCopyWithContext(ctx, input, opts...)
}

func (w *s3wrapper) CompleteMultipartUploadWithContext(ctx aws.Context, input *s3.CompleteMultipartUploadInput, opts ...request.Option) (*s3.CompleteMultipartUploadOutput, error) {
	if err := w.Wait(ctx); err != nil {
		return nil, err
	}

	return w.s3.CompleteMultipartUploadWithContext(ctx, input, opts...)
}

func (w *s3wrapper) DeleteObjectsWithContext(ctx aws.Context, input *s3.DeleteObjectsInput, opts ...request.Option) (*s3.DeleteObjectsOutput, error) {
	if err := w.Wait(ctx); err != nil {
		return nil, err
	}

	return w.s3.DeleteObjectsWithContext(ctx, input, opts...)
}

func (w *s3wrapper) GetObjectRequest(input *s3.GetObjectInput) (*request.Request, *s3.GetObjectOutput) {
	// This does not make network calls, no need to rate limit.
	return w.s3.GetObjectRequest(input)
}

func (w *s3wrapper) HeadObjectRequest(input *s3.HeadObjectInput) (*request.Request, *s3.HeadObjectOutput) {
	// This does not make network calls, no need to rate limit.
	return w.s3.HeadObjectRequest(input)
}

func (w *s3wrapper) ListObjectsV2PagesWithContext(ctx aws.Context, input *s3.ListObjectsV2Input, f func(*s3.ListObjectsV2Output, bool) bool, opts ...request.Option) error {
	if err := w.Wait(ctx); err != nil {
		return err
	}

	return w.s3.ListObjectsV2PagesWithContext(ctx, input, f, opts...)
}

func (w *s3wrapper) AbortMultipartUploadWithContext(ctx aws.Context, input *s3.AbortMultipartUploadInput, opts ...request.Option) (*s3.AbortMultipartUploadOutput, error) {
	if err := w.Wait(ctx); err != nil {
		return nil, err
	}

	return w.s3.AbortMultipartUploadWithContext(ctx, input, opts...)
}

func (w *s3wrapper) UploadPartWithContext(ctx aws.Context, input *s3.UploadPartInput, opts ...request.Option) (*s3.UploadPartOutput, error) {
	if err := w.Wait(ctx); err != nil {
		return nil, err
	}

	return w.s3.UploadPartWithContext(ctx, input, opts...)
}
