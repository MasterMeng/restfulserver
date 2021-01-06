package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	cb "github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/retry"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/msp"
	packager "github.com/hyperledger/fabric-sdk-go/pkg/fab/ccpackager/gopackager"

	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/ledger"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
)

// CreateChannel 创建通道
func CreateChannel(chID, chCfgPath, targetOrder string, signIdentity []msp.SigningIdentity, peerResMgmt *resmgmt.Client) error {

	// 判断是否已经创建
	created, err := IsCreatedChannel(chID, peerResMgmt, targetOrder)
	if err != nil {
		return err
	}
	if created {
		log.Println(fmt.Sprintf("channel %s has already created ", chID))
		return nil
	}

	//var lastConfigBlockNum uint64

	// 构建请求
	req := resmgmt.SaveChannelRequest{
		ChannelID:         chID,
		ChannelConfigPath: chCfgPath,
		SigningIdentities: signIdentity,
	}

	// 发送创建请求
	_, err = peerResMgmt.SaveChannel(req, resmgmt.WithOrdererEndpoint(targetOrder))
	if err != nil {
		return err
	}

	return nil

}

// UpdateAnchorPeer 更新锚节点
func UpdateAnchorPeer(chID, chCfgPath, targetOrder string, signIdentity []msp.SigningIdentity, peerResMgmt *resmgmt.Client) error {

	// 构建请求
	req := resmgmt.SaveChannelRequest{
		ChannelID:         chID,
		ChannelConfigPath: chCfgPath,
		SigningIdentities: signIdentity,
	}

	// 发送创建请求
	_, err := peerResMgmt.SaveChannel(req, resmgmt.WithOrdererEndpoint(targetOrder))
	if err != nil {
		return err
	}

	return nil

}

// JoinChannel 加入通道
// targetPeers与peerOrgResMgmt需是同一个org下的
// 指定身份：signPayload中选择fabsdk.context中的用户
func JoinChannel(chID, targetOrder string, targetPeers []string, peerOrgResMgmt *resmgmt.Client) error {
	var err error
	realTargets := make([]string, 0)
	// 判断是否已经加入过
	for _, target := range targetPeers {
		joined, err := IsJoinedChannel(chID, peerOrgResMgmt, target)
		if err != nil {
			return err
		}
		if joined {
			log.Println(fmt.Sprintf("%s has already joined channel %s", target, chID))
			return nil
		}
		realTargets = append(realTargets, target)

	}

	// 加入通道
	if len(realTargets) > 0 {
		err = peerOrgResMgmt.JoinChannel(
			chID,
			resmgmt.WithRetry(retry.DefaultResMgmtOpts),
			resmgmt.WithOrdererEndpoint(targetOrder),
			resmgmt.WithTargetEndpoints(realTargets...),
		)
		if err != nil {
			return err
		}

	}

	return nil
}

// InstallCC 安装chaincode
// targetPeers与peerOrgResMgmt需是同一个org下的
// cc install是针对peer的，每个peer都得执行一遍
// 指定身份：signProposal中选择fabsdk.context中的用户
func InstallCC(ccID, ccVersion, ccPath string, targetPeers []string, peerOrgResMgmt *resmgmt.Client) error {

	realTargets := make([]string, 0)
	// 判断是否已经install
	for _, target := range targetPeers {
		installed, err := IsCCInstalled(peerOrgResMgmt, ccID, ccVersion, target)
		if err != nil {
			return err
		}
		if installed {
			log.Println(fmt.Sprintf("%s has already installed cc %s:%s", target, ccID, ccVersion))
			return nil
		}
		realTargets = append(realTargets, target)

	}

	if len(realTargets) > 0 {
		ccPkg, err := packager.NewCCPackage(ccPath, "")
		if err != nil {
			return err
		}

		// 构建请求
		pwd, _ := os.Getwd()
		ccAbsPath := path.Join(pwd, ccPath)
		req := resmgmt.InstallCCRequest{
			Name:    ccID,
			Path:    ccAbsPath,
			Version: ccVersion,
			Package: ccPkg,
		}

		// install cc
		_, err = peerOrgResMgmt.InstallCC(req, resmgmt.WithRetry(retry.DefaultResMgmtOpts), resmgmt.WithTargetEndpoints(realTargets...))
		if err != nil {
			return err
		}

	}

	return nil
}

