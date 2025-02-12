package server

import (
	"encoding/hex"
	"fmt"
	"net/http"

	"errors"

	"github.com/Layr-Labs/eigenda-proxy/store"
	"github.com/Layr-Labs/eigenda-proxy/store/generated_key/eigenda"
	"github.com/Layr-Labs/eigenda-proxy/verify"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/gorilla/mux"
)

// HandleGet handles the GET_EXTRA request for commitments.
func (svr *Server) HandleGetExtra(w http.ResponseWriter, r *http.Request) error {
	extraInfo, err := svr.getExtra(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}

	svr.writeResponse(w, extraInfo)
	return err
}

func (svr *Server) getExtra(r *http.Request) ([]byte, error) {
	rawCommitmentHex, ok := mux.Vars(r)[routingVarNamePayloadHex]
	if !ok {
		return nil, fmt.Errorf("commitment not found in path: %s", r.URL.Path)
	}
	commitment, err := hex.DecodeString(rawCommitmentHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode rawCommitmentHex %s: %w", rawCommitmentHex, err)
	}

	var cert verify.Certificate
	err = rlp.DecodeBytes(commitment, &cert)
	if err != nil {
		return nil, fmt.Errorf("failed to decode DA cert to RLP format: %w", err)
	}

	sm, ok := svr.sm.(*store.Manager)
	if !ok {
		return nil, errors.New("store manager get extra info is unsupported")
	}

	client, ok := sm.GetEigenDAClient().(*eigenda.EigenDAClientProxy)
	if !ok {
		return nil, errors.New("client get extra info is unsupported")
	}

	extraInfo, err := client.GetBlobExtra(r.Context(), cert.BlobVerificationProof.BatchMetadata.BatchHeaderHash, cert.BlobVerificationProof.BlobIndex)
	if err == nil {
		return extraInfo, nil
	}

	return nil, fmt.Errorf("failed to get extra info for commitment %v", rawCommitmentHex)
}
