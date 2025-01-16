package server

import (
	"errors"
	"fmt"
	"net/http"
	"path"

	"github.com/Layr-Labs/eigenda-proxy/commitments"
	"github.com/Layr-Labs/eigenda-proxy/verify"
	"github.com/ethereum/go-ethereum/rlp"
)

// HandleGet handles the GET_EXTRA request for commitments.
func (svr *Server) HandleGetExtra(w http.ResponseWriter, r *http.Request) (commitments.CommitmentMeta, error) {
	meta, err := ReadCommitmentMeta(r)
	if err != nil {
		err = fmt.Errorf("invalid commitment mode: %w", err)
		svr.WriteBadRequest(w, err)
		return commitments.CommitmentMeta{}, err
	}

	if meta.Mode != commitments.SimpleCommitmentMode && meta.Mode != commitments.OptimismGeneric {
		err = errors.New("only eigenda backend is supported")
		svr.WriteBadRequest(w, err)
		return commitments.CommitmentMeta{}, MetaError{
			Err:  err,
			Meta: meta,
		}
	}

	key := path.Base(r.URL.Path)
	comm, err := commitments.StringToDecodedCommitment(key, meta.Mode)
	if err != nil {
		err = fmt.Errorf("failed to decode commitment from key %v (commitment mode %v): %w", key, meta.Mode, err)
		svr.WriteBadRequest(w, err)
		return commitments.CommitmentMeta{}, MetaError{
			Err:  err,
			Meta: meta,
		}
	}

	var cert verify.Certificate
	err = rlp.DecodeBytes(comm, &cert)
	if err != nil {
		err = fmt.Errorf("failed to decode DA cert to RLP format: %w", err)
		svr.WriteBadRequest(w, err)
		return commitments.CommitmentMeta{}, MetaError{
			Err:  err,
			Meta: meta,
		}
	}

	extraKey := ExtraKey(cert.BlobVerificationProof.BatchMetadata.BatchHeaderHash, cert.BlobVerificationProof.BlobIndex)

	svr.router.GetS3Store()
	for _, c := range svr.router.Caches() {
		extraInfo, err := c.Get(r.Context(), extraKey)
		if err == nil {
			svr.WriteResponse(w, extraInfo)
			return meta, nil
		}
	}

	err = fmt.Errorf("failed to get extra info for commitment %v", key)
	svr.WriteBadRequest(w, err)
	return commitments.CommitmentMeta{}, MetaError{
		Err:  err,
		Meta: meta,
	}
}
