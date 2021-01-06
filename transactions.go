package main

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/ledger/rwset"
	"github.com/hyperledger/fabric-protos-go/ledger/rwset/kvrwset"
	"github.com/hyperledger/fabric-protos-go/msp"
	"github.com/hyperledger/fabric-protos-go/peer"
)

type cachedIdentity struct {
	mspID string
	cert  *x509.Certificate
}

func getIdentity(serilizedIdentity []byte) (*cachedIdentity, error) {
	var err error

	sid := &msp.SerializedIdentity{}
	err = proto.Unmarshal(serilizedIdentity, sid)
	if err != nil {
		return nil, err
	}

	var cert *x509.Certificate
	cert, err = decodeX509Pem(sid.IdBytes)
	if err != nil {
		return nil, err
	}
	// log.Println("Unmarshal SerializedIdentity:", sid)

	return &cachedIdentity{
		mspID: sid.Mspid,
		cert:  cert,
	}, nil

}
func decodeX509Pem(certPem []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(certPem)
	if block == nil {
		return nil, fmt.Errorf("bad cert")
	}

	return x509.ParseCertificate(block.Bytes)
}

type Endorser struct {
	MSP  string `json:"msp"`
	Name string `json:"name"`
}

// TransactionDetail is the detail of transaction, but not contains RW set
type TransactionDetail struct {
	ChannelName      string `json:"channel_name"`
	ID               string `json:"id"`
	Type             string `json:"type"`
	Creator          string `json:"creator"`
	CreatorMSP       string `json:"creator_msp"`
	ChaincodeName    string `json:"chaincode_name"`
	ValidationResult string `json:"validation_result"`
	BlockNumber      uint64 `json:"block_number"`
	// TxNumber         int         `json:"tx_number"`
	CreatedAt time.Time   `json:"created_at"`
	Endorsers []*Endorser `json:"endorsers"`
	Value     *RawValue   `json:"raw"`
}

// RawValue define the raw value stored into blockchain
type RawValue struct {
	// Type        string   `json:"type"`
	ChaincodeID *peer.ChaincodeID `json:"chaincodeid"`
	Input       []string          `json:"input"`
	IDs         []string          `json:"ids"`
	// Writes      []*kvrwset.KVWrite `json:"writes"`
	// Reads       []*kvrwset.KVRead  `json:"reads"`
}

func convertEnvelopeToTXDetail(txFlag int32, env *common.Envelope) (*TransactionDetail, error) {
	// log.Println(env)
	payload, err := GetPayload(env)
	if err != nil {
		log.Printf("Unexpected error from unmarshal envelope: %v", err)
		return nil, fmt.Errorf("Unexpected error from unmarshal envelope: %v", err)
	}
	// log.Println("After GetPayload:",payload)

	chdr, err := UnmarshalChannelHeader(payload.Header.ChannelHeader)
	if err != nil {
		log.Printf("Unexpected error from unmarshal channel header: %v", err)
		return nil, fmt.Errorf("Unexpected error from unmarshal channel header: %v", err)
	}
	// log.Println("After UnmarshalChannelHeader:",chdr)

	shdr, err := GetSignatureHeader(payload.Header.SignatureHeader)
	if err != nil {
		log.Printf("Unexpected error from unmarshal signature header: %v", err)
		return nil, fmt.Errorf("Unexpected error from unmarshal signature header: %v", err)
	}
	// log.Println("After GetSignatureHeader:",shdr)

	identity, err := getIdentity(shdr.Creator)
	if err != nil {
		return nil, err
	}

	tx := &TransactionDetail{
		ID:               chdr.TxId,
		Type:             common.HeaderType_name[chdr.Type],
		CreatorMSP:       identity.mspID,
		ValidationResult: peer.TxValidationCode_name[txFlag],
		CreatedAt:        time.Unix(chdr.Timestamp.Seconds, int64(chdr.Timestamp.Nanos)),
	}

	if identity.cert != nil {
		tx.Creator = identity.cert.Subject.CommonName
	}

	hdrExt, err := GetChaincodeHeaderExtension(payload.Header)
	if err != nil {
		log.Printf("GetChaincodeHeaderExtension failed: %v", err)
		return nil, fmt.Errorf("GetChaincodeHeaderExtension failed: %v", err)
	}

	if hdrExt.ChaincodeId != nil {
		tx.ChaincodeName = hdrExt.ChaincodeId.Name
	}

	//fetch the endorsers from the envelope
	chaincodeProposalPayload, endorsements, chaincodeAction, err := parseChaincodeEnvelope(env)

	if err != nil {
		log.Printf("parseChaincodeEnvelope failed: %v", err)
		return nil, fmt.Errorf("parseChaincodeEnvelope failed: %v", err)
	}
	distinctEndorser := map[string]bool{}
	for _, e := range endorsements {
		identity, err := getIdentity(e.Endorser)
		if err != nil {
			return nil, err
		}
		userName := ""
		if identity.cert != nil {
			userName = identity.cert.Subject.CommonName
		}

		if _, ok := distinctEndorser[identity.mspID+":"+userName]; !ok {
			tx.Endorsers = append(tx.Endorsers, &Endorser{MSP: identity.mspID, Name: userName})
		}

	}

	// log.Println("Chaincode Input:", chaincodeProposalPayload.Input)
	cis := &peer.ChaincodeInvocationSpec{}
	err = proto.Unmarshal(chaincodeProposalPayload.Input, cis)
	if err != nil {
		return nil, err
	}
	tx.Value = parseChaincodeInvocationSpec(cis)
	keys, err := parseChaincodeAction(chaincodeAction, tx.ChaincodeName)
	if err != nil {
		return nil, err
	}
	tx.Value.IDs = keys
	return tx, nil
}

