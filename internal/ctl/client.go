package ctl

import (
	"context"
	"errors"
	"time"

	"github.com/flipkart-incubator/dkv/pkg/serverpb"
	"google.golang.org/grpc"
)

// A DKVClient instance is used to communicate with various DKV services
// over GRPC. It is an adapter to the underlying GRPC clients that
// exposes a simpler API to its users without having to deal with timeouts,
// contexts and other GRPC semantics.
type DKVClient struct {
	cliConn    *grpc.ClientConn
	dkvCli     serverpb.DKVClient
	dkvReplCli serverpb.DKVReplicationClient
	dkvBRCli   serverpb.DKVBackupRestoreClient
	dkvClusCli serverpb.DKVClusterClient
}

// TODO: Should these be paramterised ?
const (
	ReadBufSize  = 10 << 30
	WriteBufSize = 10 << 30
	Timeout      = 5 * time.Second
)

// NewInSecureDKVClient creates an insecure GRPC client against the
// given DKV service address.
func NewInSecureDKVClient(svcAddr string) (*DKVClient, error) {
	var dkvClnt *DKVClient
	conn, err := grpc.Dial(svcAddr, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithReadBufferSize(ReadBufSize), grpc.WithWriteBufferSize(WriteBufSize))
	if err == nil {
		dkvCli := serverpb.NewDKVClient(conn)
		dkvReplCli := serverpb.NewDKVReplicationClient(conn)
		dkvBRCli := serverpb.NewDKVBackupRestoreClient(conn)
		dkvClusCli := serverpb.NewDKVClusterClient(conn)
		dkvClnt = &DKVClient{conn, dkvCli, dkvReplCli, dkvBRCli, dkvClusCli}
	}
	return dkvClnt, err
}

// Put takes the key and value as byte arrays and invokes the
// GRPC Put method. This is a convenience wrapper.
func (dkvClnt *DKVClient) Put(key []byte, value []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()
	putReq := &serverpb.PutRequest{Key: key, Value: value}
	res, err := dkvClnt.dkvCli.Put(ctx, putReq)
	var status *serverpb.Status
	if res != nil {
		status = res.Status
	}
	return errorFromStatus(status, err)
}

// Get takes the key as byte array and invokes the
// GRPC Get method. This is a convenience wrapper.
func (dkvClnt *DKVClient) Get(key []byte) (*serverpb.GetResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()
	getReq := &serverpb.GetRequest{Key: key}
	return dkvClnt.dkvCli.Get(ctx, getReq)
}

// MultiGet takes the keys as byte arrays and invokes the
// GRPC MultiGet method. This is a convenience wrapper.
func (dkvClnt *DKVClient) MultiGet(keys ...[]byte) ([][]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()
	multiGetReq := &serverpb.MultiGetRequest{Keys: keys}
	res, err := dkvClnt.dkvCli.MultiGet(ctx, multiGetReq)
	return res.Values, err
}

// GetChanges retrieves changes since the given change number
// using the underlying GRPC GetChanges method. One can limit the
// number of changes retrieved using the maxNumChanges parameter.
// This is a convenience wrapper.
func (dkvClnt *DKVClient) GetChanges(fromChangeNum uint64, maxNumChanges uint32) (*serverpb.GetChangesResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()
	getChngsReq := &serverpb.GetChangesRequest{FromChangeNumber: fromChangeNum, MaxNumberOfChanges: maxNumChanges}
	return dkvClnt.dkvReplCli.GetChanges(ctx, getChngsReq)
}

// Backup backs up the entire keyspace into the given filesystem
// location using the underlying GRPC Backup method. This is a
// convenience wrapper.
func (dkvClnt *DKVClient) Backup(path string) error {
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()
	backupReq := &serverpb.BackupRequest{BackupPath: path}
	res, err := dkvClnt.dkvBRCli.Backup(ctx, backupReq)
	return errorFromStatus(res, err)
}

// Restore restores the entire keyspace from the given filesystem
// location using the underlying GRPC Restore method. This is a
// convenience wrapper.
func (dkvClnt *DKVClient) Restore(path string) error {
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()
	restoreReq := &serverpb.RestoreRequest{RestorePath: path}
	res, err := dkvClnt.dkvBRCli.Restore(ctx, restoreReq)
	return errorFromStatus(res, err)
}

// AddNode adds the node with the given identifier and Nexus URL to
// the Nexus cluster of which the current node is a member of.
func (dkvClnt *DKVClient) AddNode(nodeID uint32, nodeURL string) error {
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()
	addNodeReq := &serverpb.AddNodeRequest{NodeId: nodeID, NodeUrl: nodeURL}
	res, err := dkvClnt.dkvClusCli.AddNode(ctx, addNodeReq)
	return errorFromStatus(res, err)
}

// RemoveNode removes the node with the given identifier from the
// Nexus cluster of which the current node is a member of.
func (dkvClnt *DKVClient) RemoveNode(nodeID uint32) error {
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()
	remNodeReq := &serverpb.RemoveNodeRequest{NodeId: nodeID}
	res, err := dkvClnt.dkvClusCli.RemoveNode(ctx, remNodeReq)
	return errorFromStatus(res, err)
}

// Close closes the underlying GRPC client connection to DKV service
func (dkvClnt *DKVClient) Close() error {
	return dkvClnt.cliConn.Close()
}

func errorFromStatus(res *serverpb.Status, err error) error {
	switch {
	case err != nil:
		return err
	case res != nil && res.Code != 0:
		return errors.New(res.Message)
	default:
		return nil
	}
}
