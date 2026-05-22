#!/bin/bash
set -e

echo "=== Prysm Vulnerability PoC ==="

echo ""
echo "[1] Testing Reference Leakage in ToProtoUnsafe..."
# This test proves that ToProtoUnsafe returns direct references to internal state slices.
# Mutating the returned Protobuf object directly mutates the live BeaconState.
go test -v ./beacon-chain/state/state-native/ -run TestToProtoUnsafe_Mutation

echo ""
echo "[2] Testing Data Race in Background SaveState..."
# This test simulates the background goroutine in SaveFinalizedState.
# It demonstrates that the state can be corrupted during concurrent access.
# Due to the non-deterministic nature of races, this test may require multiple runs or a longer duration to trigger on all environments.
go test -v ./beacon-chain/db/kv/ -run TestRace_SaveState_ConcurrentRead

echo ""
echo "=== PoC Execution Complete ==="
