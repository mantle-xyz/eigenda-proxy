package store

import (
	"github.com/Layr-Labs/eigenda-proxy/store/generated_key/eigenda"
	"github.com/Layr-Labs/eigenda/api/clients"
)

func (m *Manager) GetEigenDAClient() clients.DisperserClient {
	if m.eigenda == nil {
		return nil
	}

	store, ok := m.eigenda.(*eigenda.Store)
	if !ok {
		return nil
	}

	return store.GetClient().Client
}