// InstantiateCC 实例化chaincode
// targetPeers与peerOrgResMgmt需是同一个org下的
// cc instantiate是针对channel的，只需执行一遍
// 指定身份：signProposal中选择fabsdk.context中的用户
func InstantiateCC(chID, ccID, ccVersion, ccPath string, ccPolicy *cb.SignaturePolicyEnvelope, targetPeers []string, peerOrgResMgmt *resmgmt.Client) error {

	// 判断是否已经instantiated
	var code string
	var err error
	for _, target := range targetPeers {
		code, err = InstantiateOrUpdate(peerOrgResMgmt, chID, ccID, ccVersion, target)
		if err != nil {
			return err
		}
		break // 针对channel而言，instantiated判断执行一遍即可
	}

	pwd, _ := os.Getwd()
	ccAbsPath := path.Join(pwd, ccPath)
	switch code {
	case "0":
		log.Println(fmt.Sprintf("channel %s has already instantiate cc %s:%s ", chID, ccID, ccVersion))
	case "1":
		// 构建请求
		req := resmgmt.InstantiateCCRequest{
			Name:    ccID,
			Path:    ccAbsPath,
			Version: ccVersion,
			Policy:  ccPolicy,
		}
		// instantiate cc
		_, err := peerOrgResMgmt.InstantiateCC(chID, req, resmgmt.WithRetry(retry.DefaultResMgmtOpts), resmgmt.WithTargetEndpoints(targetPeers...))
		if err != nil {
			return err
		}
		//log.Println(fmt.Sprintf("%v in channel %s instantiate cc %s:%s success, txId is %s", targetPeers, chID, ccID, ccVersion, resp.TransactionID))
	case "2":
		req := resmgmt.UpgradeCCRequest{
			Name:    ccID,
			Path:    ccAbsPath,
			Version: ccVersion,
			Policy:  ccPolicy,
		}
		// upgrade cc
		_, err := peerOrgResMgmt.UpgradeCC(chID, req, resmgmt.WithRetry(retry.DefaultResMgmtOpts), resmgmt.WithTargetEndpoints(targetPeers...))
		if err != nil {
			return err
		}
		//log.Println(fmt.Sprintf("%v in channel %s upgrade cc %s:%s success, txId is %s", targetPeers, chID, ccID, ccVersion, resp.TransactionID))
	}

	return nil
}

