package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	lrucache "github.com/hashicorp/golang-lru"
)

var (
	ErrTileNotFound = fmt.Errorf("not found")
)

type TileStore interface {
	GetTile(ctx context.Context, z uint32, x uint32, y uint32) ([]byte, error)
}

type S3TileStore struct {
	s3Client      *s3.Client
	bucketName    string
	tileNameTempl string
}

type CacheTileStore struct {
	tileCache     *lrucache.Cache
	tileNameTempl string
}

type ElevationTileStore struct {
	tileCache     *lrucache.Cache
	tileNameTempl string
}

func NewS3TileStore(s3Client *s3.Client, bucketName string, tileNameTempl string) (*S3TileStore, error) {
	ts := S3TileStore{
		s3Client:      s3Client,
		bucketName:    bucketName,
		tileNameTempl: tileNameTempl,
	}
	return &ts, nil
}

func (ts *S3TileStore) GetTile(ctx context.Context, z uint32, x uint32, y uint32) ([]byte, error) {

	oName := fmt.Sprintf(ts.tileNameTempl, z, x, y)
	goi := &s3.GetObjectInput{
		Bucket: aws.String(ts.bucketName),
		Key:    aws.String(oName),
	}

	goo, err := ts.s3Client.GetObject(ctx, goi)
	if err != nil {
		return nil, err
	}

	data := new(bytes.Buffer)
	data.ReadFrom(goo.Body)

	goo.Body.Close()

	return data.Bytes(), nil
}

func NewCacheTileStore(tileNameTempl string, cacheSize int) (*CacheTileStore, error) {
	tileCache, err := lrucache.New(cacheSize)
	if err != nil {
		return nil, err
	}
	ts := CacheTileStore{
		tileCache:     tileCache,
		tileNameTempl: tileNameTempl,
	}
	return &ts, nil
}

func (ts *CacheTileStore) GetTile(ctx context.Context, z uint32, x uint32, y uint32) ([]byte, error) {
	oName := fmt.Sprintf(ts.tileNameTempl, z, x, y)
	if !ts.tileCache.Contains(oName) {
		return nil, ErrTileNotFound
	}

	obj, ok := ts.tileCache.Get(oName)
	if !ok {
		return nil, errors.New("cache error")
	}

	cacheData, ok := obj.([]byte)
	if !ok {
		return nil, errors.New("cache error")
	}

	return cacheData, nil
}

func (ts *CacheTileStore) Add(z uint32, x uint32, y uint32, data []byte) {
	key := fmt.Sprintf(ts.tileNameTempl, z, x, y)
	ts.tileCache.Add(key, data)
}

func NewElevationTileStore(tileNameTempl string, cacheSize int) (*ElevationTileStore, error) {
	tileCache, err := lrucache.New(cacheSize)
	if err != nil {
		return nil, err
	}
	ts := ElevationTileStore{
		tileCache:     tileCache,
		tileNameTempl: tileNameTempl,
	}
	return &ts, nil
}

func (ts *ElevationTileStore) GetTile(ctx context.Context, z uint32, x uint32, y uint32) ([]float64, error) {
	oName := fmt.Sprintf(ts.tileNameTempl, z, x, y)
	if !ts.tileCache.Contains(oName) {
		return nil, ErrTileNotFound
	}

	obj, ok := ts.tileCache.Get(oName)
	if !ok {
		return nil, errors.New("cache error")
	}

	cacheData, ok := obj.([]float64)
	if !ok {
		return nil, errors.New("cache error")
	}

	return cacheData, nil
}

func (ts *ElevationTileStore) Add(z uint32, x uint32, y uint32, data []float64) {
	key := fmt.Sprintf(ts.tileNameTempl, z, x, y)
	ts.tileCache.Add(key, data)
}
