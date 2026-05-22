package kv

import (
	"context"
	"sync"
	"testing"

	statenative "github.com/OffchainLabs/prysm/v7/beacon-chain/state/state-native"
	"github.com/OffchainLabs/prysm/v7/config/features"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"
)

func TestRace_SaveState_ConcurrentRead(t *testing.T) {
	features.Get().EnableHistoricalSpaceRepresentation = true
	defer func() {
		features.Get().EnableHistoricalSpaceRepresentation = false
	}()

	ctx := context.Background()
	tmpDir := t.TempDir()

	s, err := NewKVStore(ctx, tmpDir)
	require.NoError(t, err)

	err = s.db.Update(func(tx *bolt.Tx) error {
		mb := tx.Bucket(migrationsBucket)
		return mb.Put(migrationStateValidatorsKey, migrationCompleted)
	})
	require.NoError(t, err)

	// Create a state with some validators.
	vals := []*ethpb.Validator{
		{EffectiveBalance: 32000000000},
		{EffectiveBalance: 32000000000},
	}
	st, err := statenative.InitializeFromProtoUnsafePhase0(&ethpb.BeaconState{
		Slot:       primitives.Slot(10),
		Validators: vals,
	})
	require.NoError(t, err)

	root := [32]byte{1, 2, 3}

	iterations := 1000
	var wg sync.WaitGroup
	wg.Add(2)

	corruptedDetected := false
	var detectionLock sync.Mutex

	// Thread 1: Continuous Saving
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			_ = s.SaveState(ctx, st, root)
		}
	}()

	// Thread 2: Continuous Copying/Reading
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			// We access the validators via ToProtoUnsafe which simulates the RPC access pattern.
			pb := st.ToProtoUnsafe()
			var num int
			switch pt := pb.(type) {
			case *ethpb.BeaconState:
				num = len(pt.Validators)
			}

			if num == 0 {
				detectionLock.Lock()
				corruptedDetected = true
				detectionLock.Unlock()
			}
		}
	}()

	wg.Wait()

	if corruptedDetected {
		t.Log("CRITICAL: State corruption detected! Concurrent reader saw 0 validators during SaveState.")
	} else {
		t.Log("No corruption detected in this run.")
	}
}