// InvokeCC 调用chaincode
// 若不指定targetpeer，则从配置文件中取
// 指定身份：signProposal中选择fabsdk.context中的用户
func InvokeCC(chClient *channel.Client, req channel.Request) (*channel.Response, error) {

	response, err := chClient.Execute(req, channel.WithRetry(retry.DefaultChannelOpts))
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// QueryCC 查询
// 指定身份：signProposal中选择fabsdk.context中的用户
func QueryCC(chClient *channel.Client, req channel.Request) (*channel.Response, error) {

	response, err := chClient.Query(req, channel.WithRetry(retry.DefaultChannelOpts))
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// QueryBlockByNum 通过块高查询区块
func QueryBlockByNum(ldgCLient *ledger.Client, num uint64, targetPeer string) (*cb.Block, error) {

	block, err := ldgCLient.QueryBlock(num, ledger.WithTargetEndpoints(targetPeer))
	if err != nil {
		return nil, err
	}

	return block, nil

}

// QueryBlockByTxID 通过TxID查询区块
func QueryBlockByTxID(ldgCLient *ledger.Client, txID string, targetPeer string) (*cb.Block, error) {

	ID := fab.TransactionID(txID)

	block, err := ldgCLient.QueryBlockByTxID(ID, ledger.WithTargetEndpoints(targetPeer))
	if err != nil {
		return nil, err
	}

	return block, nil

}

// IsCreatedChannel 判读通道是否已创建
func IsCreatedChannel(channelID string, resMgmtClient *resmgmt.Client, targetOrder string) (bool, error) {

	chCfg, err := resMgmtClient.QueryConfigFromOrderer(channelID, resmgmt.WithOrdererEndpoint(targetOrder))
	if err != nil {
		if strings.Contains(err.Error(), "NOT_FOUND") {
			return false, nil
		}
		return false, err
	}

	if chCfg.ID() == channelID {
		return true, nil
	}

	return false, nil
}

// IsJoinedChannel 通道是否已加入
// 只能一个一个peer的查询
func IsJoinedChannel(channelID string, resMgmtClient *resmgmt.Client, targetPeer string) (bool, error) {

	resp, err := resMgmtClient.QueryChannels(resmgmt.WithTargetEndpoints(targetPeer))
	if err != nil {
		return false, err
	}
	for _, chInfo := range resp.Channels {
		if chInfo.ChannelId == channelID {
			return true, nil
		}
	}
	return false, nil
}

// IsCCInstalled chaincode是否已安装
// 只能一个一个peer的查询
func IsCCInstalled(resMgmt *resmgmt.Client, ccName, ccVersion string, targetPeer string) (bool, error) {

	resp, err := resMgmt.QueryInstalledChaincodes(resmgmt.WithRetry(retry.DefaultResMgmtOpts), resmgmt.WithTargetEndpoints(targetPeer))
	if err != nil {
		return false, err
	}
	found := false
	for _, ccInfo := range resp.Chaincodes {
		if ccInfo.Name == ccName && ccInfo.Version == ccVersion {
			found = true
			break
		}
	}

	return found, nil
}

// IsCCInstantiated chaincode是否已实例化
// 只能一个一个peer的查询
func IsCCInstantiated(resMgmt *resmgmt.Client, channelID, ccName, ccVersion string, targetPeer string) (bool, error) {

	resp, err := resMgmt.QueryInstantiatedChaincodes(channelID, resmgmt.WithRetry(retry.DefaultResMgmtOpts), resmgmt.WithTargetEndpoints(targetPeer))
	if err != nil {
		return false, err
	}
	instantiated := false
	for _, ccInfo := range resp.Chaincodes {
		if ccInfo.Name == ccName && ccInfo.Version == ccVersion {
			instantiated = true
			break
		}
	}

	return instantiated, nil
}

// InstantiateOrUpdate 实例化或更新
// 只能一个一个peer的查询: 0，已instantiated；1，需要instantiate；2，需要update
func InstantiateOrUpdate(resMgmt *resmgmt.Client, channelID, ccName, ccVersion string, targetPeer string) (string, error) {
	if resMgmt == nil || channelID == "" || ccName == "" || ccVersion == "" || targetPeer == "" {
		return "", errors.New("InstantiateOrUpdate failed. some arg is null. ")
	}

	resp, err := resMgmt.QueryInstantiatedChaincodes(channelID, resmgmt.WithRetry(retry.DefaultResMgmtOpts), resmgmt.WithTargetEndpoints(targetPeer))
	if err != nil {
		return "", err
	}
	instantiated := false
	upgrade := false
	for _, ccInfo := range resp.Chaincodes {
		if ccInfo.Name == ccName && ccInfo.Version == ccVersion {
			instantiated = true
			break
		}
	}
	for _, ccInfo := range resp.Chaincodes {
		if ccInfo.Name == ccName && ccInfo.Version != ccVersion {
			upgrade = true
			break
		}
	}

	if instantiated {
		return "0", nil
	}
	if !instantiated && !upgrade {
		return "1", nil
	}
	if !instantiated && upgrade {
		return "2", nil
	}

	return "", nil
}
