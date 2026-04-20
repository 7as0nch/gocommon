/*
 * @Author: chengjiang
 * @Date: 2026-04-20 11:56:01
 * @Description: MinIO 对象存储客户端封装。
 *
 * 目前仅接入 MinIO；后续扩展 OSS/COS/S3 时，建议在本包内抽一个 ObjectStore 接口，
 * 由各实现按需满足。这里先保留"面向 MinIO 的直接封装 + 一组常用高阶方法"，
 * 减少业务侧学习成本。
 */
package oss

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Config MinIO 连接/默认桶配置。
type Config struct {
	// Endpoint 形如 "play.min.io" 或 "oss.example.com:9000"，不含协议。
	Endpoint string
	// AccessKeyID / SecretAccessKey 访问凭证。
	AccessKeyID     string
	SecretAccessKey string
	// UseSSL 是否走 https。
	UseSSL bool
	// Region 可选；国外/自建集群经常需要指定。
	Region string
	// DefaultBucket 默认桶名，未指定时由调用方传入。
	DefaultBucket string
}

// Client MinIO 封装；零值不可用，必须通过 NewClient 构造。
type Client struct {
	raw    *minio.Client
	bucket string
}

// NewClient 创建 MinIO 客户端。
// 不做强制 Ping，避免测试/离线场景构造失败；
// 第一次真正调用接口时若凭证/endpoint 有误自然会报错。
func NewClient(cfg Config) (*Client, error) {
	if cfg.Endpoint == "" {
		return nil, errors.New("oss: endpoint is required")
	}
	if cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" {
		return nil, errors.New("oss: access key or secret is empty")
	}
	opts := &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
	}
	if cfg.Region != "" {
		opts.Region = cfg.Region
	}
	cli, err := minio.New(cfg.Endpoint, opts)
	if err != nil {
		return nil, fmt.Errorf("oss: new minio client failed: %w", err)
	}
	return &Client{raw: cli, bucket: cfg.DefaultBucket}, nil
}

// Raw 返回底层 *minio.Client，方便业务调用未封装 API。
func (c *Client) Raw() *minio.Client {
	if c == nil {
		return nil
	}
	return c.raw
}

// Bucket 返回默认 bucket，若 override 非空则以 override 为准。
func (c *Client) Bucket(override string) string {
	if override != "" {
		return override
	}
	return c.bucket
}

// EnsureBucket 若 bucket 不存在则创建；已存在则无操作。
func (c *Client) EnsureBucket(ctx context.Context, bucket, region string) error {
	bucket = c.Bucket(bucket)
	if bucket == "" {
		return errors.New("oss: bucket is empty")
	}
	exists, err := c.raw.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("oss: bucket exists check failed: %w", err)
	}
	if exists {
		return nil
	}
	return c.raw.MakeBucket(ctx, bucket, minio.MakeBucketOptions{Region: region})
}

// PutObjectOptions 上传可调参数，全部为可选项。
type PutObjectOptions struct {
	ContentType string            // 不传走自动探测或 "application/octet-stream"
	UserMeta    map[string]string // 自定义元数据，会写成 x-amz-meta-*
	CacheControl string           // 可选的 Cache-Control header
}

// PutObjectStream 把 reader 内容上传为 object。
// size < 0 时走 multi-part（允许未知大小）。
func (c *Client) PutObjectStream(ctx context.Context, bucket, object string, reader io.Reader, size int64, opt PutObjectOptions) (minio.UploadInfo, error) {
	bucket = c.Bucket(bucket)
	if bucket == "" || object == "" {
		return minio.UploadInfo{}, errors.New("oss: bucket or object is empty")
	}
	putOpts := minio.PutObjectOptions{
		ContentType:  opt.ContentType,
		UserMetadata: opt.UserMeta,
		CacheControl: opt.CacheControl,
	}
	if putOpts.ContentType == "" {
		putOpts.ContentType = "application/octet-stream"
	}
	return c.raw.PutObject(ctx, bucket, object, reader, size, putOpts)
}

// PutObjectBytes 直接上传一段 bytes，size 自动推断。
func (c *Client) PutObjectBytes(ctx context.Context, bucket, object string, data []byte, opt PutObjectOptions) (minio.UploadInfo, error) {
	return c.PutObjectStream(ctx, bucket, object, bytes.NewReader(data), int64(len(data)), opt)
}

// GetObject 拿到对象 ReadCloser；调用方负责 Close。
func (c *Client) GetObject(ctx context.Context, bucket, object string) (*minio.Object, error) {
	bucket = c.Bucket(bucket)
	if bucket == "" || object == "" {
		return nil, errors.New("oss: bucket or object is empty")
	}
	return c.raw.GetObject(ctx, bucket, object, minio.GetObjectOptions{})
}

// StatObject 获取对象元信息。
func (c *Client) StatObject(ctx context.Context, bucket, object string) (minio.ObjectInfo, error) {
	bucket = c.Bucket(bucket)
	if bucket == "" || object == "" {
		return minio.ObjectInfo{}, errors.New("oss: bucket or object is empty")
	}
	return c.raw.StatObject(ctx, bucket, object, minio.StatObjectOptions{})
}

// RemoveObject 删除对象；object 不存在时返回 nil（MinIO 侧行为）。
func (c *Client) RemoveObject(ctx context.Context, bucket, object string) error {
	bucket = c.Bucket(bucket)
	if bucket == "" || object == "" {
		return errors.New("oss: bucket or object is empty")
	}
	return c.raw.RemoveObject(ctx, bucket, object, minio.RemoveObjectOptions{})
}

// PresignedGetURL 生成一条限时有效的直链下载地址；expires <= 0 时兜底 1h。
// reqParams 支持覆盖响应 header（如 response-content-disposition）。
func (c *Client) PresignedGetURL(ctx context.Context, bucket, object string, expires time.Duration, reqParams url.Values) (string, error) {
	bucket = c.Bucket(bucket)
	if bucket == "" || object == "" {
		return "", errors.New("oss: bucket or object is empty")
	}
	if expires <= 0 {
		expires = time.Hour
	}
	u, err := c.raw.PresignedGetObject(ctx, bucket, object, expires, reqParams)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

// PresignedPutURL 生成一条限时有效的上传直链。
func (c *Client) PresignedPutURL(ctx context.Context, bucket, object string, expires time.Duration) (string, error) {
	bucket = c.Bucket(bucket)
	if bucket == "" || object == "" {
		return "", errors.New("oss: bucket or object is empty")
	}
	if expires <= 0 {
		expires = time.Hour
	}
	u, err := c.raw.PresignedPutObject(ctx, bucket, object, expires)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

// ListOptions 列目录参数。
type ListOptions struct {
	Prefix    string
	Recursive bool
	MaxKeys   int
}

// ListObjects 返回 prefix 下的对象信息切片；内部一次性收集，不适合超大桶。
func (c *Client) ListObjects(ctx context.Context, bucket string, opt ListOptions) ([]minio.ObjectInfo, error) {
	bucket = c.Bucket(bucket)
	if bucket == "" {
		return nil, errors.New("oss: bucket is empty")
	}
	listOpts := minio.ListObjectsOptions{
		Prefix:    opt.Prefix,
		Recursive: opt.Recursive,
		MaxKeys:   opt.MaxKeys,
	}
	var result []minio.ObjectInfo
	for info := range c.raw.ListObjects(ctx, bucket, listOpts) {
		if info.Err != nil {
			return result, info.Err
		}
		result = append(result, info)
	}
	return result, nil
}