func parseChaincodeEnvelope(env *common.Envelope) (*peer.ChaincodeProposalPayload, []*peer.Endorsement, *peer.ChaincodeAction, error) {
	payl, err := GetPayload(env)
	if err != nil {
		log.Println(err.Error())
		return nil, nil, nil, err
	}

	tx, err := GetTransaction(payl.Data)
	if err != nil {
		log.Println(err.Error())
		return nil, nil, nil, err
	}

	if len(tx.Actions) == 0 {
		log.Println("At least one TransactionAction is required")
		return nil, nil, nil, fmt.Errorf("At least one TransactionAction is required")
	}

	actionPayload, chaincodeAction, err := GetPayloads(tx.Actions[0])
	if err != nil {
		log.Println(err.Error())
		return nil, nil, nil, err
	}

	chaincodeProposalPayload, err := GetChaincodeProposalPayload(actionPayload.ChaincodeProposalPayload)
	if err != nil {
		log.Println(err.Error())
		return nil, nil, nil, err
	}

	return chaincodeProposalPayload, actionPayload.Action.Endorsements, chaincodeAction, nil
}

func parseChaincodeInvocationSpec(cis *peer.ChaincodeInvocationSpec) *RawValue {
	raw := &RawValue{}

	// raw.Type = peer.ChaincodeSpec_Type_name[int32(cis.GetChaincodeSpec().Type)]
	raw.ChaincodeID = cis.GetChaincodeSpec().GetChaincodeId()

	args := cis.GetChaincodeSpec().GetInput().GetArgs()
	intput := make([]string, len(args))
	for i := 0; i < len(args); i++ {
		intput[i] = string(args[i])
	}
	raw.Input = intput

	return raw
}

func parseChaincodeAction(action *peer.ChaincodeAction, chaincodename string) ([]string, error) {
	resultBytes := action.GetResults()

	txRWSet := &rwset.TxReadWriteSet{}
	err := proto.Unmarshal(resultBytes, txRWSet)
	if err != nil {
		return nil, err
	}

	// log.Println("TxReadWriteSet:", txRWSet)

	var kvRwSetByte []byte

	for _, rwset := range txRWSet.NsRwset {
		if rwset.Namespace == chaincodename {
			kvRwSetByte = rwset.Rwset
		}
	}
	kvRWSet := &kvrwset.KVRWSet{}
	err = proto.Unmarshal(kvRwSetByte, kvRWSet)
	if err != nil {
		return nil, err
	}

	// log.Println(kvRWSet)
	// log.Println("Reads:", kvRWSet.Reads)

	keys := make([]string, 0)
	for _, write := range kvRWSet.Writes {
		keys = append(keys, write.Key)
	}
	return keys, nil
}

// 获取完整的读写集
func getRWSets(action *peer.ChaincodeAction) ([]*kvrwset.KVWrite, []*kvrwset.KVRead, error) {
	resultBytes := action.GetResults()

	txRWSet := &rwset.TxReadWriteSet{}
	err := proto.Unmarshal(resultBytes, txRWSet)
	if err != nil {
		return nil, nil, err
	}

	kvRwSetByte := txRWSet.NsRwset[0].Rwset
	kvRWSet := &kvrwset.KVRWSet{}
	err = proto.Unmarshal(kvRwSetByte, kvRWSet)
	if err != nil {
		return nil, nil, err
	}
	return kvRWSet.Writes, kvRWSet.Reads, nil
}
