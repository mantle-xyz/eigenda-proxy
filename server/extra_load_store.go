package server

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"

	disperser_rpc "github.com/Layr-Labs/eigenda/api/grpc/disperser"

	"github.com/Layr-Labs/eigenda-proxy/store"
	"github.com/Layr-Labs/eigenda-proxy/store/precomputed_key/redis"
	"github.com/Layr-Labs/eigenda-proxy/verify"
	"github.com/Layr-Labs/eigenda/api/clients"
	"github.com/Layr-Labs/eigenda/disperser"
	"github.com/ethereum/go-ethereum/log"
)

func ExtraKey(batchHeaderHash []byte, blobIndex uint32) []byte {
	key := fmt.Sprintf("extra-info_%x_%d", batchHeaderHash, blobIndex)
	return []byte(key)
}

func ExtraInfo(key []byte) ([]byte, error) {
	extra := make(map[string]interface{})
	extra["request_id"] = hex.EncodeToString(key)
	return json.Marshal(extra)
}

// EigenDAClientProxy is used to proxy DisperserClient in eigenda api sdk to intercept GetBlobStatus requests
// Because request_id only visible in eigenda api sdk
type EigenDAClientProxy struct {
	client clients.DisperserClient   // The actual DisperserClient instance
	s3     store.PrecomputedKeyStore // s3 used to store request_id
	redis  *redis.Store              // redis used to store request_id
	log    log.Logger
}

func NewEigenDAClientProxy(client clients.DisperserClient, log log.Logger, s3 store.PrecomputedKeyStore, redis *redis.Store) clients.DisperserClient {
	return &EigenDAClientProxy{
		client: client,
		s3:     s3,
		redis:  redis,
		log:    log,
	}
}

func (c *EigenDAClientProxy) DisperseBlob(ctx context.Context, data []byte, customQuorums []uint8) (*disperser.BlobStatus, []byte, error) {
	return c.client.DisperseBlob(ctx, data, customQuorums)
}

func (c *EigenDAClientProxy) DisperseBlobAuthenticated(ctx context.Context, data []byte, customQuorums []uint8) (*disperser.BlobStatus, []byte, error) {
	return c.client.DisperseBlobAuthenticated(ctx, data, customQuorums)
}

// Intercept GetBlobStatus requests and store request_id
func (c *EigenDAClientProxy) GetBlobStatus(ctx context.Context, key []byte) (*disperser_rpc.BlobStatusReply, error) {
	reply, err := c.client.GetBlobStatus(ctx, key)
	func() {
		defer func() {
			if r := recover(); r != nil {
				c.log.Error("panic in cache extra info", "error", r)
			}
		}()
		if reply.Status == disperser_rpc.BlobStatus_CONFIRMED || reply.Status == disperser_rpc.BlobStatus_FINALIZED {
			cert := (*verify.Certificate)(reply.Info)
			bytes, err := ExtraInfo(key)
			extraKey := ExtraKey(cert.BlobVerificationProof.BatchMetadata.BatchHeaderHash, cert.BlobVerificationProof.BlobIndex)
			if err == nil {
				if c.redis != nil {
					extraData, err := c.redis.Get(ctx, extraKey)
					if extraData == nil {
						err = c.redis.Put(ctx, extraKey, bytes)
						c.log.Info("put extra info found to redis", "key", key, "error", err)
					}
				}
				if c.s3 != nil {
					extraData, err := c.s3.Get(ctx, extraKey)
					if extraData == nil {
						err = c.s3.Put(ctx, extraKey, bytes)
						c.log.Info("put extra info found to s3", "key", key, "error", err)
					}
				}
			} else {
				c.log.Error("failed to encode DA cert to RLP format", "error", err)
			}
		}
	}()

	return reply, err
}

func (c *EigenDAClientProxy) RetrieveBlob(ctx context.Context, batchHeaderHash []byte, blobIndex uint32) ([]byte, error) {
	return c.client.RetrieveBlob(ctx, batchHeaderHash, blobIndex)
}
