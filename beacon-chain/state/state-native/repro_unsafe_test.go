package state_native

import (
	"sync"
	"testing"

	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/stretchr/testify/require"
)

func TestToProtoUnsafe_Mutation(t *testing.T) {
	// 1. Create a Gloas state with one builder
	builder := &ethpb.Builder{
		Pubkey:  make([]byte, 48),
		Balance: 32000000000,
	}
	st, err := InitializeFromProtoGloas(&ethpb.BeaconStateGloas{
		Builders: []*ethpb.Builder{builder},
	})
	if err != nil {
		t.Fatalf("Failed to initialize state: %v", err)
	}

	// 2. Get the "unsafe" proto
	protoObj := st.ToProtoUnsafe().(*ethpb.BeaconStateGloas)

	// 3. Mutate the proto
	if len(protoObj.Builders) == 0 {
		t.Fatal("Expected at least one builder")
	}
	originalBalance := protoObj.Builders[0].Balance
	protoObj.Builders[0].Balance = 999999

	// 4. Check if the internal state changed
	internalBuilders, err := st.Builders()
	if err != nil {
		t.Fatalf("Failed to get internal builders: %v", err)
	}
	if internalBuilders[0].Balance == 999999 {
		t.Logf("VULNERABILITY CONFIRMED: Mutating ToProtoUnsafe result changed internal state! (Balance: %d -> %d)", originalBalance, internalBuilders[0].Balance)
	} else {
		t.Errorf("Internal state NOT changed. Balance is still %d", internalBuilders[0].Balance)
	}
}

func TestToProtoUnsafe_Race(t *testing.T) {
	vals := []*ethpb.Validator{
		{EffectiveBalance: 32000000000},
	}
	st, err := InitializeFromProtoUnsafePhase0(&ethpb.BeaconState{
		Slot:       primitives.Slot(10),
		Validators: vals,
	})
	require.NoError(t, err)

	iterations := 1000
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			_ = st.ToProtoUnsafe()
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			_ = st.Copy()
		}
	}()

	wg.Wait()
}
